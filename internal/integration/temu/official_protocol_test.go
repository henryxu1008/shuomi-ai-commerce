package temu

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
)

func TestSignatureUsesOuterASCIISortAndExactNestedWireValues(t *testing.T) {
	c := identityClient(t, storecenter.PartitionCN, nil)
	body, err := c.encodeRequest("bg.glo.goods.add", "test-token", map[string]json.RawMessage{
		"items": json.RawMessage(`[{"z":"中文<&>","a":true}]`),
	})
	require.NoError(t, err)
	var wire map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &wire))
	require.Equal(t, `[{"z":"中文<&>","a":true}]`, string(wire["items"]))
	// Independently computed with .NET MD5 from the documented canonical text.
	// These are synthetic public fixtures, never real platform credentials.
	require.Equal(t, `"297BF02ABF995EE7D4A532370D74DBEE"`, string(wire["sign"]))
	require.JSONEq(t, `"1760000000"`, string(wire["timestamp"]))
	require.NotContains(t, string(body), "test-secret")
}

func TestEncoderRejectsReservedFieldsMalformedJSONAndOversizedBodies(t *testing.T) {
	c := identityClient(t, storecenter.PartitionCN, nil)
	for _, fields := range []map[string]json.RawMessage{
		{"type": json.RawMessage(`"different.api"`)},
		{"app_key": json.RawMessage(`"another-app"`)},
		{"access_token": json.RawMessage(`"another-token"`)},
		{"sign": json.RawMessage(`"forged"`)},
		{"timestamp": json.RawMessage(`"0"`)},
		{"data_type": json.RawMessage(`"XML"`)},
		{"version": json.RawMessage(`"other-version"`)},
		{"broken": json.RawMessage(`{"unterminated"`)},
		{"items": json.RawMessage(`"` + strings.Repeat("x", 2*1024*1024) + `"`)},
	} {
		_, err := c.encodeRequest("bg.glo.goods.add", "test-token", fields)
		require.ErrorIs(t, err, storecenter.ErrOfficialConnectionUnavailable)
	}
}
