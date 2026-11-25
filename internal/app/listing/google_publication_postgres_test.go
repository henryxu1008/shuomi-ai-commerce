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
	"sync"
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
	var insertions, refreshes, merchantRequests atomic.Int32
	var refuseRefresh atomic.Bool
	var refreshHook atomic.Pointer[func(http.ResponseWriter, *http.Request)]
	var submittedOffers sync.Map
	var processed atomic.Bool
	var failPreflight, loseResponse atomic.Bool
	var replaceImageDuringPreflight, imageReplaced atomic.Bool
	var challenge atomic.Value
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/token" {
			merchantRequests.Add(1)
		}
		switch {
		case r.URL.Path == "/token":
			if err := r.ParseForm(); err != nil {
				t.Errorf("invalid OAuth grant: %v", err)
			}
			if r.Form.Get("grant_type") == "refresh_token" {
				refreshes.Add(1)
				if hook := refreshHook.Load(); hook != nil {
					(*hook)(w, r)
					return
				}
				if refuseRefresh.Swap(false) {
					w.Header().Set("Retry-After", "1")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"error":"rate_limit_exceeded"}`))
					return
				}
				if r.Form.Get("refresh_token") != "pretend-refresh" {
					t.Error("refresh did not use the bound credential")
				}
				_, _ = w.Write([]byte(`{"access_token":"refreshed-access","token_type":"Bearer","expires_in":3600}`))
				return
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
			key := body.ContentLanguage + "~" + body.FeedLabel + "~" + body.OfferID
			submittedOffers.Store(key, true)
			// The Merchant SDK uses a URL-safe base64 identifier where required.
			submittedOffers.Store(base64.RawURLEncoding.EncodeToString([]byte(key)), true)
			if loseResponse.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"name": "accounts/123/productInputs/" + key, "product": "accounts/123/products/" + key})
		case strings.HasPrefix(r.URL.Path, "/products/v1/accounts/123/products/"):
			if replaceImageDuringPreflight.Swap(false) {
				imageReplaced.Store(true)
			}
			if failPreflight.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			remote := strings.TrimPrefix(r.URL.Path, "/products/v1/")
			productKey := strings.TrimPrefix(remote, "accounts/123/products/")
			if _, submitted := submittedOffers.Load(productKey); !processed.Load() || !submitted {
				w.WriteHeader(http.StatusNotFound)
				return
			}
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
	registration := storeapp.GoogleApplicationRegistration{Application: app, OAuth: oauth, Accounts: accountClient, Protection: protection}
	connections, err := storeapp.NewGoogleConnections(storeRepo, registration)
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
		replaced: &imageReplaced,
		inventory: asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: "product-a", TargetPlatform: "product", SourceSnapshotVersion: 1},
			Assets: []asset.ApprovedAsset{{ID: "approved-main", URL: "https://seller.example.org/desk.jpg", Role: asset.RoleMain}}},
	}, Stores: connections, Plans: planRepo, Kernel: kernel}
	if os.Getenv("GOOGLE_WORKBENCH_BROWSER") == "1" {
		refuseRefresh.Store(true)
		runGoogleWorkbenchBrowser(t, ctx, storeDB, scope, storeRepo, connections, service, store.ID(), source.ID, &processed, &refreshes, &merchantRequests)
		require.Equal(t, int32(1), insertions.Load())
		require.Equal(t, int32(3), refreshes.Load(), "temporary refusal, original confirmation and original record lookup")
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

	staleImage, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	replaceImageDuringPreflight.Store(true)
	stopped, err = service.Confirm(ctx, source.ID, staleImage.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, stopped.Items[0].Status)
	require.Zero(t, insertions.Load(), "a changed approved image must not be sent")
	saved, err := service.Result(ctx, source.ID, staleImage.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, saved.Items[0].Status)
	imageReplaced.Store(false)

	plan, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.Equal(t, int32(0), insertions.Load(), "preview must not send")
	require.NoError(t, expireGoogleGrant(storeDB, store.ID()))
	require.NoError(t, func() error {
		_, err := connections.PublicationBinding(ctx, scope.OrganizationID, store.ID())
		return err
	}(), "an expired bound store remains available for local preparation")
	require.Zero(t, refreshes.Load(), "local preparation must not refresh")
	var captured submission.AcquireExecutionCommand
	service.Kernel = googleCaptureExecution{GooglePublicationKernel: kernel, command: &captured}
	published, err := service.Confirm(ctx, source.ID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, published.Items[0].Status)
	require.Equal(t, int32(1), refreshes.Load())
	service.Kernel = kernel
	expectedPayload, err := json.Marshal([]any{plan.Version, plan.ProductHash, plan.InventoryHash,
		storeapp.GooglePublicationBinding{Target: plan.Target, Generation: plan.Generation}, plan.Offers[0]})
	require.NoError(t, err)
	require.JSONEq(t, string(expectedPayload), string(captured.Payload))
	intent := collection.StableID(scope.OrganizationID, "google-product-insert", plan.ID, plan.Offers[0].ID)
	require.Equal(t, intent, captured.IntentKey)
	executionScope := submission.ExecutionScope{OrganizationID: scope.OrganizationID}
	publishedAttempt, err := kernel.ReadIntent(ctx, executionScope, intent)
	require.NoError(t, err)
	originalPlan, err := planRepo.Read(ctx, scope, plan.ID)
	require.NoError(t, err)
	require.True(t, originalPlan.SameConfirmation(plan))
	require.Equal(t, plan.CreatedAt, originalPlan.CreatedAt)
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
	require.Len(t, options.Records, 4)
	processed.Store(true)
	observed, err := service.Result(ctx, source.ID, plan.ID)
	require.NoError(t, err)
	require.NotNil(t, observed.Items[0].Observation)
	require.Equal(t, []string{"US"}, observed.Items[0].Observation.FreeListings.Approved)
	require.Equal(t, plan.Offers[0].ID, observed.Items[0].OfferID)
	input.Variants[0].VariantID = "backup"
	lost, err := service.Prepare(ctx, uuid.NewString(), input)
	require.NoError(t, err)
	require.NoError(t, expireGoogleGrant(storeDB, store.ID()))
	secondRepo, err := storecenter.NewMemberScopedStoreRepository(storeDB.Session(&gorm.Session{NewDB: true}), googleTestAccess{scope})
	require.NoError(t, err)
	waiting := make(chan struct{})
	secondConnections, err := storeapp.NewGoogleConnections(&googleCredentialWaitProbe{GoogleConnectionStore: secondRepo, entered: waiting}, registration)
	require.NoError(t, err)
	secondEffects, err := persistence.NewOfficialEffectsRepository(productDB.Session(&gorm.Session{NewDB: true}))
	require.NoError(t, err)
	secondKernel, err := submission.NewExecutionKernel(secondEffects)
	require.NoError(t, err)
	secondService := *service
	secondService.Stores, secondService.Kernel = secondConnections, secondKernel
	replayWaiting := make(chan struct{})
	replayConnections, err := storeapp.NewGoogleConnections(&googleCredentialWaitProbe{GoogleConnectionStore: secondRepo, entered: replayWaiting}, registration)
	require.NoError(t, err)
	replayService := secondService
	replayService.Stores = replayConnections
	entered, release := make(chan struct{}), make(chan struct{})
	rotate := func(w http.ResponseWriter, r *http.Request) {
		requireBoundRefreshToken(t, r, "pretend-refresh")
		close(entered)
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write([]byte(`{"access_token":"rotated-access","refresh_token":"rotated-refresh","token_type":"Bearer","expires_in":3600}`))
	}
	refreshHook.Store(&rotate)
	type publicationReply struct {
		result listingapp.GooglePublicationResult
		err    error
	}
	confirmReply, resultReply, replayReply := make(chan publicationReply, 1), make(chan publicationReply, 1), make(chan publicationReply, 1)
	loseResponse.Store(true)
	go func() {
		result, err := service.Confirm(ctx, source.ID, lost.ID)
		confirmReply <- publicationReply{result, err}
	}()
	awaitGoogleTestSignal(t, entered)
	go func() {
		result, err := secondService.Result(ctx, source.ID, plan.ID)
		resultReply <- publicationReply{result, err}
	}()
	go func() {
		result, err := replayService.Confirm(ctx, source.ID, lost.ID)
		replayReply <- publicationReply{result, err}
	}()
	awaitGoogleTestSignal(t, waiting)
	awaitGoogleTestSignal(t, replayWaiting)
	close(release)
	confirmed, queried, replayed := <-confirmReply, <-resultReply, <-replayReply
	require.NoError(t, confirmed.err)
	require.NoError(t, replayed.err)
	require.Contains(t, []submission.ExecutionStatus{confirmed.result.Items[0].Status, replayed.result.Items[0].Status}, submission.ExecutionOutcomeUnknown)
	require.NoError(t, queried.err)
	require.NotNil(t, queried.result.Items[0].Observation)
	require.Equal(t, int32(2), refreshes.Load(), "two service instances share one exchange for this revision")
	require.Equal(t, int32(2), insertions.Load(), "one accepted operation and one response lost")

	require.NoError(t, expireGoogleGrant(storeDB, store.ID()))
	providerCount := merchantRequests.Load()
	unknown, err := service.Confirm(ctx, source.ID, lost.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, unknown.Items[0].Status)
	require.Equal(t, providerCount, merchantRequests.Load(), "replay must only return the original result")
	require.Equal(t, int32(2), refreshes.Load(), "replay must not refresh")
	verifyRotation := func(w http.ResponseWriter, r *http.Request) {
		requireBoundRefreshToken(t, r, "rotated-refresh")
		_, _ = w.Write([]byte(`{"access_token":"lookup-access","token_type":"Bearer","expires_in":3600}`))
	}
	refreshHook.Store(&verifyRotation)
	unknown, err = service.Result(ctx, source.ID, lost.ID)
	require.NoError(t, err)
	require.NotNil(t, unknown.Items[0].Observation)
	require.Equal(t, "accounts/123/dataSources/456", unknown.Items[0].Observation.DataSource)
	require.Equal(t, submission.ExecutionOutcomeUnknown, unknown.Items[0].Status, "a product observation is not linked write evidence")
	require.Equal(t, int32(2), insertions.Load())
	require.Equal(t, int32(3), refreshes.Load())
	afterLookup, err := kernel.ReadIntent(ctx, executionScope, intent)
	require.NoError(t, err)
	require.Equal(t, publishedAttempt, afterLookup, "lookup and renewal must preserve the original attempt and payload digest")
	binding, err := connections.PublicationBinding(ctx, scope.OrganizationID, store.ID())
	require.NoError(t, err)
	require.Equal(t, storeapp.GooglePublicationBinding{Target: plan.Target, Generation: plan.Generation}, binding)

	// Exercise the recovery boundaries against the same isolated Store database
	// and simulation. The fixture restores a new grant through real reauthorization.
	refreshFixture := googleRefreshFixture{ctx: ctx, db: storeDB, scope: scope, storeID: store.ID(),
		repo: storeRepo, application: registration, hook: &refreshHook, calls: &refreshes, challenge: &challenge,
		initialPlan: duplicate, originalPlan: plan, listing: service}
	refreshFixture.testRecovery(t)
}

type googleCaptureExecution struct {
	listingapp.GooglePublicationKernel
	command *submission.AcquireExecutionCommand
}

func (k googleCaptureExecution) PrepareGoogleExecution(ctx context.Context, cmd submission.AcquireExecutionCommand) (submission.PreparedAcquisition, error) {
	*k.command = cmd
	return k.GooglePublicationKernel.PrepareGoogleExecution(ctx, cmd)
}

type googleCredentialWaitProbe struct {
	storeapp.GoogleConnectionStore
	entered chan struct{}
	once    sync.Once
}

func (g *googleCredentialWaitProbe) GoogleCredential(ctx context.Context, request storecenter.GoogleCredentialRequest, claimID string) (storecenter.GoogleCredentialState, error) {
	state, err := g.GoogleConnectionStore.GoogleCredential(ctx, request, claimID)
	if err == nil && claimID != "" && state.ClaimID != claimID && state.Phase == storecenter.GoogleRefreshPossible {
		g.once.Do(func() { close(g.entered) })
	}
	return state, err
}

func requireBoundRefreshToken(t *testing.T, request *http.Request, token string) {
	t.Helper()
	if request.Form.Get("refresh_token") != token {
		t.Error("OAuth exchange did not use the expected bound token")
	}
}

func awaitGoogleTestSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("fixture did not reach the expected refresh step")
	}
}

func expireGoogleGrant(db *gorm.DB, storeID string) error {
	return db.Exec("UPDATE workbench_store_google_oauth_grants SET access_expires_at = clock_timestamp() - interval '1 second' WHERE store_id = ?", storeID).Error
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

type googleAssets struct {
	inventory asset.ApprovedAssetInventory
	replaced  *atomic.Bool
}

func (a googleAssets) GetApprovedInventory(context.Context, asset.InventoryScope) (asset.ApprovedAssetInventory, error) {
	inventory := a.inventory
	if a.replaced.Load() {
		inventory.Assets = []asset.ApprovedAsset{{ID: "new-main", URL: "https://seller.example.org/new-desk.jpg", Role: asset.RoleMain}}
	}
	return inventory, nil
}

func runGoogleWorkbenchBrowser(
	t *testing.T, ctx context.Context, db *gorm.DB, scope collection.Scope,
	repo storecenter.Repository, connections *storeapp.GoogleConnections,
	publication *listingapp.GooglePublication, storeID, sourceID string, processed *atomic.Bool, refreshes, merchantRequests *atomic.Int32,
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
	var reads atomic.Int32
	router.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == listinghttp.GoogleListingPath+"/result" && reads.Add(1) == 1 {
			if err := expireGoogleGrant(db, storeID); err != nil {
				t.Errorf("expire fixture before reading original record: %v", err)
			}
		}
		refreshBefore, merchantBefore := refreshes.Load(), merchantRequests.Load()
		c.Next()
		if strings.HasSuffix(path, "/connection/google/bind") && c.Writer.Status() == http.StatusOK {
			if err := expireGoogleGrant(db, storeID); err != nil {
				t.Errorf("expire fixture after binding: %v", err)
			}
		}
		if path == "/api/v1/workbench/stores" || strings.HasPrefix(path, listinghttp.GoogleListingPath+"/options/") || path == listinghttp.GoogleListingPath+"/prepare" {
			if refreshBefore != refreshes.Load() || merchantBefore != merchantRequests.Load() {
				t.Error("local listing and preview must make no provider request")
			}
		}
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
