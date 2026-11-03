package orgresourceadapter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	resource "task-processor/internal/ledger/orgresource"
)

func TestModelPointFingerprintKeepsOriginalClaimWhenUsagePartitionsArrive(t *testing.T) {
	fact := aicapability.InvocationRecord{InvocationID: "invocation", TenantID: "org", MemberID: "member", UserID: "actor", AgentRunID: "run", Operation: aicapability.OperationProductAgentDecision, InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched}
	original := modelPointFingerprint(fact)
	fact.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 5, CachedInputText: 2, OutputText: 2}
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 5, 2, 7
	fact.Outcome, fact.UsageKnown = aicapability.InvocationSucceeded, true
	require.Equal(t, original, modelPointFingerprint(fact), "observed counters cannot change the original reservation identity")
}

func TestOfficialPriceQuantityAndResourceDisplayUseTheSameMicrounit(t *testing.T) {
	require.Equal(t, aicapability.AIPointMicrounitV1, resource.AIPointQuantityUnitV1)
	quantity := resource.AIPointQuantity{Micropoints: 745500, UnitVersion: aicapability.AIPointMicrounitV1}
	text, err := quantity.PointsString()
	require.NoError(t, err)
	require.Equal(t, "0.7455", text)
	parsed, err := resource.ParseAIPointQuantity(text)
	require.NoError(t, err)
	require.Equal(t, quantity, parsed)
}

func TestModelSettlementRequiresMatchingCompleteOfficialUsage(t *testing.T) {
	for _, kind := range []string{"missing", "incomplete", "different-mapping", "different-counters"} {
		t.Run(kind, func(t *testing.T) {
			db, recorder, _, _, fact := modelPointFixture(t)
			require.NoError(t, recorder.ReserveAIInvocationUsage(context.Background(), "org", "member", fact.InvocationID, 20, fact.StartedAt))
			fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, fact.StartedAt.Add(time.Second)
			fact.PromptTokens, fact.TotalTokens = 3, 3
			fact.BillableUsage = syntheticUncachedUsage(3, 0)
			switch kind {
			case "missing":
				fact.BillableUsage = nil
			case "incomplete":
				fact.BillableUsage.Complete = false
			case "different-mapping":
				fact.BillableUsage.MappingVersion = "another-mapper"
			case "different-counters":
				fact.BillableUsage.InputText = 2
			}
			require.Error(t, recorder.RecordInvocation(context.Background(), fact))
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			require.EqualValues(t, 30, bucket.Reserved)
			require.Zero(t, bucket.Consumed)
		})
	}
}

func TestModelSettlementRoundsOnceAfterOfficialCacheDiscount(t *testing.T) {
	db, recorder, repo, _, fact := modelPointFixture(t)
	// New canonical claim: exact CNY price gives 1 micro/input token,
	// 0.25/cached token and 2/output token. Reserve without cache discount.
	fact.InvocationID = "cached-official"
	fact.OfficialTariff.Rates.CachedInputText = 2500
	_, err := recorder.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
	fact.BillableUsage = syntheticUncachedUsage(3, 2)
	fact.BillableUsage.CachedInputText = 2
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, fact.StartedAt.Add(time.Second)
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 5, bucket.Consumed, "floor(1+2*0.25+2*2), no per-part rounding or independent point rate")
	require.EqualValues(t, 95, bucket.Available)
	require.Zero(t, bucket.Reserved)
	n, err := repo.RecoverDue(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestPositiveOfficialUsageCanRoundToZeroWithoutBecomingNoSend(t *testing.T) {
	db, recorder, _, _, fact := modelPointFixture(t)
	fact.InvocationID = "sub-micropoint"
	fact.OfficialTariff.Rates.InputText, fact.OfficialTariff.Rates.OutputText = 1, 1
	_, err := recorder.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	fact.PromptTokens, fact.TotalTokens = 1, 1
	fact.BillableUsage = syntheticUncachedUsage(1, 0)
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, fact.StartedAt.Add(time.Second)
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 100, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.Zero(t, bucket.Consumed)
	var row organizationResourceReservationRow
	require.NoError(t, db.Where("owner_attempt_id = ?", fact.InvocationID).Take(&row).Error)
	require.Equal(t, string(resource.ReservationCommitted), row.State, "zero actual charge is a committed invocation, not a no-send release")
}

func TestExplicitResolutionKeepsOfficialUsageAndOriginalChargeSnapshot(t *testing.T) {
	db, recorder, _, _, fact := modelPointFixture(t)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	originalPrice := fact.OfficialTariff
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, fact.StartedAt.Add(time.Second)
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
	fact.BillableUsage = syntheticUncachedUsage(3, 2)
	fact.OfficialTariff.Rates.InputText = 9000000 // resolution cannot reprice the original claim
	require.NoError(t, recorder.ResolveDispatchedInvocation(ctx, fact))
	require.NoError(t, recorder.ResolveDispatchedInvocation(ctx, fact))
	got, err := recorder.ReadModelInvocation(ctx, "org", fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, originalPrice, got.OfficialTariff)
	require.Equal(t, fact.BillableUsage, got.BillableUsage)
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 7, bucket.Consumed)
	require.EqualValues(t, 93, bucket.Available)
	require.Zero(t, bucket.Reserved)
}
