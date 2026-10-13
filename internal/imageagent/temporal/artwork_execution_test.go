package temporal

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
	"testing"
)

func TestArtworkV3WirePreservesCreativeIntentWithoutAnOriginal(t *testing.T) {
	art := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创图案", Prompt: "Create brand new artwork", InputDigest: "original-intent"}
	input := ExecuteSlotV3ActivityInput{RunID: "run", Identity: imageagent.ExecutionIdentity{TenantID: "org", UserID: "actor"}, PlanRevision: 1, Slot: imageagent.Slot{ID: "artwork", Role: imageagent.SlotRoleMain}, Attempt: 1, Artwork: art}
	execution := slotExecutionInputV3(input)
	require.Equal(t, art, execution.Artwork)
	require.Nil(t, execution.ImageSet)
	require.Empty(t, execution.Slot.SourceAssetIDs)
	before := imageagent.SlotExecutionFingerprint(execution)
	art.Prompt = "changed"
	require.Equal(t, before, imageagent.SlotExecutionFingerprint(execution))
	require.NotEqual(t, before, imageagent.SlotExecutionFingerprint(slotExecutionInputV3(input)))
	for _, payload := range []any{SlotWorkflowV3Input{}, ExecuteSlotV3ActivityInput{}, EffectRecoveryWorkflowInput{}} {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "Artwork")
	}
}
