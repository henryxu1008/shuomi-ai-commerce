//go:build b1_local

package membership

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/authz"
	domain "task-processor/internal/organization/membership"
)

func TestB1PostgresNoEffectEvidenceAuditAndReservationAreAtomic(t *testing.T) {
	dsn := os.Getenv("B1_MEMBERSHIP_DATABASE_URL")
	if dsn == "" {
		t.Skip("explicit disposable B1_MEMBERSHIP_DATABASE_URL required")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1", u.Hostname())
	require.NotEmpty(t, u.Port())
	require.NotEqual(t, "5432", u.Port())
	require.Equal(t, "/b1_membership", u.Path)
	require.NotNil(t, u.User)
	require.Equal(t, "b1_implementation", u.User.Username())
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	ctx := context.Background()
	tx, err := sqlDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, InstallSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	repo, err := NewRepository(ctx, db, "project")
	require.NoError(t, err)
	for _, failAudit := range []bool{false, true} {
		t.Run(fmt.Sprint("audit_failure=", failAudit), func(t *testing.T) {
			key, target, role := uuid.NewString(), uuid.NewString(), authz.EnterpriseRoleKey("org", 2)
			op := domain.Operation{Scope: domain.OperationScope{ProjectID: "project", OrganizationID: "org", ActorID: "actor"}, Key: key, Fingerprint: strings.Repeat("a", 64), Kind: domain.CommandBusinessRoleGrant, TargetUserID: target, AuthorizationID: "grant-" + target, ExpectedVersion: strings.Repeat("b", 64), Role: role, Step: domain.StepConditionalRole, Phase: domain.PhaseReady, Revision: 1}
			op, err = repo.Begin(ctx, op)
			require.NoError(t, err)
			plan := &domain.ConditionalRolePlan{Native: domain.NativeAuthorizationRevision{InstanceID: "instance", OrganizationID: "org", ProjectID: "project", AuthorizationID: op.AuthorizationID, UserID: target, NativeSequence: "7", State: "active", Roles: []string{"listingkit_admin", "platform_admin"}}, Direction: "ADD", AfterRoles: []string{"listingkit_admin", "platform_admin", role}}
			op, err = repo.Apply(ctx, op.Scope, key, op.Revision, domain.OperationChange{Event: domain.EventDispatch, DispatchID: "attempt", ConditionalRole: plan})
			require.NoError(t, err)
			if failAudit {
				_, err = sqlDB.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE public.organization_member_audit_events ADD CONSTRAINT b1_force_audit_failure CHECK (operation_key <> '%s'::uuid)`, key))
				require.NoError(t, err)
				defer func() {
					_, cleanupErr := sqlDB.ExecContext(ctx, `ALTER TABLE public.organization_member_audit_events DROP CONSTRAINT b1_force_audit_failure`)
					require.NoError(t, cleanupErr)
				}()
			}
			e := &domain.RoleWriteEvidence{Result: domain.RoleWriteNoEffect, InstanceID: "instance", OrganizationID: "org", ProjectID: "project", AuthorizationID: op.AuthorizationID, UserID: target, ExpectedSequence: "7", RoleKey: role, Direction: "ADD", DispatchID: "attempt"}
			_, saveErr := repo.Apply(ctx, op.Scope, key, op.Revision, domain.OperationChange{Event: domain.EventProviderNoEffect, DispatchID: "attempt", NoEffect: e})
			read, err := repo.Read(ctx, op.Scope, key)
			require.NoError(t, err)
			var count int
			require.NoError(t, sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM public.organization_member_audit_events WHERE operation_key=$1 AND operation='role'`, key).Scan(&count))
			if failAudit {
				require.Error(t, saveErr)
				require.Equal(t, domain.PhaseDispatched, read.Phase)
				require.Nil(t, read.ProviderRoleResult)
				require.Zero(t, count)
			} else {
				require.NoError(t, saveErr)
				require.Equal(t, domain.PhaseRejected, read.Phase)
				require.NotNil(t, read.ProviderRoleResult)
				require.Equal(t, 1, count)
			}
			op.Key = uuid.NewString()
			op.Phase = domain.PhaseReady
			op.Revision = 1
			op.DispatchID = ""
			op.ConditionalRole = nil
			op.ProviderRoleResult = nil
			_, reserveErr := repo.Begin(ctx, op)
			if failAudit {
				require.True(t, errors.Is(reserveErr, domain.ErrConflict), "unconfirmed result released target")
			} else {
				require.NoError(t, reserveErr)
			}
		})
	}
}
