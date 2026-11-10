package target

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestLocalTrialRecordCannotBecomePublicationByCompletingDraft(t *testing.T) {
	// A marker received from durable storage must survive decoding and make
	// even a malicious ready flag ineligible for publication.
	var value TargetRecord
	require.NoError(t, json.Unmarshal([]byte(`{"input":{"localTrial":{"contractVersion":1,"trialId":"test"}},"merchant":{"site":"shein-us"},"result":{"ready_for_upload":true}}`), &value))
	require.False(t, PublicationReady(value))
	require.False(t, ValidPlatformRecord(value), "incomplete/unknown provenance must fail closed")
}
