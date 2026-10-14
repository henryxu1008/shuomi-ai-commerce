package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/commercetool"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/selection"
	"task-processor/internal/product/sourcing"
)

func TestSelectionHTTPUsesRealConfigurationToolModelLedgerAndReceipt(t *testing.T) {
	testProductAgentOwners(t, "selection")
}
func mustSelectionJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
func testSelectionHTTPThroughNativeOwners(t *testing.T, f *acquisitionHTTPFixture, parent *productAgentApplication, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, calls *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	live := &productReviewLiveOrganizationAccess{resolver: deps.organizationResolver, now: time.Now}
	authority, err := collection.NewContextAuthorizer(live, permissions)
	require.NoError(t, err)
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	repository, err := collectionstore.NewRepository(ctx, f.owner, func(tx *gorm.DB) (collectionstore.OwnPublisher, error) {
		return productsourcing.NewOwnProductWriter(tx, authority)
	})
	require.NoError(t, err)
	sources, err := productsourcing.NewPublishedAcquisitionReader(ctx, f.owner, live, permissions)
	require.NoError(t, err)
	collections, err := collection.NewService(repository, authority, sources)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(f.owner, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	collections.WithSnapshots(snapshots)
	actor := authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "B", EffectiveOrganizationID: "B", UserID: "admin", EffectiveMemberID: "member-B-admin", Roles: []string{"listingkit_admin"}, TokenExpiresAt: time.Now().Add(time.Minute)})
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer admin")
	require.NoError(t, err)
	in := selectionInput{ContextID: uuid.NewString()}
	for _, title := range []string{"已保存商品A", "已保存商品B"} {
		receipt, err := collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "create_product", Product: &collection.OwnProduct{Title: title}})
		require.NoError(t, err)
		item, err := collections.ReadItem(actor, receipt.ItemID)
		require.NoError(t, err)
		in.Candidates = append(in.Candidates, selectionChoice{ItemID: item.Item.ID, ExpectedRevision: strconv.FormatInt(item.Item.Revision, 10), OriginalPublicationID: item.Item.Source.PublicationID, OriginalVersion: strconv.FormatUint(item.Item.Source.Version, 10)})
	}
	a, err := buildProductSelectionApplication(ctx, parent, collections, f.owner, deps.organizationResolver, permissions)
	require.NoError(t, err)
	routes := productSelectionRoutes(a, (productReviewCapabilityBinder{}).Bind)
	app := buildIsolatedApplicationHTTPServer(routes, deps, 2*time.Minute)
	server := httptest.NewServer(app.Handler)
	defer server.Close()
	call := func(method, path, key string, body any, status int) []byte {
		t.Helper()
		raw := ""
		if body != nil {
			raw = string(mustSelectionJSON(t, body))
		}
		code, data, err := acquisitionHTTPRequest(server, method, path, "admin", "B", key, raw)
		require.NoError(t, err)
		require.Equal(t, status, code, string(data))
		return data
	}
	raw := call("POST", productSelectionBase+"/inspect", uuid.NewString(), in, http.StatusOK)
	var facts selection.Inspection
	require.NoError(t, json.Unmarshal(raw, &facts))
	require.Len(t, facts.Products, 2)
	require.Zero(t, calls.Load(), "reading facts does not invoke a paid model")
	scope := agent.Scope{OrganizationID: "B", ActorID: "admin"}
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.SelectionAgentID, Operation: "enable", Absent: true})
	require.NoError(t, err)
	identity := agent.ExecutionIdentity{OrganizationID: "B", ActorID: "admin", MemberID: "member-B-admin"}
	subject := agentconfig.ResolvedModeSubject{Subject: agentconfig.ModeSubject{Kind: agentconfig.RegisteredAgentSubject, ID: agentconfig.SelectionAgentID}, Definition: commercetool.AgentRef{ID: agentconfig.SelectionAgentID, Version: agentconfig.SelectionAgentVersion}}
	mode, err := parent.configuration.ExecuteMode(ctx, agentconfig.ModeCommand{Identity: identity, Subject: subject, Key: uuid.NewString(), Mode: agent.ModeReadOnly})
	require.NoError(t, err)
	key := uuid.NewString()
	preparation := selectionPrepareInput{selectionInput: in, RequestKey: key, ExpectedModeRevision: mode.Revision, GoalSummary: "根据真实资料比较并指出缺项"}
	raw = call("POST", productSelectionBase+"/prepare", key, preparation, http.StatusOK)
	var prepared struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}
	require.NoError(t, json.Unmarshal(raw, &prepared))
	path := productSelectionBase + "/contexts/" + in.ContextID + "/runs/" + key
	body := struct {
		ConfigurationSnapshotRef agent.ConfigurationSnapshotRef `json:"configurationSnapshotRef"`
	}{prepared.ConfigurationSnapshotRef}
	raw = call("POST", productSelectionBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	var result selectionRunDTO
	require.NoError(t, json.Unmarshal(raw, &result))
	require.True(t, result.Validated, string(raw))
	require.Equal(t, agent.ModeReadOnly, result.ExecutionMode)
	require.EqualValues(t, 2, calls.Load(), "one inspect decision and one proposal decision")
	require.NotNil(t, result.Proposal)
	require.Len(t, result.Inspection.Products, 2)
	replay := call("POST", productSelectionBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	require.JSONEq(t, string(raw), string(replay))
	require.EqualValues(t, 2, calls.Load())
	_, err = parent.configuration.Execute(ctx, agentconfig.Command{Scope: scope, Key: uuid.NewString(), AgentID: agentconfig.SelectionAgentID, Operation: "disable", Expected: 1})
	require.NoError(t, err)
	history := call("GET", path, "", nil, http.StatusOK)
	require.JSONEq(t, string(raw), string(history), "disable must preserve exact committed history")
	replay = call("POST", productSelectionBase+"/contexts/"+in.ContextID+"/runs", key, body, http.StatusOK)
	require.JSONEq(t, string(raw), string(replay), "exact replay after disable returns the old receipt")
	require.EqualValues(t, 2, calls.Load())
	_, err = collections.Mutate(actor, uuid.NewString(), collection.Mutation{Action: "archive_item", ItemID: in.Candidates[1].ItemID, ExpectedRevision: 1})
	require.NoError(t, err)
	call("GET", path, "", nil, http.StatusNotFound)
	require.EqualValues(t, 2, calls.Load(), "revoked candidate history causes no redispatch")
}
