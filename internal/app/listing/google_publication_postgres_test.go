package listingapp_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	googleadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/googlemerchant"
	persistence "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	google "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/googlemerchant"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type googleTestAccess struct{ scope collection.Scope }

func (a googleTestAccess) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if org != a.scope.OrganizationID {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: a.scope.ActorID, MemberID: a.scope.MemberID, Administrator: true, CanWrite: true}, nil
}

type googleTestTransport struct{ fixture *url.URL }

func (m googleTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL.Scheme, clone.URL.Host = m.fixture.Scheme, m.fixture.Host
	clone.Host = m.fixture.Host
	return http.DefaultTransport.RoundTrip(clone)
}

func TestGoogleMockAuthorizationPublicationAndSavedResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	storeDB, productDB := googleFlowDatabases(t, ctx)
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	storeRepo, err := storecenter.NewMemberScopedStoreRepository(storeDB, googleTestAccess{scope})
	require.NoError(t, err)
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{
		ID: uuid.NewString(), OrganizationID: scope.OrganizationID, ActorSubject: scope.ActorID, Name: "Google test store",
		Platform: "googlemerchant", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	store, _, err = storeRepo.CreateOrReplay(ctx, scope.OrganizationID, store)
	require.NoError(t, err)
	var insertions atomic.Int32
	var processed atomic.Bool
	var failPreflight, loseResponse atomic.Bool
	var challenge atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("invalid OAuth grant: %v", err)
			}
			digest := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
			expected := challenge.Load()
			if code, ok := strings.CutPrefix(r.Form.Get("code"), "browser-consent-"); ok {
				expected = code
			}
			if base64.RawURLEncoding.EncodeToString(digest[:]) != expected {
				t.Error("PKCE challenge was not paired with the callback")
			}
			_, _ = w.Write([]byte(`{"access_token":"pretend-access","refresh_token":"pretend-refresh","token_type":"Bearer","expires_in":3600,"scope":"https://www.googleapis.com/auth/content"}`))
		case r.URL.Path == "/accounts/v1/accounts":
			_, _ = w.Write([]byte(`{"accounts":[{"name":"accounts/123","accountId":"123","accountName":"Mock merchant"}]}`))
		case r.URL.Path == "/datasources/v1/accounts/123/dataSources":
			_, _ = w.Write([]byte(`{"dataSources":[{"name":"accounts/123/dataSources/456","dataSourceId":"456","input":"API","displayName":"Shuomi products","primaryProductDataSource":{"feedLabel":"US","contentLanguage":"en"}}]}`))
		case r.URL.Path == "/datasources/v1/accounts/123/dataSources/456":
			_, _ = w.Write([]byte(`{"name":"accounts/123/dataSources/456","dataSourceId":"456","input":"API","primaryProductDataSource":{"feedLabel":"US","contentLanguage":"en","destinations":[{"destination":"FREE_LISTINGS","state":"ENABLED"}]}}`))
		case r.URL.Path == "/products/v1/accounts/123/productInputs:insert" && r.Method == http.MethodPost:
			insertions.Add(1)
			var body struct{ OfferID, ContentLanguage, FeedLabel string }
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || r.URL.Query().Get("dataSource") != "accounts/123/dataSources/456" {
				t.Errorf("bad insert %v %s", err, r.URL)
			}
			if loseResponse.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			key := body.ContentLanguage + "~" + body.FeedLabel + "~" + body.OfferID
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "accounts/123/productInputs/" + key, "product": "accounts/123/products/" + key})
		case strings.HasPrefix(r.URL.Path, "/products/v1/accounts/123/products/"):
			if failPreflight.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			if !processed.Load() || insertions.Load() == 0 {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			remote := strings.TrimPrefix(r.URL.Path, "/products/v1/")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": remote, "dataSource": "accounts/123/dataSources/456",
				"productStatus": map[string]any{"destinationStatuses": []map[string]any{{"reportingContext": "FREE_LISTINGS", "approvedCountries": []string{"US"}}}}})
		default:
			t.Errorf("unexpected provider request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	fixture, err := url.Parse(server.URL)
	require.NoError(t, err)
	mockClient := &http.Client{Transport: googleTestTransport{fixture}}
	accountClient, err := googleadapter.NewClient(mockClient)
	require.NoError(t, err)
	app := google.Application{ID: "demo-client", Revision: "v1", CallbackURL: "https://app.test/workbench/stores/googlemerchant/callback"}
	oauth, err := googleadapter.NewOAuth(app, "mock-secret", mockClient)
	require.NoError(t, err)
	protection, err := googleadapter.NewCredentialProtection("test-key", make([]byte, 32))
	require.NoError(t, err)
	connections, err := storeapp.NewGoogleConnections(storeRepo, storeapp.GoogleApplicationRegistration{Application: app, OAuth: oauth, Accounts: accountClient, Protection: protection})
	require.NoError(t, err)
	now := time.Now().UTC()
	// The isolated listing fixture already has a Store service entitlement.
	require.NoError(t, storeDB.Exec("UPDATE workbench_stores SET service_status='active',service_started_at=?,service_expires_at=? WHERE id=?", now.Add(-time.Minute), now.Add(time.Hour), store.ID()).Error)

	roles := googleLiveRoles{}
	permissions := authz.DefaultListingKitAuthorizer()
	authorize, err := preparation.NewContextAuthorizer(roles, permissions)
	require.NoError(t, err)
	collectionAuth, err := collection.NewContextAuthorizer(roles, permissions)
	require.NoError(t, err)
	source := preparation.SourceItem{ID: uuid.NewString(), PreparationID: uuid.NewString(), CollectionItemID: uuid.NewString(), CollectionRevision: 1,
		Source: collection.Source{ProductKey: "product-a", PublicationID: uuid.NewString(), Version: 1, Kind: "own"}}
	snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: "product-a"},
		PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Desk organizer", Description: "Bamboo organizer", Variants: []catalog.Variant{{SourceID: "default", Title: "Standard"}, {SourceID: "backup", Title: "Alternate"}}}}
	collections, err := collection.NewService(googleCollections{}, collectionAuth, googleUnusedSources{})
	require.NoError(t, err)
	sourceReader := googleSources{source: source}
	preparations, err := preparation.NewService(sourceReader, collections, authorize)
	require.NoError(t, err)
	selector, err := preparation.NewSourceSelector(preparations, collections, sourceReader, googleCatalog{snapshot})
	require.NoError(t, err)
	planRepo, err := persistence.NewGooglePlanStore(ctx, productDB)
	require.NoError(t, err)
	effects, err := persistence.NewOfficialEffectsRepository(productDB)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(effects)
	require.NoError(t, err)
	service := &listingapp.GooglePublication{Authorization: authorize, Sources: selector, Assets: googleAssets{
		inventory: asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: "product-a", TargetPlatform: "product", SourceSnapshotVersion: 1},
			Assets: []asset.ApprovedAsset{{ID: "approved-main", URL: "https://seller.example.org/desk.jpg", Role: asset.RoleMain}}},
	}, Stores: connections, Plans: planRepo, Kernel: kernel}
	if os.Getenv("GOOGLE_WORKBENCH_BROWSER") == "1" {
		runGoogleWorkbenchBrowser(t, ctx, storeDB, scope, storeRepo, connections, service, store.ID(), source.ID, &processed)
		require.Equal(t, int32(1), insertions.Load())
		return
	}
	begin, err := connections.Begin(ctx, scope.OrganizationID, store.ID(), store.Version())
	require.NoError(t, err)
	authorizationURL, err := url.Parse(begin.AuthorizationURL)
	require.NoError(t, err)
	challenge.Store(authorizationURL.Query().Get("code_challenge"))
	require.NoError(t, connections.Complete(ctx, scope.OrganizationID, store.ID(), begin.AttemptID, authorizationURL.Query().Get("state"), "mock-consent-code"))
	accounts, err := connections.Accounts(ctx, scope.OrganizationID, store.ID(), begin.AttemptID, "")
	require.NoError(t, err)
	require.Equal(t, "123", accounts.Accounts[0].ID)
	sources, err := connections.Sources(ctx, scope.OrganizationID, store.ID(), begin.AttemptID, "123", "")
	require.NoError(t, err)
	require.Equal(t, "456", sources.Sources[0].ID)
	bound, err := connections.Bind(ctx, scope.OrganizationID, store.ID(), begin.AttemptID, storecenter.GoogleAccountSelection{
		AccountID: "123", DataSourceID: "456", ContentLanguage: "en", FeedLabel: "US", DedicatedSourceConfirmed: true,
	})
	require.NoError(t, err)
	require.Equal(t, storecenter.ConnectionStatusConnected, bound.Status)
	options, err := service.Options(ctx, source.ID)
	require.NoError(t, err)
	require.Equal(t, "Desk organizer", options.Title)
	input := listingapp.GooglePrepareInput{SourceID: source.ID, StoreID: store.ID(), ImageID: "approved-main",
		Variants: []listingapp.GoogleProductInput{{VariantID: "default", Link: "https://seller.example.org/desk", Price: "19.95", Currency: "USD", Availability: "IN_STOCK", Condition: "NEW"}}}
	preflightPlan, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	failPreflight.Store(true)
	stopped, err := service.Confirm(ctx, source.ID, preflightPlan.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, stopped.Items[0].Status)
	require.Zero(t, insertions.Load())
	failPreflight.Store(false)
	plan, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.Equal(t, int32(0), insertions.Load(), "preview must not send")
	published, err := service.Confirm(ctx, source.ID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, published.Items[0].Status)
	_, err = service.Confirm(ctx, source.ID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, int32(1), insertions.Load(), "replay must preserve the original write")
	duplicate, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	_, err = service.Confirm(ctx, source.ID, duplicate.ID)
	require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded)
	require.Equal(t, int32(1), insertions.Load(), "the processed product still returns 404, but the original accepted write blocks a fresh plan")
	options, err = service.Options(ctx, source.ID)
	require.NoError(t, err)
	require.Len(t, options.Records, 3)
	processed.Store(true)
	observed, err := service.Result(ctx, source.ID, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, observed.Items[0].Observation)
	require.Equal(t, []string{"US"}, observed.Items[0].Observation.FreeListings.Approved)
	require.Equal(t, plan.Offers[0].ID, observed.Items[0].OfferID)
	processed.Store(false)
	input.Variants[0].VariantID = "backup"
	lost, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	loseResponse.Store(true)
	unknown, err := service.Confirm(ctx, source.ID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, unknown.Items[0].Status)
	unknown, err = service.Confirm(ctx, source.ID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, unknown.Items[0].Status)
	require.Equal(t, int32(2), insertions.Load(), "loss and retry of one response must not cause another write")
	processed.Store(true)
	unknown, err = service.Result(ctx, source.ID, lost.ID)
	require.NoError(t, err)
	require.NotNil(t, unknown.Items[0].Observation)
	require.Equal(t, submission.ExecutionOutcomeUnknown, unknown.Items[0].Status, "a product observation is not linked write evidence")

}

func googleFlowDatabases(t *testing.T, ctx context.Context) (*gorm.DB, *gorm.DB) {
	t.Helper()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("google_store"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("synthetic-google-test"), tcpostgres.BasicWaitStrategies())
	if container != nil {
		t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	}
	require.NoError(t, err)
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(connection string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(connection), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	storeDB := open(dsn)
	require.NoError(t, storeDB.Exec("CREATE DATABASE google_listing").Error)
	productDSN, err := url.Parse(dsn)
	require.NoError(t, err)
	productDSN.Path = "/google_listing"
	productDB := open(productDSN.String())
	pool, err := storeDB.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallGoogleMerchantSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, storeDB, storecenter.RuntimeCapabilities{GoogleMerchant: true, TikTokShop: true, WalmartUS: true}))
	require.NoError(t, persistence.InstallSchema(productDB))
	require.NoError(t, persistence.InstallGooglePlanSchema(ctx, productDB))
	return storeDB, productDB
}

type googleLiveRoles struct{}

func (googleLiveRoles) ResolveLiveRoles(context.Context, string, string) ([]string, error) {
	return []string{"listingkit_admin"}, nil
}

type googleCollections struct{ collection.Repository }
type googleUnusedSources struct {
	sourcing.PublishedAcquisitionReader
}
type googleSources struct {
	preparation.Repository
	source preparation.SourceItem
}

func (s googleSources) ReadRetainedSource(_ context.Context, scope collection.Scope, id string) (preparation.SourceItem, error) {
	if id != s.source.ID || scope.ActorID != "actor-a" {
		return preparation.SourceItem{}, preparation.ErrNotFound
	}
	return s.source, nil
}

type googleCatalog struct{ snapshot catalog.PublishedSnapshot }

func (c googleCatalog) GetSnapshot(context.Context, catalog.SnapshotIdentity, uint64) (catalog.PublishedSnapshot, error) {
	return c.snapshot, nil
}

type googleAssets struct{ inventory asset.ApprovedAssetInventory }

func (a googleAssets) GetApprovedInventory(context.Context, asset.InventoryScope) (asset.ApprovedAssetInventory, error) {
	return a.inventory, nil
}

func runGoogleWorkbenchBrowser(
	t *testing.T, ctx context.Context, db *gorm.DB, scope collection.Scope,
	repo storecenter.Repository, connections *storeapp.GoogleConnections,
	publication *listingapp.GooglePublication, storeID, sourceID string, processed *atomic.Bool,
) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if c.GetHeader("Authorization") != "Bearer fixture-google-access" || c.GetHeader("X-Requested-Organization-ID") != scope.OrganizationID {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		identity := authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID,
			UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)}
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), identity))
	})
	audit, err := storecenter.NewGormAuditRepository(db)
	require.NoError(t, err)
	stores, err := storecenter.NewService(repo, audit, googleStatus{connections}, time.Now)
	require.NoError(t, err)
	handler, err := storehttp.NewHandler(stores)
	require.NoError(t, err)
	require.NoError(t, handler.SetGoogleConnections(connections))
	registry := kernelmodule.NewRegistry()
	require.NoError(t, storehttp.NewModule(handler).Register(registry))
	registry.AddRoutes(listinghttp.GoogleRoutes(publication, func(ctx context.Context, _ string) (context.Context, error) { return ctx, nil })...)
	for _, route := range registry.Routes() {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	processed.Store(true)
	args := []string{"exec", "playwright", "test", "e2e/google-merchant-isolated.spec.ts", "--project=chromium", "--workers=1", "--reporter=line"}
	executable := "pnpm"
	if runtime.GOOS == "windows" {
		executable, args = "cmd.exe", append([]string{"/d", "/c", "pnpm"}, args...)
	}
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir, err = filepath.Abs("../../../web/listingkit-ui")
	require.NoError(t, err)
	command.Env = append(os.Environ(),
		"CI=1", "AUTH_SECRET=isolated-google-workbench-browser-test", "AUTH_URL=http://127.0.0.1:3210",
		"LISTINGKIT_PUBLIC_BASE_URL=http://127.0.0.1:3210",
		"ZITADEL_ISSUER_URL=https://google-workbench-test.invalid", "ZITADEL_CLIENT_ID=google-workbench-test",
		"NEXT_PUBLIC_GOOGLE_MERCHANT_ENABLED=true", "LISTINGKIT_SERVICE_API_BASE="+server.URL+"/api/v1",
		"GOOGLE_TEST_STORE="+storeID, "GOOGLE_TEST_SOURCE="+sourceID)
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	t.Log(string(output))
}

type googleStatus struct{ connections *storeapp.GoogleConnections }

func (s googleStatus) Status(ctx context.Context, input storecenter.ConnectionStatusInput) (storecenter.ConnectionStatus, error) {
	status, err := s.connections.Read(ctx, input.OrganizationID, input.StoreID)
	return status.Status, err
}
