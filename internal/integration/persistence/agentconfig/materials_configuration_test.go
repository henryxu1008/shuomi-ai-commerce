package agentconfigpersistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
)

func priorMaterialsSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`ALTER TABLE agent_configuration.template_revisions
 DROP CONSTRAINT template_revisions_schema_version_check, DROP CONSTRAINT template_revisions_check,
 ADD CONSTRAINT template_revisions_schema_version_check `+contentSchemaVersionDDL+`,
 ADD CONSTRAINT template_revisions_check `+contentParametersDDL).Error)
}

func materialsRuntimeRole(t *testing.T, db *gorm.DB) string {
	t.Helper()
	role := "materials_runtime_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" NOLOGIN").Error)
	t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
	require.NoError(t, GrantRuntime(db, role))
	return role
}

func TestMaterialsSchemaUpdatePreservesExistingTemplatesAndContentReadiness(t *testing.T) {
	db, store, _ := fixture(t)
	priorMaterialsSchema(t, db)
	ctx := context.Background()
	require.NoError(t, VerifyContentConfigurationSchema(ctx, db))
	require.ErrorIs(t, VerifyMaterialsConfigurationSchema(ctx, db), agentconfig.ErrUnavailable)
	scope := agent.Scope{OrganizationID: "retained-org", ActorID: "actor"}
	enable := command(scope, "enable", 0)
	enable.Absent = true
	_, err := store.Execute(ctx, enable)
	require.NoError(t, err)
	create := command(scope, "create-template", 0)
	create.Input = agentconfig.TemplateInput{Name: "准确旧模板", TargetPlatform: "shein"}
	receipt, err := store.Execute(ctx, create)
	require.NoError(t, err)
	before, err := store.ReadTemplate(ctx, scope, create.AgentID, receipt.TemplateID, 1)
	require.NoError(t, err)
	role := materialsRuntimeRole(t, db)
	for n := 0; n < 2; n++ {
		require.NoError(t, UpdateMaterialsConfigurationSchema(ctx, db, role))
		require.NoError(t, VerifyMaterialsConfigurationSchema(ctx, db))
		require.NoError(t, VerifyContentConfigurationSchema(ctx, db), "new exact checks must retain content readiness")
		require.NoError(t, UpdateContentConfigurationSchema(ctx, db, role), "the original explicit content updater remains idempotent")
		after, err := store.ReadTemplate(ctx, scope, create.AgentID, receipt.TemplateID, 1)
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
	var count int64
	require.NoError(t, db.Table("agent_configuration.organization_agents").Where("agent_id=?", agentconfig.MaterialsAgentID).Count(&count).Error)
	require.Zero(t, count, "installer never activates, maps or backfills a new agent")
	var unsafe bool
	require.NoError(t, db.Raw(`SELECT has_schema_privilege(?,'agent_configuration','CREATE') OR has_any_column_privilege(?,'agent_configuration.template_revisions','UPDATE')`, role, role).Scan(&unsafe).Error)
	require.False(t, unsafe)
}

func TestMaterialsConfigurationStoresExactTemplateAndFrozenModeBinding(t *testing.T) {
	db, store, _ := fixture(t)
	ctx := context.Background()
	require.NoError(t, VerifyMaterialsConfigurationSchema(ctx, db))
	scope := agent.Scope{OrganizationID: "materials-org", ActorID: "actor"}
	enable := command(scope, "enable", 0)
	enable.AgentID = agentconfig.MaterialsAgentID
	enable.Absent = true
	_, err := store.Execute(ctx, enable)
	require.NoError(t, err)
	create := command(scope, "create-template", 0)
	create.AgentID = enable.AgentID
	create.Input = agentconfig.TemplateInput{Name: "平台资料模板", TargetPlatform: "shein"}
	r, err := store.Execute(ctx, create)
	require.NoError(t, err)
	template, err := store.ReadTemplate(ctx, scope, enable.AgentID, r.TemplateID, 1)
	require.NoError(t, err)
	require.Equal(t, agentconfig.MaterialsParameterSchema, template.SchemaVersion)
	require.Equal(t, create.Input, template.TemplateInput)
	profile := aicapability.ModelProfile{ClientName: "materials", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "model", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "routing", AdapterPolicyVersion: "adapter", PromptVersion: agentconfig.MaterialsPromptVersion, OutputSchemaVersion: agentconfig.MaterialsActionVersion, UsageMappingVersion: "usage", CostPricingVersion: "cost", PointTariff: aicapability.ModelPointTariff{PriceVersion: "points", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 1000000}, Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 1000, MaximumCompletionTokens: 100, MaximumInputBytes: 4096, MaximumOutputBytes: 4096, DeadlineBound: time.Minute}
	c := agentconfig.StartCommand{Scope: scope, MemberID: "member", ExpectedModeRevision: "0", AgentID: agentconfig.MaterialsAgentID, AgentVersion: agentconfig.MaterialsAgentVersion, Template: &agentconfig.TemplateRef{TemplateID: r.TemplateID, Revision: "1"}, ExecutionModelProfile: profile, Request: agent.Request{Key: uuid.NewString(), Binding: agent.Binding{ContextKind: "listing-target-record", ContextID: uuid.NewString(), ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}, PolicyVersion: agentconfig.MaterialsPolicyVersion, PromptVersion: agentconfig.MaterialsPromptVersion, Limits: agent.Limits{Steps: 6, ModelCalls: 3, Tokens: 2000, CostMicros: 2000, Currency: "CNY", Runtime: time.Minute}}}
	frozen, err := store.Prepare(ctx, c)
	require.NoError(t, err)
	ref := agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: frozen.ID, Digest: frozen.Digest}
	replay, err := store.Prepare(ctx, c)
	require.NoError(t, err)
	require.Equal(t, frozen, replay)
	loaded, err := store.LoadSnapshot(ctx, scope, ref)
	require.NoError(t, err)
	require.Equal(t, frozen, loaded)
	identity := agent.ExecutionIdentity{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: c.MemberID}
	mode, err := store.CheckRegisteredTextExecutionMode(ctx, identity, ref, frozen.Request.ModeSnapshotRef)
	require.NoError(t, err)
	require.Equal(t, agentconfig.MaterialsAgentID, mode.Subject.Subject.ID)
	changed := c
	changed.Request.Binding.ContextID = uuid.NewString()
	_, err = store.Match(ctx, changed, ref)
	require.ErrorIs(t, err, agentconfig.ErrConflict)
	changed = c
	changed.KnowledgeBaseID = uuid.NewString()
	_, err = store.Prepare(ctx, changed)
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
}

func TestMaterialsAndContentSchemaRejectChangedOrUnvalidatedContract(t *testing.T) {
	for _, sql := range []string{
		`ALTER TABLE agent_configuration.template_revisions DROP CONSTRAINT template_revisions_schema_version_check; ALTER TABLE agent_configuration.template_revisions ADD CONSTRAINT template_revisions_schema_version_check CHECK(true)`,
		`ALTER TABLE agent_configuration.template_revisions DROP CONSTRAINT template_revisions_check; ALTER TABLE agent_configuration.template_revisions ADD CONSTRAINT template_revisions_check CHECK(content_parameters IS NULL) NOT VALID`,
	} {
		db, _, _ := fixture(t)
		role := materialsRuntimeRole(t, db)
		require.NoError(t, db.Exec(sql).Error)
		require.ErrorIs(t, VerifyMaterialsConfigurationSchema(context.Background(), db), agentconfig.ErrUnavailable)
		require.ErrorIs(t, VerifyContentConfigurationSchema(context.Background(), db), agentconfig.ErrUnavailable)
		require.ErrorIs(t, UpdateMaterialsConfigurationSchema(context.Background(), db, role), agentconfig.ErrInvalid)
	}
}
