package agentconfigpersistence

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

const priorContentSchema = `ALTER TABLE agent_configuration.template_revisions
 DROP CONSTRAINT template_revisions_check,
 DROP CONSTRAINT template_revisions_schema_version_check,
 DROP COLUMN content_parameters,
 ADD CONSTRAINT template_revisions_schema_version_check CHECK(schema_version IN ('title-config-v1','image-config-v1')),
 ADD CONSTRAINT template_revisions_check CHECK((schema_version='title-config-v1' AND agent_id<>'product.image.agent' AND target_platform<>'product' AND image_parameters IS NULL)
 OR (schema_version='image-config-v1' AND agent_id='product.image.agent' AND default_knowledge_base_id IS NULL AND image_parameters IS NOT NULL AND octet_length(image_parameters) BETWEEN 2 AND 65536))`

func currentContentSchemaChecks(t *testing.T, db *gorm.DB) {
	t.Helper()
	var checks []struct{ Name, Definition string }
	require.NoError(t, db.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass AND conname IN ('template_revisions_check','template_revisions_schema_version_check') ORDER BY conname`).Scan(&checks).Error)
	t.Log(checks)
}

func TestContentSchemaAbsentLeavesCurrentTitleConfigurationUsable(t *testing.T) {
	db, store, _ := fixture(t)
	require.NoError(t, db.Exec(priorContentSchema).Error)
	require.NoError(t, VerifySchema(context.Background(), db))
	require.ErrorIs(t, VerifyContentConfigurationSchema(context.Background(), db), agentconfig.ErrUnavailable)
	scope := agent.Scope{OrganizationID: "title-org", ActorID: "actor"}
	enable := command(scope, "enable", 0)
	enable.Absent = true
	_, err := store.Execute(context.Background(), enable)
	require.NoError(t, err)
	create := command(scope, "create-template", 0)
	create.Input = agentconfig.TemplateInput{Name: "Title without content schema", TargetPlatform: "shein"}
	receipt, err := store.Execute(context.Background(), create)
	require.NoError(t, err, "ordinary title insert must not reference the uninstalled column")
	view, err := store.ReadTemplate(context.Background(), scope, create.AgentID, receipt.TemplateID, 1)
	require.NoError(t, err)
	require.Equal(t, create.Input, view.TemplateInput)
	require.Nil(t, view.Content)
}

func TestContentSchemaUpdatePreservesFactsAndRepeatsWithoutWrites(t *testing.T) {
	db, store, _ := fixture(t)
	require.NoError(t, db.Exec(priorContentSchema).Error)
	updated, shapeErr := verifyContentSchemaShape(db)
	if shapeErr != nil {
		currentContentSchemaChecks(t, db)
	}
	require.NoError(t, shapeErr)
	require.False(t, updated)
	scope := agent.Scope{OrganizationID: "existing-org", ActorID: "actor"}
	enable := command(scope, "enable", 0)
	enable.Absent = true
	_, err := store.Execute(context.Background(), enable)
	require.NoError(t, err)
	create := command(scope, "create-template", 0)
	create.Input = agentconfig.TemplateInput{Name: "Preserved title", TargetPlatform: "shein"}
	receipt, err := store.Execute(context.Background(), create)
	require.NoError(t, err)
	before, err := store.ReadTemplate(context.Background(), scope, create.AgentID, receipt.TemplateID, 1)
	require.NoError(t, err)
	role := "content_runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN").Error)
	t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
	require.NoError(t, GrantRuntime(db, role))
	for i := 0; i < 2; i++ {
		require.NoError(t, UpdateContentConfigurationSchema(context.Background(), db, role))
		require.NoError(t, VerifyContentConfigurationSchema(context.Background(), db))
		after, err := store.ReadTemplate(context.Background(), scope, create.AgentID, receipt.TemplateID, 1)
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
	var count int64
	require.NoError(t, db.Table("agent_configuration.template_revisions").Where("content_parameters IS NOT NULL").Count(&count).Error)
	require.Zero(t, count, "update does not backfill or create content facts")
	var unsafe bool
	require.NoError(t, db.Raw(`SELECT has_schema_privilege(?,'agent_configuration','CREATE') OR has_any_column_privilege(?,'agent_configuration.template_revisions','UPDATE')`, role, role).Scan(&unsafe).Error)
	require.False(t, unsafe)
}

func TestContentSchemaUpdateRejectsDriftBeforeChangingFactsOrColumns(t *testing.T) {
	for _, test := range []struct {
		name, sql string
		current   bool
	}{
		{"partial column", `ALTER TABLE agent_configuration.template_revisions ADD COLUMN content_parameters bytea`, false},
		{"wrong column type", `ALTER TABLE agent_configuration.template_revisions ADD COLUMN content_parameters text`, false},
		{"loosened schema check", `ALTER TABLE agent_configuration.template_revisions DROP CONSTRAINT template_revisions_schema_version_check; ALTER TABLE agent_configuration.template_revisions ADD CONSTRAINT template_revisions_schema_version_check CHECK(true)`, false},
		{"unknown agent check", `ALTER TABLE agent_configuration.template_revisions ADD CONSTRAINT unknown_agent_check CHECK(agent_id<>'product.content.agent')`, false},
		{"content default", `ALTER TABLE agent_configuration.template_revisions ALTER COLUMN content_parameters SET DEFAULT ''::bytea`, true},
		{"foreign table owner", `ALTER TABLE agent_configuration.template_revisions OWNER TO permission_test`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			db, _, _ := fixture(t)
			if !test.current {
				require.NoError(t, db.Exec(priorContentSchema).Error)
			}
			sql := test.sql
			if test.name == "foreign table owner" {
				// This isolated database must own its fixture role; the test
				// cannot depend on a role left by another package or CI setup.
				foreign := "content_foreign_" + strings.ReplaceAll(uuid.NewString(), "-", "")
				require.NoError(t, db.Exec("CREATE ROLE "+foreign+" NOLOGIN").Error)
				t.Cleanup(func() {
					_ = db.Exec("DROP OWNED BY " + foreign + " CASCADE").Error
					_ = db.Exec("DROP ROLE " + foreign).Error
				})
				sql = strings.ReplaceAll(sql, "permission_test", foreign)
			}
			require.NoError(t, db.Exec(sql).Error)
			role := "content_runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
			require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN").Error)
			t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
			require.NoError(t, GrantRuntime(db, role))
			var before []struct{ Name, Definition string }
			require.NoError(t, db.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass ORDER BY conname`).Scan(&before).Error)
			require.ErrorIs(t, UpdateContentConfigurationSchema(context.Background(), db, role), agentconfig.ErrInvalid)
			var after []struct{ Name, Definition string }
			require.NoError(t, db.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass ORDER BY conname`).Scan(&after).Error)
			require.Equal(t, before, after)
		})
	}
}
