package supplychainapp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	tiktok "task-processor/internal/marketplace/tiktokshop"
	walmart "task-processor/internal/marketplace/walmart"
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

func TestUploadPlanContinuesOnlyAnEvidencedWalmartRepair(t *testing.T) {
	uploader, scope, records, merchant, _ := walmartFlowFixture(t)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	accepted, err := uploader.Upload(ctx, scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	receipt, err := uploader.dependencies.Receipts.ReadOfficial(ctx, scope, accepted.AttemptID)
	require.NoError(t, err)
	sku := records.saved.Walmart.SKUs[0].SellerSKU
	repair := *records.saved.Input.Walmart
	repair.SKUs = append([]walmart.SKUInput(nil), repair.SKUs...)
	repair.SKUs[0].Price = "11"
	repair.RepairFrom, repair.RepairSKUs = receipt.ID, []string{sku}
	merchantKey := walmart.SKUNamespace(merchant.binding.SupplierIdentityHash, merchant.binding.Walmart.GrantID)
	draft, err := walmart.BuildDraft(repair, merchant.rules, merchantKey, records.saved.Source.Source.ProductKey, []string{repair.SKUs[0].SourceSKUKey})
	require.NoError(t, err)
	records.saved.ID, records.saved.Revision = uuid.NewString(), records.saved.Revision+1
	records.saved.Input.Walmart, records.saved.Walmart = &repair, &draft
	require.True(t, record.PublicationReady(records.saved))

	result := submission.WalmartResultRecord{ID: uuid.NewString(), ReceiptID: receipt.ID, Generation: 2, Latest: true, RepairableSKUs: []string{sku}, Observation: walmart.FeedObservation{FeedID: receipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Failed: 1, Items: []walmart.ItemResult{{SKU: sku, IngestionStatus: "DATA_ERROR"}}}}
	results := &walmartFlowResults{records: []submission.WalmartResultRecord{result}}
	reader := UploadPlanInputs{Application: &Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Records: records, Authorization: &uploadAuth{scope: scope}, PublicationReceipts: uploader.dependencies.Receipts, TargetMerchants: &rulesFixture{binding: merchant.binding}, WalmartResults: results}, Operations: struct {
		preparation.UploadPlanOperationReader
	}{}}
	plan := preparation.UploadPlan{Owner: scope, Input: preparation.UploadPlanInput{PreparationID: records.saved.Source.PreparationID, StoreID: merchant.binding.StoreID, PublicationMode: "platform"}}
	item := preparation.UploadPlanItem{SourceID: records.saved.Source.ID, SourceVersion: records.saved.Source.Source.Version}
	facts, err := reader.ReadUploadPlanFacts(ctx, plan, item)
	require.NoError(t, err)
	require.Equal(t, receipt.RecordID, facts.ExistingPublicationRecordID)
	require.Equal(t, preparation.OperationUpload, preparation.NextUploadPlanStep(plan.Input, item, facts))
	require.Equal(t, &preparation.WalmartUploadApproval{SourceID: item.SourceID, RecordID: records.saved.ID, Revision: records.saved.Revision, WireHash: draft.WireHash}, facts.WalmartApproval)
	preview := preparation.NewUploadPlanPreviewDigest(plan)
	preview.Add(preparation.UploadPlanItemView{Item: item, Facts: facts, Step: preparation.OperationUpload})
	results.records[0].ID = uuid.NewString()
	refreshed, err := reader.ReadUploadPlanFacts(ctx, plan, item)
	require.NoError(t, err)
	refreshedPreview := preparation.NewUploadPlanPreviewDigest(plan)
	refreshedPreview.Add(preparation.UploadPlanItemView{Item: item, Facts: refreshed, Step: preparation.OperationUpload})
	require.NotEqual(t, preview.Sum(), refreshedPreview.Sum(), "a new observation needs a new approval preview")
	usedItem := item
	usedItem.UploadOperationID = uuid.NewString()
	for _, status := range []string{preparation.ItemPending, preparation.ItemUnknown, preparation.ItemFailed, preparation.ItemSucceeded} {
		facts.UploadStatus = status
		require.NotEqual(t, preparation.OperationUpload, preparation.NextUploadPlanStep(plan.Input, usedItem, facts), "the original plan cannot dispatch a new operation")
	}
	for _, change := range []func(*submission.WalmartResultRecord){
		func(result *submission.WalmartResultRecord) { result.ReceiptID = uuid.NewString() },
		func(result *submission.WalmartResultRecord) { result.RepairableSKUs = []string{"unrelated-sku"} },
		func(result *submission.WalmartResultRecord) { result.Observation.Partial = true },
		func(result *submission.WalmartResultRecord) { result.RepairableSKUs = nil },
	} {
		other := result
		change(&other)
		results.records = []submission.WalmartResultRecord{other}
		facts, err = reader.ReadUploadPlanFacts(ctx, plan, item)
		require.NoError(t, err)
		require.Equal(t, "channel_exists", preparation.NextUploadPlanStep(plan.Input, item, facts))
	}
	require.Len(t, merchant.submitted, 1, "a plan preview does not send a Feed")
}
