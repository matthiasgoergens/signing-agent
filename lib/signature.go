package lib

import (
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/pkg/errors"

	"gitlab.qredo.com/custody-engine/automated-approver/defs"

	"gitlab.qredo.com/custody-engine/automated-approver/api"
	"gitlab.qredo.com/custody-engine/automated-approver/util"
)

// Sign signs a payload
func (h *autoApprover) Sign(agentID, messageHex string) (*api.SignResponse, error) {
	msg, err := hex.DecodeString(messageHex)
	if err != nil {
		return nil, defs.ErrBadRequest().WithDetail("invalid_message_hex").Wrap(err)
	}

	if len(msg) > 64 {
		return nil, defs.ErrBadRequest().WithDetail("invalid_message_hex_size").Wrap(fmt.Errorf("invalid message hex size %s ", strconv.Itoa(len(msg))))
	}

	agent := h.store.GetAgent(agentID)
	if agent == nil {
		return nil, defs.ErrNotFound().WithDetail("core_client_seed").Wrap(fmt.Errorf("get lib client seed from secrets store %s", agentID))
	}

	signature, err := util.BLSSign(agent.BLSSeed, msg)
	if err != nil {
		return nil, errors.Wrapf(err, "sign message for lib client %s", agentID)
	}

	return &api.SignResponse{
		SignatureHex: hex.EncodeToString(signature),
		SignerID:     agentID,
	}, nil
}

// Verify verifies a signature
func (h *autoApprover) Verify(req *api.VerifyRequest) error {
	msg, err := hex.DecodeString(req.MessageHashHex)
	if err != nil {
		return defs.ErrBadRequest().WithDetail("invalid_message_hex").Wrap(err)
	}
	if len(msg) > 64 {
		return defs.ErrBadRequest().WithDetail("invalid_message_hex_size").Wrap(fmt.Errorf("invalid message hex size " + strconv.Itoa(len(msg))))
	}

	sig, err := hex.DecodeString(req.SignatureHex)
	if err != nil {
		return defs.ErrBadRequest().WithDetail("invalid_signature_hex").Wrap(err)
	}

	agent := h.store.GetAgent(req.SignerID)
	if agent == nil {
		return defs.ErrNotFound().WithDetail("signer_not_found").Wrap(fmt.Errorf("get signer %s", req.SignerID))
	}

	if err := util.BLSVerify(agent.BLSSeed, msg, sig); err != nil {
		return defs.ErrForbidden().WithDetail("invalid_signature").Wrap(err)
	}

	return nil
}
