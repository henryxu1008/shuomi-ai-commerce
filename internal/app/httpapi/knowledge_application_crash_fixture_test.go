//go:build integration

package httpapi

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/app/runtime/currentapplication"
	"task-processor/internal/app/schema/productagentruntime"
	registrySchema "task-processor/internal/app/schema/sourceaccountregistry"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
	resourceadapter "task-processor/internal/integration/orgresource"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	knowledgestore "task-processor/internal/integration/persistence/knowledge"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	"task-processor/internal/ledger/orgresource"
	productasset "task-processor/internal/product/asset"
)

type knowledgeApplicationFixture struct {
	core, commercial, agent, product, knowledgeDB *gorm.DB
	knowledge                                     *consumerKnowledgeFixture
	config                                        currentapplication.Config
	operation                                     acquisitionResultDTO
}

// Reuse the disposable PG and native installers; no supplied/shared DSN is
// consumed. Every serving connection is a non-owner, bounded runtime role.
func newKnowledgeApplicationFixture(t *testing.T) *knowledgeApplicationFixture {
	t.Helper()
	ctx := context.Background()
	root, connection := commercialPostgres(t)
	baseDSN := root.Dialector.(*postgres.Dialector).DSN
	password := strings.ReplaceAll(uuid.NewString(), "-", "")
	for _, role := range []string{"source_account_runtime", "commercial_owner_runtime", "product_agent_runtime", "title_credential_writer", "product_review_runtime", "asset_reader", "knowledge_runtime", "knowledge_owner", "product_agent_owner"} {
		require.NoError(t, root.Exec("CREATE ROLE "+role+" LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS PASSWORD '"+password+"'").Error)
	}
	// Existing acquisition fixture needs an initializer, never a serving role.
	require.NoError(t, root.Exec("CREATE ROLE issue398_owner LOGIN SUPERUSER PASSWORD '"+password+"'").Error)
	t.Setenv("ISSUE398_TEST_DSN", baseDSN+" user=issue398_owner password="+password)
	f := newAcquisitionHTTPFixture(t)
	seed := f.server(t)
	op := acquisitionHTTPCall(t, seed, "POST", productAcquisitionBase, "admin", "B", uuid.NewString(), `{"source":"https://detail.1688.com/offer/981645030344.html"}`, 200)
	seed.Close()
	var productName string
	require.NoError(t, f.owner.Raw("SELECT current_database()").Scan(&productName).Error)

	open := func(name, user string) *gorm.DB {
		dsn := baseDSN + " dbname=" + name
		if user != "issue347" {
			dsn += " user=" + user + " password=" + password
		}
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		// Never print a connection string or its credentials on failure.
		require.True(t, err == nil, "isolated database connection failed")
		pool, err := db.DB()
		require.NoError(t, err)
		pool.SetMaxOpenConns(4)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	create := func(name, owner string) *gorm.DB {
		require.NoError(t, root.Exec("CREATE DATABASE "+name+" OWNER "+owner).Error)
		require.NoError(t, root.Exec("REVOKE ALL ON DATABASE "+name+" FROM PUBLIC").Error)
		db := open(name, owner)
		require.NoError(t, db.Exec("REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
		return db
	}
	// Keep original fact owners separate, on the same newly created PG instance.
	commercial := create("commercial", "issue347")
	agentDB := create("product_agent", "product_agent_owner")
	knowledgeDB := create("knowledge", "knowledge_owner")
	require.NoError(t, registrySchema.Migrate(ctx, root))
	require.NoError(t, root.Exec("REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, root.Exec("GRANT CONNECT ON DATABASE issue347 TO source_account_runtime; GRANT USAGE ON SCHEMA public TO source_account_runtime").Error)
	for table, privileges := range run1AllowedPrivileges["source_account_runtime"] {
		require.NoError(t, root.Exec("GRANT "+strings.Join(privileges, ",")+" ON public."+table+" TO source_account_runtime").Error)
	}
	require.NoError(t, root.Exec("GRANT USAGE,SELECT ON SEQUENCE public.account_business_profile_audit_events_id_seq TO source_account_runtime").Error)

	require.NoError(t, commercialstore.AutoMigrate(commercial))
	require.NoError(t, resourceadapter.AutoMigrate(commercial))
	require.NoError(t, commercial.Exec("GRANT CONNECT ON DATABASE commercial TO commercial_owner_runtime; GRANT USAGE ON SCHEMA public TO commercial_owner_runtime; GRANT SELECT ON ALL TABLES IN SCHEMA public TO commercial_owner_runtime").Error)
	for _, table := range []string{"saas_organization_resource_buckets", "saas_organization_resource_operations", "saas_organization_resource_reservations", "saas_organization_resource_debts", "saas_member_ai_point_limits", "saas_member_ai_point_months", "saas_member_resource_positions"} {
		require.NoError(t, commercial.Exec("GRANT INSERT,UPDATE ON public."+table+" TO commercial_owner_runtime").Error)
	}
	for _, table := range []string{"saas_organization_resource_source_claims", "saas_organization_resource_events", "saas_organization_resource_audit_logs"} {
		require.NoError(t, commercial.Exec("GRANT INSERT ON public."+table+" TO commercial_owner_runtime").Error)
	}
	require.NoError(t, commercial.Exec("GRANT USAGE,SELECT ON ALL SEQUENCES IN SCHEMA public TO commercial_owner_runtime").Error)
	limit, err := resourceadapter.NewGormMemberLimitRepository(commercial, resourceadapter.TransactionConfig{})
	require.NoError(t, err)
	_, err = limit.SetMonthlyLimit(ctx, orgresource.SetMemberLimitExecution{OrganizationID: "B", MemberID: "grant-B", ActorID: "admin", OperationID: "synthetic-limit", Target: 10000000})
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, commercial.Exec("INSERT INTO saas_organization_resource_buckets (organization_id,resource_type,available,allocated,reserved,consumed,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?)", "B", "ai_point", 10000000, 0, 0, 0, now, now).Error)

	require.NoError(t, aistore.AutoMigrateInvocationLedger(agentDB))
	require.NoError(t, agentDB.Exec("GRANT SELECT ON public.ai_invocations TO product_agent_runtime").Error)
	// PrepareRun consumes the same owner/role/schema contract as the normal CLI.
	_, err = productagentruntime.PrepareRun(ctx, open("product_agent", "issue347"))
	require.NoError(t, err)
	initializer := open("product_agent", "issue347")
	require.NoError(t, configstore.InstallSchema(initializer))
	require.NoError(t, configstore.GrantRuntime(initializer, "product_agent_runtime"))
	configuration, err := configstore.New(initializer)
	require.NoError(t, err)
	_, err = configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AgentID: "product.title.agent", Operation: "enable", Absent: true})
	require.NoError(t, err)

	require.NoError(t, InstallProductReviewSchema(f.owner))
	require.NoError(t, assetstore.AutoMigrate(f.owner))
	for _, role := range []string{"product_review_runtime", "asset_reader"} {
		require.NoError(t, f.owner.Exec("GRANT CONNECT ON DATABASE "+productName+" TO "+role+"; GRANT USAGE ON SCHEMA public TO "+role).Error)
	}
	require.NoError(t, f.owner.Exec("GRANT SELECT,INSERT ON public.product_title_proposals,public.product_title_operations TO product_review_runtime; GRANT UPDATE(state,payload) ON public.product_title_proposals TO product_review_runtime; GRANT SELECT ON public.product_snapshot_versions,public.product_snapshot_heads,public.product_source_publications,public.product_source_publication_receipts TO product_review_runtime").Error)
	require.NoError(t, f.owner.Exec("GRANT SELECT ON public.product_approved_assets,public.product_approval_receipts,public.product_approved_inventory_heads,public.product_approved_inventory_version_heads TO asset_reader").Error)
	assets, err := assetstore.NewRepository(f.owner)
	require.NoError(t, err)
	version, err := strconv.ParseUint(op.CatalogVersion, 10, 64)
	require.NoError(t, err)
	_, err = assets.CommitApproval(ctx, productasset.ApprovalCommit{TenantID: "B", ProductKey: op.ProductKey, TargetPlatform: "shein", SourceSnapshotVersion: version, ActionID: "synthetic-approved", Assets: []productasset.ApprovedAsset{{ID: "synthetic-main", RunID: "synthetic-image", PlanRevision: 1, SlotID: "main", Attempt: 1, Role: productasset.RoleMain, URL: "https://example.test/main.png"}}})
	require.NoError(t, err)
	kf := newConsumerKnowledgeFixture(t, knowledgeDB)
	require.NoError(t, knowledgestore.GrantRuntime(ctx, knowledgeDB))
	// The disposable cluster's default databases must not grant cross-owner
	// CONNECT/TEMP through PUBLIC. This is also enforced by normal startup.
	var databases []string
	require.NoError(t, root.Raw("SELECT datname FROM pg_database WHERE datallowconn").Scan(&databases).Error)
	for _, name := range databases {
		require.NoError(t, root.Exec("REVOKE ALL ON DATABASE "+pgx.Identifier{name}.Sanitize()+" FROM PUBLIC").Error)
	}
	require.NoError(t, knowledgestore.VerifyRuntime(ctx, open("knowledge", "knowledge_runtime")))

	dbConfig := func(name, role string) currentapplication.DatabaseConfig {
		return currentapplication.DatabaseConfig{Host: "127.0.0.1", Port: connection.Port, User: role, Password: password, Database: name, MaxConnections: 4}
	}
	commercialConfig, productConfig := dbConfig("commercial", "commercial_owner_runtime"), dbConfig(productName, "source_acquisition_runtime")
	// The acquisition fixture chooses its own runtime password; use its exact
	// generated credential, not a privileged owner connection.
	productDSN := f.db.Dialector.(*postgres.Dialector).DSN
	parsed := strings.Split(productDSN, "password=")
	productConfig.Password = strings.Fields(parsed[len(parsed)-1])[0]
	cfg := currentapplication.Config{SchemaVersion: 1, Listen: currentapplication.ListenConfig{Host: "127.0.0.1"}, SourceAccountDatabase: dbConfig("issue347", "source_account_runtime"), CommercialOwnerDatabase: &commercialConfig, ProductAcquisitionDatabase: &productConfig,
		ProductAgent: &currentapplication.ProductAgentConfig{Enabled: true, Database: dbConfig("product_agent", "product_agent_runtime"), ReviewDatabase: dbConfig(productName, "product_review_runtime"), AssetDatabase: dbConfig(productName, "asset_reader"), AllowedOrganizationIDs: []string{"B"}, Currency: "CNY", Steps: 12, ModelCalls: 6, Tokens: 5000000, CostMicros: 5000000, RuntimeSeconds: 60},
		Knowledge:    &currentapplication.KnowledgeConfig{Enabled: true, Database: dbConfig("knowledge", "knowledge_runtime"), Storage: currentapplication.KnowledgeStorageConfig{Region: "us-east-1", Bucket: "synthetic", AccessKeyID: "synthetic-access", SecretAccessKey: "synthetic-secret", Mode: "aws"}}}
	return &knowledgeApplicationFixture{core: root, commercial: commercial, agent: agentDB, product: f.owner, knowledgeDB: knowledgeDB, knowledge: kf, config: cfg, operation: op}
}

func (f *knowledgeApplicationFixture) configureHTTP(t *testing.T, identityOrigin, modelOrigin, sinkOrigin string) {
	t.Helper()
	f.config.Identity = currentapplication.IdentityConfig{IssuerURL: identityOrigin, AuthorizationAPIURL: identityOrigin, ClientID: "fixture-client", ClientSecret: "fixture-secret", ProjectID: "project"}
	f.config.Knowledge.Storage.Endpoint = sinkOrigin
	f.config.Knowledge.ParserEndpoint = sinkOrigin
	credential := openai.AIClientCredential{TenantID: "B", ClientName: "default", APIKey: "synthetic-only", BaseURL: modelOrigin + "/v1", Model: "gemini-2.5-flash", APIStyle: "grsai", Enabled: true, TimeoutSecond: 3}
	credentials := openai.NewOrganizationOnlyCredentialResolver(f.agent)
	require.NoError(t, credentials.SaveCredential(context.Background(), credential))
	row, err := credentials.GetCredential(context.Background(), "B", "", "default")
	require.NoError(t, err)
	f.config.ProductAgent.TextPolicies = map[string]governed.RoutePolicy{"B": {AdmittedCredentialVersion: governed.CredentialVersion(*row), AdmittedEndpointIdentityDigest: governed.EndpointIdentityDigest(credential.BaseURL), Profile: aicapability.ModelProfile{ClientName: "default", ProviderID: "grsai", AdapterKind: "openai-compatible", ModelID: credential.Model, RoutingPolicyVersion: "route-v1", AdapterPolicyVersion: "adapter-v1", PromptVersion: "product-title-agent-v1", OutputSchemaVersion: "product-title-action-v1", UsageMappingVersion: "usage-v1", CostPricingVersion: "fixture-v1", PointTariff: aicapability.ModelPointTariff{PriceVersion: "synthetic-points-v1", InputPointsPerMillionTokens: 1000000, OutputPointsPerMillionTokens: 2000000}, Currency: "CNY", InputMicrosPerMillion: 300000, OutputMicrosPerMillion: 2000000, MaximumPromptTokens: 1048576, MaximumCompletionTokens: 8192, MaximumInputBytes: 64 << 10, MaximumOutputBytes: 16 << 10, DeadlineBound: 3 * time.Second}}}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	f.config.Listen.Port = listener.Addr().(*net.TCPAddr).Port
	require.NoError(t, listener.Close())
	// Validate every configured remote destination before the binary can run.
	for _, origin := range []string{identityOrigin, modelOrigin, sinkOrigin} {
		require.True(t, strings.HasPrefix(origin, "http://127.0.0.1:"), "only test-owned loopback HTTP peers are admitted")
	}
}

func applicationCrashCount(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Table(table).Count(&count).Error)
	return count
}

func applicationCrashAddress(cfg currentapplication.Config) string {
	return fmt.Sprintf("http://%s", cfg.ListenAddress())
}
