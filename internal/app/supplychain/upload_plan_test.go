package supplychainapp

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	tiktok "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/review"
)

func TestUploadPlanReusesOriginalOptimizationAdmission(t *testing.T) {
	calls := 0
	app := &Application{AuthorizeOptimization: func(_ context.Context, in preparation.OperationInput) error {
		calls++
		require.Equal(t, "quoted-template", in.TitleTemplateID)
		return preparation.ErrForbidden
	}}
	reader := UploadPlanInputs{Application: app}
	in := preparation.OperationInput{Action: preparation.OperationOptimize, TitleTemplateID: "quoted-template"}
	err := reader.AuthorizeUploadPlanStep(context.Background(), preparation.UploadPlan{}, in)
	require.ErrorIs(t, err, preparation.ErrForbidden)
	require.Equal(t, 1, calls)
	app.AuthorizeOptimization = nil
	require.ErrorIs(t, reader.AuthorizeUploadPlanStep(context.Background(), preparation.UploadPlan{}, in), record.ErrNotReady)
}

func TestUploadPlanRequiresItsOwnApplyReceiptAndExactNewTarget(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	id := uuid.NewString()
	original := record.TargetRecord{EffectiveVersion: 1, Source: preparation.SourceItem{Source: collection.Source{ProductKey: "product"}}}
	saved := original
	saved.EffectiveVersion = 2
	saved.ApplyReceiptID = id
	reviewed := review.Record{ID: id, Org: scope.OrganizationID, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: "product", BaseVersion: 1}, State: "accepted"}
	for _, state := range []string{"pending", "accepted", "applied"} {
		reviewed.State = state
		required, rejected, err := uploadPlanReviewState(scope, id, original, saved, reviewed)
		require.NoError(t, err)
		require.True(t, required, "state text without an actual Apply receipt cannot authorize uploading")
		require.False(t, rejected)
	}
	reviewed.State = "applied"
	reviewed.Receipt = &review.Receipt{ProposalID: id, Actor: scope.ActorID, ProductVersion: 2}
	required, rejected, err := uploadPlanReviewState(scope, id, original, saved, reviewed)
	require.NoError(t, err)
	require.False(t, required)
	require.False(t, rejected)
	saved.ApplyReceiptID = uuid.NewString()
	required, _, err = uploadPlanReviewState(scope, id, original, saved, reviewed)
	require.NoError(t, err)
	require.True(t, required, "an unrelated edit/approval does not complete this plan's review")
	reviewed.Owner = "other-actor"
	_, _, err = uploadPlanReviewState(scope, id, original, saved, reviewed)
	require.ErrorIs(t, err, record.ErrConflict)
}

func TestUploadPlanPublicationSummaryShowsOriginalNativeTimeAndDoesNotInventScheduler(t *testing.T) {
	_, _, records, _, _, _, _ := uploadFixture(t)
	saved := records.saved
	saved.Result.Product.SKCs[0].ShelfWay = "2"
	saved.Result.Product.SKCs[0].HopeOnSaleDate = "2099-01-01 08:00:00"
	require.Contains(t, uploadPlanPublicationSummary(saved)[0], "2099-01-01 08:00:00")
}

func TestUploadPlanDistinguishesNativeTikTokPublicationFromRetainedAssets(t *testing.T) {
	require.Equal(t, "failed", uploadPlanTikTokOutcome(&submission.TikTokEffectReceipt{Publication: &tiktok.GlobalPublishResult{State: tiktok.PublicationFailed}}))
	for _, state := range []tiktok.ChannelState{tiktok.SubmittedForReview, tiktok.RemoteDraft} {
		require.Equal(t, "accepted", uploadPlanTikTokOutcome(&submission.TikTokEffectReceipt{Publication: &tiktok.GlobalPublishResult{State: state}}))
	}
	require.Equal(t, "accepted", uploadPlanTikTokOutcome(&submission.TikTokEffectReceipt{Local: &tiktok.CreatedLocalProduct{}, LocalSaveMode: "AS_DRAFT"}))
	require.Equal(t, "asset_retained", uploadPlanTikTokOutcome(&submission.TikTokEffectReceipt{Global: &tiktok.CreatedGlobalProduct{}}))
	require.Equal(t, "asset_retained", uploadPlanTikTokOutcome(&submission.TikTokEffectReceipt{Uploaded: &tiktok.UploadedImage{}}))
}
