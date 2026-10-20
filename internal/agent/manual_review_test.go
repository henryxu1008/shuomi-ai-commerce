package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/product/enrichment"
)

func TestManualReviewOnlyIsFrozenButAbsentOnHistoricalWire(t *testing.T) {
	var original Request
	old, err := json.Marshal(original)
	require.NoError(t, err)
	require.NotContains(t, string(old), "ManualReviewOnly")
	var fields map[string]any
	require.NoError(t, json.Unmarshal(old, &fields))
	fields["ManualReviewOnly"] = true
	command, err := json.Marshal(fields)
	require.NoError(t, err)
	var frozen Request
	require.NoError(t, json.Unmarshal(command, &frozen))
	wire, err := json.Marshal(frozen)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"ManualReviewOnly":true`, "the text runtime must retain the caller's explicit manual-only consent")
	require.False(t, frozen.ValidContentSelection(), "manual-only localization cannot be smuggled into an unrelated title request")
}

func TestManualReviewOnlyAllowsOnlyFrozenDefaultMode(t *testing.T) {
	r := Request{ManualReviewOnly: true, PolicyVersion: enrichment.ContentPolicyVersion, ContentSelection: enrichment.ContentSelection{TargetLanguage: "en", Title: true, SellingPoints: true}}
	for _, mode := range []ExecutionMode{ModeDefault, ModeReadOnly, ModeAutoApprove, ModeFullAccess} {
		require.Equal(t, mode == ModeDefault, r.AllowsManualReviewMode(mode), string(mode))
	}
}
