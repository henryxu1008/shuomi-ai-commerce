package orgresourceadapter

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/imageagent"
	"task-processor/internal/ledger/orgresource"
	"testing"
)

func TestImageOfficialActualUsageReleasesRemainderDebtFirstExactlyOnce(t *testing.T) {
	db, repo, owner, reserved := reservedImageSettlementFixture(t)
	require.NoError(t, db.Create(&organizationResourceDebtRow{QuantityUnit: orgresource.AIPointQuantityUnitV1, OrganizationID: "org-1", ResourceType: "ai_point", Amount: 2}).Error)
	var err error
	owner.fact, _, err = owner.fact.BeginDispatch()
	require.NoError(t, err)
	owner.fact, err = owner.fact.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "synthetic-result", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result", UsageKnown: true, InputTokens: 50, OutputTokens: 50, TotalTokens: 100, BillableUsage: syntheticImageUsage(50, 50)})
	require.NoError(t, err)
	settled, err := repo.FinalizeImageGeneration(context.Background(), owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.EqualValues(t, 12, reserved.Points)
	require.EqualValues(t, 12, settled.ReservedMicropoints)
	require.EqualValues(t, 6, settled.Points, "actual official usage, not the old fixed per-image amount")
	require.Equal(t, orgresource.AIPointQuantityUnitV1, settled.UnitVersion)
	assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 92, 0, 6)
	assertDebt(t, db, "org-1", orgresource.ResourceAIPoint, 0)
	again, err := repo.FinalizeImageGeneration(context.Background(), owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, settled, again)
	var event organizationResourceEventRow
	require.NoError(t, db.Where("operation_id = ?", settled.OperationID).Take(&event).Error)
	require.EqualValues(t, 6, event.ConsumedDelta)
	require.EqualValues(t, 6, event.GrossCredit)
	require.EqualValues(t, 2, event.DebtRepaid)
	require.EqualValues(t, 4, event.NetCredit)
}
func TestImageMissingOrOutOfBoundActualUsageKeepsOriginalReservation(t *testing.T) {
	for _, kind := range []string{"missing", "incomplete", "over-bound"} {
		t.Run(kind, func(t *testing.T) {
			db, repo, owner, _ := reservedImageSettlementFixture(t)
			var err error
			owner.fact, _, err = owner.fact.BeginDispatch()
			require.NoError(t, err)
			success := imageagent.GenerationSuccess{ResponseID: "synthetic-result", ResultDigest: strings.Repeat("c", 64), ResultUnavailable: "invalid_result"}
			if kind != "missing" {
				success.UsageKnown = true
				success.InputTokens = 101
				success.TotalTokens = 101
				success.BillableUsage = syntheticImageUsage(101, 0)
				if kind == "incomplete" {
					success.BillableUsage.Complete = false
				}
			}
			owner.fact, err = owner.fact.RecordSuccess(success)
			require.NoError(t, err)
			_, err = repo.FinalizeImageGeneration(context.Background(), owner.fact.Intent.Identity)
			require.ErrorIs(t, err, orgresource.ErrOwnerNotTerminal)
			assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 88, 12, 0)
		})
	}
}

func TestPositiveImageUsageCanCommitZeroMicroPointsAndReleaseOnlyItsOriginalReserve(t *testing.T) {
	db, repo, owner, reserved := reservedImageSettlementFixture(t)
	var err error
	owner.fact, _, err = owner.fact.BeginDispatch()
	require.NoError(t, err)
	owner.fact, err = owner.fact.RecordSuccess(imageagent.GenerationSuccess{ResponseID: "synthetic-tiny-result", ResultDigest: strings.Repeat("d", 64), ResultUnavailable: "invalid_result", UsageKnown: true, InputTokens: 1, TotalTokens: 1, BillableUsage: syntheticImageUsage(1, 0)})
	require.NoError(t, err)
	settled, err := repo.FinalizeImageGeneration(context.Background(), owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, "committed", settled.State)
	require.Zero(t, settled.Points)
	require.Equal(t, reserved.Points, settled.ReservedMicropoints)
	require.Equal(t, orgresource.AIPointQuantityUnitV1, settled.UnitVersion)
	assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 100, 0, 0)
	again, err := repo.FinalizeImageGeneration(context.Background(), owner.fact.Intent.Identity)
	require.NoError(t, err)
	require.Equal(t, settled, again)
	assertBucket(t, db, "org-1", orgresource.ResourceAIPoint, 100, 0, 0)
}
