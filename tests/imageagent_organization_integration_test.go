package tests

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	aistore "task-processor/internal/aicapability/store"
	imageworker "task-processor/internal/app/worker/imageagent"
	"task-processor/internal/authidentity"
	zitadel "task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
	imagestore "task-processor/internal/imageagent/store"
	imagetemporal "task-processor/internal/imageagent/temporal"
	imagetools "task-processor/internal/imageagent/tools"
	openai "task-processor/internal/integration/openai"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	platformtemporal "task-processor/internal/platform/temporal"
	"task-processor/internal/product/catalog"
	"task-processor/internal/workbenchcontext"
)

// Only identity issuance, the remote IAM response, Temporal's server boundary
// and object/provider I/O are controlled external substitutes. Scoped admission,
// grant parsing/resolution/authorization, Catalog/run/effect persistence,
// Temporal Client/converter/Activities, Manager and Review adapter are real.
type scope339TemporalServer struct {
	sdkclient.Client
	mu         sync.Mutex
	payload    *commonpb.Payloads
	options    sdkclient.StartWorkflowOptions
	starts     int
	failNext   bool
	executions map[string]bool
}

func (s *scope339TemporalServer) ExecuteWorkflow(_ context.Context, options sdkclient.StartWorkflowOptions, _ interface{}, args ...interface{}) (sdkclient.WorkflowRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.starts++
	if s.executions == nil {
		s.executions = map[string]bool{}
	}
	s.executions[options.ID] = true
	payload, err := converter.GetDefaultDataConverter().ToPayloads(args...)
	if err != nil {
		return nil, err
	}
	s.payload = payload
	s.options = options
	if s.failNext {
		s.failNext = false
		return nil, fmt.Errorf("controlled accepted start response lost")
	}
	return nil, nil
}
func (s *scope339TemporalServer) QueryWorkflow(context.Context, string, string, string, ...interface{}) (converter.EncodedValue, error) {
	return nil, serviceerror.NewNotFound("controlled server has no running workflow")
}
func (s *scope339TemporalServer) input(t *testing.T) imagetemporal.WorkflowInput {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	var input imagetemporal.WorkflowInput
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(s.payload, &input))
	return input
}

type scope339IAM struct {
	revoked atomic.Bool
	failed  atomic.Bool
	calls   atomic.Int32
}

func (s *scope339IAM) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.calls.Add(1)
	if s.failed.Load() {
		http.Error(w, "controlled authorization unavailable", 503)
		return
	}
	var req struct {
		Filters []map[string]json.RawMessage `json:"filters"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || len(req.Filters) < 2 {
		http.Error(w, "invalid query", 400)
		return
	}
	var users struct {
		IDs []string `json:"ids"`
	}
	_ = json.Unmarshal(req.Filters[0]["inUserIds"], &users)
	if len(users.IDs) != 1 {
		http.Error(w, "missing actor", 400)
		return
	}
	orgs := []string{"B", "C"}
	if len(req.Filters) == 3 {
		if r.Header.Get("Authorization") != "Bearer service339" {
			http.Error(w, "wrong service credential", 401)
			return
		}
		var org struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(req.Filters[2]["organizationId"], &org)
		orgs = []string{org.ID}
	}
	entries := []any{}
	if !s.revoked.Load() {
		for _, org := range orgs {
			role := authz.EnterpriseRoleKey(org, 1)
			if users.IDs[0] == "admin" {
				role = "listingkit_admin"
			}
			entries = append(entries, map[string]any{"id": "grant-" + org, "project": map[string]string{"id": "project"}, "organization": map[string]string{"id": org}, "user": map[string]string{"id": users.IDs[0]}, "state": "STATE_ACTIVE", "roles": []any{map[string]string{"key": role}}})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"pagination": map[string]string{"totalResult": fmt.Sprint(len(entries)), "appliedLimit": "100"}, "authorizations": entries})
}

type scope339Verifier struct{}

type scope339ImageRolePolicy struct{}

func (scope339ImageRolePolicy) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, key := range keys {
		if key == authz.EnterpriseRoleKey(org, 1) {
			result[key] = []string{"images"}
		}
	}
	return result, nil
}

func (scope339Verifier) Verify(_ context.Context, token string) (authidentity.AuthenticatedIdentity, error) {
	if token != "actor" && token != "admin" {
		return authidentity.AuthenticatedIdentity{}, fmt.Errorf("controlled invalid identity")
	}
	return authidentity.AuthenticatedIdentity{HomeOrganizationID: "A", TenantID: "A", UserID: token, TokenExpiresAt: time.Now().Add(time.Hour)}, nil
}

type scope339Fixture struct {
	db            *gorm.DB
	iam           *scope339IAM
	authClient    *zitadel.AuthorizationClient
	auth          *authz.ListingKitAuthorizer
	temporal      *scope339TemporalServer
	catalogs      map[string]imageagent.AssetCatalog
	providerCalls atomic.Int32
}

func newScope339Fixture(t *testing.T) *scope339Fixture {
	t.Helper()
	dsn := os.Getenv("ISSUE339_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated PostgreSQL ISSUE339_TEST_DSN")
	}
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	root, err := gorm.Open(postgres.Open(dsn), cfg)
	require.NoError(t, err)
	schema := "scope339_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE SCHEMA "+schema).Error)
	db, err := gorm.Open(postgres.Open(dsn+" search_path="+schema), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		raw, _ := db.DB()
		_ = raw.Close()
		_ = root.Exec("DROP SCHEMA " + schema + " CASCADE").Error
		raw, _ = root.DB()
		_ = raw.Close()
	})
	require.NoError(t, imagestore.AutoMigrateOrganizationScope(db))
	require.NoError(t, catalogstore.AutoMigrate(db))
	require.NoError(t, db.AutoMigrate(&openai.AIClientCredential{}))
	require.NoError(t, aistore.AutoMigrateInvocationLedger(db))
	f := &scope339Fixture{db: db, iam: &scope339IAM{}, temporal: &scope339TemporalServer{}, catalogs: map[string]imageagent.AssetCatalog{}}
	upstream := httptest.NewServer(f.iam)
	t.Cleanup(upstream.Close)
	f.authClient = zitadel.NewAuthorizationClient(upstream.URL, upstream.Client())
	f.auth, err = authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	f.auth.SetRolePolicyReader(scope339ImageRolePolicy{})
	repo, err := catalogstore.NewRepository(db)
	require.NoError(t, err)
	for _, org := range []string{"B", "C"} {
		published, err := repo.PublishSnapshot(context.Background(), catalog.PublishRequest{Identity: catalog.SnapshotIdentity{TenantID: org, ProductKey: "product"}, PublicationID: "scope339-publication", Snapshot: catalog.ProductSnapshot{Title: "Controlled shoes", Images: []catalog.Image{{URL: "https://source.example/" + org + ".png", Role: "source", Width: 1200, Height: 1200}}}})
		require.NoError(t, err)
		url := published.Snapshot.Images[0].URL
		value, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{
			Assets:         []imageagent.AuthorizedAsset{{ID: "catalog-image-1", Type: imageagent.AuthorizedAssetSource, URL: url, SourceURL: url, DisplayURL: url, Label: "Product image", Width: 1200, Height: 1200}},
			ProductContext: imageagent.ProductContextRef{ProductID: published.Identity.ProductKey, Title: published.Snapshot.Title, SourceSnapshotVersion: published.Version},
		})
		require.NoError(t, err)
		value.Manifest.Hash = imageagent.CatalogSnapshotHash(value.Assets, value.ProductContext)
		f.catalogs[org] = value
	}
	return f
}

// service composes the current domain owner; it installs no HTTP routes.
func (f *scope339Fixture) service(t *testing.T) *imageagent.Service {
	t.Helper()
	service, err := imageagent.NewService(imagestore.NewOrganizationRepository(f.db), imagetemporal.NewOrganizationClient(f.temporal), scope339Catalog{catalogs: f.catalogs}, imageagent.WithOrganizationScope(), imageagent.WithTenantStartGate(imageagent.TenantAllowlistStartGate{Enabled: true, AllowedTenantIDs: []string{"B", "C"}}))
	require.NoError(t, err)
	return service
}

// context uses the same verified identity, resolver and scoped permission owner
// as current HTTP controls, with the original bounded request lifetime.
func (f *scope339Fixture) context(t *testing.T, actor, organization, permission string) (context.Context, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	identity, err := (scope339Verifier{}).Verify(ctx, actor)
	if err != nil {
		return nil, err
	}
	policy := httproute.OrganizationAccessPolicyLiveWrite
	if permission == authz.PermissionImageAgentRead {
		policy = httproute.OrganizationAccessPolicyCachedRead
	}
	resolver := workbenchcontext.NewResolver(workbenchcontext.NewGrantResolver(f.authClient, workbenchcontext.NewGrantCache(time.Now)), "project", "v1", nil)
	identity, err = resolver.Resolve(ctx, policy, workbenchcontext.ResolveInput{Identity: identity, BearerToken: actor, RequestedOrganizationID: organization})
	if err != nil {
		return nil, err
	}
	ctx = authidentity.WithAuthenticatedIdentity(ctx, identity)
	allowed, err := authz.AuthorizeOrganization(ctx, f.auth, identity.UserID, identity.EffectiveOrganizationID, identity.Roles, permission)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, imageagent.ErrIdentityRequired
	}
	return ctx, nil
}

// A fixed domain port supplies only the published synthetic values. Production
// acquisition catalog selection/ownership remains tested at its current owner.
type scope339Catalog struct {
	catalogs map[string]imageagent.AssetCatalog
}

func (c scope339Catalog) Resolve(_ context.Context, scope imageagent.AssetCatalogScope) (imageagent.AssetCatalog, error) {
	catalog, ok := c.catalogs[scope.TenantID]
	if !ok {
		return imageagent.AssetCatalog{}, imageagent.ErrIdentityRequired
	}
	return catalog, nil
}

func scope339StartInput() imageagent.StartRunInput {
	return imageagent.StartRunInput{
		RunID: "run339", BusinessTaskID: "catalog-input", TargetPlatform: "shein",
		ImagePolicyContext: imageagent.ImagePolicyContext{Country: "us", Family: "default", SceneCategory: "shoes"},
		Mode:               imageagent.RunModeManual, IdempotencyKey: "run-key339", MaxConcurrentSlots: 1,
		Budget: imageagent.Budget{MaxImages: 12, MaxAgentSteps: 20, MaxModelCalls: 30, MaxRepairAttemptsPerSlot: 2, MaxElapsed: 500 * time.Second,
			EnabledLimits: imageagent.BudgetLimitImages | imageagent.BudgetLimitAgentSteps | imageagent.BudgetLimitModelCalls | imageagent.BudgetLimitRepairAttemptsPerSlot | imageagent.BudgetLimitElapsed},
		Plan: imageagent.Plan{Revision: 1, IdempotencyKey: "plan-key339", SourceAssetIDs: []string{"catalog-image-1"}, Slots: []imageagent.Slot{{ID: "slot1", Role: imageagent.SlotRoleMain, Status: imageagent.SlotStatusPending, SourceAssetIDs: []string{"catalog-image-1"}, IdempotencyKey: "slot-key339"}}},
	}
}

func TestOrganizationScopeServicePersistenceAndActivity(t *testing.T) {
	f := newScope339Fixture(t)
	ctx, err := f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.NoError(t, err)
	require.NoError(t, f.service(t).Start(ctx, scope339StartInput()))
	input := f.temporal.input(t)
	require.Equal(t, "B", input.Identity.TenantID)
	require.Equal(t, "actor", input.Identity.UserID)
	require.Equal(t, "run339", input.Identity.RunID)
	require.Equal(t, imageagent.OrganizationScopeProtocol, input.Identity.ScopeProtocol)
	require.Equal(t, imagetemporal.OrganizationTaskQueue, f.temporal.options.TaskQueue)
	require.True(t, strings.HasPrefix(f.temporal.options.ID, "organization-v1:"))
	repo := imagestore.NewOrganizationRepository(f.db)
	scope := imageagent.RunScope{TenantID: "B", OwnerUserID: "actor", RunID: "run339"}
	stored, err := repo.GetProjection(context.Background(), scope)
	require.NoError(t, err)
	require.Equal(t, input.AssetCatalog, stored.AssetCatalog)
	require.Contains(t, stored.AssetCatalog.Assets[0].URL, "/B.png")
	// Rebuild the current service/repository and retry the typed command.
	require.NoError(t, f.service(t).Start(ctx, scope339StartInput()))
	restored := f.temporal.input(t)
	require.Equal(t, input.Identity, restored.Identity)
	readContext, err := f.context(t, "actor", "B", authz.PermissionImageAgentRead)
	require.NoError(t, err)
	_, err = f.service(t).Get(readContext, "run339")
	require.NoError(t, err)
	starts := f.temporal.starts
	for _, tc := range []struct {
		name, actor, org string
		mutate           func(*imageagent.StartRunInput)
	}{
		{"missing organization", "actor", "", nil},
		{"cross organization identity", "actor", "C", nil},
		{"cross organization key", "actor", "C", func(i *imageagent.StartRunInput) { i.RunID = "other339" }},
		{"changed payload", "actor", "B", func(i *imageagent.StartRunInput) { i.Plan.Slots[0].IdempotencyKey = "different" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := scope339StartInput()
			if tc.mutate != nil {
				tc.mutate(&request)
			}
			callContext, callErr := f.context(t, tc.actor, tc.org, authz.PermissionImageAgentWrite)
			if callErr == nil && tc.org == "C" {
				// C may create its own inputs. This negative case reuses B's
				// captured transport owner under C's selected organization.
				captured, ok := authidentity.AuthenticatedIdentityFromContext(callContext)
				require.True(t, ok)
				captured.TenantID = "B"
				callContext = authidentity.WithAuthenticatedIdentity(callContext, captured)
			}
			if callErr == nil {
				callErr = f.service(t).Start(callContext, request)
			}
			require.Error(t, callErr)
			require.Equal(t, starts, f.temporal.starts)
		})
	}
	tampered, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	require.True(t, ok)
	tampered.TenantID = "C"
	require.ErrorIs(t, f.service(t).Start(authidentity.WithAuthenticatedIdentity(ctx, tampered), scope339StartInput()), imageagent.ErrIdentityRequired)
	require.Equal(t, starts, f.temporal.starts)
	for _, actor := range []string{"actor", "admin"} {
		org := "C"
		if actor == "admin" {
			org = "B"
		}
		callContext, callErr := f.context(t, actor, org, authz.PermissionImageAgentRead)
		require.NoError(t, callErr)
		_, callErr = f.service(t).Get(callContext, "run339")
		require.ErrorIs(t, callErr, imageagent.ErrRunNotFound)
		callContext, callErr = f.context(t, actor, org, authz.PermissionImageAgentWrite)
		require.NoError(t, callErr)
		require.ErrorIs(t, f.service(t).RestartFailed(callContext, "run339"), imageagent.ErrRunNotFound)
	}
	f.iam.revoked.Store(true)
	_, err = f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.Error(t, err)
	f.iam.revoked.Store(false)
	f.iam.failed.Store(true)
	_, err = f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.Error(t, err)
	f.iam.failed.Store(false)
	require.Equal(t, starts, f.temporal.starts)
	legacy := imagestore.NewGormRepository(f.db)
	_, err = legacy.GetProjection(context.Background(), scope)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = legacy.GetAssetCatalog(context.Background(), scope)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = legacy.ListEvents(context.Background(), scope, 0, 10)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = legacy.CommitProjection(context.Background(), imageagent.ProjectionCommit{Scope: scope, Snapshot: stored})
	require.Error(t, err)
	oldRun := stored.Run
	oldRun.ID = "old339"
	oldRun.ScopeProtocol = ""
	oldRun.IdempotencyKey = "old-key339"
	oldScope := imageagent.ScopeForRun(oldRun)
	_, err = legacy.InitializeRun(context.Background(), imageagent.ProjectionInitialization{Scope: oldScope, Run: oldRun, Plan: stored.Plan, Catalog: stored.AssetCatalog, Snapshot: imageagent.RunProjection{Run: oldRun, Plan: stored.Plan}, CommitID: "old-initialize", EventType: "run.initialized", EventPayload: json.RawMessage(`{}`)})
	require.NoError(t, err)
	_, err = repo.GetProjection(context.Background(), oldScope)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = repo.GetAssetCatalog(context.Background(), oldScope)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = repo.ListEvents(context.Background(), oldScope, 0, 10)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	oldInput := scope339StartInput()
	oldInput.RunID = "old339"
	require.Error(t, f.service(t).Start(ctx, oldInput))
	oldProjection, err := legacy.GetProjection(context.Background(), oldScope)
	require.NoError(t, err)
	require.Empty(t, oldProjection.Run.ScopeProtocol)
	for _, tc := range []struct {
		name       string
		repository imageagent.Repository
		projection imageagent.RunProjection
	}{
		{"organization cannot mutate historical", repo, oldProjection},
		{"historical cannot mutate organization", legacy, stored},
	} {
		t.Run(tc.name, func(t *testing.T) {
			direct := tc.repository.(interface {
				UpdateRun(context.Context, imageagent.RunScope, int64, imageagent.RunMutation) error
				AppendPlan(context.Context, imageagent.RunScope, int64, imageagent.Plan) error
			})
			scope := imageagent.ScopeForRun(tc.projection.Run)
			err := direct.UpdateRun(context.Background(), scope, tc.projection.Run.Version, imageagent.RunMutation{Status: imageagent.RunStatusFailed, CurrentNode: "must-not-write", ActivePlanRevision: 1})
			require.ErrorIs(t, err, imageagent.ErrRunNotFound)
			// Also reject a replay of an existing plan before consulting its receipt.
			require.ErrorIs(t, direct.AppendPlan(context.Background(), scope, 1, tc.projection.Plan), imageagent.ErrRunNotFound)
			plan := tc.projection.Plan
			plan.Revision = 2
			plan.ParentRevision = 1
			plan.IdempotencyKey = "must-not-append"
			require.ErrorIs(t, direct.AppendPlan(context.Background(), scope, 1, plan), imageagent.ErrRunNotFound)
		})
	}
	current, err := repo.GetProjection(context.Background(), scope)
	require.NoError(t, err)
	require.Equal(t, stored.Run, current.Run)
	scope339AssertEffectMode(t, repo, legacy, stored)
	scope339AssertEffectMode(t, legacy, repo, oldProjection)
	f.verifyActivity(t, restored)
	f.verifyRecoveryCommand(t)
}

func TestOrganizationRealTemporalImageBudgetDeniesBeforeFirstProvider(t *testing.T) {
	address := os.Getenv("ISSUE487_TEMPORAL_ADDRESS")
	if address == "" {
		t.Skip("requires isolated Temporal ISSUE487_TEMPORAL_ADDRESS")
	}
	f := newScope339Fixture(t)
	var providerCalls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { providerCalls.Add(1) }))
	defer provider.Close()
	logger := logrus.New()
	manager := review334Manager(t, f.db, provider.URL, logger)
	capabilities, err := imageworker.BuildOrganizationImageCapabilities(manager, f.db, logger)
	require.NoError(t, err)
	executor := imagetools.NewProductImageSlotExecutor(imagetools.Dependencies{
		SubjectExtractor: capabilities.SubjectExtractor, WhiteBackgroundRenderer: capabilities.WhiteBackgroundRenderer,
		SceneRenderer: capabilities.SceneRenderer, Reviewer: capabilities.Reviewer,
		UsageQuoter: capabilities.UsageQuoter, ProfileResolver: capabilities.ProfileResolver,
	})
	repository := imagestore.NewOrganizationRepository(f.db)
	authorizer := imageworker.OrganizationExecutionAuthorizer{Client: f.authClient, ServiceToken: func(context.Context) (string, error) { return "service339", nil }, ProjectID: "project", Authorizer: f.auth}
	activities, err := imagetemporal.NewActivities(imagetemporal.ActivityDependencies{
		Repository: repository, SlotExecutor: executor, StagedSlotExecutor: executor,
		Publisher: scope339NoPublication{}, PublisherV3: scope339NoPublication{}, ArtifactStore: &scope339Artifacts{},
		ExecutionAuthorizer: authorizer,
	})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	client, closeClient, err := platformtemporal.Dial(ctx, platformtemporal.Config{Address: address, Namespace: "default"})
	require.NoError(t, err)
	defer closeClient()
	worker, err := imagetemporal.NewWorker(imagetemporal.WorkerConfig{Client: client, Activities: activities, WireMode: imagetemporal.WorkerWireModeOrganization})
	require.NoError(t, err)
	require.NoError(t, worker.Start())
	defer worker.Stop()
	request := scope339StartInput()
	request.Budget.MaxImages = 1
	request.RunID = "run487-" + uuid.NewString()[:8]
	request.TargetPlatform = "product"
	request.ImagePolicyContext = imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
	requestContext, err := f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.NoError(t, err)
	require.NoError(t, f.service(t).Start(requestContext, request))
	input := f.temporal.input(t)
	_, err = client.ExecuteWorkflow(ctx, f.temporal.options, "ImageAgentOrganizationWorkflowV1", input)
	require.NoError(t, err)
	defer func() {
		_ = client.TerminateWorkflow(context.Background(), f.temporal.options.ID, "", "controlled isolated test cleanup")
	}()
	scope := imageagent.RunScope{TenantID: "B", OwnerUserID: "actor", RunID: input.RunID}
	for {
		projection, getErr := repository.GetProjection(ctx, scope)
		require.NoError(t, getErr)
		if projection.Run.Status == imageagent.RunStatusBlocked || projection.Run.Status == imageagent.RunStatusFailed {
			require.Zero(t, providerCalls.Load(), "quota must deny before first Extract provider dispatch")
			require.Equal(t, imageagent.RunStatusBlocked, projection.Run.Status)
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("real Temporal worker did not persist a quota block before deadline")
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func scope339AssertEffectMode(t *testing.T, owner, other imageagent.Repository, projection imageagent.RunProjection) {
	t.Helper()
	ctx := context.Background()
	id := imageagent.SlotExternalEffectIdentity{RunScope: imageagent.ScopeForRun(projection.Run), PlanRevision: projection.Plan.Revision, SlotID: projection.Plan.Slots[0].ID, Attempt: 47}
	reservation := imageagent.SlotEffectV3Reservation{Identity: id, IdempotencyKey: "protocol-effect339", InputFingerprint: "controlled-effect339"}
	proper := owner.(imageagent.SlotExternalEffectV3Repository)
	wrong := other.(imageagent.SlotExternalEffectV3Repository)
	_, claimed, err := proper.ReserveSlotProviderV3(ctx, reservation)
	require.NoError(t, err)
	require.True(t, claimed)
	ownerKey, err := imageagent.ArtifactOwnerKey(id.OwnerUserID)
	require.NoError(t, err)
	hash := strings.Repeat("a", 64)
	manifest := imageagent.StagingManifest{Assets: []imageagent.StagedAssetRef{{ObjectKey: fmt.Sprintf("image-agent/staging/%s/%s/%s/1/%s/47/0-%s.png", id.TenantID, ownerKey, id.RunID, id.SlotID, hash), SHA256: hash, SizeBytes: 1, ContentType: "image/png", Width: 1, Height: 1, SourceAssetID: projection.Plan.SourceAssetIDs[0], Operations: []string{"render_scene_model"}}}}
	_, err = wrong.PrepareSlotStagingV3(ctx, reservation, manifest)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrong.CommitSlotStagedV3(ctx, reservation, "controlled-fingerprint")
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrong.RecordSlotReviewOutcomeV3(ctx, imageagent.SlotReviewUsageReservation{Identity: id}, imageagent.SlotReviewOutcomeNeedsHuman)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrong.BlockSlotEffectV3(ctx, imageagent.SlotEffectV3BlockTransition{Reservation: reservation, Phase: imageagent.SlotEffectV3RecoveryBlocked, Code: imageagent.SlotRecoveryBlockedCode})
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrong.(imageagent.RecoveryBlockedSlotEffectV3Repository).RestoreRecoveryBlockedEffectV3(ctx, reservation)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrong.(imageagent.ReviewRetrySlotEffectV3Repository).ResumeReviewRetrySlotV3(ctx, reservation)
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	unchanged, err := proper.GetSlotExternalEffectV3(ctx, id)
	require.NoError(t, err)
	require.Equal(t, imageagent.SlotEffectV3ProviderClaimed, unchanged.Phase)
	oldReservation := imageagent.SlotExternalEffectReservation{Identity: id, IdempotencyKey: "protocol-v2-effect339", InputFingerprint: "controlled-v2-effect339"}
	properV2 := owner.(imageagent.SlotExternalEffectRepository)
	wrongV2 := other.(imageagent.SlotExternalEffectRepository)
	_, claimed, err = properV2.ReserveSlotExternalEffect(ctx, oldReservation)
	require.NoError(t, err)
	require.True(t, claimed)
	_, err = wrongV2.StoreSlotGeneratedOutput(ctx, oldReservation, imageagent.SlotGeneratedOutput{SlotID: id.SlotID, Attempt: id.Attempt, SourceAssetID: projection.Plan.SourceAssetIDs[0], Assets: []imageagent.GeneratedAsset{{URL: "https://synthetic.example/existing.png", Width: 1, Height: 1}}})
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	_, err = wrongV2.CompleteSlotPublication(ctx, oldReservation, imageagent.SlotExecutionResult{SlotID: id.SlotID, Attempt: id.Attempt, Candidates: []imageagent.AssetCandidate{{AssetID: "controlled-existing", URL: "https://synthetic.example/existing.png"}}})
	require.ErrorIs(t, err, imageagent.ErrRunNotFound)
	v2Unchanged, err := properV2.GetSlotExternalEffect(ctx, id)
	require.NoError(t, err)
	require.Equal(t, imageagent.SlotExternalEffectProviderStarted, v2Unchanged.Phase)
}

func TestOrganizationScopeInitializationFailureAndStartResponseLoss(t *testing.T) {
	f := newScope339Fixture(t)
	ctx, err := f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.NoError(t, err)
	require.NoError(t, f.db.Callback().Create().Before("gorm:create").Register("scope339:fail-event", func(tx *gorm.DB) {
		if tx.Statement.Table == "image_agent_v2_events" {
			tx.AddError(fmt.Errorf("controlled event write failure"))
		}
	}))
	require.Error(t, f.service(t).Start(ctx, scope339StartInput()))
	for _, table := range []string{"image_agent_v2_runs", "image_agent_v2_plans", "image_agent_v2_asset_catalog_manifests", "image_agent_v2_projection_snapshots", "image_agent_v2_events"} {
		var count int64
		require.NoError(t, f.db.Table(table).Count(&count).Error)
		require.Zero(t, count)
	}
	require.Zero(t, f.temporal.starts)
	require.NoError(t, f.db.Callback().Create().Remove("scope339:fail-event"))
	f.temporal.failNext = true
	require.Error(t, f.service(t).Start(ctx, scope339StartInput()))
	first := f.temporal.input(t)
	require.NoError(t, f.service(t).Start(ctx, scope339StartInput()))
	require.Equal(t, first.Identity, f.temporal.input(t).Identity)
	require.Equal(t, 2, f.temporal.starts)
	require.Len(t, f.temporal.executions, 1)
	var count int64
	require.NoError(t, f.db.Table("image_agent_v2_runs").Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestOrganizationCredentialAdmissionChecksVerifiedScopeBeforeResolver(t *testing.T) {
	f := newScope339Fixture(t)
	var calls atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer provider.Close()
	credentials := openai.NewGormCredentialResolver(f.db)
	cfg := openai.NewClientConfig("synthetic-B", "review-test", provider.URL+"/v1", 1)
	for _, name := range []string{"default", "image_gpt_image_2"} {
		require.NoError(t, credentials.SaveCredential(context.Background(), openai.AIClientCredential{TenantID: "B", UserID: "actor", ClientName: name, APIKey: "synthetic-B", BaseURL: provider.URL + "/v1", Model: "review-test", Enabled: true, TimeoutSecond: 1}))
	}
	manager, err := openai.NewManager(&openai.ManagerConfig{Clients: map[string]*openai.ClientConfig{"default": cfg, "image_gpt_image_2": cfg}})
	require.NoError(t, err)
	_, err = imageworker.BuildOrganizationImageCapabilities(manager, f.db, logrus.New())
	require.NoError(t, err)
	// This supplemental negative matrix deliberately creates mismatched contexts
	// to test the App adapter. It is not the full-chain scope evidence above.
	for _, verified := range []authidentity.AuthenticatedIdentity{
		{}, {TenantID: "A", UserID: "actor", HomeOrganizationID: "A"},
		{TenantID: "B", UserID: "other", EffectiveOrganizationID: "B"},
		{TenantID: "C", UserID: "actor", EffectiveOrganizationID: "C"},
	} {
		ctx := authidentity.WithAuthenticatedIdentity(context.Background(), verified)
		ctx = openai.WithIdentity(ctx, openai.Identity{TenantID: "B", UserID: "actor"})
		_, err := manager.ResolveEffectiveClientRoute(ctx, "default")
		require.ErrorIs(t, err, openai.ErrClientConfigurationUnavailable)
	}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "B", UserID: "actor", HomeOrganizationID: "A", EffectiveOrganizationID: "B"})
	_, err = manager.ResolveEffectiveClientRoute(ctx, "default")
	require.NoError(t, err)
	require.Zero(t, calls.Load())
}

func (f *scope339Fixture) verifyRecoveryCommand(t *testing.T) {
	repo := imagestore.NewOrganizationRepository(f.db)
	scope := imageagent.RunScope{TenantID: "B", OwnerUserID: "actor", RunID: "run339"}
	current, err := repo.GetProjection(context.Background(), scope)
	require.NoError(t, err)
	next := current
	next.Run.Status = imageagent.RunStatusBlocked
	next.Run.CurrentNode = "blocked"
	next.Run.Version++
	next.Run.Block = &imageagent.Block{Code: imageagent.SlotReviewRequiredCode, SlotID: "slot1"}
	next.Slots = append([]imageagent.SlotProjection(nil), current.Slots...)
	next.Slots[0].Slot.Status = imageagent.SlotStatusBlocked
	next.Slots[0].Attempt = 1
	next.Slots[0].ErrorCode = imageagent.SlotReviewRequiredCode
	_, err = repo.CommitProjection(context.Background(), imageagent.ProjectionCommit{Scope: scope, CommitID: "scope339-blocked", ExpectedProjectionVersion: current.ProjectionVersion, ExpectedRunVersion: current.Run.Version, Snapshot: next, EventType: "run.blocked", EventPayload: json.RawMessage(`{}`), RunMutation: &imageagent.RunMutation{Status: next.Run.Status, CurrentNode: next.Run.CurrentNode, ActivePlanRevision: next.Run.ActivePlanRevision, Block: next.Run.Block}, SlotMutation: &imageagent.SlotProjectionMutation{PlanRevision: 1, Projection: next.Slots[0], Result: imageagent.SlotResult{SlotID: "slot1", Attempt: 1, Status: imageagent.SlotStatusBlocked, ErrorCode: imageagent.SlotReviewRequiredCode}, Attempt: imageagent.StepAttempt{TenantID: "B", OwnerUserID: "actor", RunID: "run339", PlanRevision: 1, SlotID: "slot1", Attempt: 1, Node: "execute_slot", IdempotencyKey: "scope339-blocked-attempt", Outcome: "blocked", ErrorCategory: imageagent.SlotReviewRequiredCode}}})
	require.NoError(t, err)
	ctx, err := f.context(t, "actor", "B", authz.PermissionImageAgentWrite)
	require.NoError(t, err)
	require.NoError(t, f.service(t).RecoverEffect(ctx, "run339", "slot1", 1, 1, "recover339"))
	var recovery imagetemporal.EffectRecoveryWorkflowInput
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(f.temporal.payload, &recovery))
	require.Equal(t, "run339", recovery.Identity.RunID)
	require.Equal(t, "B", recovery.Identity.TenantID)
	require.Equal(t, "actor", recovery.Identity.UserID)
	require.Equal(t, "catalog-input", recovery.Identity.BusinessTaskID)
	require.Equal(t, imagetemporal.OrganizationTaskQueue, f.temporal.options.TaskQueue)
	require.True(t, strings.HasPrefix(f.temporal.options.ID, "organization-v1:"))
}

// The object store substitute can read an existing synthetic staging manifest;
// all generation/publication methods panic if accidentally invoked.
type scope339Artifacts struct {
	imagetemporal.DurableArtifactStore
	reads atomic.Int32
}

func (s *scope339Artifacts) PublicURL(key string) string { return "https://assets.example/" + key }
func (s *scope339Artifacts) RecoverSlotArtifacts(_ context.Context, _ imageagent.SlotExternalEffectIdentity, _ imageagent.StagingManifest) (objectstore.PreparedSlotArtifacts, error) {
	s.reads.Add(1)
	return objectstore.PreparedSlotArtifacts{}, nil
}
func (s *scope339Artifacts) EnsureStaged(context.Context, objectstore.PreparedSlotArtifacts) error {
	return nil
}

type scope339NoPublication struct {
	imageagent.ApprovedAssetPublisher
	imageagent.ApprovedAssetPublisherV3
}

func (f *scope339Fixture) verifyActivity(t *testing.T, wf imagetemporal.WorkflowInput) {
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.providerCalls.Add(1)
		require.Equal(t, "Bearer synthetic-B", r.Header.Get("Authorization"))
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		body, _ := io.ReadAll(r.Body)
		require.Contains(t, string(body), "/B.png")
		require.NotContains(t, string(body), "/A.png")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"scope339-review","model":"review-test","choices":[{"index":0,"message":{"role":"assistant","content":"{\"score\":0.2,\"needs_human_review\":true,\"reasons\":[\"controlled review\"]}"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":4,"total_tokens":7}}`)
	}))
	defer provider.Close()
	credentials := openai.NewGormCredentialResolver(f.db)
	for _, org := range []string{"A"} {
		for _, name := range []string{"default", "image_gpt_image_2"} {
			require.NoError(t, credentials.SaveCredential(context.Background(), openai.AIClientCredential{TenantID: org, UserID: "actor", ClientName: name, APIKey: "synthetic-" + org, BaseURL: provider.URL + "/v1", Model: "review-test", Enabled: true, TimeoutSecond: 1}))
		}
	}
	cfg := openai.NewClientConfig("static-must-not-be-used", "review-test", provider.URL+"/v1", 1)
	manager, err := openai.NewManager(&openai.ManagerConfig{Clients: map[string]*openai.ClientConfig{"default": cfg, "image_gpt_image_2": cfg}})
	require.NoError(t, err)
	capabilities, err := imageworker.BuildOrganizationImageCapabilities(manager, f.db, logger)
	require.NoError(t, err)
	executor := imagetools.NewProductImageSlotExecutor(imagetools.Dependencies{SubjectExtractor: capabilities.SubjectExtractor, WhiteBackgroundRenderer: capabilities.WhiteBackgroundRenderer, SceneRenderer: capabilities.SceneRenderer, Reviewer: capabilities.Reviewer, UsageQuoter: capabilities.UsageQuoter, ProfileResolver: capabilities.ProfileResolver})
	repo := imagestore.NewOrganizationRepository(f.db)
	effects := repo.(imageagent.SlotExternalEffectV3Repository)
	artifacts := &scope339Artifacts{}
	authorizer := imageworker.OrganizationExecutionAuthorizer{Client: f.authClient, ServiceToken: func(context.Context) (string, error) { return "service339", nil }, ProjectID: "project", Authorizer: f.auth}
	activities, err := imagetemporal.NewActivities(imagetemporal.ActivityDependencies{Repository: repo, SlotExecutor: executor, Publisher: scope339NoPublication{}, PublisherV3: scope339NoPublication{}, StagedSlotExecutor: executor, ArtifactStore: artifacts, ExecutionAuthorizer: authorizer})
	require.NoError(t, err)
	input := imagetemporal.ExecuteSlotV3ActivityInput{RunID: wf.RunID, Identity: wf.Identity, PlanRevision: wf.Plan.Revision, Slot: wf.Plan.Slots[0], Attempt: 1, IdempotencyKey: "review339", BudgetAuthorization: true, BudgetPolicy: wf.BudgetPolicy, ReviewActionID: "review-action339", TargetPlatform: wf.TargetPlatform, ImagePolicyContext: wf.ImagePolicyContext, AssetCatalog: wf.AssetCatalog}
	// Capture and decode the actual Activity envelope too; no authidentity or
	// aiidentity context is inserted by this test at the consumer end.
	data, err := converter.GetDefaultDataConverter().ToPayloads(input)
	require.NoError(t, err)
	input = imagetemporal.ExecuteSlotV3ActivityInput{}
	require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(data, &input))
	execution := imageagent.SlotExecutionInput{RunID: input.RunID, TenantID: input.Identity.TenantID, UserID: input.Identity.UserID, PlanRevision: input.PlanRevision, Slot: input.Slot, Attempt: input.Attempt, IdempotencyKey: input.IdempotencyKey, TargetPlatform: input.TargetPlatform, ImagePolicyContext: input.ImagePolicyContext, AssetCatalog: input.AssetCatalog, ProductContext: input.AssetCatalog.ProductContext}
	reservation := imageagent.SlotEffectV3Reservation{Identity: imageagent.SlotExternalEffectIdentity{RunScope: imageagent.RunScope{TenantID: input.Identity.TenantID, OwnerUserID: input.Identity.UserID, RunID: input.RunID}, PlanRevision: input.PlanRevision, SlotID: input.Slot.ID, Attempt: input.Attempt}, IdempotencyKey: input.IdempotencyKey, InputFingerprint: imageagent.SlotExecutionFingerprint(execution)}
	// This is a controlled existing generation receipt, not a substitute for
	// Review quoting. The actual Activity obtains and validates its own quote.
	reservation.Policy = wf.BudgetPolicy
	maximum := imageagent.UsageVector{Images: 1, AgentSteps: 1}
	reservation.Quote = imageagent.SlotUsageQuote{Fingerprint: "existing-staging-quote339", Maximum: maximum, Operations: []imageagent.SlotUsageOperation{{Name: "render_scene", Fingerprint: "existing-staging-operation339", MaximumOutputs: 1, Maximum: maximum}}}
	_, claimed, err := effects.ReserveSlotProviderV3(context.Background(), reservation)
	require.NoError(t, err)
	require.True(t, claimed)
	sum := sha256.Sum256([]byte("existing synthetic staging evidence"))
	hash := hex.EncodeToString(sum[:])
	owner, err := imageagent.ArtifactOwnerKey(input.Identity.UserID)
	require.NoError(t, err)
	manifest := imageagent.StagingManifest{Assets: []imageagent.StagedAssetRef{{ObjectKey: fmt.Sprintf("image-agent/staging/%s/%s/%s/%d/%s/1/0-%s.png", input.Identity.TenantID, owner, input.RunID, input.PlanRevision, input.Slot.ID, hash), SHA256: hash, SizeBytes: 35, ContentType: "image/png", Width: 1200, Height: 1200, SourceAssetID: input.Slot.SourceAssetIDs[0], Operations: []string{"render_scene_model"}}}}
	_, err = effects.PrepareSlotStagingV3(context.Background(), reservation, manifest)
	require.NoError(t, err)
	for _, tc := range []struct {
		name   string
		mutate func(*imagetemporal.ExecuteSlotV3ActivityInput)
	}{
		{"missing protocol", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.Identity.ScopeProtocol = "" }},
		{"unknown protocol", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.Identity.ScopeProtocol = "other" }},
		{"organization tamper", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.Identity.TenantID = "A" }},
		{"actor tamper", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.Identity.UserID = "admin" }},
		{"run tamper", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.RunID = "other" }},
		{"business context tamper", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.Identity.BusinessTaskID = "other" }},
		{"catalog tamper", func(i *imagetemporal.ExecuteSlotV3ActivityInput) { i.AssetCatalog.ProductContext.Title = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := input
			tc.mutate(&bad)
			_, err := activities.ReviewStagedSlotV3(context.Background(), bad)
			require.Error(t, err)
			require.Zero(t, f.providerCalls.Load())
			require.Zero(t, artifacts.reads.Load())
		})
	}
	f.iam.revoked.Store(true)
	_, err = activities.ReviewStagedSlotV3(context.Background(), input)
	require.Error(t, err)
	f.iam.revoked.Store(false)
	f.iam.failed.Store(true)
	_, err = activities.ReviewStagedSlotV3(context.Background(), input)
	require.Error(t, err)
	f.iam.failed.Store(false)
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = activities.ReviewStagedSlotV3(canceled, input)
	require.Error(t, err)
	expired, stop := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer stop()
	_, err = activities.ReviewStagedSlotV3(expired, input)
	require.Error(t, err)
	require.Zero(t, f.providerCalls.Load())
	require.Zero(t, artifacts.reads.Load())
	// The old recorded Review success/failure/Token matrix retires with its
	// owner. Current credential routing is checked directly by
	// TestOrganizationCredentialAdmissionChecksVerifiedScopeBeforeResolver;
	// the production closed Review seam remains covered in the worker tests.

}

func TestServiceAuthorizationRequiresExactOrganization(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Equal(t, "Bearer synthetic-service", r.Header.Get("Authorization"))
		var request struct {
			Filters []map[string]json.RawMessage `json:"filters"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Len(t, request.Filters, 3)
		require.JSONEq(t, `{"id":"org-b"}`, string(request.Filters[2]["organizationId"]))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pagination":{"totalResult":"0"},"authorizations":[]}`))
	}))
	defer server.Close()
	client := zitadel.NewAuthorizationClient(server.URL, server.Client())
	_, err := client.ListServiceProjectAuthorizations(context.Background(), "synthetic-service", "actor", "project", "")
	require.Error(t, err)
	require.Zero(t, calls)
	grants, err := client.ListServiceProjectAuthorizations(context.Background(), "synthetic-service", "actor", "project", "org-b")
	require.NoError(t, err)
	require.Empty(t, grants)
	require.Equal(t, 1, calls)
}

func review334Manager(t *testing.T, db *gorm.DB, endpoint string, logger *logrus.Logger) *openai.Manager {
	t.Helper()
	credentials := openai.NewGormCredentialResolver(db)
	for _, org := range []string{"B", "C"} {
		for _, name := range []string{"default", "image_gpt_image_2"} {
			require.NoError(t, credentials.SaveCredential(context.Background(), openai.AIClientCredential{TenantID: org, UserID: "actor", ClientName: name, APIKey: "SENSITIVE-KEY-" + org, BaseURL: endpoint + "/v1", Model: "review-test", Enabled: true, TimeoutSecond: 1}))
		}
	}
	cfg := openai.NewClientConfig("SENSITIVE-STATIC", "review-test", endpoint+"/v1", 1)
	cfg.MaxRetries = 3
	cfg.RetryDelay = time.Millisecond
	cfg.Logger = openai.AdaptLogrus(logrus.NewEntry(logger))
	manager, err := openai.NewManager(&openai.ManagerConfig{Clients: map[string]*openai.ClientConfig{"default": cfg, "image_gpt_image_2": cfg}, Logger: cfg.Logger})
	require.NoError(t, err)
	return manager
}
