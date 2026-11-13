package httpapi

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/app/productsourcing"
	reportapp "task-processor/internal/app/reportcenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/aiworkbench/einorouter"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	reviewproposal "task-processor/internal/product/review/proposal"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type revocableOwnProductReader struct {
	delegate ownProductReader
	denied   atomic.Bool
}

func (r *revocableOwnProductReader) ReadItem(ctx context.Context, id string) (collection.ItemDetail, error) {
	if r.denied.Load() {
		return collection.ItemDetail{}, collection.ErrForbidden
	}
	return r.delegate.ReadItem(ctx, id)
}

func TestChatContentTasksUseOriginalOwnersForMultipleSources(t *testing.T) {
	for _, mode := range []string{"content flow chat", "own content chat", "own content chat full_access", "full runtime content flow chat", "full runtime own content chat", "full runtime content flow chat mini"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, mode) })
	}
}

// Uses the existing domain/HTTP fixture, task-owned PostgreSQL and local model;
// no serving instance, real credentials or provider are involved.
func testChatContentThroughOwners(t *testing.T, f *acquisitionHTTPFixture, m productAgentModule, deps routeAuthDependencies, auth *authz.ListingKitAuthorizer, configuration *configstore.Store, operationID string, calls, sharedPlannerCalls *atomic.Int32, own, automatic, fullRuntime, scheduled bool) {
	ctx := context.Background()
	a := m.application.content
	miniRuntime := a.config.ContentTextPolicies["B"].Profile.AdapterPolicyVersion == governed.OpenAIMiniTextAdapterPolicy
	require.NotNil(t, a)
	if fullRuntime {
		require.NotNil(t, m.application.runtime)
		require.NotNil(t, m.application.assets)
	} else {
		require.Nil(t, m.application.runtime)
		require.Nil(t, m.application.assets)
	}
	_, err := configuration.Execute(ctx, agentconfig.Command{Scope: agent.Scope{OrganizationID: "B", ActorID: "admin"}, Key: uuid.NewString(), AgentID: agentconfig.ContentAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	if automatic {
		_, e := configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}, Subject: a.modeSubject(), Key: uuid.NewString(), Mode: agent.ModeFullAccess})
		require.NoError(t, e)
	}
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)})
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, err)
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, err := collection.NewContextAuthorizer(live, auth)
	require.NoError(t, err)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repo, err := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, err)
	published, err := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, auth)
	require.NoError(t, err)
	collections, err := collection.NewService(repo, authority, published)
	require.NoError(t, err)
	collections.WithSnapshots(a.snapshots)
	ownedProducts := &revocableOwnProductReader{delegate: collections}
	a.ownProducts = collections
	if own && os.Getenv("CODEX_PRODUCT_CONTEXT_FIXTURE") == "1" {
		a.ownProducts = ownedProducts
	}
	var binding agent.Binding
	if own {
		saved, e := collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "create_product", Product: &collection.OwnProduct{Title: "Saved own product", Description: "Saved own product description", Images: []string{"https://images.example.org/own.png"}}})
		require.NoError(t, e)
		source, e := a.ownSource(actor, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)}, saved.ItemID, nil)
		require.NoError(t, e)
		binding = source.Binding
	} else {
		binding, err = a.currentAcquisitionBinding(actor, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Hour)}, operationID, "product")
		require.NoError(t, err)
	}
	selection := reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true, SellingPoints: true}
	if !own {
		selection.Title = true
		selection.SEOKeywords = true
	}
	selected := aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: binding.ContextKind, ID: binding.ContextID, CatalogVersion: binding.CatalogVersion, PublicationID: binding.PublicationID}, Capability: aiworkbench.ContentTaskKind, Parameters: selection}
	require.NoError(t, workstore.InstallSchema(f.owner))
	require.NoError(t, f.owner.Exec(`DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='ai_workbench_runtime') THEN CREATE ROLE ai_workbench_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$`).Error)
	require.NoError(t, f.owner.Exec("ALTER ROLE ai_workbench_runtime PASSWORD 'chat808-fixture'").Error)
	require.NoError(t, workstore.GrantRuntime(f.owner, "ai_workbench_runtime"))
	var database string
	require.NoError(t, f.owner.Raw("SELECT current_database()").Scan(&database).Error)
	require.NoError(t, f.owner.Exec(`GRANT CONNECT ON DATABASE "`+database+`" TO ai_workbench_runtime`).Error)
	db, err := gorm.Open(postgres.Open(os.Getenv("ISSUE398_TEST_DSN")+" dbname="+database+" user=ai_workbench_runtime password=chat808-fixture"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	planningCalls := &atomic.Int32{}
	if fullRuntime {
		planningCalls = sharedPlannerCalls
	}
	const projectID = "project-fixture"
	var projectRevoked atomic.Bool
	var failSend atomic.Bool
	var projectReads atomic.Int32
	var goal atomic.Value
	goal.Store("original project goal")
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		planningCalls.Add(1)
		var wire struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&wire))
		require.Contains(t, wire.Messages[len(wire.Messages)-1].Content, binding.ContextID)
		require.Contains(t, wire.Messages[len(wire.Messages)-1].Content, goal.Load())
		if os.Getenv("CODEX_PRODUCT_CONTEXT_FIXTURE") == "1" {
			require.Contains(t, wire.Messages[len(wire.Messages)-1].Content, `"product_context"`)
			if own {
				require.Contains(t, wire.Messages[len(wire.Messages)-1].Content, "Saved own product description")
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"planner-fixture","choices":[{"message":{"content":"{\"mode\":\"READY\",\"assistant_text\":\"请确认内容方案\",\"goal_summary\":\"整理商品内容\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30,"prompt_tokens_details":{"cached_tokens":0}}}`))
	}))
	defer provider.Close()
	credentials := newHTTPChannelFixture(f.owner)
	require.NoError(t, credentials.seed(ctx, httpModelFixture{TenantID: "B", ClientName: "chat", APIKey: "fixture-only", BaseURL: provider.URL + "/v1", Model: "planner-fixture", APIStyle: "openai", Enabled: true, TimeoutSecond: 3}))
	credential, err := credentials.record(ctx, "B", "", "chat")
	require.NoError(t, err)
	policy := a.config.ContentTextPolicies["B"]
	policy.Profile.ClientName = "chat"
	policy.Profile.ProviderID = "openai-compatible"
	policy.Profile.ModelID = "planner-fixture"
	policy.Profile.OfficialTariff.Model = policy.Profile.ModelID
	policy.Profile.PromptVersion = "ai-workbench-chat-plan-v3"
	policy.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	policy.Profile.MaximumInputBytes = 128 << 10
	policy.Profile.MaximumCompletionTokens = 4096
	policy.Profile.ConnectionID = credential.ClientName
	policy.Profile.CredentialVersion = credential.reference
	policy.AdmittedCredentialVersion = httpCredentialReference(*credential)
	policy.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(credential.BaseURL)
	if fullRuntime {
		policy = a.config.ContentTextPolicies["B"]
		policy.Profile.PromptVersion = "ai-workbench-chat-plan-v3"
		policy.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
		policy.Profile.MaximumInputBytes = 128 << 10
		policy.Profile.MaximumCompletionTokens = 4096
		if miniRuntime {
			policy.Profile.MaximumPromptTokens = 4096
			policy.Profile.MaximumInputBytes = 64 << 10
			policy.Profile.MaximumCompletionTokens = 512
		}
	} else {
		reader, ok := a.config.CurrentTextPolicies.(syntheticCurrentTextPolicies)
		require.True(t, ok)
		reader.planning = map[string]governed.RoutePolicy{"B": policy}
		a.config.CurrentTextPolicies = reader
	}
	if os.Getenv("CODEX_PRODUCT_CONTEXT_FIXTURE") == "1" {
		policy.Profile.PromptVersion = "ai-workbench-chat-plan-v4"
		if !fullRuntime {
			reader := a.config.CurrentTextPolicies.(syntheticCurrentTextPolicies)
			reader.planning = map[string]governed.RoutePolicy{"B": policy}
			a.config.CurrentTextPolicies = reader
		}
	}
	if fullRuntime {
		testFullChatApplicationComposition(t, f, m.application.config, deps, db, policy, operationID)
	}
	if fullRuntime && miniRuntime {
		// A deployment descriptor has no frozen tariff; the shared price owner
		// supplies it at use time. Startup must still admit its valid shape.
		unpriced := policy
		unpriced.Profile.OfficialTariff = aicapability.OfficialModelTariff{}
		require.NoError(t, unpriced.ShapeProfile().ValidateShape())
		require.Error(t, unpriced.ShapeProfile().Validate())
		_, err := buildAIWorkbenchModule(ctx, AIWorkbenchDependencies{DB: db, ContentTasks: true, AllowedOrganizationIDs: []string{"B"}, PlanningTextPolicies: map[string]governed.RoutePolicy{"B": unpriced}}, m.application)
		require.NoError(t, err, "a shape-valid planning descriptor must not need a frozen tariff at startup")
		invalid := unpriced
		invalid.Profile.MaximumOutputBytes = 0
		_, err = buildAIWorkbenchModule(ctx, AIWorkbenchDependencies{DB: db, ContentTasks: true, AllowedOrganizationIDs: []string{"B"}, PlanningTextPolicies: map[string]governed.RoutePolicy{"B": invalid}}, m.application)
		require.Error(t, err, "invalid planning shape must still be rejected")
	}
	var routePolicies map[string]governed.RoutePolicy
	var storeRouteCalls atomic.Int32
	routeStarted, routeRelease := make(chan struct{}), make(chan struct{})
	var releaseRoute, startRoute sync.Once
	runStoreRoute := os.Getenv("CODEX_STORE_QUERY_FIXTURE") == "1" && !fullRuntime && !own && !scheduled
	if runStoreRoute {
		routeProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			storeRouteCalls.Add(1)
			startRoute.Do(func() { close(routeStarted) })
			<-routeRelease
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"route-fixture","choices":[{"message":{"content":"{\"kind\":\"STORE_OVERVIEW\",\"status\":\"ALL_NON_DELETED\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30,"prompt_tokens_details":{"cached_tokens":0}}}`))
		}))
		t.Cleanup(routeProvider.Close)
		t.Cleanup(func() { releaseRoute.Do(func() { close(routeRelease) }) })
		require.NoError(t, credentials.seed(ctx, httpModelFixture{TenantID: "B", ClientName: "store-router", APIKey: "fixture-only", BaseURL: routeProvider.URL + "/v1", Model: "route-fixture", APIStyle: "openai", Enabled: true, TimeoutSecond: 3}))
		routeCredential, e := credentials.record(ctx, "B", "", "store-router")
		require.NoError(t, e)
		routePolicy := policy
		routePolicy.Profile.ClientName = "store-router"
		routePolicy.Profile.ConnectionID = "store-router"
		routePolicy.Profile.ModelID = "route-fixture"
		routePolicy.Profile.OfficialTariff.Model = "route-fixture"
		routePolicy.Profile.PromptVersion = einorouter.PromptVersion
		routePolicy.Profile.OutputSchemaVersion = einorouter.SchemaVersion
		routePolicy.Profile.CredentialVersion = routeCredential.reference
		routePolicy.AdmittedCredentialVersion = httpCredentialReference(*routeCredential)
		routePolicy.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(routeCredential.BaseURL)
		routePolicies = map[string]governed.RoutePolicy{"B": routePolicy}
		reader, ok := a.config.CurrentTextPolicies.(syntheticCurrentTextPolicies)
		require.True(t, ok)
		reader.storeRoute = routePolicies
		a.config.CurrentTextPolicies = reader
	}
	wm, err := buildAIWorkbenchModule(ctx, AIWorkbenchDependencies{DB: db, ContentTasks: true, AllowedOrganizationIDs: []string{"B"}, PlanningTextPolicies: map[string]governed.RoutePolicy{"B": policy}, StoreRouteTextPolicies: routePolicies}, m.application)
	require.NoError(t, err)
	app := wm.(aiWorkbenchModule).application
	if runStoreRoute {
		require.Equal(t, "AVAILABLE", wm.(aiWorkbenchModule).StoreRouteReadiness(ctx, "B"))
		require.Equal(t, "UNAVAILABLE", wm.(aiWorkbenchModule).StoreRouteReadiness(ctx, "A"))
		require.NoError(t, storecenter.AutoMigrateStoreRepository(f.owner))
		require.NoError(t, storecenter.AutoMigrateAuditRepository(f.owner))
		catalog, e := storecenter.NewCatalog(f.owner, currentStoreMemberAuthorizer{authorizer: auth}, nil)
		require.NoError(t, e)
		app.attachStoreOverview(catalog)
	}
	app.scopes.reader.Project = func(_ context.Context, owner aiworkbench.Scope, id string) (aiworkbench.ProjectScopeFacts, error) {
		if owner != (aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}) || id != projectID || projectRevoked.Load() || failSend.Load() && projectReads.Add(1) == 2 {
			return aiworkbench.ProjectScopeFacts{}, aiworkbench.ErrNotFound
		}
		return aiworkbench.ProjectScopeFacts{ID: projectID, Title: "Fixture project", Goal: goal.Load().(string), Revision: 1}, nil
	}
	require.Equal(t, aiworkbench.ContentTaskKind, wm.(aiWorkbenchModule).ExecutionKind())
	require.Equal(t, "AVAILABLE", wm.(aiWorkbenchModule).PlanningReadiness(ctx, "B"))
	require.Equal(t, "UNAVAILABLE", wm.(aiWorkbenchModule).TitleReadiness(ctx, "B"))
	server := httptest.NewServer(buildCurrentApplicationHTTPServer(append(m.routes, wm.(aiWorkbenchModule).routes...), deps).Handler)
	defer server.Close()
	call := func(method, path, key string, body any, status int) []byte {
		var rawBody string
		if body != nil {
			b, e := json.Marshal(body)
			require.NoError(t, e)
			rawBody = string(b)
		}
		code, raw, e := acquisitionHTTPRequest(server, method, path, "admin", "B", key, rawBody)
		require.NoError(t, e)
		require.Equal(t, status, code, string(raw))
		return raw
	}
	if runStoreRoute {
		var general struct{ Conversation aiworkbench.Conversation }
		body := workbenchCreateFixtureBody(t, server, "admin", "B", aiworkbench.ScopeSelection{Kind: "general"}, false)
		code, raw, e := acquisitionHTTPRequest(server, "POST", workbenchChatBase, "admin", "B", uuid.NewString(), body)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		require.NoError(t, json.Unmarshal(raw, &general))
		intentKey := uuid.NewString()
		path := workbenchChatBase + "/" + general.Conversation.ID + "/intent"
		question := `{"content":"我有几个店铺"}`
		type routeResult struct {
			code int
			raw  []byte
			err  error
		}
		first := make(chan routeResult, 1)
		go func() {
			code, raw, e := acquisitionHTTPRequest(server, "POST", path, "admin", "B", intentKey, question)
			first <- routeResult{code, raw, e}
		}()
		select {
		case <-routeStarted:
		case <-time.After(5 * time.Second):
			t.Fatal("controlled route was not dispatched")
		}
		code, raw, e = acquisitionHTTPRequest(server, "POST", path, "admin", "B", intentKey, question)
		require.NoError(t, e)
		require.Equal(t, 503, code, string(raw), "a concurrent original request must remain pending")
		releaseRoute.Do(func() { close(routeRelease) })
		initial := <-first
		require.NoError(t, initial.err)
		require.Equal(t, 200, initial.code, string(initial.raw))
		require.Contains(t, string(initial.raw), `"COMPLETE"`)
		code, raw, e = acquisitionHTTPRequest(server, "POST", path, "admin", "B", intentKey, question)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		require.EqualValues(t, 1, storeRouteCalls.Load())
		stored, found, e := app.store.ReplayStoreRoute(ctx, aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}, general.Conversation.ID, intentKey, "member-B-admin", "我有几个店铺")
		require.NoError(t, e)
		require.True(t, found)
		fact, e := a.config.Ledger.ReadModelInvocation(ctx, "B", stored.InvocationID)
		require.NoError(t, e)
		require.Equal(t, aicapability.InvocationSucceeded, fact.Outcome)
		settlement, e := a.points.InspectModelInvocationReservation(ctx, "B", stored.InvocationID)
		require.NoError(t, e)
		require.Equal(t, "committed", settlement.State)
		t.Log("store route fixture: one local model response, same-key concurrent retry, original invocation settled")
		code, raw, e = acquisitionHTTPRequest(server, "GET", workbenchChatBase+"/"+general.Conversation.ID, "admin", "B", "", "")
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		require.Contains(t, string(raw), `"currentStoreOverview"`)
		require.Contains(t, string(raw), `"total":0`)
		// Reproduce a settled model output lost before Chat T1; the durable AI
		// owner stores only the output digest and must not dispatch it again.
		var lostConversation struct{ Conversation aiworkbench.Conversation }
		code, raw, e = acquisitionHTTPRequest(server, "POST", workbenchChatBase, "admin", "B", uuid.NewString(), body)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		require.NoError(t, json.Unmarshal(raw, &lostConversation))
		owner := aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}
		lostKey := uuid.NewString()
		gateway := app.storeRoute.Gateway
		admitted, e := gateway.Admit(actor, owner, lostConversation.Conversation.ID, "member-B-admin", "店铺情况")
		require.NoError(t, e)
		_, _, e = app.store.BeginStoreRoute(actor, owner, lostConversation.Conversation.ID, lostKey, admitted)
		require.NoError(t, e)
		frozen, found, e := app.store.ReplayStoreRoute(actor, owner, lostConversation.Conversation.ID, lostKey, "member-B-admin", "店铺情况")
		require.NoError(t, e)
		require.True(t, found)
		decision, e := gateway.Decide(actor, frozen)
		require.NoError(t, e)
		require.NoError(t, gateway.VerifyDecision(actor, frozen, decision))
		require.NoError(t, f.owner.Exec("UPDATE ai_workbench.commands SET planner_deadline=clock_timestamp()-INTERVAL '2 seconds',route_commit_deadline=clock_timestamp()-INTERVAL '1 second' WHERE idempotency_key=?", lostKey).Error)
		lost, e := app.storeRoute.Route(actor, owner, lostConversation.Conversation.ID, lostKey, "member-B-admin", "店铺情况")
		require.NoError(t, e)
		require.Equal(t, "ROUTING_RESULT_LOST", lost.State)
		lost, e = app.storeRoute.Route(actor, owner, lostConversation.Conversation.ID, lostKey, "member-B-admin", "店铺情况")
		require.NoError(t, e)
		require.Equal(t, "ROUTING_RESULT_LOST", lost.State)
		require.EqualValues(t, 2, storeRouteCalls.Load())
		lostMessages, _, e := app.store.ListMessagesPage(actor, owner, lostConversation.Conversation.ID, 0, 50)
		require.NoError(t, e)
		require.Len(t, lostMessages, 2)
		require.Contains(t, lostMessages[1].Content, "未在期限内保存")
		settlement, e = a.points.InspectModelInvocationReservation(ctx, "B", frozen.InvocationID)
		require.NoError(t, e)
		require.Equal(t, "committed", settlement.State)
		t.Log("store route fixture: lost output recovered once after deadline without redispatch")
		explicitKey := uuid.NewString()
		explicitPath := workbenchChatBase + "/" + general.Conversation.ID + "/store-overview"
		code, raw, e = acquisitionHTTPRequest(server, "POST", explicitPath, "admin", "B", explicitKey, `{"status":"ALL_NON_DELETED"}`)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		code, raw, e = acquisitionHTTPRequest(server, "POST", explicitPath, "admin", "B", explicitKey, `{"status":"ALL_NON_DELETED"}`)
		require.NoError(t, e)
		require.Equal(t, 200, code, string(raw))
		require.Contains(t, string(raw), `"replay":true`)
		require.EqualValues(t, 2, storeRouteCalls.Load())
		t.Log("store route fixture: explicit overview replayed without model call")
	}
	// Chat's explicit allowlist may be narrower than the shared content owner.
	a.config.AllowedOrganizationIDs = []string{"A", "B"}
	deniedStatus, _, deniedErr := acquisitionHTTPRequest(server, "POST", workbenchChatBase, "admin", "A", uuid.NewString(), `{}`)
	require.NoError(t, deniedErr)
	require.NotEqual(t, 200, deniedStatus, "Chat must enforce its own admitted organizations")
	a.config.AllowedOrganizationIDs = []string{"B"}
	var sourcePreview struct {
		Subject    aiworkbench.Subject
		ProductKey string
	}
	require.NoError(t, json.Unmarshal(call("GET", "/api/v1/workbench/chat/subjects/"+binding.ContextKind+"/"+binding.ContextID, "", nil, 200), &sourcePreview))
	require.Equal(t, selected.Subject, sourcePreview.Subject)
	var created struct{ Conversation aiworkbench.Conversation }
	createBody := workbenchCreateFixtureBody(t, server, "admin", "B", aiworkbench.ScopeSelection{Kind: "project", ProjectID: projectID}, false)
	var createPayload map[string]any
	require.NoError(t, json.Unmarshal([]byte(createBody), &createPayload))
	createKey := uuid.NewString()
	require.NoError(t, json.Unmarshal(call("POST", workbenchChatBase, createKey, createPayload, 200), &created))
	require.NotNil(t, created.Conversation.BusinessScope)
	require.Equal(t, projectID, created.Conversation.BusinessScope.ProjectID)
	call("POST", workbenchChatBase, createKey, createPayload, 200)
	messagePath := workbenchChatBase + "/" + created.Conversation.ID + "/messages"
	body := map[string]any{"content": "整理为英文内容", "taskSelection": selected}
	failSend.Store(true)
	var deniedTurn struct{ State aiworkbench.PlanningState }
	require.NoError(t, json.Unmarshal(call("POST", messagePath, uuid.NewString(), body, 200), &deniedTurn))
	require.Equal(t, aiworkbench.PlanningFailedBeforeDispatch, deniedTurn.State)
	require.Zero(t, planningCalls.Load(), "last scope gate must stop the first dispatch")
	failSend.Store(false)
	require.Error(t, db.Exec("UPDATE ai_workbench.conversations SET business_scope = NULL WHERE id = ?", created.Conversation.ID).Error)
	messageKey := uuid.NewString()
	var planned struct {
		ProposalID string
		Proposal   workbenchProposalCard
	}
	require.NoError(t, json.Unmarshal(call("POST", messagePath, messageKey, body, 200), &planned))
	require.NotEmpty(t, planned.ProposalID)
	require.Equal(t, &selected, planned.Proposal.TaskSelection)
	require.Equal(t, a.config.Limits.Tokens, planned.Proposal.MaximumTokens)
	require.Equal(t, a.config.Limits.ModelCalls, planned.Proposal.MaximumModelCalls)
	call("POST", messagePath, messageKey, body, 200)
	require.EqualValues(t, 1, planningCalls.Load(), "same turn replay must not call planner again")
	var pendingConversation, pendingKey string
	var pendingInput aiworkbench.MessageInput
	if own && os.Getenv("CODEX_PRODUCT_CONTEXT_FIXTURE") == "1" {
		owner := aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}
		metadataOnly := *app
		metadataOnly.plan, metadataOnly.agent = nil, nil
		require.True(t, metadataOnly.productHistoryUnavailable(actor, owner, created.Conversation.ID, math.MaxInt64))
		ownedProducts.denied.Store(true)
		page := call("GET", workbenchChatBase+"/"+created.Conversation.ID, "", nil, 200)
		require.NotContains(t, string(page), "请确认内容方案")
		require.Contains(t, string(page), "相关商品当前不可访问")
		replay := call("POST", messagePath, messageKey, body, 202)
		require.NotContains(t, string(replay), "proposalId")
		require.EqualValues(t, 1, planningCalls.Load(), "revocation cannot redispatch")
		ownedProducts.denied.Store(false)
		page = call("GET", workbenchChatBase+"/"+created.Conversation.ID, "", nil, 200)
		require.Contains(t, string(page), "请确认内容方案")
		var pending struct{ Conversation aiworkbench.Conversation }
		require.NoError(t, json.Unmarshal(call("POST", workbenchChatBase, uuid.NewString(), createPayload, 200), &pending))
		pendingConversation, pendingKey = pending.Conversation.ID, uuid.NewString()
		pendingInput = aiworkbench.MessageInput{Content: "整理为英文内容", TaskSelection: &selected}
		prepare, e := app.plan.Admission(actor, owner, pendingConversation, pendingInput)
		require.NoError(t, e)
		_, _, e = app.store.AppendUser(actor, owner, pendingConversation, pendingKey, pendingInput, prepare)
		require.NoError(t, e)
	}
	before := calls.Load()
	require.Zero(t, before, "planning never starts a content run")
	if scheduled {
		testScheduleExistingChatProposal(t, ctx, f, app, planned.ProposalID, calls, &projectRevoked, deps)
		return
	}
	confirmPath := workbenchChatBase + "/" + created.Conversation.ID + "/proposals/" + planned.ProposalID + "/confirm"
	key := uuid.NewString()
	goal.Store("changed while awaiting confirmation")
	call("POST", confirmPath, uuid.NewString(), nil, 409)
	require.Zero(t, calls.Load())
	goal.Store("original project goal")
	var confirmed struct {
		Task   workbenchTaskView
		Replay bool
	}
	var competing [2]struct {
		key  string
		code int
		body []byte
		err  error
	}
	var confirmations sync.WaitGroup
	for index := range competing {
		competing[index].key = uuid.NewString()
		confirmations.Add(1)
		go func(index int) {
			defer confirmations.Done()
			result := &competing[index]
			result.code, result.body, result.err = acquisitionHTTPRequest(server, "POST", confirmPath, "admin", "B", result.key, "")
		}(index)
	}
	confirmations.Wait()
	createdCount := 0
	for _, result := range competing {
		require.NoError(t, result.err)
		if result.code == http.StatusOK {
			createdCount++
			key = result.key
			require.NoError(t, json.Unmarshal(result.body, &confirmed))
		} else {
			require.Equal(t, http.StatusConflict, result.code, string(result.body))
		}
	}
	require.Equal(t, 1, createdCount, "different confirmation keys for one proposal must share the original T1 fence")
	require.NotEmpty(t, confirmed.Task.ID)
	expectedContentCalls := int32(3)
	if miniRuntime {
		expectedContentCalls = 2
	}
	require.EqualValues(t, expectedContentCalls, calls.Load())
	call("POST", confirmPath, key, nil, 200)
	require.EqualValues(t, expectedContentCalls, calls.Load())
	call("POST", confirmPath, uuid.NewString(), nil, 409)
	require.EqualValues(t, expectedContentCalls, calls.Load(), "a new browser key cannot execute the same confirmed proposal twice")
	// A different browser has no sessionStorage confirmation key. The owner
	// must still return its durable receipt, even if execution readiness changes.
	var reopened struct {
		Proposals []struct {
			ConfirmedTaskID string `json:"confirmedTaskId"`
		}
	}
	require.NoError(t, json.Unmarshal(call("GET", workbenchChatBase+"/"+created.Conversation.ID, "", nil, 200), &reopened))
	require.Len(t, reopened.Proposals, 1)
	require.Equal(t, confirmed.Task.ID, reopened.Proposals[0].ConfirmedTaskID)
	require.NoError(t, json.Unmarshal(call("POST", messagePath, messageKey, body, 200), &planned))
	require.Equal(t, confirmed.Task.ID, planned.Proposal.ConfirmedTaskID)
	require.EqualValues(t, 1, planningCalls.Load())
	owner := aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}
	storedTask, err := app.store.GetTask(ctx, owner, confirmed.Task.ID)
	require.NoError(t, err)
	runScope := agent.Scope{OrganizationID: owner.OrganizationID, ActorID: owner.ActorID}
	native, nextNative, err := a.configuration.Recent(ctx, runScope, "member-B-admin", agentconfig.ContentAgentID, "", 20, a.store, func(agent.Binding) error { return nil })
	require.NoError(t, err)
	require.Empty(t, native, "scoped content is reachable through its exact Task, not a standalone run")
	require.Empty(t, nextNative)
	_, _, err = a.configuration.Recent(ctx, runScope, "member-B-admin", agentconfig.ContentAgentID, storedTask.ConfigurationSnapshotRef.ID, 20, a.store, func(agent.Binding) error { return nil })
	require.ErrorIs(t, err, agentconfig.ErrInvalid)
	var scopedPage struct {
		Tasks      []workbenchTaskView
		Incomplete bool
	}
	require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"?scopeKind=project&projectId="+projectID, "", nil, 200), &scopedPage))
	require.Len(t, scopedPage.Tasks, 1)
	require.Equal(t, confirmed.Task.ID, scopedPage.Tasks[0].ID)
	require.NotNil(t, scopedPage.Tasks[0].BusinessScope)
	require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"?scopeKind=general", "", nil, 200), &scopedPage))
	require.Empty(t, scopedPage.Tasks)
	projectRevoked.Store(true)
	originalCreate := call("POST", workbenchChatBase, createKey, createPayload, 200)
	require.NotContains(t, string(originalCreate), "conversationAccepted")
	require.NotContains(t, string(originalCreate), "Fixture project")
	rejectedCreate := call("POST", workbenchChatBase, uuid.NewString(), createPayload, 404)
	require.JSONEq(t, `{"code":"NOT_FOUND","conversationAccepted":false}`, string(rejectedCreate))
	call("GET", workbenchTaskBase+"/"+confirmed.Task.ID, "", nil, 404)
	rawHidden := call("GET", workbenchChatBase+"/"+created.Conversation.ID, "", nil, 200)
	require.NotContains(t, string(rawHidden), "Fixture project")
	require.NotContains(t, string(rawHidden), "请确认内容方案")
	call("GET", workbenchTaskBase+"?scopeKind=project&projectId="+projectID, "", nil, 404)
	projectRevoked.Store(false)
	if automatic {
		require.Equal(t, aiworkbench.TaskCompleted, confirmed.Task.State)
		require.NotNil(t, confirmed.Task.PolicyAdoption)
		require.Equal(t, "product.content.policy-adopt-v1", confirmed.Task.PolicyAdoption.ActionVersion)
		var read struct{ Task workbenchTaskView }
		require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"/"+confirmed.Task.ID, "", nil, 200), &read))
		require.Equal(t, confirmed.Task.PolicyAdoption, read.Task.PolicyAdoption)
		var listed struct{ Tasks []workbenchTaskView }
		require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase, "", nil, 200), &listed))
		require.Len(t, listed.Tasks, 1)
		require.Equal(t, read.Task.PolicyAdoption, listed.Tasks[0].PolicyAdoption)
		require.Equal(t, selection, read.Task.PolicyAdoption.Execution.ContentSelection)
		return
	}
	raw := call("POST", workbenchTaskBase+"/"+confirmed.Task.ID+"/review", uuid.NewString(), nil, 200)
	var reviewed struct{ Task workbenchTaskView }
	require.NoError(t, json.Unmarshal(raw, &reviewed))
	require.NotEmpty(t, reviewed.Task.ReviewID, string(raw))
	view := titleCall(t, server, "GET", titleBasePath+"/"+reviewed.Task.ReviewID, "admin", "B", "", "", 200)
	sources := reportapp.Sources{Reviews: a.reviews, Policy: auth}
	_, denied := sources.Read(actor, rc.Scope{OrganizationID: "B", ActorID: "admin"}, "TITLE_REVIEW", reviewed.Task.ReviewID)
	require.ErrorIs(t, denied, rc.ErrNotFound)
	projectRevoked.Store(true)
	call("GET", titleBasePath+"/"+reviewed.Task.ReviewID, "", nil, 404)
	projectRevoked.Store(false)
	require.Len(t, view.Content.After, len(selection.Fields()))
	accepted := titleDecision(t, server, view, "accept", "admin", "", 200)
	applied := titleApply(t, server, accepted, uuid.NewString(), 200)
	require.NotNil(t, applied.Receipt)
	if pendingConversation != "" {
		pending, e := app.service.Message(actor, owner, pendingConversation, pendingKey, pendingInput)
		require.NoError(t, e)
		require.Equal(t, aiworkbench.PlanningFailedBeforeDispatch, pending.State)
		require.EqualValues(t, 1, planningCalls.Load(), "a newer product head must stop the original pending call")
	}
	var completed struct{ Task workbenchTaskView }
	require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"/"+confirmed.Task.ID, "", nil, 200), &completed))
	require.Equal(t, aiworkbench.TaskCompleted, completed.Task.State)
	require.NoError(t, json.Unmarshal(call("GET", workbenchChatBase+"/"+created.Conversation.ID, "", nil, 200), &reopened))
	require.Equal(t, confirmed.Task.ID, reopened.Proposals[0].ConfirmedTaskID, "a newer adopted product version does not erase the prior task receipt")
	call("POST", confirmPath, key, nil, 200)
	require.EqualValues(t, expectedContentCalls, calls.Load(), "receipt-first replay after Apply cannot dispatch again")
	stale := call("POST", messagePath, uuid.NewString(), body, 409)
	require.Contains(t, string(stale), "PROPOSAL_STALE")
	require.EqualValues(t, 1, planningCalls.Load())
	if miniRuntime {
		var balance struct {
			Reserved int64
			Consumed int64
		}
		require.NoError(t, f.owner.Raw("SELECT reserved, consumed FROM saas_organization_resource_buckets WHERE organization_id = ? AND resource_type = ?", "B", "ai_point").Scan(&balance).Error)
		require.Zero(t, balance.Reserved, "all three synthetic requests settle their own holds")
		require.Greater(t, balance.Consumed, int64(0), "actual synthetic usage is charged by the original resource owner")
		var settlements int64
		require.NoError(t, f.owner.Raw("SELECT count(*) FROM saas_organization_resource_reservations WHERE organization_id = ? AND state = 'committed' AND settlement_operation_id IS NOT NULL", "B").Scan(&settlements).Error)
		require.EqualValues(t, 3, settlements, "one planning plus two content calls")
		t.Logf("synthetic resource owner: settled=%d consumed=%d micropoints reserved=%d", settlements, balance.Consumed, balance.Reserved)
	}
	require.NoError(t, json.Unmarshal(call("GET", "/api/v1/workbench/chat/subjects/"+binding.ContextKind+"/"+binding.ContextID, "", nil, 200), &sourcePreview))
	require.Equal(t, strconv.FormatUint(applied.Receipt.ProductVersion, 10), sourcePreview.Subject.CatalogVersion)
	require.Equal(t, applied.Receipt.PublicationID, sourcePreview.Subject.PublicationID)
	selected.Subject.CatalogVersion = strconv.FormatUint(applied.Receipt.ProductVersion, 10)
	selected.Subject.PublicationID = applied.Receipt.PublicationID
	body["taskSelection"] = selected
	require.NoError(t, json.Unmarshal(call("POST", messagePath, uuid.NewString(), body, 200), &planned))
	require.Equal(t, selected.Subject, planned.Proposal.TaskSelection.Subject)
	frozen := a.config.Limits
	a.config.Limits.Tokens++
	changedPath := workbenchChatBase + "/" + created.Conversation.ID + "/proposals/" + planned.ProposalID + "/confirm"
	call("POST", changedPath, uuid.NewString(), nil, 409)
	a.config.Limits = frozen
	require.EqualValues(t, expectedContentCalls, calls.Load())
	stored, err := app.store.GetTask(ctx, aiworkbench.Scope{OrganizationID: "B", ActorID: "admin"}, confirmed.Task.ID)
	require.NoError(t, err)
	var request agent.Request
	require.NoError(t, json.Unmarshal(stored.ExecutionRequest, &request))
	require.Equal(t, selection, request.ContentSelection)
	require.Equal(t, binding, request.Binding)
	require.NoError(t, f.owner.Exec(`ALTER TABLE ai_workbench.execution_proposals DROP CONSTRAINT ai_workbench_proposal_capability`).Error)
	require.Error(t, workstore.VerifySchema(ctx, db), "missing capability check must stop serving")
	require.NoError(t, f.owner.Exec(`ALTER TABLE ai_workbench.execution_proposals ADD CONSTRAINT ai_workbench_proposal_capability CHECK (true)`).Error)
	require.Error(t, workstore.VerifySchema(ctx, db), "a same-named weakened check must stop serving")
	for _, who := range [][2]string{{"admin", "A"}, {"reviewer", "B"}} {
		code, _, e := acquisitionHTTPRequest(server, "GET", workbenchTaskBase+"/"+stored.ID, who[0], who[1], "", "")
		require.NoError(t, e)
		require.NotEqual(t, 200, code)
		code, _, e = acquisitionHTTPRequest(server, "GET", workbenchChatBase+"/"+created.Conversation.ID, who[0], who[1], "", "")
		require.NoError(t, e)
		require.NotEqual(t, 200, code)
	}
}
