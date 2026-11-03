package preparation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUploadPlanContinuesAfterCompletionAndReviewWithoutReexecutingOwners(t *testing.T) {
	input := UploadPlanInput{Rule: "standard", PublicationMode: "platform"}
	item := UploadPlanItem{SourceID: "source"}
	facts := UploadPlanFacts{}
	require.Equal(t, OperationAdapt, NextUploadPlanStep(input, item, facts))
	facts.HasTarget = true
	require.Equal(t, "missing", NextUploadPlanStep(input, item, facts))
	facts.Ready = true
	require.Equal(t, OperationUpload, NextUploadPlanStep(input, item, facts))
	input.TitleTemplateID = "approved-template"
	require.Equal(t, OperationOptimize, NextUploadPlanStep(input, item, facts))
	item.OptimizeOperationID = "original-optimization"
	facts.OptimizationStatus = ItemUnknown
	require.Equal(t, "unknown", NextUploadPlanStep(input, item, facts))
	facts.OptimizationStatus = ItemReview
	facts.ReviewRequired = true
	require.Equal(t, "review", NextUploadPlanStep(input, item, facts))
	facts.ReviewRequired = false
	require.Equal(t, OperationUpload, NextUploadPlanStep(input, item, facts), "applied content continues to upload, never starts a second optimization")
	facts.Ready = false
	require.Equal(t, "missing", NextUploadPlanStep(input, item, facts), "applied content must pass validation again")
}

func TestUploadPlanPreservesEveryOriginalUploadOutcomeAcrossNewTargetVersions(t *testing.T) {
	for _, status := range []string{ItemPending, ItemRunning, ItemUnknown, ItemFailed, ItemDenied, ItemCancelled, ItemSucceeded} {
		t.Run(status, func(t *testing.T) {
			item := UploadPlanItem{UploadOperationID: "original-send"}
			facts := UploadPlanFacts{HasTarget: true, Ready: true, UploadStatus: status}
			step := NextUploadPlanStep(UploadPlanInput{}, item, facts)
			require.NotEqual(t, OperationUpload, step, "an edited/new head must never authorize another send for this plan")
			require.Equal(t, status, step)
		})
	}
	item := UploadPlanItem{UploadOperationID: "original-unknown"}
	require.Equal(t, ItemSucceeded, NextUploadPlanStep(UploadPlanInput{}, item, UploadPlanFacts{UploadStatus: ItemUnknown, UploadConfirmed: true}))
}

func TestUploadPlanDoesNotTurnFailureCancellationOrRejectionIntoNewModelCall(t *testing.T) {
	for _, status := range []string{ItemPending, ItemRunning, ItemUnknown, ItemFailed, ItemDenied, ItemCancelled} {
		require.Equal(t, status, NextUploadPlanStep(UploadPlanInput{TitleTemplateID: "t"}, UploadPlanItem{OptimizeOperationID: "op"}, UploadPlanFacts{HasTarget: true, Ready: true, OptimizationStatus: status}))
	}
	require.Equal(t, "rejected", NextUploadPlanStep(UploadPlanInput{TitleTemplateID: "t"}, UploadPlanItem{OptimizeOperationID: "op"}, UploadPlanFacts{HasTarget: true, Ready: true, OptimizationStatus: ItemReview, ReviewRejected: true}))
}
