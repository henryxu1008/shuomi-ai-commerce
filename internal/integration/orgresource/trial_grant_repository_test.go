package orgresourceadapter

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"task-processor/internal/ledger/orgresource"
)

type trialRepositoryAuthorizer struct {
	approval orgresource.TrialAIGrantApproval
}

func (a trialRepositoryAuthorizer) AuthorizeTrialAIGrant(context.Context, orgresource.Principal, string) (orgresource.TrialAIGrantApproval, error) {
	return a.approval, nil
}

func trialRepositoryService(t *testing.T, db *gorm.DB) (*orgresource.TrialAIGrantService, *GormRepository, orgresource.TrialAIGrantInput, trialRepositoryAuthorizer) {
	t.Helper()
	repository, err := NewGormRepository(db, TransactionConfig{})
	require.NoError(t, err)
	auth := trialRepositoryAuthorizer{orgresource.TrialAIGrantApproval{OrganizationID: "trial-org", OperatorID: "trial-operator", EvidenceID: "human-trial-approval", GrantEnabled: true}}
	service, err := orgresource.NewTrialAIGrantService(repository, auth)
	require.NoError(t, err)
	input := orgresource.TrialAIGrantInput{OrganizationID: "trial-org", OperationID: "trial-operation", Principal: orgresource.Principal{ID: "trial-operator", Kind: orgresource.PrincipalTrustedTrialOperator}}
	return service, repository, input, auth
}

func TestTrialAIGrantPreservesDebtReservationAndImmutableReplay(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	require.NoError(t, db.Create(&organizationResourceBucketRow{OrganizationID: "trial-org", ResourceType: "ai_point", Available: 50, Reserved: 30, Consumed: 20}).Error)
	require.NoError(t, db.Create(&organizationResourceDebtRow{OrganizationID: "trial-org", ResourceType: "ai_point", Amount: 250}).Error)
	service, repository, input, auth := trialRepositoryService(t, db)
	first, err := service.Grant(context.Background(), input)
	require.NoError(t, err)
	require.False(t, first.Replayed)
	require.Equal(t, "10000", first.Snapshot.Quantity)
	require.Equal(t, "250", first.Snapshot.DebtRepaid)
	require.Equal(t, "9750", first.Snapshot.NetCredit)
	require.Equal(t, "9800", first.Snapshot.BalanceAfter)
	assertBucket(t, db, "trial-org", orgresource.ResourceAIPoint, 9800, 30, 20)
	assertDebt(t, db, "trial-org", orgresource.ResourceAIPoint, 0)
	for _, operation := range []string{input.OperationID, "another-operation"} {
		input.OperationID = operation
		replay, replayErr := service.Grant(context.Background(), input)
		require.NoError(t, replayErr)
		require.True(t, replay.Replayed)
		require.Equal(t, first.Snapshot, replay.Snapshot)
	}
	auth.approval.GrantEnabled = false
	service, err = orgresource.NewTrialAIGrantService(repository, auth)
	require.NoError(t, err)
	read, found, err := service.Inspect(context.Background(), input)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, first.Snapshot, read.Snapshot)
	auth.approval.GrantEnabled = true
	auth.approval.EvidenceID = "replaced-evidence"
	service, _ = orgresource.NewTrialAIGrantService(repository, auth)
	_, err = service.Grant(context.Background(), input)
	require.ErrorIs(t, err, orgresource.ErrIdempotencyKeyConflict)
	for _, table := range []string{"operations", "source_claims", "events", "audit_logs"} {
		assertTableCount(t, db, "saas_organization_resource_"+table, 1)
	}
	var operation organizationResourceOperationRow
	require.NoError(t, db.Take(&operation).Error)
	require.Equal(t, "human-trial-approval", operation.ApprovalEvidenceID)
}

func TestTrialAIGrantAuditFailureRollsBackAllEffects(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	require.NoError(t, db.Exec(`CREATE TRIGGER fail_trial_audit BEFORE INSERT ON saas_organization_resource_audit_logs BEGIN SELECT RAISE(ABORT,'audit unavailable'); END`).Error)
	service, _, input, _ := trialRepositoryService(t, db)
	_, err := service.Grant(context.Background(), input)
	require.Error(t, err)
	for _, table := range []string{"buckets", "debts", "operations", "source_claims", "events", "audit_logs"} {
		assertTableCount(t, db, "saas_organization_resource_"+table, 0)
	}
}

func TestTrialAIGrantLostCommitReplyReadsOriginalReceipt(t *testing.T) {
	db := openSQLiteStore(t)
	require.NoError(t, AutoMigrate(db))
	service, repository, input, _ := trialRepositoryService(t, db)
	repository.afterCommit = func() error { return errors.New("synthetic response loss") }
	result, err := service.Grant(context.Background(), input)
	require.NoError(t, err)
	require.True(t, result.Replayed)
	require.Equal(t, "10000", result.Snapshot.BalanceAfter)
	assertTableCount(t, db, "saas_organization_resource_events", 1)
}
