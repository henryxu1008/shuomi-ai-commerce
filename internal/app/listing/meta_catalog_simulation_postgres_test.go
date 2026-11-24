package listingapp_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	listinghttp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	metaadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/metacatalog"
	persistence "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const metaCallback = "http://127.0.0.1:3210/workbench/stores/meta-catalog/callback"

type metaGraphFixture struct {
	writes           atomic.Int32
	retailer         atomic.Value
	loseResponse     atomic.Bool
	differentCatalog atomic.Bool
	occupiedRetailer atomic.Bool
}

func (f *metaGraphFixture) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	query := r.URL.Query()
	switch r.URL.Path {
	case "/dialog/oauth":
		if query.Get("redirect_uri") != metaCallback || query.Get("client_id") != "shuomi-meta-simulation" || query.Get("state") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		callback := metaCallback + "?" + url.Values{"state": {query.Get("state")}, "code": {"meta-consent"}}.Encode()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Referrer-Policy", "no-referrer")
		_, _ = fmt.Fprintf(w, "<!doctype html><html lang=\"zh-CN\"><title>模拟 Meta 授权</title><h1>模拟 Meta 授权</h1><form action=\"%s\" method=\"get\"><input type=\"hidden\" name=\"state\" value=\"%s\"><input type=\"hidden\" name=\"code\" value=\"meta-consent\"><button>允许硕米管理商品目录</button></form></html>", html.EscapeString(callback), html.EscapeString(query.Get("state")))
	case "/fixture/writes":
		_ = json.NewEncoder(w).Encode(f.writes.Load())
	case "/v27.0/oauth/access_token":
		if query.Get("code") != "meta-consent" || query.Get("redirect_uri") != metaCallback {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"synthetic-meta-token"}`))
	case "/v27.0/debug_token":
		if query.Get("input_token") != "synthetic-meta-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"app_id": "shuomi-meta-simulation", "user_id": "789", "is_valid": true, "expires_at": time.Now().Add(time.Hour).Unix(),
			"scopes": []string{"business_management", "catalog_management", "whatsapp_business_management", "whatsapp_business_messaging", "public_profile"},
		}})
	default:
		if r.Header.Get("Authorization") != "Bearer synthetic-meta-token" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		f.graph(w, r)
	}
}

func (f *metaGraphFixture) graph(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/v27.0/me/businesses":
		_, _ = w.Write([]byte(`{"data":[{"id":"101","name":"模拟企业"}]}`))
	case "/v27.0/101/owned_product_catalogs":
		_, _ = w.Write([]byte(`{"data":[{"id":"123","name":"模拟商品目录"},{"id":"124","name":"未授予管理权的目录"}]}`))
	case "/v27.0/123/assigned_users":
		_, _ = w.Write([]byte(`{"data":[{"id":"789","tasks":["MANAGE"]}]}`))
	case "/v27.0/124/assigned_users":
		_, _ = w.Write([]byte(`{"data":[{"id":"789","tasks":["ADVERTISE"]}]}`))
	case "/v27.0/101/owned_whatsapp_business_accounts":
		_, _ = w.Write([]byte(`{"data":[{"id":"456","name":"模拟 WhatsApp 账号"},{"id":"457","name":"其他 WhatsApp 账号"}]}`))
	case "/v27.0/456/product_catalogs":
		id := "123"
		if f.differentCatalog.Load() {
			id = "124"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"id": id}}})
	case "/v27.0/457/product_catalogs":
		_, _ = w.Write([]byte(`{"data":[{"id":"124"}]}`))
	case "/v27.0/123/products":
		var input struct {
			Upsert     *bool  `json:"allow_upsert"`
			Price      int64  `json:"price"`
			RetailerID string `json:"retailer_id"`
		}
		if r.Method != http.MethodPost || json.NewDecoder(r.Body).Decode(&input) != nil ||
			input.Upsert == nil || *input.Upsert || input.Price != 1995 || !strings.HasPrefix(input.RetailerID, "sm-") {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if f.occupiedRetailer.Load() {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("{\"error\":{\"type\":\"OAuthException\",\"code\":10800}}"))
			return
		}
		f.writes.Add(1)
		if f.loseResponse.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		f.retailer.Store(input.RetailerID)
		_, _ = w.Write([]byte(`{"id":"555"}`))
	case "/v27.0/555":
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "555", "retailer_id": f.retailer.Load(), "product_catalog": map[string]string{"id": "123"}})
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestMetaMockAuthorizationPublicationAndSavedResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	storeDB, productDB := googleFlowDatabases(t, ctx)
	require.NoError(t, storecenter.InstallMetaCatalogSimulationSchema(ctx, storeDB))
	require.NoError(t, persistence.InstallMetaPlanSchema(ctx, productDB))
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: "org-a",
		EffectiveOrganizationID: "org-a", UserID: "actor-a", EffectiveMemberID: "member-a", TokenExpiresAt: time.Now().Add(5 * time.Minute)})
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	repo, err := storecenter.NewMemberScopedStoreRepository(storeDB, googleTestAccess{scope})
	require.NoError(t, err)
	newStore, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: scope.OrganizationID,
		ActorSubject: scope.ActorID, Name: "Meta 模拟店铺", Platform: "metacatalog", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now()})
	require.NoError(t, err)
	store, _, err := repo.CreateOrReplay(ctx, scope.OrganizationID, newStore)
	require.NoError(t, err)
	require.NoError(t, storeDB.Exec("UPDATE workbench_stores SET service_status='active', service_started_at=?, service_expires_at=? WHERE id=?",
		time.Now().Add(-time.Minute), time.Now().Add(time.Hour), store.ID()).Error)

	graph := &metaGraphFixture{}
	upstream := httptest.NewServer(graph)
	defer upstream.Close()
	adapter, err := metaadapter.NewSimulation(upstream.URL)
	require.NoError(t, err)
	var key [32]byte
	_, err = rand.Read(key[:])
	require.NoError(t, err)
	cipher, err := credentialprotection.NewAESGCM("meta-fixture", key[:])
	require.NoError(t, err)
	connection, err := storecenter.NewMetaCatalogSimulation(repo, adapter, cipher, metaCallback)
	require.NoError(t, err)
	publication, sourceID := newMetaPublication(t, ctx, productDB, connection)
	org := scope.OrganizationID

	if os.Getenv("META_WORKBENCH_BROWSER") == "1" {
		runMetaBrowser(t, ctx, storeDB, scope, repo, connection, publication, store.ID(), sourceID, upstream.URL)
		require.EqualValues(t, 1, graph.writes.Load())
		return
	}

	begin, err := connection.Begin(ctx, org, store.ID(), int64(store.Version()))
	require.NoError(t, err)
	redirect, err := url.Parse(begin.AuthorizationURL)
	require.NoError(t, err)
	state := redirect.Query().Get("state")
	require.Error(t, connection.Complete(ctx, "org-b", store.ID(), begin.AttemptID, state, "meta-consent"))
	require.Error(t, connection.Complete(ctx, org, store.ID(), begin.AttemptID, strings.Repeat("a", 43), "meta-consent"))
	require.NoError(t, connection.Complete(ctx, org, store.ID(), begin.AttemptID, state, "meta-consent"))
	require.Error(t, connection.Complete(ctx, org, store.ID(), begin.AttemptID, state, "meta-consent"))
	choices, err := connection.Choices(ctx, org, store.ID(), begin.AttemptID)
	require.NoError(t, err)
	require.Len(t, choices.Catalogs, 1)
	_, err = connection.Bind(ctx, org, store.ID(), begin.AttemptID, "124", "")
	require.Error(t, err)
	_, err = connection.Bind(ctx, org, store.ID(), begin.AttemptID, "123", "457")
	require.Error(t, err)
	view, err := connection.Bind(ctx, org, store.ID(), begin.AttemptID, "123", "456")
	require.NoError(t, err)
	require.True(t, view.Connected)

	input := listingapp.MetaPrepareInput{SourceID: sourceID, StoreID: store.ID(), VariantID: "default", ImageID: "approved-main",
		ProductURL: "https://seller.example.org/desk", Price: "19.95"}
	options, err := publication.Options(ctx, sourceID)
	require.NoError(t, err)
	require.Len(t, options.Images, 1)
	bad := input
	bad.ImageID = "approved-gallery"
	_, err = publication.Prepare(ctx, uuid.NewString(), bad)
	require.Error(t, err)
	bad = input
	bad.Price = "19.950"
	_, err = publication.Prepare(ctx, uuid.NewString(), bad)
	require.ErrorIs(t, err, preparation.ErrInvalid)
	plan, err := publication.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.EqualValues(t, 1995, plan.Product.PriceCents)
	require.Equal(t, "Desk organizer · Standard", plan.Product.Title)
	require.Zero(t, graph.writes.Load())
	result, err := publication.Confirm(ctx, sourceID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, result.Status)
	result, err = publication.Confirm(ctx, sourceID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, "555", result.RemoteID)
	require.EqualValues(t, 1, graph.writes.Load())
	otherPlan, err := publication.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	_, err = publication.Confirm(ctx, sourceID, otherPlan.ID)
	require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded)
	result, err = publication.Result(ctx, sourceID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, &storecenter.MetaObservation{Catalog: "found", WhatsApp: "matched", Facebook: "unverified"}, result.Observation)
	graph.differentCatalog.Store(true)
	result, err = publication.Result(ctx, sourceID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, "different", result.Observation.WhatsApp)

	graph.occupiedRetailer.Store(true)
	input.VariantID = "alternate"
	occupied, err := publication.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	result, err = publication.Confirm(ctx, sourceID, occupied.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionFailedDefinitive, result.Status)
	require.Empty(t, result.RemoteID)
	require.EqualValues(t, 1, graph.writes.Load())
	graph.occupiedRetailer.Store(false)

	graph.loseResponse.Store(true)
	input.VariantID = "alternate"
	lost, err := publication.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	result, err = publication.Confirm(ctx, sourceID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
	result, err = publication.Result(ctx, sourceID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
	otherLost, err := publication.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	_, err = publication.Confirm(ctx, sourceID, otherLost.ID)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
	require.EqualValues(t, 2, graph.writes.Load())
	require.NoError(t, connection.Disconnect(ctx, org, store.ID(), view.Version))
	result, err = publication.Result(ctx, sourceID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, "555", result.RemoteID)
	require.Equal(t, "unverified", result.Observation.Catalog)
}

func newMetaPublication(t *testing.T, ctx context.Context, db *gorm.DB, connection *storecenter.MetaCatalogSimulation) (*listingapp.MetaPublication, string) {
	t.Helper()
	roles := googleLiveRoles{}
	permissions := authz.DefaultListingKitAuthorizer()
	authorization, err := preparation.NewContextAuthorizer(roles, permissions)
	require.NoError(t, err)
	collectionAuth, err := collection.NewContextAuthorizer(roles, permissions)
	require.NoError(t, err)
	source := preparation.SourceItem{ID: uuid.NewString(), PreparationID: uuid.NewString(), CollectionItemID: uuid.NewString(), CollectionRevision: 1,
		Source: collection.Source{ProductKey: "product-a", PublicationID: uuid.NewString(), Version: 1, Kind: "own"}}
	snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: "org-a", ProductKey: "product-a"},
		PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{
			Title: "Desk organizer", Brand: "Bamboo Home", Description: "Bamboo desk organizer",
			Variants: []catalog.Variant{{SourceID: "default", Title: "Standard"}, {SourceID: "alternate", Title: "Alternate"}},
		}}
	collections, err := collection.NewService(googleCollections{}, collectionAuth, googleUnusedSources{})
	require.NoError(t, err)
	reader := googleSources{source: source}
	preparations, err := preparation.NewService(reader, collections, authorization)
	require.NoError(t, err)
	sources, err := preparation.NewSourceSelector(preparations, collections, reader, googleCatalog{snapshot})
	require.NoError(t, err)
	plans, err := persistence.NewMetaPlanStore(ctx, db)
	require.NoError(t, err)
	repository, err := persistence.NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(repository)
	require.NoError(t, err)
	assets := googleAssets{replaced: new(atomic.Bool), inventory: asset.ApprovedAssetInventory{
		Scope:  asset.InventoryScope{TenantID: "org-a", ProductKey: "product-a", TargetPlatform: "product", SourceSnapshotVersion: 1},
		Assets: []asset.ApprovedAsset{{ID: "approved-main", URL: "https://seller.example.org/desk.jpg", Role: asset.RoleMain}, {ID: "approved-gallery", URL: "https://seller.example.org/extra.jpg", Role: asset.RoleGallery}},
	}}
	return &listingapp.MetaPublication{Authorization: authorization, Sources: sources, Assets: assets, Stores: connection, Plans: plans, Kernel: kernel}, source.ID
}

type metaFixtureConnectionStatus struct {
	connections *storecenter.MetaCatalogSimulation
}

func (s metaFixtureConnectionStatus) Status(ctx context.Context, input storecenter.ConnectionStatusInput) (storecenter.ConnectionStatus, error) {
	view, err := s.connections.Read(ctx, input.OrganizationID, input.StoreID)
	if err != nil {
		return storecenter.ConnectionStatusUnavailable, err
	}
	if view.Connected {
		return storecenter.ConnectionStatusConnected, nil
	}
	return storecenter.ConnectionStatusDisconnected, nil
}

func runMetaBrowser(t *testing.T, ctx context.Context, db *gorm.DB, scope collection.Scope,
	repo storecenter.Repository, connection *storecenter.MetaCatalogSimulation, publication *listingapp.MetaPublication, storeID, sourceID, graphOrigin string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer fixture-meta-access" || c.GetHeader("X-Requested-Organization-ID") != scope.OrganizationID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		c.Request = c.Request.WithContext(ctx)
	})
	audit, err := storecenter.NewGormAuditRepository(db)
	require.NoError(t, err)
	service, err := storecenter.NewService(repo, audit, metaFixtureConnectionStatus{connection}, time.Now)
	require.NoError(t, err)
	handler, err := storehttp.NewHandler(service)
	require.NoError(t, err)
	registry := kernelmodule.NewRegistry()
	require.NoError(t, storehttp.NewModule(handler).Register(registry))
	registry.AddRoutes(storehttp.MetaCatalogSimulationRoutes(handler, connection)...)
	registry.AddRoutes(listinghttp.MetaCatalogSimulationRoutes(publication, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil })...)
	for _, route := range registry.Routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	server := httptest.NewServer(router)
	defer server.Close()

	args := []string{"exec", "playwright", "test", "e2e/meta-catalog-isolated.spec.ts", "--project=chromium", "--workers=1", "--reporter=line"}
	executable := "pnpm"
	if runtime.GOOS == "windows" {
		executable, args = "cmd.exe", append([]string{"/d", "/c", "pnpm"}, args...)
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir, err = filepath.Abs("../../../web/listingkit-ui")
	require.NoError(t, err)
	command.Env = append(os.Environ(), "CI=1", "AUTH_SECRET=isolated-meta-workbench-browser-test", "AUTH_URL=http://127.0.0.1:3210",
		"LISTINGKIT_PUBLIC_BASE_URL=http://127.0.0.1:3210", "ZITADEL_ISSUER_URL=https://meta-workbench-test.invalid", "ZITADEL_CLIENT_ID=meta-workbench-test",
		"NEXT_PUBLIC_META_CATALOG_SIMULATION=true", "LISTINGKIT_SERVICE_API_BASE="+server.URL+"/api/v1",
		"META_TEST_STORE="+storeID, "META_TEST_SOURCE="+sourceID, "META_GRAPH_ORIGIN="+graphOrigin)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	t.Log(string(output))
}
