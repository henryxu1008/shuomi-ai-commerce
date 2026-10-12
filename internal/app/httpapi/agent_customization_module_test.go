package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	confighttp "task-processor/internal/agentconfig/httpapi"
	d "task-processor/internal/agentcustomization"
	customhttp "task-processor/internal/agentcustomization/httpapi"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	store "task-processor/internal/integration/persistence/agentcustomization"
	kernelmodule "task-processor/internal/kernel/module"
)

func TestAgentCustomizationModuleRejectsMissingDuplicateAndAliasedPool(t *testing.T) {
	source, custom, tool, configuration := &gorm.DB{}, &gorm.DB{}, &gorm.DB{}, &gorm.DB{}
	for _, candidate := range []struct {
		name    string
		options []CurrentApplicationOption
		allowed bool
	}{
		{"disabled", nil, true},
		{"independent", []CurrentApplicationOption{WithAgentCustomization(custom)}, true},
		{"quality execution missing mode owner", []CurrentApplicationOption{WithAgentCustomization(custom), WithSupplyChain(SupplyChainDependencies{AssetDB: tool})}, false},
		{"quality execution mode owner", []CurrentApplicationOption{WithAgentCustomization(custom), WithSupplyChain(SupplyChainDependencies{AssetDB: tool}), WithAgentConfiguration(configuration)}, true},
		{"quality execution aliased mode owner", []CurrentApplicationOption{WithAgentCustomization(custom), WithSupplyChain(SupplyChainDependencies{AssetDB: tool}), WithAgentConfiguration(custom)}, false},
		{"missing", []CurrentApplicationOption{WithAgentCustomization(nil)}, false},
		{"duplicate", []CurrentApplicationOption{WithAgentCustomization(custom), WithAgentCustomization(custom)}, false},
		{"source alias", []CurrentApplicationOption{WithAgentCustomization(source)}, false},
		{"tool market independent", []CurrentApplicationOption{WithAgentCustomization(custom), WithToolMarket(ToolMarketDependencies{DB: tool})}, true},
		{"tool market alias", []CurrentApplicationOption{WithAgentCustomization(custom), WithToolMarket(ToolMarketDependencies{DB: custom})}, false},
		{"project independent", []CurrentApplicationOption{WithAgentCustomization(custom), WithProjectCenter(tool)}, true},
		{"project alias", []CurrentApplicationOption{WithAgentCustomization(custom), WithProjectCenter(custom)}, false},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			options := currentApplicationOptions{}
			for _, apply := range candidate.options {
				apply(&options)
			}
			err := validateAgentCustomizationPool(options, source)
			if candidate.allowed {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAgentCustomizationModulePostgresNormalMiddlewareFlow(t *testing.T) {
	dsn := os.Getenv("ISSUE611_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated ISSUE611_TEST_DSN")
	}
	ctx := context.Background()
	root, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	name := "module611_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	_, err = root.Exec("CREATE DATABASE " + name)
	require.NoError(t, err)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = "/" + name
	db, err := sql.Open("pgx", u.String())
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	role := "module611_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	t.Cleanup(func() {
		db.Exec("RESET ROLE")
		db.Exec("DROP OWNED BY " + role)
		db.Exec("DROP ROLE " + role)
		db.Close()
		root.Exec("DROP DATABASE " + name + " WITH (FORCE)")
		root.Close()
	})
	require.NoError(t, store.InstallSchema(ctx, db))
	modeORM, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
	require.NoError(t, err)
	modePool, err := modeORM.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = modePool.Close() })
	require.NoError(t, configstore.InstallSchema(modeORM))
	modeRepo, err := configstore.New(modeORM)
	require.NoError(t, err)
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	_, err = buildAgentCustomizationModule(ctx, orm)
	require.ErrorIs(t, err, d.ErrUnavailable, "installer is not a serving identity")
	_, err = db.Exec("CREATE ROLE " + role + " NOLOGIN")
	require.NoError(t, err)
	require.NoError(t, store.GrantRuntime(ctx, db, role))
	_, err = db.Exec("SET ROLE " + role)
	require.NoError(t, err)
	draftProbe := &customizationDraftProbe{record: uuid.NewString()}
	m, err := buildAgentCustomizationModule(ctx, orm, draftProbe)
	require.NoError(t, err)
	privateService := m.(agentCustomizationModule).handler.Service.(*d.Service)
	modeHandler := &confighttp.Handler{Repository: modeRepo, Catalog: productAgentCatalog{}, Authorize: func(ctx context.Context, _ ...string) (agent.Scope, error) {
		i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok || i.EffectiveOrganizationID != i.TenantID || i.EffectiveMemberID == "" {
			return agent.Scope{}, agentconfig.ErrForbidden
		}
		return agent.Scope{OrganizationID: i.TenantID, ActorID: i.UserID}, nil
	}}
	privateService.WithExecutionModes(privateQualityMode{repository: modeRepo, resolve: agentModeSubjectResolver(modeHandler, privateService), identity: func(ctx context.Context) (agent.ExecutionIdentity, error) {
		return privateQualityExecutionIdentity(ctx, modeHandler)
	}})
	registry := kernelmodule.NewRegistry()
	require.NoError(t, m.Register(registry))
	require.Len(t, registry.Routes(), 13)
	authorizer, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	identity := authidentity.AuthenticatedIdentity{UserID: "verified-user", EffectiveMemberID: "private-mode-member", Roles: []string{"platform_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)}
	events := []string{}
	router := gin.New()
	mountRoutesWithAuthDependencies(router, registry.Routes(), routeAuthDependencies{
		workbenchVerifier:    mountedVerifierStub{identity: identity},
		organizationResolver: mountedOrganizationResolverStub{events: &events, roles: []string{"listingkit_admin"}},
		authorizer:           authorizer, auditRecorder: &mountedWorkbenchAuditStub{},
	})
	request := func(method, path, body, key, version string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer synthetic-verified-token")
		r.Header.Set("X-Requested-Organization-ID", "org-a")
		if body != "" {
			r.Header.Set("Content-Type", "application/json")
		}
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		if version != "" {
			r.Header.Set("If-Match", `"`+version+`"`)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	key := uuid.NewString()
	body := `{"name":"需求","scenario":"商品维护","direction":"OTHER","description":"整理资料","contactName":"测试","contactMethod":"test-only","consent":true}`
	w := request("POST", customhttp.Base, body, key, "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var receipt d.Receipt
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &receipt))
	replayed := request("POST", customhttp.Base, body, key, "")
	require.Equal(t, w.Body.String(), replayed.Body.String())
	for i, update := range []string{
		`{"stage":"EVALUATING","note":"专员评估"}`,
		`{"stage":"PROPOSED","note":"给出方案","proposal":"方案和报价在线下确认"}`,
		`{"stage":"DEVELOPING","note":"开始开发","offlineConfirmation":"线下方案费用已确认"}`,
		`{"stage":"DELIVERED","note":"交付操作说明","deliverQualityAgent":true}`,
	} {
		w = request("POST", customhttp.AdminBase+"/"+receipt.RequestID+"/progress", update, uuid.NewString(), receipt.Version)
		require.Equal(t, 200, w.Code, w.Body.String())
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &receipt))
		require.Equal(t, d.Stages[i+1], receipt.Stage)
	}
	w = request("GET", customhttp.Base+"/"+receipt.RequestID, "", "", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var detail d.Detail
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &detail))
	require.Equal(t, "org-a", detail.Request.OrganizationID)
	require.Equal(t, "verified-user", detail.Request.CreatedBy)
	require.Equal(t, d.Delivered, detail.Request.Stage)
	require.Len(t, detail.Events, 5)
	require.NotEmpty(t, detail.Request.DeliveryID)
	w = request("GET", customhttp.PrivateBase, "", "", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var deliveries d.DeliveryPage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &deliveries))
	require.Len(t, deliveries.Items, 1)
	require.Equal(t, "2.0.0", deliveries.Items[0].Version, "new deliveries consume platform drafts rather than manual input")
	reportKey := uuid.NewString()
	reportPath := customhttp.PrivateBase + "/" + deliveries.Items[0].ID + "/reports"
	w = request("POST", reportPath, `{"name":"收纳盒","material":"","dimensions":"20x10","description":"","specifications":[]}`, reportKey, "")
	require.Equal(t, 400, w.Code, "manual execution is retired")
	w = request("POST", reportPath, `{"recordId":"`+draftProbe.record+`","expectedRevision":1,"expectedModeRevision":"0"}`, reportKey, "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var report d.QualityRun
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &report))
	require.Equal(t, agent.ModeDefault, report.ExecutionMode)
	require.True(t, report.ModeSnapshotRef.Valid())
	require.NotEmpty(t, report.Draft.Issues)
	w = request("GET", reportPath, "", "", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var reports d.QualityRunPage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &reports))
	require.Len(t, reports.Items, 1)
	require.Equal(t, report.ID, reports.Items[0].ID)
	w = request("GET", reportPath+"/"+report.ID, "", "", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	var readBack d.QualityRun
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &readBack))
	require.Equal(t, report, readBack)
	modeIdentity := agent.ExecutionIdentity{OrganizationID: report.OrganizationID, ActorID: report.ActorID, MemberID: identity.EffectiveMemberID}
	modeSubject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.PrivateDeliverySubject, ID: report.DeliveryID}, Definition: commercetool.AgentRef{ID: report.Definition, Version: report.Version}}
	for index, mode := range []agent.ExecutionMode{agent.ModeAutoApprove, agent.ModeReadOnly, agent.ModeFullAccess} {
		receipt, err := modeRepo.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: modeIdentity, Subject: modeSubject, Key: uuid.NewString(), Mode: mode, ExpectedRevision: uint64(index)})
		require.NoError(t, err)
		w = request("POST", reportPath, `{"recordId":"`+draftProbe.record+`","expectedRevision":1,"expectedModeRevision":"`+receipt.Revision+`"}`, uuid.NewString(), "")
		require.Equal(t, 200, w.Code, w.Body.String())
		var current d.QualityRun
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &current))
		require.Equal(t, mode, current.ExecutionMode)
		require.Equal(t, receipt.Revision, current.ModeRevision)
		require.Equal(t, report.Draft, current.Draft, "all modes remain diagnostic")
	}
	w = request("GET", reportPath+"/"+report.ID, "", "", "")
	require.Equal(t, 200, w.Code, w.Body.String())
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &readBack))
	require.Equal(t, agent.ModeDefault, readBack.ExecutionMode, "history retains its frozen mode after widening")
	w = request("POST", reportPath, `{"recordId":"`+draftProbe.record+`","expectedRevision":1,"expectedModeRevision":"0"}`, reportKey, "")
	require.Equal(t, 200, w.Code, w.Body.String(), "lost response must recover the original report after a mode change")
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &readBack))
	require.Equal(t, report, readBack)
	draftProbe.denied = true
	w = request("POST", reportPath, `{"recordId":"`+draftProbe.record+`","expectedRevision":1,"expectedModeRevision":"0"}`, reportKey, "")
	require.Equal(t, 403, w.Code, "original receipt still requires current source authorization")
	draftProbe.denied = false
	w = request("POST", reportPath, `{"recordId":"`+draftProbe.record+`","expectedRevision":1,"expectedModeRevision":"0"}`, uuid.NewString(), "")
	require.Equal(t, 412, w.Code, "new operation cannot use an old mode revision")
	var reportsCount int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM agent_customization.quality_runs").Scan(&reportsCount))
	require.Equal(t, 4, reportsCount, "replay and rejected new operation create no additional report")
	var count int
	require.NoError(t, db.QueryRow("SELECT count(*) FROM agent_customization.requests").Scan(&count))
	require.Equal(t, 1, count, "replay did not create another request")
}

type customizationDraftProbe struct {
	record string
	denied bool
}

func (p *customizationDraftProbe) Inspect(ctx context.Context, scope d.Scope, in d.DraftSelection, _ bool) (d.DraftSnapshot, error) {
	if p.denied {
		return d.DraftSnapshot{}, d.ErrForbidden
	}
	if _, ok := ctx.Value(productReviewCapabilityContextKey{}).(productReviewRequestCapability); !ok {
		return d.DraftSnapshot{}, d.ErrForbidden
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.UserID != scope.ActorID || identity.EffectiveOrganizationID != scope.OrganizationID {
		return d.DraftSnapshot{}, d.ErrForbidden
	}
	if in.RecordID != p.record || in.ExpectedRevision != 1 {
		return d.DraftSnapshot{}, d.ErrRevision
	}
	h := strings.Repeat("a", 64)
	return d.DraftSnapshot{DraftBinding: d.DraftBinding{RecordID: p.record, Revision: 1, SourceID: p.record, PreparationID: p.record, StoreID: p.record, Platform: "shein", Site: "shein-us", ProductKey: "own:fixture", ProductVersion: "1", Title: "收纳盒", RecordHash: h, ProductHash: h, RulesHash: h, InventoryHash: h, SavedAt: time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)}, Issues: []d.DraftIssue{{Code: "missing", Field: "category_id", Message: "选择类目"}}}, nil
}
