package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/integration/operationscockpitauth"
	cockpitstore "task-processor/internal/integration/persistence/operationscockpit"
	cockpit "task-processor/internal/operationscockpit"
	"task-processor/internal/storecenter"
	observations "task-processor/internal/storecenter/observations"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"

	"task-processor/internal/shopdiagnosis"
)

func TestDiagnosisHTTPUsesRealConfigurationToolModelLedgerAndReceipt(t *testing.T) {
	testProductAgentOwners(t, "diagnosis")
}
func mustDiagnosisJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
func testDiagnosisHTTPThroughNativeOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	require.NoError(t, storecenter.AutoMigrateStoreRepository(f.owner))
	records, err := storecenter.NewGormStoreRepository(f.owner)
	require.NoError(t, err)
	shopID := uuid.NewString()
	shop, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: shopID, OrganizationID: "B", ActorSubject: "admin", Name: "经营测试店铺", Platform: "temu", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	_, _, err = records.CreateOrReplay(ctx, "B", shop)
	require.NoError(t, err)
	require.NoError(t, cockpitstore.Install(ctx, f.owner))
	require.NoError(t, cockpitstore.GrantRuntime(ctx, f.owner, "source_acquisition_runtime"))
	require.NoError(t, f.owner.Exec("GRANT SELECT,UPDATE ON workbench_stores,workbench_store_member_grants TO source_acquisition_runtime").Error)
	boundary, err := operationscockpitauth.New(deps.organizationResolver, permissions, f.owner)
	require.NoError(t, err)
	repository, err := cockpitstore.New(ctx, f.db, boundary, time.Now)
	require.NoError(t, err)
	actor, cancel := context.WithTimeout(authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)}), time.Minute)
	defer cancel()
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, err)
	actor, err = boundary.Bind(actor, "Bearer admin")
	require.NoError(t, err)
	factID := uuid.NewString()
	fact := cockpit.FactInput{Period: cockpit.Period{Start: "2026-10-01", End: "2026-10-07"}, Amounts: cockpit.Amounts{Revenue: 100000, Procurement: 50000}, Note: "private note must stay undisclosed"}
	_, err = repository.Execute(actor, cockpit.Command{Scope: cockpit.Scope{OrganizationID: "B", ActorID: "admin"}, Operation: "fact_create", Key: uuid.NewString(), ID: factID, StoreID: shopID, Fact: &fact})
	require.NoError(t, err)
	spy := &diagnosisObservationSpy{}
	gate := &diagnosisModuleGate{Authorizer: boundary, stores: true, advice: true}
	reads := &shopdiagnosis.NativeReader{Repository: repository, Boundary: gate, Observations: spy}
	a, err := buildShopDiagnosisApplication(ctx, parent, reads, boundary.Bind, f.owner, deps.organizationResolver, permissions)
	require.NoError(t, err)
	in := diagnosisInput{ContextID: uuid.NewString(), StoreID: shopID, StartDate: fact.Period.Start, EndDate: fact.Period.End}
	routes := shopDiagnosisRoutes(a, a.bindRequest)
	app := buildIsolatedApplicationHTTPServer(routes, deps, 2*time.Minute)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	call := func(method, path, key string, body any, status int) []byte {
		t.Helper()
		raw := ""
		if body != nil {
			raw = string(mustDiagnosisJSON(t, body))
		}
		code, data, err := acquisitionHTTPRequest(server, method, path, "admin", "B", key, raw)
		require.NoError(t, err)
		require.Equal(t, status, code, string(data))
		return data
	}
	raw := call("POST", shopDiagnosisBase+"/inspect", uuid.NewString(), in, http.StatusOK)
	var facts shopdiagnosis.Inspection
	require.NoError(t, json.Unmarshal(raw, &facts))
	require.NotEmpty(t, facts.Facts)
	require.NotContains(t, string(raw), "private note")
	require.Zero(t, spy.calls.Load())
	require.Zero(t, calls.Load(), "reading facts does not invoke a paid model")
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	for _, capability := range productAgentCapabilities(actor, deps.organizationResolver, func(context.Context, ...string) (agent.Scope, error) { return scope, nil }, nil, parent) {
		if capability.ID == "text.generate" {
			require.Equal(t, "NEEDS_CONFIGURATION", capability.Readiness, "diagnosis-only must not ready an unconfigured title consumer")
		}
	}
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.DiagnosisAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.DiagnosisAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.DiagnosisAgentID, Version: agentconfig.DiagnosisAgentVersion}}
	mode, err := parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	key := uuid.NewString()
	preparation := diagnosisPrepareInput{diagnosisInput: in, RequestKey: key, ExpectedModeRevision: mode.Revision, GoalSummary: "根据真实资料比较并指出缺项"}
	raw = call("POST", shopDiagnosisBase+"/prepare", key, preparation, http.StatusOK)
	var prepared struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}
	require.NoError(t, json.Unmarshal(raw, &prepared))
	path := shopDiagnosisBase + "/contexts/" + in.ContextID + "/runs/" + key
	body := struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}{prepared.ConfigurationSnapshotRef}
	for _, revoked := range []string{"stores", "advice"} {
		gate.stores = revoked != "stores"
		gate.advice = revoked != "advice"
		call("POST", shopDiagnosisBase+"/inspect", uuid.NewString(), in, http.StatusForbidden)
		call("POST", shopDiagnosisBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusForbidden)
		require.Zero(t, calls.Load(), "each missing module permission must prevent actual governed model dispatch")
	}
	gate.stores, gate.advice = true, true
	raw = call("POST", shopDiagnosisBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	var result diagnosisRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Validated, string(raw))
	require.Equal(t, agent.ModeReadOnly, result.ExecutionMode)
	require.EqualValues(t, 2, calls.Load(), "one inspect decision and one proposal decision")
	require.NotNil(t, result.Proposal)
	require.NotEmpty(t, result.Inspection.Facts)
	replay := call("POST", shopDiagnosisBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	require.JSONEq(t, string(raw), string(replay))
	require.EqualValues(t, 2, calls.Load())
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.DiagnosisAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	history := call("GET", path, "", nil, http.StatusOK)
	require.JSONEq(t, string(raw), string(history), "disable must preserve exact committed history")
	replay = call("POST", shopDiagnosisBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	require.JSONEq(t, string(raw), string(replay), "exact replay after disable returns the old receipt")
	require.EqualValues(t, 2, calls.Load())
	// Editing native facts changes future dispatch identity, not retained advice.
	fact.Amounts.Revenue = 120000
	_, err = repository.Execute(actor, cockpit.Command{Scope: cockpit.Scope{OrganizationID: "B", ActorID: "admin"}, Operation: "fact_update", Key: uuid.NewString(), ID: factID, StoreID: shopID, Expected: 1, Fact: &fact})
	require.NoError(t, err)
	call("GET", path, "", nil, http.StatusOK)
	require.NoError(t, f.owner.Exec("UPDATE workbench_stores SET record_status='deleted',deleted_at=now() WHERE id=?", shopID).Error)
	call("GET", path, "", nil, http.StatusForbidden)
	require.EqualValues(t, 2, calls.Load())
	require.Zero(t, spy.calls.Load(), "facts, prepare, start and history never enter TEMU Access.Open")

}

type diagnosisObservationSpy struct{ calls atomic.Int32 }

func (s *diagnosisObservationSpy) List(context.Context, observations.Scope, observations.Query) (observations.Result, error) {
	s.calls.Add(1)
	return observations.Result{}, observations.ErrUnavailable
}

type diagnosisModuleGate struct {
	*operationscockpitauth.Authorizer
	stores, advice bool
}

func (g *diagnosisModuleGate) Current(ctx context.Context, scope cockpit.Scope) (cockpit.Access, error) {
	a, e := g.Authorizer.Current(ctx, scope)
	a.StoresRead = a.StoresRead && g.stores
	a.AdviceRead = a.AdviceRead && g.advice
	return a, e
}
