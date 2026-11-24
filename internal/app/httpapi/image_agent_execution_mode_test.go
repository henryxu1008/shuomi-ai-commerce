package httpapi

import (
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImagePolicyRequiresExactCompleteSet(t *testing.T) {
	plan := imageagent.Plan{Revision: 1, Slots: []imageagent.Slot{{ID: "main", Recipe: &imageagent.ImageSlotRecipe{Placement: imageagent.ImagePlacement{Group: "carousel", Order: 1}}}}}
	p := imageagent.RunProjection{Run: imageagent.Run{ID: "run"}, Plan: plan, ResultDigest: "digest", Slots: []imageagent.SlotProjection{{Slot: imageagent.Slot{ID: "main", Status: imageagent.SlotStatusAccepted}, Attempt: 1, Candidates: []imageagent.AssetCandidate{{AssetID: "asset"}}}}}
	command := asset.ImageSetCommand{Choices: []asset.ImageSetChoice{{Kind: "generated", RunID: "run", PlanRevision: 1, ResultDigest: "digest", SlotID: "main", Attempt: 1, AssetID: "asset", Presentation: asset.ImagePresentation{Group: "carousel", Order: 1}}}}
	require.True(t, completeImagePolicySelection(p, command))
	p.Plan.Set = &imageagent.ImageSetPlan{Source: imageagent.ImageSourceBinding{OwnProduct: true}}
	require.False(t, completeImagePolicySelection(p, command), "enterprise-owned supplier output requires a human adoption")
	p.Plan.Set = nil
	p.Slots[0].Candidates = append(p.Slots[0].Candidates, imageagent.AssetCandidate{AssetID: "other"})
	require.False(t, completeImagePolicySelection(p, command), "policy cannot guess which candidate the user meant")
	p.Slots[0].Candidates = p.Slots[0].Candidates[:1]
	command.Choices[0].AssetID = "foreign"
	require.False(t, completeImagePolicySelection(p, command))
	command.Choices[0].AssetID = "asset"
	p.Slots[0].Slot.Status = imageagent.SlotStatusBlocked
	require.False(t, completeImagePolicySelection(p, command), "partial or unknown work cannot be adopted")
}
