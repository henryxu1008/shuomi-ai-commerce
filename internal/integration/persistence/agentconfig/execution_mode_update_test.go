package agentconfigpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"testing"
)

func TestExecutionModeUpdateRetainsFactsAndExistingGrants(t *testing.T) {
	db, store, _ := fixture(t)
	scope := agent.Scope{OrganizationID: "retained", ActorID: "admin"}
	c := command(scope, "enable", 0)
	c.Absent = true
	before, err := store.Execute(context.Background(), c)
	require.NoError(t, err)
	role := "mode_update_" + uuid.NewString()[:8]
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS").Error)
	require.NoError(t, GrantRuntime(db, role))
	// This is the disposable fixture's pre-update schema, never a retained app.
	require.NoError(t, db.Exec("DROP TABLE agent_configuration.execution_mode_snapshots, agent_configuration.execution_modes").Error)
	var aclBefore string
	require.NoError(t, db.Raw(`SELECT relacl::text FROM pg_class WHERE oid='agent_configuration.commands'::regclass`).Scan(&aclBefore).Error)
	require.NoError(t, InstallExecutionModes(db, role))
	require.NoError(t, InstallExecutionModes(db, role))
	var aclAfter string
	require.NoError(t, db.Raw(`SELECT relacl::text FROM pg_class WHERE oid='agent_configuration.commands'::regclass`).Scan(&aclAfter).Error)
	require.Equal(t, aclBefore, aclAfter)
	after, err := store.Execute(context.Background(), c)
	require.NoError(t, err)
	require.Equal(t, before, after)
	var count int64
	require.NoError(t, db.Table("agent_configuration.execution_modes").Count(&count).Error)
	require.Zero(t, count)
	var granted bool
	require.NoError(t, db.Raw(`SELECT has_table_privilege(?,'agent_configuration.execution_mode_snapshots','UPDATE,DELETE,TRUNCATE')`, role).Scan(&granted).Error)
	require.False(t, granted)
	require.NoError(t, db.Raw(`SELECT has_column_privilege(?,'agent_configuration.execution_modes','mode','UPDATE')`, role).Scan(&granted).Error)
	require.True(t, granted)
}
