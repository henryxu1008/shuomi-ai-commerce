package listingapp

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/product/review"
)

func TestCockpitListingQueuesAreExclusiveAndUseCurrentTargetVersion(t *testing.T) {
	_, scope, records, _, _, _, _ := uploadFixture(t)
	saved := records.saved
	proposalID := uuid.NewString()
	item := preparation.OperationItem{SourceID: saved.Source.ID, RecordID: saved.ID, RecordRevision: saved.Revision, Status: preparation.ItemReview, ResultReference: proposalID}
	proposal := review.Record{ID: proposalID, Org: scope.OrganizationID, Owner: scope.ActorID, Input: review.CreateInput{ProductKey: saved.Source.Source.ProductKey, BaseVersion: saved.EffectiveVersion}}

	for _, tc := range []struct {
		name, operationState, reviewState string
		stale, pending, needsReview       bool
	}{
		{"review required", preparation.ItemReview, "pending", false, false, true},
		{"accepted, waiting to apply", preparation.ItemReview, "accepted", false, false, false},
		{"applied to a previous candidate", preparation.ItemReview, "applied", false, false, false},
		{"review rejected", preparation.ItemReview, "rejected", false, true, false},
		{"obsolete review", preparation.ItemReview, "pending", true, true, false},
		{"running optimization", preparation.ItemRunning, "", false, false, false},
		{"unknown optimization result", preparation.ItemUnknown, "", false, false, false},
		{"failed optimization, still needs processing", preparation.ItemFailed, "", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operation, value := item, proposal
			operation.Status, value.State = tc.operationState, tc.reviewState
			if tc.stale {
				operation.RecordRevision++
			}
			projection := ReviewProjection{Facts: stageFactsFixture{item: operation}, Reviews: stageReviewFixture{value: value}}
			pending, needsReview, err := projection.metricQueues(context.Background(), scope, saved)
			require.NoError(t, err)
			require.Equal(t, tc.pending, pending)
			require.Equal(t, tc.needsReview, needsReview)
		})
	}
	proposal.Owner = "different-actor"
	projection := ReviewProjection{Facts: stageFactsFixture{item: item}, Reviews: stageReviewFixture{value: proposal}}
	_, _, err := projection.metricQueues(context.Background(), scope, saved)
	require.ErrorIs(t, err, record.ErrConflict)
}

func TestCockpitCurrentReviewOverridesExistingUploadReceipt(t *testing.T) {
	uploader, scope, records, _, _, receipts, _ := uploadFixture(t)
	saved := records.saved
	ctx := context.Background()
	result, err := uploader.Upload(ctx, scope, uuid.NewString(), saved.ID)
	require.NoError(t, err)
	require.Equal(t, saved.ID, result.PublishedRecordID)
	app := Application{PublicationReceipts: receipts, StageProjection: ReviewProjection{Facts: stageFactsFixture{}}}
	var counts listingMetricCounts
	require.NoError(t, app.countTargetQueues(ctx, scope, saved, saved.Merchant, &counts))
	require.Equal(t, listingMetricCounts{}, counts, "an uploaded current record is no longer pending")

	proposalID := uuid.NewString()
	item := preparation.OperationItem{SourceID: saved.Source.ID, RecordID: saved.ID, RecordRevision: saved.Revision, Status: preparation.ItemReview, ResultReference: proposalID}
	app.StageProjection = ReviewProjection{
		Facts: stageFactsFixture{item: item},
		Reviews: stageReviewFixture{value: review.Record{ID: proposalID, Org: scope.OrganizationID, Owner: scope.ActorID, State: "pending",
			Input: review.CreateInput{ProductKey: saved.Source.Source.ProductKey, BaseVersion: saved.EffectiveVersion}}},
	}
	require.NoError(t, app.countTargetQueues(ctx, scope, saved, saved.Merchant, &counts))
	require.Equal(t, listingMetricCounts{Review: 1}, counts)
}

func TestCockpitMerchantBindingExcludesOldMerchantAndAcceptsRefreshedExpiry(t *testing.T) {
	_, _, records, _, _, _, _ := uploadFixture(t)
	saved := records.saved.Merchant
	current := saved
	current.ServiceExpiresAt = current.ServiceExpiresAt.Add(time.Hour)
	require.True(t, sameMetricsMerchant(saved, current))
	current.SupplierIdentityHash = "another-merchant"
	require.False(t, sameMetricsMerchant(saved, current))
}
