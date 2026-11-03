package orgresourceadapter

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/ledger/orgresource"
	"testing"
	"time"
)

type modelTestAuthorizer struct{ err error }

func (a *modelTestAuthorizer) AuthorizeModelInvocation(context.Context, aicapability.InvocationRecord) error {
	return a.err
}

func modelPointFixture(t *testing.T, operations ...aicapability.Operation) (*gorm.DB, *aistore.GormInvocationRecorder, *GormModelInvocationRepository, *modelTestAuthorizer, aicapability.InvocationRecord) {
	t.Helper()
	resource, native := openSQLiteStore(t), openSQLiteStore(t)
	require.NoError(t, AutoMigrate(resource))
	require.NoError(t, aistore.AutoMigrateInvocationLedger(native))
	now := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)
	fact := aicapability.InvocationRecord{InvocationID: "invocation", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", Operation: aicapability.OperationProductAgentDecision, Outcome: aicapability.InvocationDispatched, StartedAt: now, MaximumPromptTokens: 10, MaximumCompletionTokens: 10, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic", FXVersion: "fixture-fx-identity", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "fixture-model", Channel: "fixture-channel", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}}, ModelID: "fixture-model", Currency: "CNY"}
	if len(operations) == 1 {
		fact.Operation = operations[0]
	}
	if len(operations) == 1 && operations[0] == aicapability.OperationAIWorkbenchChatPlan {
		fact.Operation = operations[0]
		fact.Capability = aicapability.CapabilityAIWorkbenchChatPlanning
		fact.AgentRunID = ""
	}
	balance := int64(100)
	if fact.Operation == aicapability.OperationProductImageAnalysis {
		fact.Capability, fact.ModelID = aicapability.CapabilityProductImageAnalysis, aicapability.SingleImageMiniModel
		fact.OfficialTariff.Model, fact.OfficialTariff.Channel = fact.ModelID, "openai-direct"
		fact.OfficialTariff.Rates.CachedInputText = 1000
		fact.MaximumPromptTokens, fact.MaximumCompletionTokens = 5000, 64
		balance = 10000
	}
	recorder := aistore.NewGormInvocationRecorder(native)
	_, err := recorder.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	auth := &modelTestAuthorizer{}
	repo, err := NewGormModelInvocationRepository(resource, TransactionConfig{}, recorder, auth)
	require.NoError(t, err)
	repo.now = func() time.Time { return now }
	limits, err := NewGormMemberLimitRepository(resource, TransactionConfig{})
	require.NoError(t, err)
	limits.now = repo.now
	_, err = limits.SetMonthlyLimit(context.Background(), orgresource.SetMemberLimitExecution{UnitVersion: "ai-micropoint-v1", OrganizationID: "org", MemberID: "member", ActorID: "admin", OperationID: "limit", Target: balance})
	require.NoError(t, err)
	require.NoError(t, resource.Omit("Reservations", "Debts").Create(&organizationResourceBucketRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org", ResourceType: "ai_point", Available: balance}).Error)
	recorder.SetUsageSettler(repo)
	return resource, recorder, repo, auth, fact
}

func TestSingleImagePointsUseOneOriginalReservationAndActualCachedUsage(t *testing.T) {
	db, recorder, repo, _, fact := modelPointFixture(t, aicapability.OperationProductImageAnalysis)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 5064, fact.StartedAt))
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 5064, fact.StartedAt))
	var held organizationResourceBucketRow
	require.NoError(t, db.Take(&held).Error)
	require.EqualValues(t, 5128, held.Reserved)
	require.ErrorIs(t, repo.ReleaseAIInvocationUsage(ctx, "org", fact.InvocationID), orgresource.ErrOwnerNotTerminal, "a dispatched unknown image cannot release or replay")
	fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationUsageObservedFailed, true, fact.StartedAt.Add(time.Second)
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 30, 5, 35
	fact.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 30, CachedInputText: 10, OutputText: 5}
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	// Synthetic tariff: (30-10)*1 + 10*0.1 + 5*2 = 31 micro-points.
	// The image is already in P; there is no second per-image charge.
	require.EqualValues(t, 31, bucket.Consumed)
	require.EqualValues(t, 9969, bucket.Available)
	require.Zero(t, bucket.Reserved)
	var month memberAIPointMonthRow
	require.NoError(t, db.Take(&month).Error)
	require.EqualValues(t, 31, month.Consumed)
	require.Zero(t, month.Reserved)
	var reservation organizationResourceReservationRow
	require.NoError(t, db.Take(&reservation).Error)
	require.Equal(t, fact.AgentRunID, reservation.BusinessScope)
	require.Equal(t, string(orgresource.ReservationCommitted), reservation.State)
	var events []organizationResourceEventRow
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 2, "reserve and one settlement, including exact original-key replay")
}

func TestAdvisoryAgentsModelPointsUseOriginalRunAndSettleOnce(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationListingMaterialsDecision, aicapability.OperationProductResearchDecision, aicapability.OperationProductSelectionDecision, aicapability.OperationShopDiagnosisDecision} {
		t.Run(string(op), func(t *testing.T) {
			db, recorder, _, _, fact := modelPointFixture(t, op)
			ctx := context.Background()
			require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, fact.StartedAt))
			require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, fact.StartedAt))
			var reservation organizationResourceReservationRow
			require.NoError(t, db.Where("owner_attempt_id=?", fact.InvocationID).Take(&reservation).Error)
			require.Equal(t, fact.AgentRunID, reservation.BusinessScope)
			fact.Outcome, fact.UsageKnown = aicapability.InvocationUsageObservedFailed, true
			fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
			fact.BillableUsage = syntheticUncachedUsage(3, 2)
			fact.FinishedAt = fact.StartedAt.Add(time.Second)
			require.NoError(t, recorder.RecordInvocation(ctx, fact))
			require.NoError(t, recorder.RecordInvocation(ctx, fact))
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			require.Zero(t, bucket.Reserved)
			require.EqualValues(t, 7, bucket.Consumed)
			require.EqualValues(t, 93, bucket.Available)

		})
	}
}

func TestAdvisoryReferenceUsageKeepsOriginalLedgerAndUnknownReservation(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationProductResearchDecision, aicapability.OperationProductSelectionDecision} {
		for _, known := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/basic-totals-known=%t", op, known), func(t *testing.T) {
				db, recorder, repo, _, fact := modelPointFixture(t, op)
				ctx := context.Background()
				start := fact.StartedAt
				fact.InvocationID = "reference-invocation"
				fact.OfficialTariff.FXVersion = "cny-identity-v1"
				fact.OfficialTariff.UsageVersion = aicapability.ReferenceTextTotalsV1
				fact.OfficialTariff.PriceVersion = "reference-config-v1"
				_, err := recorder.ClaimInvocation(ctx, fact)
				require.NoError(t, err)
				require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, start))
				fingerprint := modelPointFingerprint(fact)
				fact.Outcome, fact.UsageKnown, fact.FinishedAt = aicapability.InvocationSucceeded, true, start.Add(time.Second)
				fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
				fact.BillableUsage = &aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceTextTotalsV1, Complete: true, InputText: 3, OutputText: 2}
				if known {
					require.True(t, fact.OfficialTariff.IsReferenceEstimate())
				} else {
					fact.BillableUsage = nil
				}
				require.Equal(t, fingerprint, modelPointFingerprint(fact))
				err = recorder.RecordInvocation(ctx, fact)
				if known {
					require.NoError(t, err)
					require.NoError(t, recorder.RecordInvocation(ctx, fact))
				} else {
					require.Error(t, err)
					require.Error(t, repo.ReleaseAIInvocationUsage(ctx, fact.TenantID, fact.InvocationID))
				}
				var bucket organizationResourceBucketRow
				require.NoError(t, db.Take(&bucket).Error)
				if known {
					require.EqualValues(t, 7, bucket.Consumed)
					require.Zero(t, bucket.Reserved)
					require.EqualValues(t, 93, bucket.Available)
				} else {
					require.Zero(t, bucket.Consumed)
					require.EqualValues(t, 30, bucket.Reserved)
					require.EqualValues(t, 70, bucket.Available)
				}
				var events []organizationResourceEventRow
				require.NoError(t, db.Find(&events).Error)
				if known {
					require.Len(t, events, 2, "exact replay does not settle twice")
				} else {
					require.Len(t, events, 1, "only original reserve remains")
				}
			})
		}
	}
}

func TestPlannerModelPointsBindChatPlanScopeWithoutAgentRun(t *testing.T) {
	db, recorder, _, _, fact := modelPointFixture(t, aicapability.OperationAIWorkbenchChatPlan)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, fact.TenantID, fact.MemberID, fact.InvocationID, 20, fact.StartedAt))
	var reservation organizationResourceReservationRow
	require.NoError(t, db.Where("owner_attempt_id = ?", fact.InvocationID).Take(&reservation).Error)
	require.Equal(t, "chat-plan:"+fact.InvocationID, reservation.BusinessScope)
	fact.Outcome = aicapability.InvocationUsageObservedFailed
	fact.UsageKnown, fact.EstimatedCostKnown = true, true
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
	fact.BillableUsage = syntheticUncachedUsage(3, 2)
	fact.FinishedAt = fact.StartedAt.Add(time.Second)
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	var settled organizationResourceReservationRow
	require.NoError(t, db.Where("owner_attempt_id = ?", fact.InvocationID).Take(&settled).Error)
	require.Equal(t, "chat-plan:"+fact.InvocationID, settled.BusinessScope)
	require.Equal(t, string(orgresource.ReservationCommitted), settled.State)
}

func TestModelPointsActualUsageReturnsRemainderDebtFirstToOriginalMonth(t *testing.T) {
	db, recorder, r, auth, fact := modelPointFixture(t)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	require.Error(t, recorder.ReserveAIInvocationUsage(ctx, "org", "other", fact.InvocationID, 20, fact.StartedAt))
	require.NoError(t, db.Create(&organizationResourceDebtRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org", ResourceType: "ai_point", Amount: 4}).Error)
	r.now = func() time.Time { return fact.StartedAt.Add(2 * time.Minute) }
	auth.err = orgresource.ErrForbidden
	fact.Outcome = aicapability.InvocationUsageObservedFailed
	fact.UsageKnown = true
	fact.PromptTokens = 3
	fact.CompletionTokens = 2
	fact.TotalTokens = 5
	fact.BillableUsage = syntheticUncachedUsage(3, 2)
	fact.FinishedAt = r.now()
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	require.NoError(t, recorder.RecordInvocation(ctx, fact))
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 89, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 7, bucket.Consumed)
	var month memberAIPointMonthRow
	require.NoError(t, db.Take(&month).Error)
	require.Equal(t, orgresource.AIPointMonthStart(fact.StartedAt), month.MonthStart)
	require.Zero(t, month.Reserved)
	require.EqualValues(t, 7, month.Consumed)
	var debt organizationResourceDebtRow
	require.NoError(t, db.Take(&debt).Error)
	require.Zero(t, debt.Amount)
	var events []organizationResourceEventRow
	require.NoError(t, db.Order("created_at,event_id").Find(&events).Error)
	require.Len(t, events, 2)
	require.EqualValues(t, 23, events[1].GrossCredit)
	require.EqualValues(t, 4, events[1].DebtRepaid)
	require.EqualValues(t, 19, events[1].NetCredit)
	require.Error(t, r.SettleAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 999, fact.FinishedAt))
}

func TestModelPointsReleaseRequiresNativeNoDispatchAndFencesLateReserve(t *testing.T) {
	for _, scenario := range []string{"held-no-dispatch", "no-hold", "generic-failure", "unknown"} {
		t.Run(scenario, func(t *testing.T) {
			db, recorder, r, _, fact := modelPointFixture(t)
			ctx := context.Background()
			if scenario != "no-hold" {
				require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
			}
			if scenario != "unknown" {
				fact.Outcome = aicapability.InvocationFailed
				fact.FinishedAt = fact.StartedAt.Add(time.Second)
				fact.UsageKnown = true
				if scenario != "generic-failure" {
					fact.ErrorCode = "reservation_failed_before_dispatch"
				}
				err := recorder.RecordInvocation(ctx, fact)
				if scenario == "generic-failure" {
					require.ErrorIs(t, err, orgresource.ErrOwnerNotTerminal)
				} else {
					require.NoError(t, err)
				}
			}
			if scenario == "unknown" {
				require.ErrorIs(t, r.ReleaseAIInvocationUsage(ctx, "org", fact.InvocationID), orgresource.ErrOwnerNotTerminal)
			}
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			if scenario == "generic-failure" || scenario == "unknown" {
				require.EqualValues(t, 30, bucket.Reserved)
				require.EqualValues(t, 70, bucket.Available)
			} else {
				require.Zero(t, bucket.Reserved)
				require.EqualValues(t, 100, bucket.Available)
			}
			if scenario == "no-hold" {
				require.Error(t, r.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
			}
		})
	}
}

func TestModelPointsRecoverPersistedTerminalAfterSettlementRollback(t *testing.T) {
	db, recorder, r, auth, fact := modelPointFixture(t)
	ctx := context.Background()
	require.NoError(t, recorder.ReserveAIInvocationUsage(ctx, "org", "member", fact.InvocationID, 20, fact.StartedAt))
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_model_audit", func(tx *gorm.DB) {
		if tx.Statement.Table == "saas_organization_resource_audit_logs" {
			tx.AddError(errors.New("synthetic audit failure"))
		}
	}))
	fact.Outcome = aicapability.InvocationSucceeded
	fact.UsageKnown = true
	fact.PromptTokens = 3
	fact.TotalTokens = 3
	fact.BillableUsage = syntheticUncachedUsage(3, 0)
	fact.FinishedAt = fact.StartedAt.Add(time.Second)
	require.Error(t, recorder.RecordInvocation(ctx, fact))
	got, err := recorder.ReadModelInvocation(ctx, "org", fact.InvocationID)
	require.NoError(t, err)
	require.Equal(t, aicapability.InvocationSucceeded, got.Outcome)
	require.NoError(t, db.Callback().Create().Remove("reject_model_audit"))
	restarted, err := NewGormModelInvocationRepository(db, TransactionConfig{}, recorder, auth)
	require.NoError(t, err)
	restarted.now = r.now
	auth.err = orgresource.ErrForbidden
	n, err := restarted.RecoverDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = restarted.RecoverDue(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 97, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 3, bucket.Consumed)
}

func TestModelPointsAdmissionCannotExceedEitherLimitOrUseOldMonth(t *testing.T) {
	for _, scenario := range []string{"member-limit", "enterprise-balance", "revoked", "old-month"} {
		t.Run(scenario, func(t *testing.T) {
			db, recorder, r, auth, fact := modelPointFixture(t)
			switch scenario {
			case "member-limit":
				require.NoError(t, db.Model(&memberAIPointLimitRow{}).Where("organization_id = ?", "org").Update("monthly_limit", 29).Error)
			case "enterprise-balance":
				require.NoError(t, db.Model(&organizationResourceBucketRow{}).Where("organization_id = ?", "org").Update("available", 29).Error)
			case "revoked":
				auth.err = orgresource.ErrForbidden
			case "old-month":
				r.now = func() time.Time { return fact.StartedAt.Add(2 * time.Minute) }
			}
			require.Error(t, recorder.ReserveAIInvocationUsage(context.Background(), "org", "member", fact.InvocationID, 20, fact.StartedAt))
			var count int64
			require.NoError(t, db.Model(&organizationResourceReservationRow{}).Count(&count).Error)
			require.Zero(t, count)
			var bucket organizationResourceBucketRow
			require.NoError(t, db.Take(&bucket).Error)
			require.Zero(t, bucket.Reserved)
		})
	}
}

func TestModelPointsBoundedRecoveryAdvancesUnknownBeforeReadingProof(t *testing.T) {
	db, recorder, r, _, fact := modelPointFixture(t)
	ctx := context.Background()
	require.NoError(t, db.Model(&memberAIPointLimitRow{}).Where("organization_id = ?", "org").Update("monthly_limit", 3000).Error)
	require.NoError(t, db.Model(&organizationResourceBucketRow{}).Where("organization_id = ?", "org").Update("available", 3000).Error)
	for i := 0; i < 55; i++ {
		original := fact
		original.InvocationID = fmt.Sprintf("scan-%02d", i)
		_, err := recorder.ClaimInvocation(ctx, original)
		require.NoError(t, err)
		require.NoError(t, r.ReserveAIInvocationUsage(ctx, "org", "member", original.InvocationID, 20, original.StartedAt))
		require.NoError(t, db.Model(&organizationResourceReservationRow{}).Where("owner_attempt_id = ?", original.InvocationID).Update("next_check_at", fact.StartedAt.Add(time.Duration(i-55)*time.Second)).Error)
		if i >= 50 {
			recorder.SetUsageSettler(nil)
			original.Outcome = aicapability.InvocationSucceeded
			original.UsageKnown = true
			original.PromptTokens = 1
			original.TotalTokens = 1
			original.BillableUsage = syntheticUncachedUsage(1, 0)
			original.FinishedAt = original.StartedAt.Add(time.Second)
			require.NoError(t, recorder.RecordInvocation(ctx, original))
		}
	}
	n, err := r.RecoverDue(ctx)
	require.NoError(t, err)
	require.Zero(t, n)
	n, err = r.RecoverDue(ctx)
	require.NoError(t, err)
	require.Equal(t, 5, n)
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Take(&bucket).Error)
	require.EqualValues(t, 1500, bucket.Reserved)
	require.EqualValues(t, 5, bucket.Consumed)
}

// syntheticUncachedUsage is explicit synthetic provider evidence, not a fallback for absent partitions.
func syntheticUncachedUsage(input, output int64) *aicapability.BillableModelUsage {
	return &aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: input, OutputText: output}
}
