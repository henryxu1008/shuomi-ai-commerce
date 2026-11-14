package agent

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	reviewproposal "task-processor/internal/product/review/proposal"
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
	r := Request{ManualReviewOnly: true, PolicyVersion: reviewproposal.ContentPolicyVersion, ContentSelection: reviewproposal.ContentSelection{TargetLanguage: "en", Title: true, SellingPoints: true}}
	for _, mode := range []ExecutionMode{ModeDefault, ModeReadOnly, ModeAutoApprove, ModeFullAccess} {
		require.Equal(t, mode == ModeDefault, r.AllowsManualReviewMode(mode), string(mode))
	}
}

func TestScheduledSourceAllowsOnlyFrozenManualModes(t *testing.T) {
	r := Request{Key: "c93e6a9a-7e23-5ca1-b43e-96049ee01cf8", ScheduledSource: ScheduledSourceRef{PlanID: "fe865313-b1b8-4ea9-9e01-ea3f359295f3", OccurrenceID: "c93e6a9a-7e23-5ca1-b43e-96049ee01cf8"}, ManualReviewOnly: true, PolicyVersion: reviewproposal.ContentPolicyVersion, ContentSelection: reviewproposal.ContentSelection{TargetLanguage: "en", Title: true}}
	for _, mode := range []ExecutionMode{ModeDefault, ModeReadOnly, ModeAutoApprove, ModeFullAccess} {
		require.Equal(t, mode == ModeDefault || mode == ModeReadOnly, r.AllowsManualReviewMode(mode), string(mode))
	}
	r.ManualReviewOnly = false
	require.False(t, r.ValidContentSelection(), "a scheduled source cannot remove the manual restriction")
	r.ManualReviewOnly = true
	r.ScheduledSource.OccurrenceID = ""
	require.False(t, r.ValidContentSelection(), "partial scheduled ancestry is invalid")
}
