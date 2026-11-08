package orgresourceadapter

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/ledger/orgresource"
)

func rejectedInvocationFixture(t *testing.T, resource, native *gorm.DB) (*aistore.GormInvocationRecorder, *GormModelInvocationRepository, aicapability.InvocationRecord) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, AutoMigrate(resource))
	require.NoError(t, aistore.AutoMigrateInvocationLedger(native))
	start := time.Date(2026, 11, 8, 7, 56, 27, 0, time.UTC)
	fact := aicapability.InvocationRecord{
		InvocationID: "original:step:1", AgentRunID: "original", TenantID: "org", UserID: "actor", MemberID: "member", InputHash: "input",
		ProviderID: "openai", ModelID: aicapability.SingleImageMiniModel, Operation: aicapability.OperationOwnProductIntake, Capability: aicapability.CapabilityProductEnrichText,
		StartedAt: start, Outcome: aicapability.InvocationDispatched, MaximumPromptTokens: 6660, MaximumCompletionTokens: 4000, Currency: "CNY",
		OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "USD", CNYMicrosPerCurrencyUnit: 6716800, Model: aicapability.SingleImageMiniModel, Channel: "openai-direct", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, CachedInputText: 75000, OutputText: 4500000}},
	}
	recorder := aistore.NewGormInvocationRecorder(native)
	claimed, err := recorder.ClaimInvocation(ctx, fact)
	require.NoError(t, err)
	require.True(t, claimed)
	repo, err := NewGormModelInvocationRepository(resource, TransactionConfig{}, recorder, &modelTestAuthorizer{})
	require.NoError(t, err)
	repo.now = func() time.Time { return start }
	limits, err := NewGormMemberLimitRepository(resource, TransactionConfig{})
	require.NoError(t, err)
	limits.now = repo.now
	_, err = limits.SetMonthlyLimit(ctx, orgresource.SetMemberLimitExecution{UnitVersion: "ai-micropoint-v1", OrganizationID: "org", MemberID: "member", ActorID: "operator", OperationID: "limit", Target: 30000000})
	require.NoError(t, err)
	require.NoError(t, resource.Omit("Reservations", "Debts").Create(&organizationResourceBucketRow{QuantityUnit: "ai-micropoint-v1", OrganizationID: "org", ResourceType: "ai_point", Available: 10000000000}).Error)
	recorder.SetUsageSettler(repo)
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 10660, start))
	fact, found, err := recorder.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
	require.NoError(t, err)
	require.True(t, found)
	var reservation organizationResourceReservationRow
	require.NoError(t, resource.Take(&reservation).Error)
	require.EqualValues(t, 15445282, reservation.Quantity)
	p := &aicapability.ProviderRejectionProof{
		Version: aicapability.ProviderRejectionProofV1, BindingDigest: aicapability.InvocationBindingDigest(fact), DispatchFingerprint: modelPointFingerprint(fact),
		ReservationID: reservation.ReservationID, ReservedMicropoints: reservation.Quantity, MemberMonth: *reservation.MemberMonthStart,
		OperatorID: "operator", ApprovalEvidenceID: "approved-isolated-recovery", OperationKey: aicapability.ProviderRejectionOperationKey(fact.InvocationID), ObservedAt: start.Add(time.Hour),
		Evidence: aicapability.ProviderRejectionEvidence{Source: "provider_request_receipt", Reason: "billing_inactive", HTTPStatus: 429, RejectedAt: start.Add(2 * time.Second), RequestIDHash: strings.Repeat("a", 64), ProviderRecordDigest: strings.Repeat("b", 64), NoChargeReceiptHash: strings.Repeat("c", 64), NoChargeEvidenceDigest: strings.Repeat("d", 64), ProjectIdentityHash: strings.Repeat("e", 64), KeyIdentityHash: strings.Repeat("f", 64), MatchingRequests: 1, Handoffs: 1},
	}
	p.Digest = aicapability.ProviderRejectionDigest(*p)
	fact.ProviderRejectionProof = p
	fact.Outcome, fact.FinishedAt = aicapability.InvocationFailed, p.Evidence.RejectedAt
	fact.EstimatedCostKnown = true
	fact.ErrorCategory, fact.ErrorCode = aicapability.ErrorProviderRejected, aicapability.ProviderRejectedUnbilledCode
	return recorder, repo, fact
}

func assertRejectedReservationSettledOnce(t *testing.T, db *gorm.DB) {
	t.Helper()
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 10000000000, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.Zero(t, bucket.Consumed)
	var counter memberAIPointMonthRow
	require.NoError(t, db.Take(&counter).Error)
	require.Zero(t, counter.Reserved)
	require.Zero(t, counter.Consumed)
	var reservation organizationResourceReservationRow
	require.NoError(t, db.Take(&reservation).Error)
	require.Equal(t, "committed", reservation.State)
	var events []organizationResourceEventRow
	require.NoError(t, db.Order("created_at,event_id").Find(&events).Error)
	require.Len(t, events, 2)
	var released int64
	for _, event := range events {
		require.Zero(t, event.ConsumedDelta)
		if event.ReservedDelta < 0 {
			released -= event.ReservedDelta
		}
	}
	require.EqualValues(t, 15445282, released)
}

func TestProviderRejectionOriginalZeroSettlementAndReplay(t *testing.T) {
	resource := openSQLiteStore(t)
	recorder, repo, fact := rejectedInvocationFixture(t, resource, openSQLiteStore(t))
	ctx := context.Background()
	for range 3 {
		require.NoError(t, recorder.ResolveDispatchedInvocation(ctx, fact))
		_, err := repo.RecoverDue(ctx)
		require.NoError(t, err)
	}
	assertRejectedReservationSettledOnce(t, resource)
	stored, found, err := recorder.FindInvocation(ctx, "org", "member", fact.InvocationID, "input")
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, stored.UsageKnown)
	require.Nil(t, stored.BillableUsage)
	require.Equal(t, fact.ProviderRejectionProof.Digest, stored.ProviderRejectionProof.Digest)
	require.Error(t, repo.ReleaseAIInvocationUsage(ctx, "org", fact.InvocationID), "a sent rejection never takes the no-dispatch release path")
	changed := fact
	changed.ProviderRejectionProof = nil
	raw, err := json.Marshal(fact.ProviderRejectionProof)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &changed.ProviderRejectionProof))
	changed.ProviderRejectionProof.ApprovalEvidenceID = "different-approval"
	changed.ProviderRejectionProof.Digest = aicapability.ProviderRejectionDigest(*changed.ProviderRejectionProof)
	require.Error(t, recorder.ResolveDispatchedInvocation(ctx, changed))
	assertRejectedReservationSettledOnce(t, resource)
}

func TestProviderRejectionUsesOriginalMonthAndDebtFirstAfterRevocation(t *testing.T) {
	resource := openSQLiteStore(t)
	recorder, repo, fact := rejectedInvocationFixture(t, resource, openSQLiteStore(t))
	require.NoError(t, resource.Create(&organizationResourceDebtRow{QuantityUnit: "ai-micropoint-v1", OrganizationID: "org", ResourceType: "ai_point", Amount: 2000000}).Error)
	repo.now = func() time.Time { return time.Date(2026, 12, 1, 0, 1, 0, 0, time.UTC) }
	repo.authorizer.(*modelTestAuthorizer).err = orgresource.ErrForbidden
	for range 2 {
		require.NoError(t, recorder.ResolveDispatchedInvocation(context.Background(), fact))
	}
	var bucket organizationResourceBucketRow
	require.NoError(t, resource.Take(&bucket).Error)
	require.EqualValues(t, 9998000000, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.Zero(t, bucket.Consumed)
	var counters []memberAIPointMonthRow
	require.NoError(t, resource.Find(&counters).Error)
	require.Len(t, counters, 1)
	require.Equal(t, fact.ProviderRejectionProof.MemberMonth, counters[0].MonthStart)
	require.Zero(t, counters[0].Reserved)
	require.Zero(t, counters[0].Consumed)
	var debt organizationResourceDebtRow
	require.NoError(t, resource.Take(&debt).Error)
	require.Zero(t, debt.Amount)
	var event organizationResourceEventRow
	require.NoError(t, resource.Where("reserved_delta < 0").Take(&event).Error)
	require.EqualValues(t, 15445282, event.GrossCredit)
	require.EqualValues(t, 2000000, event.DebtRepaid)
}

func TestProviderRejectionPersistedProofRequiresCanonicalOwnerEncoding(t *testing.T) {
	for _, alteration := range []struct {
		name   string
		change func(string) string
	}{
		{"noncanonical", func(raw string) string { return " " + raw }},
		{"unknown field", func(raw string) string { return strings.TrimSuffix(raw, "}") + `,"unknown":true}` }},
		{"duplicate field", func(raw string) string {
			return strings.TrimSuffix(raw, "}") + `,"version":"own-intake-openai-unbilled-v1"}`
		}},
	} {
		t.Run(alteration.name, func(t *testing.T) {
			native := openSQLiteStore(t)
			recorder, _, fact := rejectedInvocationFixture(t, openSQLiteStore(t), native)
			require.NoError(t, recorder.RecordInvocation(context.Background(), fact))
			var raw string
			require.NoError(t, native.Table("ai_invocations").Select("provider_rejection_proof_json").Scan(&raw).Error)
			require.NoError(t, native.Exec("UPDATE ai_invocations SET provider_rejection_proof_json=? WHERE invocation_id=?", alteration.change(raw), fact.InvocationID).Error)
			_, _, err := recorder.FindInvocation(context.Background(), "org", "member", fact.InvocationID, "input")
			require.Error(t, err)
		})
	}
}
