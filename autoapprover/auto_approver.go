// Package autoapprover provides a mechanism to receive action information as bytes.
// The action data is analyzed and if it meets the requirements, the action is approved.
// It supports approval retrying based on defined intervals

package autoapprover

import (
	"encoding/json"

	"go.uber.org/zap"

	"signing-agent/config"
	"signing-agent/hub"
	"signing-agent/lib"
)

type AutoApprover struct {
	hub.FeedClient
	log                  *zap.SugaredLogger
	cfgAutoApproval      *config.AutoApprove
	core                 lib.SigningAgentClient
	syncronizer          ActionSyncronizer
	lastError            error
	loadBalancingEnabled bool
}

// NewAutoApprover returns a new *AutoApprover instance initialized with the provided parameters
// The AutoApprover has an internal FeedClient which means it will be stopped when the service stops
// or the Feed channel is closed on the sender side
func NewAutoApprover(core lib.SigningAgentClient, log *zap.SugaredLogger, config *config.Config, syncronizer ActionSyncronizer) *AutoApprover {
	return &AutoApprover{
		FeedClient:           hub.NewFeedClient(true),
		log:                  log,
		cfgAutoApproval:      &config.AutoApprove,
		core:                 core,
		syncronizer:          syncronizer,
		loadBalancingEnabled: config.LoadBalancing.Enable,
	}
}

// Listen is constantly listening for messages on the Feed channel.
// The Feed channel is always closed by the sender. Then this happens, the AutoApprover stops
func (a *AutoApprover) Listen() {
	for {
		if message, ok := <-a.Feed; !ok {
			//channel was closed by the sender
			a.log.Info("AutoApproval: stopped")
			return
		} else {
			a.handleMessage(message)
		}
	}
}

func (a *AutoApprover) handleMessage(message []byte) {
	var action actionInfo
	if err := json.Unmarshal(message, &action); err == nil {
		if action.IsNotExpired() {
			if a.shouldHandleAction(action.ID) {
				go a.handleAction(&action)
			}
		} else {
			a.log.Infof("AutoApproval: action [%v] has expired", action.ID)
		}
	} else {
		a.log.Errorf("AutoApproval: error [%v] while unmarshaling the message [%v]", err, string(message))
		a.lastError = err
	}
}

func (a *AutoApprover) shouldHandleAction(actionId string) bool {
	if a.loadBalancingEnabled {
		//check if the action was already picked up by another signing agent
		if !a.syncronizer.ShouldHandleAction(actionId) {
			a.log.Debugf("AutoApproval: action [%v] was already approved!", actionId)
			return false
		}
	}

	return true
}

func (a *AutoApprover) handleAction(action *actionInfo) {
	if a.loadBalancingEnabled {
		if err := a.syncronizer.AcquireLock(); err != nil {
			a.log.Warnf("AutoApproval, mutex lock: %v action [%v]", err, action.ID)
			return
		}
		defer func() {
			if err := a.syncronizer.Release(action.ID); err != nil {
				a.log.Warnf("AutoApproval, mutex unlock: %v action [%v]", err, action.ID)
			}
		}()
	}

	a.approveAction(action.ID, action.AgentID)
}

func (a *AutoApprover) approveAction(actionId, agentId string) {
	timer := newRetryTimer(a.cfgAutoApproval.RetryInterval, a.cfgAutoApproval.RetryIntervalMax)
	for {
		if err := a.core.ActionApprove(actionId); err == nil {
			a.log.Infof("AutoApproval: action [%v] approved automatically", actionId)
			return
		} else {
			a.log.Errorf("AutoApproval: approval failed for [actionID:%v]. Error msg: %v", agentId, actionId, err)

			if timer.isTimeOut() {
				a.log.Warnf("AutoApproval: auto action approve failed [actionID:%v]", actionId)
				return
			}

			a.log.Warnf("AutoApproval: auto approve action is repeated [actionID:%v] ", actionId)
			timer.retry()
		}
	}
}
