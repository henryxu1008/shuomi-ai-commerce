package orgresourceadapter

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"task-processor/internal/imageagent"
	"task-processor/internal/ledger/orgresource"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type imageFactTestOwner struct {
	mu   sync.Mutex
	fact imageagent.GenerationFact
}

func TestImageSettlementReleaseRepaysDebtBeforeAvailableAndReplays(t *testing.T) {
	db, repository, owner, receipt := reservedImageSettlementFixture(t)
	ctx := context.Background()
	require.NoError(t, db.Create(&organizationResourceDebtRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org-1", ResourceType: "ai_point", Amount: 5}).Error)
	var err error
	owner.fact, err = owner.fact.RecordNoGeneration()
	require.NoError(t, err)
	settled, err := repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, "released", settled.State)
	require.Equal(t, receipt.ReservationID, settled.ReservationID)
	assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 95, 0, 0)
	assertDebt(t, db, "org-1", orgresource.ResourceAIPoint, 0)
	var event organizationResourceEventRow
	require.NoError(t, db.Where("organization_id = ? AND operation_id = ?", "org-1", settled.OperationID).Take(&event).Error)
	require.Equal(t, "release", event.Reason)
	require.EqualValues(t, 12, event.GrossCredit)
	require.EqualValues(t, 5, event.DebtRepaid)
	require.EqualValues(t, 7, event.NetCredit)
	var month memberAIPointMonthRow
	require.NoError(t, db.Where("organization_id = ? AND member_id = ?", "org-1", "member-1").Take(&month).Error)
	require.Zero(t, month.Reserved)
	require.Zero(t, month.Consumed)
	again, err := repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, settled, again)
	for table, count := range map[string]int64{"saas_organization_resource_operations": 3, "saas_organization_resource_events": 2, "saas_organization_resource_audit_logs": 3} {
		assertTableCount(t, db, table, count)
	}
}

func TestImageSettlementRejectsBalanceOverflowWithoutPartialEffects(t *testing.T) {
	for _, balance := range []string{"available", "consumed"} {
		t.Run(balance, func(t *testing.T) {
			db, repository, owner, receipt := reservedImageSettlementFixture(t)
			ctx := context.Background()
			var err error
			if balance == "available" {
				owner.fact, err = owner.fact.RecordNoGeneration()
			} else {
				owner.fact, _, err = owner.fact.BeginDispatch()
				require.NoError(t, err)
				owner.fact, err = owner.fact.RecordSuccess(imageagent.GenerationSuccess{BillableUsage: syntheticImageUsage(100, 100), UsageKnown: true, InputTokens: 100, OutputTokens: 100, TotalTokens: 200, ResponseID: "response-1", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"})
			}
			require.NoError(t, err)
			require.NoError(t, db.Model(&organizationResourceBucketRow{}).Where("organization_id = ? AND resource_type = ?", "org-1", "ai_point").Update(balance, int64(math.MaxInt64)).Error)
			_, err = repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			require.ErrorIs(t, err, orgresource.ErrInvalidInput)
			available, consumed := int64(88), int64(0)
			if balance == "available" {
				available = math.MaxInt64
			} else {
				consumed = math.MaxInt64
			}
			assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, available, 12, consumed)
			assertImageSettlementStillReserved(t, db, owner.fact, receipt)
			require.NoError(t, db.Model(&organizationResourceBucketRow{}).Where("organization_id = ? AND resource_type = ?", "org-1", "ai_point").Updates(map[string]any{"available": 88, "consumed": 0}).Error)
			settled, err := repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			require.NoError(t, err)
			if balance == "consumed" {
				require.Equal(t, "committed", settled.State)
				var event organizationResourceEventRow
				require.NoError(t, db.Where("organization_id = ? AND operation_id = ?", "org-1", settled.OperationID).Take(&event).Error)
				require.Equal(t, "commit", event.Reason)
			} else {
				require.Equal(t, "released", settled.State)
			}
		})
	}
}

func TestImageSettlementEventOrAuditFailureRollsBackDebtAndEntireHold(t *testing.T) {
	for _, table := range []string{"saas_organization_resource_events", "saas_organization_resource_audit_logs"} {
		t.Run(table, func(t *testing.T) {
			db, repository, owner, receipt := reservedImageSettlementFixture(t)
			ctx := context.Background()
			require.NoError(t, db.Create(&organizationResourceDebtRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org-1", ResourceType: "ai_point", Amount: 5}).Error)
			var err error
			owner.fact, err = owner.fact.RecordNoGeneration()
			require.NoError(t, err)
			failure := errors.New("synthetic image settlement persistence failure")
			require.NoError(t, db.Callback().Create().Before("gorm:create").Register("reject_image_settlement", func(tx *gorm.DB) {
				if tx.Statement.Table == table {
					tx.AddError(failure)
				}
			}))
			_, err = repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			require.ErrorIs(t, err, failure)
			assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 88, 12, 0)
			assertDebt(t, db, "org-1", orgresource.ResourceAIPoint, 5)
			assertImageSettlementStillReserved(t, db, owner.fact, receipt)
			require.NoError(t, db.Callback().Create().Remove("reject_image_settlement"))
			settled, err := repository.FinalizeImageGeneration(ctx, owner.fact.Intent.Identity)
			require.NoError(t, err)
			require.Equal(t, "released", settled.State)
			assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 95, 0, 0)
			assertDebt(t, db, "org-1", orgresource.ResourceAIPoint, 0)
		})
	}
}

func reservedImageSettlementFixture(t *testing.T, reference ...imageagent.ImageBilling) (*gorm.DB, *GormImageGenerationRepository, *imageFactTestOwner, imageagent.GenerationReservationReceipt) {
	t.Helper()
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	now := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)
	limits, err := NewGormMemberLimitRepository(db, TransactionConfig{})
	require.NoError(t, err)
	limits.now = func() time.Time { return now }
	_, err = limits.SetMonthlyLimit(context.Background(), orgresource.SetMemberLimitExecution{UnitVersion: "ai-micropoint-v1", OrganizationID: "org-1", MemberID: "member-1", ActorID: "admin-1", OperationID: "limit-1", Target: 20})
	require.NoError(t, err)
	require.NoError(t, db.Omit("Reservations", "Debts").Create(&organizationResourceBucketRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org-1", ResourceType: "ai_point", Available: 100}).Error)
	fact, err := imageagent.NewGenerationFact(imageagent.GenerationIntent{Billing: syntheticImageBilling("gpt-image-2.5", "price-1", 12), Identity: imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: "org-1", OwnerUserID: "actor-1", RunID: "run-1"}, PlanRevision: 1, SlotID: "main-1", Attempt: 1}, MemberID: "member-1", CatalogHash: "catalog-v1:" + strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), PromptVersion: "prompt-1", RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1", Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price-1", Points: 12, LimitVersion: 1, MonthStart: orgresource.AIPointMonthStart(now)})
	require.NoError(t, err)
	if len(reference) == 1 {
		intent := fact.Intent
		intent.Billing = reference[0]
		intent.Provider, intent.Model, intent.Protocol = "openai", reference[0].OfficialTariff.Model, "openai-images-sync-v1"
		intent.PriceVersion = reference[0].OfficialTariff.PriceVersion
		fact, err = imageagent.NewGenerationFact(intent)
		require.NoError(t, err)
	}
	owner := &imageFactTestOwner{fact: fact}
	repository, err := NewGormImageGenerationRepository(db, TransactionConfig{}, owner, &imagePointTestAuthorizer{})
	require.NoError(t, err)
	repository.now = func() time.Time { return now }
	receipt, err := repository.ReserveImageGeneration(context.Background(), fact.Intent.Identity)
	require.NoError(t, err)
	owner.fact, err = owner.fact.BindReservation(receipt)
	require.NoError(t, err)
	return db, repository, owner, receipt
}

func assertImageSettlementStillReserved(t *testing.T, db *gorm.DB, fact imageagent.GenerationFact, receipt imageagent.GenerationReservationReceipt) {
	t.Helper()
	var reservation organizationResourceReservationRow
	require.NoError(t, db.Where("organization_id = ? AND reservation_id = ?", "org-1", receipt.ReservationID).Take(&reservation).Error)
	require.Equal(t, string(orgresource.ReservationReserved), reservation.State)
	require.Nil(t, reservation.SettlementOperationID)
	require.Nil(t, reservation.SettledAt)
	var month memberAIPointMonthRow
	require.NoError(t, db.Where("organization_id = ? AND member_id = ? AND month_start = ?", "org-1", "member-1", fact.Intent.MonthStart).Take(&month).Error)
	require.EqualValues(t, 12, month.Reserved)
	require.Zero(t, month.Consumed)
	for table, count := range map[string]int64{"saas_organization_resource_operations": 2, "saas_organization_resource_events": 1, "saas_organization_resource_audit_logs": 2} {
		assertTableCount(t, db, table, count)
	}
}

func (o *imageFactTestOwner) ReadGenerationFact(_ context.Context, id imageagent.SlotExternalEffectIdentity) (imageagent.GenerationFact, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.fact.Intent.Identity != id {
		return imageagent.GenerationFact{}, imageagent.ErrIdentityRequired
	}
	return o.fact, nil
}

type imagePointTestAuthorizer struct{ err error }

func (a *imagePointTestAuthorizer) AuthorizeImageGeneration(context.Context, imageagent.GenerationIntent) error {
	return a.err
}

func TestImagePointsReserveBothLimitsAndSettleOriginalMonth(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	ctx := context.Background()
	now := time.Date(2026, 9, 30, 23, 59, 0, 0, time.UTC)
	limits, err := NewGormMemberLimitRepository(db, TransactionConfig{})
	require.NoError(t, err)
	limits.now = func() time.Time { return now }
	set := orgresource.SetMemberLimitExecution{UnitVersion: "ai-micropoint-v1", OrganizationID: "org-1", MemberID: "member-1", ActorID: "admin-1", OperationID: "limit-1", Target: 20}
	_, err = limits.SetMonthlyLimit(ctx, set)
	require.NoError(t, err)
	require.NoError(t, db.Omit("Reservations", "Debts").Create(&organizationResourceBucketRow{QuantityUnit: resourceQuantityUnit("ai_point"), OrganizationID: "org-1", ResourceType: "ai_point", Available: 100}).Error)
	fact, err := imageagent.NewGenerationFact(imageagent.GenerationIntent{Billing: syntheticImageBilling("gpt-image-2.5", "price-1", 12), Identity: imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: "org-1", OwnerUserID: "actor-1", RunID: "run-1"}, PlanRevision: 1, SlotID: "main-1", Attempt: 1}, MemberID: "member-1", CatalogHash: "catalog-v1:" + strings.Repeat("a", 64), SourceDigest: strings.Repeat("b", 64), PromptVersion: "prompt-1", RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1", Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", PriceVersion: "price-1", Points: 12, LimitVersion: 1, MonthStart: orgresource.AIPointMonthStart(now)})
	require.NoError(t, err)
	owner := &imageFactTestOwner{fact: fact}
	auth := &imagePointTestAuthorizer{}
	r, err := NewGormImageGenerationRepository(db, TransactionConfig{}, owner, auth)
	require.NoError(t, err)
	r.now = func() time.Time { return now }
	receipt, err := r.ReserveImageGeneration(ctx, fact.Intent.Identity)
	require.NoError(t, err)
	replay, err := r.ReserveImageGeneration(ctx, fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	read, err := limits.ReadMonthlyLimit(ctx, "org-1", "member-1")
	require.NoError(t, err)
	require.EqualValues(t, 12, read.Reserved)
	require.Zero(t, read.Consumed)
	set.OperationID = "limit-too-low"
	set.ExpectedVersion = 1
	set.Target = 11
	_, err = limits.SetMonthlyLimit(ctx, set)
	require.ErrorIs(t, err, orgresource.ErrMemberLimitExceeded)
	set.OperationID = "raise-limit"
	set.Target = 30
	_, err = limits.SetMonthlyLimit(ctx, set)
	require.NoError(t, err)
	owner.fact, err = owner.fact.BindReservation(receipt)
	require.NoError(t, err)
	owner.fact, _, err = owner.fact.BeginDispatch()
	require.NoError(t, err)
	owner.fact, err = owner.fact.MarkUnknown()
	require.NoError(t, err)
	_, err = r.FinalizeImageGeneration(ctx, fact.Intent.Identity)
	require.ErrorIs(t, err, orgresource.ErrOwnerNotTerminal)
	now = now.Add(2 * time.Minute)
	auth.err = orgresource.ErrForbidden
	_, err = r.ReserveImageGeneration(ctx, fact.Intent.Identity)
	require.ErrorIs(t, err, orgresource.ErrForbidden)
	owner.fact, err = owner.fact.RecordSuccess(imageagent.GenerationSuccess{BillableUsage: syntheticImageUsage(100, 100), UsageKnown: true, InputTokens: 100, OutputTokens: 100, TotalTokens: 200, ResponseID: "response-1", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"})
	require.NoError(t, err)
	settled, err := r.FinalizeImageGeneration(ctx, fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, "committed", settled.State)
	again, err := r.FinalizeImageGeneration(ctx, fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, settled, again)
	// Audit reads the one canonical debit, not the reserve or a reconstructed
	// workflow outcome. Rebuilding the reader must preserve the same fact.
	auditReader, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	auditPage, err := auditReader.ListImagePointDebits(ctx, "org-1", "actor-1", 1, nil)
	require.NoError(t, err)
	require.Len(t, auditPage.Items, 1)
	require.Equal(t, "member-1", auditPage.Items[0].MemberID)
	require.Equal(t, "run-1", auditPage.Items[0].RunID)
	require.Equal(t, "price-1", auditPage.Items[0].PriceVersion)
	require.EqualValues(t, 12, auditPage.Items[0].Points)
	require.Equal(t, fact.IntentID, auditPage.Items[0].IntentID)
	require.Nil(t, auditPage.Next)
	for _, scope := range [][2]string{{"org-2", "actor-1"}, {"org-1", "other-actor"}} {
		empty, readErr := auditReader.ListImagePointDebits(ctx, scope[0], scope[1], 1, nil)
		require.NoError(t, readErr)
		require.Empty(t, empty.Items)
	}
	position := orgresource.ImagePointAuditPosition{CreatedAt: auditPage.Items[0].CreatedAt, EventID: auditPage.Items[0].EventID}
	nextAudit, err := auditReader.ListImagePointDebits(ctx, "org-1", "actor-1", 1, &position)
	require.NoError(t, err)
	require.Empty(t, nextAudit.Items)
	var old memberAIPointMonthRow
	require.NoError(t, db.Where("organization_id = ? AND member_id = ? AND month_start = ?", "org-1", "member-1", fact.Intent.MonthStart).Take(&old).Error)
	require.Zero(t, old.Reserved)
	require.EqualValues(t, 12, old.Consumed)
	current, err := limits.ReadMonthlyLimit(ctx, "org-1", "member-1")
	require.NoError(t, err)
	require.Zero(t, current.Reserved)
	require.Zero(t, current.Consumed)
	var bucket organizationResourceBucketRow
	require.NoError(t, db.Where("organization_id = ? AND resource_type = ?", "org-1", "ai_point").Take(&bucket).Error)
	require.EqualValues(t, 88, bucket.Available)
	require.Zero(t, bucket.Reserved)
	require.EqualValues(t, 12, bucket.Consumed)
	// A genuinely new attempt must satisfy both limits; no previous token
	// allocation or enterprise funds make an unassigned member eligible.
	for _, mode := range []string{"missing_member", "member_limit", "enterprise_balance"} {
		t.Run(mode, func(t *testing.T) {
			next := fact.Intent
			next.Identity.RunID = "run-" + mode
			next.MonthStart = orgresource.AIPointMonthStart(now)
			next.LimitVersion = 2
			switch mode {
			case "missing_member":
				next.MemberID = "unassigned"
			case "member_limit":
				next.Points = 31
			case "enterprise_balance":
				next.Points = 12
				require.NoError(t, db.Model(&organizationResourceBucketRow{}).Where("organization_id = ? AND resource_type = ?", "org-1", "ai_point").Update("available", 1).Error)
			}
			next.Billing = syntheticImageBilling(next.Model, next.PriceVersion, next.Points)
			owner.fact, err = imageagent.NewGenerationFact(next)
			require.NoError(t, err)
			auth.err = nil
			_, err = r.ReserveImageGeneration(ctx, next.Identity)
			want := orgresource.ErrMemberLimitUnavailable
			if mode == "member_limit" {
				want = orgresource.ErrMemberLimitExceeded
			}
			if mode == "enterprise_balance" {
				want = orgresource.ErrInsufficientBalance
			}
			require.ErrorIs(t, err, want)
			var count int64
			require.NoError(t, db.Model(&organizationResourceReservationRow{}).Count(&count).Error)
			require.EqualValues(t, 1, count)
		})
	}
}
