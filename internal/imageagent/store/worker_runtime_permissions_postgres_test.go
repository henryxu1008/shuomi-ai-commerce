package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	aicapabilitystore "task-processor/internal/aicapability/store"
	"task-processor/internal/integration/openai"
	assetpersistence "task-processor/internal/integration/persistence/product/asset"
	routeconfigstore "task-processor/internal/integration/persistence/routeconfig"
)

func TestOrganizationImageWorkerRuntimeRoleHasOnlyCurrentWorkerGrants(t *testing.T) {
	if os.Getenv("ISSUE487_TEST_DSN") != "" && os.Getenv("ISSUE487_EXCLUSIVE_POSTGRES") != "ISOLATED_TRIAL_ONLY" {
		t.Skip("result=SKIP: requires isolated ISSUE487_TEST_DSN and exclusive confirmation")
	}
	dsn := imageRuntimePermissionTestDSN(t)
	ctx := context.Background()
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	rootPool, err := root.DB()
	require.NoError(t, err)
	name := "issue487_worker_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	role := "issue487_worker_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	owner, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	ownerPool, err := owner.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ownerPool.Close())
		require.NoError(t, root.Exec("DROP DATABASE "+name+" WITH (FORCE)").Error)
		require.NoError(t, root.Exec("DROP ROLE "+role).Error)
		require.NoError(t, rootPool.Close())
	})
	require.NoError(t, owner.AutoMigrate(&openai.AIClientCredential{}))
	require.NoError(t, aicapabilitystore.AutoMigrateInvocationLedger(owner))
	require.NoError(t, AutoMigrateOrganizationScope(owner))
	require.False(t, owner.Migrator().HasTable("image_agent_v2_slot_external_effects"), "new installs must not create the retired ledger")
	// Synthetic forbidden relation keeps the original excess-grant and access-denial assertions meaningful.
	require.NoError(t, owner.Exec(`CREATE TABLE image_agent_v2_slot_external_effects(value text)`).Error)
	require.NoError(t, assetpersistence.AutoMigrate(owner))
	require.NoError(t, owner.Exec(`CREATE TABLE unrelated_secret(value text)`).Error)
	require.NoError(t, owner.Exec("CREATE ROLE "+role+" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	sum := sha256.Sum256([]byte(dsn + role))
	password := hex.EncodeToString(sum[:])
	require.NoError(t, owner.Exec("ALTER ROLE "+role+" PASSWORD '"+password+"'").Error)
	require.NoError(t, grantOrganizationWorkerRuntimePermissions(ctx, owner, role))
	runtimeDB, err := gorm.Open(postgres.Open(dsn+" dbname="+name+" user="+role+" password="+password), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := runtimeDB.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(4)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
	require.Error(t, verifyOrganizationWorkerRuntimePermissions(ctx, owner, role))
	require.NoError(t, routeconfigstore.InstallSchema(ctx, owner))
	require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role), "installation alone does not require optional grants")
	require.Error(t, routeconfigstore.VerifyReader(ctx, runtimeDB, true), "managed startup still requires explicit reader grants")
	require.NoError(t, grantOrganizationWorkerRuntimePermissions(ctx, owner, role))
	require.Error(t, routeconfigstore.VerifyReader(ctx, runtimeDB, true), "base worker maintenance must not auto-grant managed reads")
	require.NoError(t, owner.Exec(routeconfigstore.GrantSQL(role, false)).Error)
	require.NoError(t, routeconfigstore.VerifyReader(ctx, runtimeDB, true))
	require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role), "explicitly installed managed image reads must be admitted")
	for _, relation := range []string{"ai_route_config_revisions", "ai_route_config_current", "ai_route_config_options", "ai_client_credential_bindings"} {
		t.Run("missing_managed_read_"+relation, func(t *testing.T) {
			require.NoError(t, owner.Exec("REVOKE SELECT ON "+relation+" FROM "+role).Error)
			require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
			require.Error(t, routeconfigstore.VerifyReader(ctx, runtimeDB, true))
			require.NoError(t, owner.Exec("GRANT SELECT ON "+relation+" TO "+role).Error)
			require.NoError(t, routeconfigstore.VerifyReader(ctx, runtimeDB, true))
		})
		t.Run("wrong_managed_kind_"+relation, func(t *testing.T) {
			kind := "TABLE"
			if relation == "ai_client_credential_bindings" {
				kind = "VIEW"
			}
			require.NoError(t, owner.Exec("REVOKE SELECT ON "+relation+" FROM "+role).Error)
			require.NoError(t, owner.Exec("ALTER "+kind+" "+relation+" RENAME TO original_managed_relation").Error)
			if kind == "TABLE" {
				require.NoError(t, owner.Exec("CREATE VIEW "+relation+" AS SELECT * FROM original_managed_relation").Error)
			} else {
				require.NoError(t, owner.Exec("CREATE TABLE "+relation+" AS SELECT * FROM original_managed_relation WITH NO DATA").Error)
			}
			require.NoError(t, owner.Exec("GRANT SELECT ON "+relation+" TO "+role).Error)
			require.Error(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
			if kind == "TABLE" {
				require.NoError(t, owner.Exec("DROP VIEW "+relation).Error)
			} else {
				require.NoError(t, owner.Exec("DROP TABLE "+relation).Error)
			}
			require.NoError(t, owner.Exec("ALTER "+kind+" original_managed_relation RENAME TO "+relation).Error)
			require.NoError(t, owner.Exec("GRANT SELECT ON "+relation+" TO "+role).Error)
			require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
		})
	}
	for _, tc := range []struct{ name, grant, revoke string }{
		{"delete_run", "GRANT DELETE ON image_agent_v2_runs TO " + role, "REVOKE DELETE ON image_agent_v2_runs FROM " + role},
		{"legacy_effect", "GRANT SELECT ON image_agent_v2_slot_external_effects TO " + role, "REVOKE SELECT ON image_agent_v2_slot_external_effects FROM " + role},
		{"other_table", "GRANT SELECT ON unrelated_secret TO " + role, "REVOKE SELECT ON unrelated_secret FROM " + role},
		{"missing_invocation_write", "REVOKE UPDATE ON ai_invocations FROM " + role, "GRANT UPDATE ON ai_invocations TO " + role},
		{"missing_asset_head_write", "REVOKE UPDATE ON product_approved_inventory_version_heads FROM " + role, "GRANT UPDATE ON product_approved_inventory_version_heads TO " + role},
		{"credential_write", "GRANT UPDATE ON ai_client_credentials TO " + role, "REVOKE UPDATE ON ai_client_credentials FROM " + role},
		{"managed_revision_write", "GRANT INSERT ON ai_route_config_revisions TO " + role, "REVOKE INSERT ON ai_route_config_revisions FROM " + role},
		{"managed_current_column_write", "GRANT UPDATE(revision_id) ON ai_route_config_current TO " + role, "REVOKE UPDATE(revision_id) ON ai_route_config_current FROM " + role},
		{"managed_metadata_write", "GRANT UPDATE ON ai_client_credential_bindings TO " + role, "REVOKE UPDATE ON ai_client_credential_bindings FROM " + role},
		{"managed_current_rls", "ALTER TABLE ai_route_config_current ENABLE ROW LEVEL SECURITY", "ALTER TABLE ai_route_config_current DISABLE ROW LEVEL SECURITY"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, owner.Exec(tc.grant).Error)
			require.Error(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
			require.NoError(t, owner.Exec(tc.revoke).Error)
			require.NoError(t, verifyOrganizationWorkerRuntimePermissions(ctx, runtimeDB, role))
		})
	}
	require.Error(t, runtimeDB.Exec("DELETE FROM image_agent_v2_runs").Error)
	require.Error(t, runtimeDB.Exec("SELECT value FROM unrelated_secret").Error)
	require.Error(t, runtimeDB.Exec("SELECT * FROM image_agent_v2_slot_external_effects").Error)
	require.Error(t, runtimeDB.Exec("UPDATE ai_client_credentials SET enabled=false").Error)
}
