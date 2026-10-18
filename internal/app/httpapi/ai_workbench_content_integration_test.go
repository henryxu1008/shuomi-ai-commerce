package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/openai"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/enrichment"
)

func TestChatContentTasksUseOriginalOwnersForMultipleSources(t *testing.T) {
	for _, mode := range []string{"content flow chat", "own content chat", "own content chat full_access"} {
		t.Run(mode, func(t *testing.T) { testProductAgentOwners(t, mode) })
	}
}

// Uses the existing domain/HTTP fixture, task-owned PostgreSQL and local model;
// no serving instance, real credentials or provider are involved.
func testChatContentThroughOwners(t *testing.T, f *acquisitionHTTPFixture, m productAgentModule, deps routeAuthDependencies, auth *authz.ListingKitAuthorizer, configuration *configstore.Store, operationID string, calls *atomic.Int32, own, automatic bool) {
	ctx := context.Background()
	a := m.application.content
	require.NotNil(t, a)
	require.Nil(t, m.application.runtime)
	require.Nil(t, m.application.assets)
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
	a.ownProducts = collections
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
	selection := enrichment.ContentSelection{TargetLanguage: "en-US", Description: true, SellingPoints: true}
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
	var planningCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		planningCalls.Add(1)
		var wire struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&wire))
		require.Contains(t, wire.Messages[len(wire.Messages)-1].Content, binding.ContextID)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"planner-fixture","choices":[{"message":{"content":"{\"mode\":\"READY\",\"assistant_text\":\"请确认内容方案\",\"goal_summary\":\"整理商品内容\"}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":10,"total_tokens":30}}`))
	}))
	defer provider.Close()
	credentials := openai.NewOrganizationOnlyCredentialResolver(f.owner)
	require.NoError(t, credentials.SaveCredential(ctx, openai.AIClientCredential{TenantID: "B", ClientName: "chat", APIKey: "fixture-only", BaseURL: provider.URL + "/v1", Model: "planner-fixture", APIStyle: "openai", Enabled: true, TimeoutSecond: 3}))
	credential, err := credentials.GetCredential(ctx, "B", "", "chat")
	require.NoError(t, err)
	policy := a.config.ContentTextPolicies["B"]
	policy.Profile.ClientName = "chat"
	policy.Profile.ProviderID = "openai-compatible"
	policy.Profile.ModelID = "planner-fixture"
	policy.Profile.PromptVersion = "ai-workbench-chat-plan-v2"
	policy.Profile.OutputSchemaVersion = "ai-workbench-plan-decision-v1"
	policy.Profile.MaximumInputBytes = 128 << 10
	policy.Profile.MaximumCompletionTokens = 4096
	policy.AdmittedCredentialVersion = governed.CredentialVersion(*credential)
	policy.AdmittedEndpointIdentityDigest = governed.EndpointIdentityDigest(credential.BaseURL)
	wm, err := buildAIWorkbenchModule(ctx, AIWorkbenchDependencies{DB: db, ContentTasks: true, AllowedOrganizationIDs: []string{"B"}, PlanningTextPolicies: map[string]governed.RoutePolicy{"B": policy}}, m.application)
	require.NoError(t, err)
	app := wm.(aiWorkbenchModule).application
	require.Equal(t, aiworkbench.ContentTaskKind, wm.(aiWorkbenchModule).ExecutionKind())
	require.Equal(t, "AVAILABLE", wm.(aiWorkbenchModule).PlanningReadiness(ctx, "B"))
	require.Equal(t, "UNAVAILABLE", wm.(aiWorkbenchModule).TitleReadiness(ctx, "B"))
	server := httptest.NewServer(buildIsolatedApplicationHTTPServer(append(m.routes, wm.(aiWorkbenchModule).routes...), deps, 2*time.Minute).Handler)
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
	var created struct{ Conversation struct{ ID string } }
	require.NoError(t, json.Unmarshal(call("POST", workbenchChatBase, uuid.NewString(), map[string]any{}, 200), &created))
	messagePath := workbenchChatBase + "/" + created.Conversation.ID + "/messages"
	body := map[string]any{"content": "整理为英文内容", "taskSelection": selected}
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
	before := calls.Load()
	require.Zero(t, before, "planning never starts a content run")
	confirmPath := workbenchChatBase + "/" + created.Conversation.ID + "/proposals/" + planned.ProposalID + "/confirm"
	key := uuid.NewString()
	var confirmed struct {
		Task   workbenchTaskView
		Replay bool
	}
	require.NoError(t, json.Unmarshal(call("POST", confirmPath, key, nil, 200), &confirmed))
	require.NotEmpty(t, confirmed.Task.ID)
	require.EqualValues(t, 3, calls.Load())
	call("POST", confirmPath, key, nil, 200)
	require.EqualValues(t, 3, calls.Load())
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
	require.Len(t, view.Content.After, len(selection.Fields()))
	accepted := titleDecision(t, server, view, "accept", "admin", "", 200)
	applied := titleApply(t, server, accepted, uuid.NewString(), 200)
	require.NotNil(t, applied.Receipt)
	var completed struct{ Task workbenchTaskView }
	require.NoError(t, json.Unmarshal(call("GET", workbenchTaskBase+"/"+confirmed.Task.ID, "", nil, 200), &completed))
	require.Equal(t, aiworkbench.TaskCompleted, completed.Task.State)
	call("POST", confirmPath, key, nil, 200)
	require.EqualValues(t, 3, calls.Load(), "receipt-first replay after Apply cannot dispatch again")
	stale := call("POST", messagePath, uuid.NewString(), body, 409)
	require.Contains(t, string(stale), "PROPOSAL_STALE")
	require.EqualValues(t, 1, planningCalls.Load())
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
	require.EqualValues(t, 3, calls.Load())
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
	}
}
