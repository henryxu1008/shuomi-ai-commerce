package skillpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	"testing"
)

func TestRuntimeGrantsKeepBodiesAndPolicyIdentityImmutable(t *testing.T) {
	db, _ := fixture(t)
	ctx := context.Background()
	role := "skill688_runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	inherited := role + "_source"
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS").Error)
	require.NoError(t, db.Exec("CREATE ROLE "+inherited+" NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS").Error)
	t.Cleanup(func() {
		require.NoError(t, db.Exec("DROP OWNED BY "+role+", "+inherited).Error)
		require.NoError(t, db.Exec("DROP ROLE "+role).Error)
		require.NoError(t, db.Exec("DROP ROLE "+inherited).Error)
	})
	require.NoError(t, configstore.GrantRuntime(db, role))
	require.NoError(t, GrantRuntime(ctx, db, role))
	var allowed bool
	require.NoError(t, db.Raw("SELECT has_column_privilege(?,'skill_capability.versions','files','UPDATE') OR has_column_privilege(?,'skill_capability.versions','digest','UPDATE') OR has_column_privilege(?,'agent_configuration.skill_admissions','organization_id','UPDATE') OR has_table_privilege(?,'skill_capability.skills','DELETE,TRUNCATE')", role, role, role, role).Scan(&allowed).Error)
	require.False(t, allowed)
	// Column/inherited privileges survive table-level revocation. Refuse the
	// unsafe effective authority instead of altering another role's grants.
	require.NoError(t, db.Exec("GRANT UPDATE(organization_id) ON agent_configuration.skill_admissions TO "+inherited).Error)
	require.NoError(t, db.Exec("GRANT "+inherited+" TO "+role).Error)
	require.Error(t, GrantRuntime(ctx, db, role))
	require.NoError(t, db.Exec("REVOKE UPDATE(organization_id) ON agent_configuration.skill_admissions FROM "+inherited).Error)
	require.NoError(t, db.Exec("GRANT UPDATE(allow_export) ON skill_capability.versions TO "+inherited).Error)
	require.Error(t, GrantRuntime(ctx, db, role))
}
