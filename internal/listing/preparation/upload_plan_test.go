package preparation

import (
	"context"
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

func TestUploadPlanKeepsExistingChannelReceiptBoundToItsOriginalVersion(t *testing.T) {
	facts := UploadPlanFacts{HasTarget: true, Ready: true, Target: TargetApproval{RecordID: "new-version"}, ExistingPublicationRecordID: "original-version", PublicationOutcome: "accepted"}
	require.Equal(t, "channel_exists", NextUploadPlanStep(UploadPlanInput{}, UploadPlanItem{}, facts), "a retained platform channel cannot be called a successful upload of new content")
	facts.Target.RecordID = "original-version"
	require.Equal(t, ItemSucceeded, NextUploadPlanStep(UploadPlanInput{}, UploadPlanItem{}, facts))
}

func TestUploadPlanManualPlatformOpensCompletionWithoutInventingAdaptation(t *testing.T) {
	require.Equal(t, "missing", NextUploadPlanStep(UploadPlanInput{}, UploadPlanItem{}, UploadPlanFacts{ManualTargetRequired: true}))
}

func TestCancelledUploadPlanCannotBindANewStep(t *testing.T) {
	s := &UploadPlanService{inputs: &planFactsOnly{}}
	v, err := s.resolve(context.Background(), UploadPlan{Cancelled: true}, UploadPlanItem{})
	require.NoError(t, err)
	require.Equal(t, ItemCancelled, v.Step)
}

type planFactsOnly struct{ UploadPlanInputs }

func (*planFactsOnly) ReadUploadPlanFacts(context.Context, UploadPlan, UploadPlanItem) (UploadPlanFacts, error) {
	return UploadPlanFacts{}, nil
}

func TestUploadPlanRetainedFailureAndAssetsNeverBecomeAnUploadedItem(t *testing.T) {
	for _, operation := range []string{"", "original-upload"} {
		facts := UploadPlanFacts{HasTarget: true, Ready: true, Target: TargetApproval{RecordID: "original"}, ExistingPublicationRecordID: "original", PublicationOutcome: "failed", UploadStatus: ItemUnknown, UploadConfirmed: true}
		require.Equal(t, ItemFailed, NextUploadPlanStep(UploadPlanInput{}, UploadPlanItem{UploadOperationID: operation}, facts))
		facts.PublicationOutcome = "asset_retained"
		facts.UploadResultOutcome = "asset_retained"
		facts.UploadConfirmed = false
		require.Equal(t, "asset_retained", NextUploadPlanStep(UploadPlanInput{}, UploadPlanItem{UploadOperationID: operation}, facts))
	}
}
