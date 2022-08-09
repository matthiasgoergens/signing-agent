package lib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeaders(t *testing.T) {

	t.Run(
		"GenTimestamp",
		func(t *testing.T) {
			var req = &Request{}
			assert.Empty(t, req.Timestamp)
			GenTimestamp(req)
			assert.NotEmpty(t, req.Timestamp)
		})

	t.Run(
		"GetHttpHeaders",
		func(t *testing.T) {
			var req = &Request{}
			req.ApiKey = "test"
			req.Signature = "test"
			GenTimestamp(req)
			headers := GetHttpHeaders(req)
			assert.Equal(t, req.ApiKey, headers.Get("x-api-key"))
			assert.Equal(t, req.Signature, headers.Get("x-sign"))
			assert.Equal(t, req.Timestamp, headers.Get("x-timestamp"))
		})

	t.Run(
		"SignRequest",
		func(t *testing.T) {
			// setup test private key
			generatePrivateKey(t, TestDataPrivatePEMFilePath)
			defer func() {
				os.Remove(TestDataPrivatePEMFilePath)
			}()
			// setup request
			var req = &Request{
				Uri:  "https://test-domain/",
				Body: []byte(`{"name": "Test Data"}`),
			}
			GenTimestamp(req)
			LoadRSAKey(req, TestDataPrivatePEMFilePath)
			assert.NotEmpty(t, req.RsaKey)
			// make the signature
			assert.Empty(t, req.Signature)
			err := SignRequest(req)
			assert.NoError(t, err)
			assert.NotEmpty(t, req.Signature)

		})

}
