//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"testing"
)

func TestObservedModeUpdatePreservesHistoricalProposalAndPrivileges(t *testing.T) {
	db := workbenchDB(t)
	store, err := New(db)
	require.NoError(t, err)
	scope := aiworkbench.Scope{OrganizationID: "retained", ActorID: "owner"}
	_, p := readyTaskFixture(t, store, scope)
	var row proposalRow
	require.NoError(t, db.Where("id=?", p.ID).Take(&row).Error)
	row.ObservedModeRevision = ""
	oldDigest := proposalDigest(row)
	require.NoError(t, db.Exec("UPDATE ai_workbench.execution_proposals SET digest=? WHERE id=?", oldDigest, p.ID).Error)
	// Simulate the exact old schema inside this disposable fixture.
	require.NoError(t, db.Exec("ALTER TABLE ai_workbench.execution_proposals DROP COLUMN observed_mode_revision").Error)
	const role = "ai_workbench_runtime"
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	require.NoError(t, GrantRuntime(db, role))
	var before string
	require.NoError(t, db.Raw("SELECT relacl::text FROM pg_class WHERE oid='ai_workbench.execution_proposals'::regclass").Scan(&before).Error)
	require.NoError(t, InstallObservedExecutionMode(db, role))
	require.NoError(t, InstallObservedExecutionMode(db, role))
	var after string
	require.NoError(t, db.Raw("SELECT relacl::text FROM pg_class WHERE oid='ai_workbench.execution_proposals'::regclass").Scan(&after).Error)
	require.Equal(t, before, after)
	// The retained serving process reconnects after an explicit schema update.
	reopened, err := gorm.Open(postgres.New(postgres.Config{DSN: db.Dialector.(*postgres.Dialector).DSN, PreferSimpleProtocol: true}), &gorm.Config{})
	require.NoError(t, err)
	raw, err := reopened.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	store, err = New(reopened)
	require.NoError(t, err)
	retained, err := store.GetProposal(context.Background(), scope, p.ID)
	require.NoError(t, err)
	require.Equal(t, oldDigest, retained.Digest)
	require.Empty(t, retained.ObservedModeRevision)
	require.False(t, validProposal(&retained), "historical metadata cannot authorize a new proposal")
	var count int64
	require.NoError(t, db.Table("ai_workbench.execution_proposals").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.Error(t, db.Exec("UPDATE ai_workbench.execution_proposals SET observed_mode_revision='00' WHERE id=?", p.ID).Error)
}
