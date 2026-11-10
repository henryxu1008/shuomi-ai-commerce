//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	listingapp "task-processor/internal/app/listing"
	current "task-processor/internal/app/runtime/currentapplication"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	coreconfig "task-processor/internal/core/config"
	"task-processor/internal/httproute"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	"task-processor/internal/integration/shein"
	kernelmodule "task-processor/internal/kernel/module"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
	contextapi "task-processor/internal/workbenchcontext/httpapi"
	"testing"
	"time"
)

func TestDraftCoexistenceRetainedCopy(t *testing.T) {
	dir := os.Getenv("DRAFT_COEXISTENCE_COPY_DIR")
	if dir == "" {
		t.Skip("explicit retained-copy validation only")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "copy-config.private.json"))
	require.NoError(t, err)
	var cfg struct {
		IsolatedPg      string                 `json:"isolatedPg"`
		Runtime         current.Config         `json:"runtime"`
		StoreID         string                 `json:"storeId"`
		OfficialStoreID string                 `json:"officialStoreId"`
		SchemaOwner     current.DatabaseConfig `json:"schemaOwner"`
		StoreKey        string                 `json:"storeKey"`
	}
	require.NoError(t, json.Unmarshal(raw, &cfg))
	require.True(t, strings.HasPrefix(cfg.IsolatedPg, "issue1158-coexist-"))
	require.NotNil(t, cfg.Runtime.ProductAcquisitionDatabase)
	require.NotNil(t, cfg.Runtime.SourceMaterials)
	open := func(d current.DatabaseConfig) *gorm.DB {
		require.Equal(t, "127.0.0.1", d.Host)
		require.NotEqual(t, 5434, d.Port)
		db, e := gorm.Open(postgres.Open(fmt.Sprintf("host=127.0.0.1 port=%d user=%s password=%s dbname=%s sslmode=disable", d.Port, d.User, d.Password, d.Database)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		pool.SetMaxOpenConns(4)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	product, stores, assets := open(*cfg.Runtime.ProductAcquisitionDatabase), open(cfg.Runtime.StoreCenter.Database), open(cfg.Runtime.SourceMaterials.AssetDatabase)
	owner := open(cfg.SchemaOwner)
	require.NoError(t, officialstore.VerifyOfficialSchema(context.Background(), product), "logical restore must admit the equivalent schema")
	for _, mutation := range []string{
		`ALTER TABLE listing_submission_official_intents DROP CONSTRAINT listing_submission_official_intents_kind_check`,
		`ALTER TABLE listing_submission_official_intents DROP CONSTRAINT listing_submission_official_intents_kind_check; ALTER TABLE listing_submission_official_intents ADD CONSTRAINT listing_submission_official_intents_kind_check CHECK (kind IN ('publish','image','global_create','shelf_on','other'))`,
		`ALTER TABLE listing_walmart_sku_bindings DROP CONSTRAINT listing_walmart_sku_bindings_environment_check`,
		`ALTER TABLE listing_walmart_sku_bindings DROP CONSTRAINT listing_walmart_sku_bindings_environment_check; ALTER TABLE listing_walmart_sku_bindings ADD CONSTRAINT listing_walmart_sku_bindings_environment_check CHECK (environment IN ('production','sandbox','other'))`,
	} {
		rollback := errors.New("rollback negative fixture")
		require.ErrorIs(t, owner.Transaction(func(tx *gorm.DB) error {
			require.NoError(t, tx.Exec(mutation).Error)
			require.Error(t, officialstore.VerifyOfficialSchema(context.Background(), tx), "missing or relaxed constraints must still fail")
			return rollback
		}), rollback)
	}
	require.NoError(t, officialstore.VerifyOfficialSchema(context.Background(), product))
	scope := collection.Scope{OrganizationID: "394513989062623234", ActorID: "394513839896395778", MemberID: "394513991663091714"}
	grants := &localCopyGrants{scope: scope}
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(acquisitionCurrentPolicies{})
	resolver := workbenchcontext.NewResolver(grants, "project", "v1", nil)
	deps := routeAuthDependencies{workbenchVerifier: titleVerifier{}, organizationResolver: resolver, authorizer: permissions, auditRecorder: workbenchcontext.NewStructuredAuditRecorder(logrus.New())}
	actor := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, Roles: []string{"listingkit_admin"}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: scope.OrganizationID, AuthorizationID: scope.MemberID, ProjectID: "project", Roles: []string{"listingkit_admin"}}}, TokenExpiresAt: time.Now().Add(time.Hour)})
	actor, err = (productReviewCapabilityBinder{}).Bind(actor, "Bearer "+scope.ActorID)
	require.NoError(t, err)
	storeRepo, err := storecenter.NewMemberScopedStoreRepository(stores, currentStoreMemberAuthorizer{authorizer: permissions})
	require.NoError(t, err)

	savedStore, err := storeRepo.Get(actor, scope.OrganizationID, cfg.StoreID)
	require.NoError(t, err)
	policy := cfg.Runtime.ControlledLocalDraft.Policy()
	module, err := buildControlledLocalDraftModule(actor, product, stores, ControlledLocalDraftDependencies{Policy: policy, AssetDB: assets}, deps, permissions, cfg.Runtime.SupplyMarket != nil, false, true, true, true, false)
	require.NoError(t, err)
	core, err := buildNativeDraftReadCore(actor, product, deps, permissions, nil, cfg.Runtime.SupplyMarket != nil, false, true, true, true, false)
	require.NoError(t, err)
	localBefore, err := module.app.ReadTarget(actor, policy.Binding.SourceID, cfg.StoreID)
	require.NoError(t, err)
	require.EqualValues(t, 3, localBefore.Revision)
	sourceBefore, err := core.app.Source(actor, policy.Binding.SourceID)
	require.NoError(t, err)
	contextBefore, err := module.app.LocalDraftContext(actor, policy.Binding.SourceID, policy.Binding.CompletionDraft)
	require.NoError(t, err)
	require.True(t, contextBefore.WritesEnabled)
	write := func(name string, value any) {
		raw, e := json.Marshal(value)
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), raw, 0600))
	}
	write("source-before.private.json", sourceBefore)
	write("test-r3-before.private.json", localBefore)
	// Only native Store mutations in the restored copy. No credentials/network
	// of the retained enterprise or a provider are consumed by this fixture.
	proposed, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: cfg.OfficialStoreID, OrganizationID: scope.OrganizationID, ActorSubject: scope.ActorID, Name: "【隔离验证 #1236】正式用途合成店铺", Platform: "shein", Region: "US", CreateIdempotencyKey: cfg.StoreKey, OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	officialStore, _, err := storeRepo.CreateOrReplay(actor, scope.OrganizationID, proposed)
	require.NoError(t, err)
	protection, err := shein.NewCredentialProtection("synthetic-key", bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	provider := &coexistProvider{}
	apps, err := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{{Provider: provider, Protection: protection, Type: storecenter.ApplicationSelfOperated}})
	require.NoError(t, err)
	connections, err := storeapp.NewOfficialConnections(storeRepo, apps)
	require.NoError(t, err)
	if officialStore.ConnectionRef() == "" {
		begin, e := connections.Begin(actor, storecenter.OfficialConnectionCommand{OrganizationID: scope.OrganizationID, StoreID: officialStore.ID(), AttemptID: "12361236-1236-4236-8236-000000000002", ExpectedStoreVersion: officialStore.Version(), ApplicationID: "synthetic-app"})
		require.NoError(t, e)
		u, e := url.Parse(begin.AuthorizationURL)
		require.NoError(t, e)
		_, query, _ := strings.Cut(u.Fragment, "?")
		q, e := url.ParseQuery(query)
		require.NoError(t, e)
		_, e = connections.Complete(actor, storecenter.CompleteOfficialConnection{OrganizationID: scope.OrganizationID, StoreID: officialStore.ID(), AttemptID: begin.AttemptID, AppID: "synthetic-app", State: q.Get("state"), TempToken: "synthetic-temp"})
		require.NoError(t, e)
	}
	baseStore, err := storecenter.NewGormStoreRepository(stores)
	require.NoError(t, err)
	require.NoError(t, stores.WithContext(actor).Transaction(func(tx *gorm.DB) error {
		v, e := baseStore.LockServiceState(actor, tx, storecenter.ServiceStoreIdentity{OrganizationID: scope.OrganizationID, StoreID: officialStore.ID()})
		if e != nil {
			return e
		}
		if v.State.ServiceStatus == storecenter.ServiceStatusActive {
			return nil
		}
		state, e := storecenter.ActivateStoreService(v.State, storecenter.ConnectionStatusConnected, time.Now().UTC())
		if e != nil {
			return e
		}
		return baseStore.ApplyServiceState(actor, tx, storecenter.ServiceStoreMutation{Identity: v.Identity, ExpectedVersion: v.Version, ExpectedConnectionRef: v.ConnectionRef, State: state, ActorSubject: scope.ActorID, OccurredAt: time.Now().UTC()})
	}))
	identity := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if grants.revoked.Load() {
			fmt.Fprint(w, `{"pagination":{"totalResult":"0"},"authorizations":[]}`)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"pagination": map[string]string{"totalResult": "1"}, "authorizations": []any{map[string]any{"id": scope.MemberID, "project": map[string]string{"id": "project"}, "organization": map[string]string{"id": scope.OrganizationID, "name": "Isolated"}, "user": map[string]string{"id": scope.ActorID}, "state": "STATE_ACTIVE", "roles": []any{map[string]string{"key": "listingkit_admin"}}}}})
	}))
	t.Cleanup(identity.Close)
	coreCfg := &coreconfig.Config{}
	coreCfg.ListingKit.Zitadel.AuthorizationAPIURL = identity.URL
	coreCfg.ListingKit.Zitadel.ProjectID = "project"
	coreCfg.ListingKit.Zitadel.TenantDirectoryToken = "synthetic-only"
	var lifecycle listingapp.OperationWorker = coexistNoWorker{}
	var activities *listingapp.OperationActivities
	full, err := buildListingModule(actor, product, stores, ListingDependencies{AssetDB: assets, Starter: coexistNoWorker{}, Worker: &lifecycle, localDraftPolicy: &policy, NewWorker: func(a *listingapp.OperationActivities) (listingapp.OperationWorker, error) {
		activities = a
		return lifecycle, nil
	}}, deps, permissions, apps, coreCfg, nil, storecenter.RuntimeCapabilities{Observations: cfg.Runtime.StoreCenter.Observations != nil, OperationsCockpit: cfg.Runtime.StoreCenter.OperationsCockpit}, cfg.Runtime.SupplyMarket != nil, false, true, true, true, false)
	require.NoError(t, err)
	require.NotNil(t, activities)
	require.NotEqual(t, module.Name(), full.Name())
	// The formal Target is prepared from the canonical Source, never test Input.
	baseline := record.TargetInput{SourceID: sourceBefore.Source.ID, StoreID: officialStore.ID(), EffectiveVersion: sourceBefore.Source.Source.Version, Draft: coexistInitialDraft(sourceBefore)}
	formal, err := full.app.ReadTarget(actor, policy.Binding.SourceID, officialStore.ID())
	if errors.Is(err, record.ErrNotFound) {
		r, e := full.app.Targets.Create(actor, "12361236-1236-4236-8236-000000000003", baseline)
		require.NoError(t, e)
		formal = r.Record
	} else {
		require.NoError(t, err)
	}
	require.Nil(t, formal.LocalTrial)
	write("formal-base.private.json", formal)
	materials, err := buildSourceMaterialsModule(actor, product, assets, deps, permissions, core.collections, full.app)
	require.NoError(t, err)
	completion, err := composeCompletionDrafts(actor, product, deps, permissions, core.collections, materials.app)
	require.NoError(t, err)
	registry := kernelmodule.NewRegistry()
	require.NoError(t, contextapi.NewModule(contextapi.NewHandlerWithWorkbenchAuthorizer(permissions)).Register(registry))
	routes := append(registry.Routes(), module.routes...)
	routes = append(routes, full.routes...)
	for _, r := range append(materials.routes, completion.routes...) {
		if r.Method == http.MethodGet && !routeExists(routes, r.Method, r.Path) {
			routes = append(routes, r)
		}
	}
	officialStore, err = storeRepo.Get(actor, scope.OrganizationID, officialStore.ID())
	require.NoError(t, err)
	routes = append(routes, httproute.Descriptor{Method: "GET", Path: "/api/v1/workbench/stores", Module: "store-center", Permission: authz.PermissionWorkbenchStoreRead, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead, RequestTimeout: 5 * time.Second, Handler: func(c *gin.Context) {
		items := []gin.H{}
		for _, s := range []*storecenter.Store{savedStore, officialStore} {
			items = append(items, gin.H{"id": s.ID(), "name": s.Name(), "platform": "shein", "region": "US", "externalStoreId": "", "recordStatus": "active", "serviceStatus": s.ServiceStatus(), "serviceStartedAt": coexistUTC(s.ServiceStartedAt()), "serviceExpiresAt": coexistUTC(s.ServiceExpiresAt()), "connectionStatus": map[bool]string{true: "connected", false: "disconnected"}[s.ID() == officialStore.ID()], "version": s.Version(), "createdAt": s.CreatedAt().UTC(), "updatedAt": s.UpdatedAt().UTC()})
		}
		c.JSON(200, gin.H{"items": items, "pagination": gin.H{"page": 1, "pageSize": 100, "total": len(items)}})
	}})
	server := buildHTTPServerFromRoutesAtWithAuthDependencies("127.0.0.1", 0, routes, deps)
	ts := httptest.NewUnstartedServer(server.Handler)
	if previous, e := os.ReadFile(filepath.Join(dir, "go.json")); e == nil {
		var v struct {
			GoOrigin string `json:"goOrigin"`
		}
		require.NoError(t, json.Unmarshal(previous, &v))
		u, e := url.Parse(v.GoOrigin)
		require.NoError(t, e)
		require.Equal(t, "127.0.0.1", u.Hostname())
		listener, e := net.Listen("tcp", u.Host)
		require.NoError(t, e)
		_ = ts.Listener.Close()
		ts.Listener = listener
	}
	ts.Start()
	t.Cleanup(ts.Close)
	write("go.json", map[string]any{"goOrigin": ts.URL, "token": scope.ActorID, "organizationId": scope.OrganizationID, "userId": scope.ActorID, "storeId": savedStore.ID(), "officialStoreId": officialStore.ID(), "sourceId": policy.Binding.SourceID, "preparationId": sourceBefore.Source.PreparationID, "candidateRecordId": policy.Binding.CompletionDraft.RecordID, "candidateRevision": 1, "localRecordId": localBefore.ID, "formalRecordId": formal.ID})
	t.Log("COEXIST_COPY_READY")
	deadline := time.Now().Add(40 * time.Minute)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(filepath.Join(dir, "stop-browser")); e == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, e := os.Stat(filepath.Join(dir, "stop-browser")); e != nil {
		t.Fatal("bounded browser window expired")
	}
	runCoexistenceAssertions(t, actor, dir, product, stores, assets, scope, policy, module, full, core.app, localBefore, sourceBefore, formal, activities, grants, write)
	require.Zero(t, provider.publishes.Load())
	require.Zero(t, provider.transforms.Load())
}
func routeExists(routes []httproute.Descriptor, method, path string) bool {
	for _, r := range routes {
		if r.Method == method && r.Path == path {
			return true
		}
	}
	return false
}

type coexistNoWorker struct{}

func (coexistNoWorker) Ensure(context.Context, string, string) error {
	return errors.New("fixture forbids workflow execution")
}
func (coexistNoWorker) Start() error { return nil }
func (coexistNoWorker) Stop()        {}
func coexistInitialDraft(source listingapp.SourceDetail) goods.OfficialDraftInput {
	draft := record.InitialSheinDraft(source.Product)
	draft.Product.CategoryID = 8123
	draft.Product.ProductTypeID = 8456
	return draft
}

// Controlled external provider port; production Store, official access,
// rules reader, source selection, R1, Target and transaction remain native.
type coexistProvider struct {
	postgresConnectionProvider
	publishes  atomic.Int32
	transforms atomic.Int32
}

func coexistOfficialRules() goods.OfficialRuleSnapshot {
	r := listingapp.ControlledLocalRules()
	r.Authority = ""
	r.Categories[0].ID = 8123
	r.Categories[0].ProductTypeID = 8456
	r.Categories[0].Name = "Qualified fixture apparel"
	r.Attributes.ProductTypeID = 8456
	r.Attributes.Attributes[0].ID = 812
	r.Attributes.Attributes[0].Name = "Color"
	r.Attributes.Attributes[0].Options[0].ID = 834
	r.Attributes.Attributes[0].Options[1].ID = 835
	r.Attributes.Attributes[1].ID = 813
	r.Attributes.Attributes[1].Name = "Size"
	r.Attributes.Attributes[1].Options[0].ID = 836
	r.Attributes.Attributes[1].Options[1].ID = 837
	r.Warehouses[0].Code = "official-us"
	r.Warehouses[0].Name = "Official fixture US warehouse"
	return r
}
func (*coexistProvider) QueryProductSites(context.Context, storecenter.OfficialMerchantCredential) ([]model.MainSite, error) {
	return coexistOfficialRules().Sites, nil
}
func (*coexistProvider) QueryProductWarehouses(context.Context, storecenter.OfficialMerchantCredential) ([]model.Warehouse, error) {
	return coexistOfficialRules().Warehouses, nil
}
func (*coexistProvider) QueryProductCategories(context.Context, storecenter.OfficialMerchantCredential) ([]model.Category, error) {
	return coexistOfficialRules().Categories, nil
}
func (*coexistProvider) QueryProductFillStandards(context.Context, storecenter.OfficialMerchantCredential, int64) (model.FillStandards, error) {
	return coexistOfficialRules().Fill, nil
}
func (*coexistProvider) QueryProductAttributes(context.Context, storecenter.OfficialMerchantCredential, int64) (model.AttributeTemplate, error) {
	return coexistOfficialRules().Attributes, nil
}
func (*coexistProvider) QueryProductLinkedRules(_ context.Context, _ storecenter.OfficialMerchantCredential, q model.LinkedRulesRequest) ([]model.LinkedRules, error) {
	v := []model.LinkedRules{}
	for _, g := range q.Groups {
		v = append(v, model.LinkedRules{GroupID: g.ID, Attributes: []model.LinkedAttributeRule{}})
	}
	return v, nil
}
func (*coexistProvider) QueryProductBrands(context.Context, storecenter.OfficialMerchantCredential) ([]model.Brand, error) {
	return []model.Brand{}, nil
}
func (*coexistProvider) QueryProductPublishPermission(context.Context, storecenter.OfficialMerchantCredential, string) (model.PublishPermission, error) {
	return model.PublishPermission{}, errors.New("fixture forbids publication")
}
func (p *coexistProvider) PublishProduct(context.Context, storecenter.OfficialMerchantCredential, model.PublishProduct) (model.PublishResult, error) {
	p.publishes.Add(1)
	return model.PublishResult{}, errors.New("fixture forbids publication")
}
func (p *coexistProvider) TransformProductImage(context.Context, storecenter.OfficialMerchantCredential, model.TransformImage) (model.TransformedImage, error) {
	p.transforms.Add(1)
	return model.TransformedImage{}, errors.New("fixture forbids image effects")
}

func runCoexistenceAssertions(t *testing.T, ctx context.Context, dir string, product, stores, assets *gorm.DB, scope collection.Scope, p record.LocalTrialPolicy, local controlledLocalDraftModule, full listingModule, core *listingapp.Application, localBefore record.TargetRecord, sourceBefore listingapp.SourceDetail, baseline record.TargetRecord, activities *listingapp.OperationActivities, grants *localCopyGrants, write func(string, any)) {
	t.Helper()
	final, e := full.app.ReadTarget(ctx, p.Binding.SourceID, baseline.Input.StoreID)
	require.NoError(t, e)
	require.EqualValues(t, 3, final.Revision)
	require.Nil(t, final.LocalTrial)
	write("formal-saved.private.json", final)
	same, e := local.app.ReadTarget(ctx, p.Binding.SourceID, p.Binding.StoreID)
	require.NoError(t, e)
	require.Equal(t, localBefore, same)
	after, e := core.Source(ctx, p.Binding.SourceID)
	require.NoError(t, e)
	require.Equal(t, sourceBefore, after)
	rows := []string{}
	for _, g := range final.Input.Draft.Product.SKCs {
		for _, sku := range g.SKUs {
			rows = append(rows, sku.SupplierSKU)
			if sku.SupplierSKU == "QA1195-BLK-M" {
				require.Len(t, sku.Stock, 1)
				require.Zero(t, sku.Stock[0].Quantity)
				require.Equal(t, "official-us", sku.Stock[0].WarehouseID)
			} else {
				require.Empty(t, sku.Stock)
			}
			require.Empty(t, sku.Prices)
		}
	}
	require.Equal(t, []string{"QA1195-BLK-M", "QA1195-BLK-L", "QA1195-WHT-M"}, rows)
	require.Len(t, final.Input.Draft.Product.SKCs, 2)
	require.Empty(t, final.Input.Draft.Images)
	require.Empty(t, final.Result.SubmissionPayload)
	require.False(t, record.PublicationReady(final))
	require.False(t, record.PublicationReady(same))
	require.EqualValues(t, 8123, final.Input.Draft.Product.CategoryID)
	require.EqualValues(t, 812, final.Input.Draft.Product.SKCs[0].SaleAttribute.AttributeID)
	_, e = full.app.ReadRecord(ctx, localBefore.ID)
	require.ErrorIs(t, e, record.ErrForbidden)
	_, e = local.app.ReadRecord(ctx, final.ID)
	require.ErrorIs(t, e, record.ErrForbidden)
	// Original command ownership, replay and concurrent retry retain one Target.
	var commands []struct {
		Key      string
		RecordID string
		StoreID  string
	}
	require.NoError(t, product.Raw("SELECT c.command_key AS key,c.record_id,r.store_id FROM listing_target_record_commands c JOIN listing_target_records r ON r.id=c.record_id ORDER BY r.revision").Scan(&commands).Error)
	require.Len(t, commands, 6)
	var formalKey string
	for _, c := range commands {
		right, wrong := full.app, local.app
		if c.StoreID == p.Binding.StoreID {
			right, wrong = local.app, full.app
		} else {
			formalKey = c.Key
		}
		receipt, e := right.ReadTargetCommand(ctx, c.Key)
		require.NoError(t, e)
		_, e = wrong.ReadTargetCommand(ctx, c.Key)
		require.ErrorIs(t, e, record.ErrForbidden)
		replay, e := right.Targets.Create(ctx, c.Key, receipt.Record.Input)
		require.NoError(t, e)
		require.True(t, replay.Replayed)
		require.Equal(t, receipt.Record, replay.Record)
	}
	command, e := full.app.ReadTargetCommand(ctx, formalKey)
	require.NoError(t, e)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := full.app.Targets.Create(ctx, formalKey, command.Record.Input)
			if e == nil && (!r.Replayed || r.Record.ID != command.Record.ID) {
				e = errors.New("changed replay")
			}
			errs <- e
		}()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	// All captured formal ports reject the pilot Store, including the worker's
	// merchant reader, without relying on app.Rules replacement.
	guarded := baseline.Input
	guarded.StoreID = p.Binding.StoreID
	_, e = full.app.QueryRules(ctx, guarded)
	require.Error(t, e)
	_, e = full.app.TargetMerchants.ReadTargetMerchant(ctx, scope, p.Binding.StoreID)
	require.Error(t, e)
	_, e = activities.TargetMerchants.ReadTargetMerchant(ctx, scope, p.Binding.StoreID)
	require.Error(t, e)
	_, e = full.app.Targets.Create(ctx, "12361236-1236-4236-8236-000000000021", localBefore.Input)
	require.ErrorIs(t, e, record.ErrForbidden)
	stripped := localBefore.Input
	stripped.LocalTrial = nil
	stripped.SheinImport = nil
	stripped.SheinSalesMapping = nil
	_, e = full.app.Targets.Create(ctx, "12361236-1236-4236-8236-000000000022", stripped)
	require.Error(t, e)
	_, e = full.app.Uploader.Upload(ctx, scope, "12361236-1236-4236-8236-000000000023", localBefore.ID)
	require.ErrorIs(t, e, record.ErrForbidden)
	_, e = local.app.Targets.Create(ctx, "12361236-1236-4236-8236-000000000024", final.Input)
	require.ErrorIs(t, e, record.ErrForbidden)
	// IT-A: a general formal input guard is not local bootstrap admission.
	full.app.ValidateTargetInput = func(collection.Scope, record.TargetInput) error { return nil }
	query := baseline.Input
	query.ExpectedRevision = 0
	r, e := full.app.QueryRules(ctx, query)
	require.NoError(t, e)
	require.Nil(t, r.InitialInput)
	// Fault after record/head mutation must rollback all writes; original key
	// remains NOT_FOUND, current head and local R3 remain unchanged.
	selection := final.Input.SheinSalesMapping.Selection
	selection.Rows = []record.SheinSalesRow{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 1, SKUIndex: 0}}
	preview, e := full.app.Targets.PreviewSheinSalesMapping(ctx, record.SheinSalesRequest{SourceID: p.Binding.SourceID, BaseRecordID: final.ID, CompletionDraft: p.Binding.CompletionDraft, Selection: selection})
	require.NoError(t, e)
	require.NotNil(t, preview.Input)
	require.NoError(t, product.Callback().Create().Before("gorm:create").Register("coexist-fail-command", func(tx *gorm.DB) {
		if tx.Statement.Table == "listing_target_record_commands" {
			tx.AddError(errors.New("isolated command insert fault"))
		}
	}))
	_, e = full.app.Targets.Create(ctx, "12361236-1236-4236-8236-000000000025", *preview.Input)
	require.NoError(t, product.Callback().Create().Remove("coexist-fail-command"))
	require.ErrorIs(t, e, record.ErrUnavailable)
	unchanged, e := full.app.ReadTarget(ctx, p.Binding.SourceID, final.Input.StoreID)
	require.NoError(t, e)
	require.Equal(t, final, unchanged)
	_, e = full.app.ReadTargetCommand(ctx, "12361236-1236-4236-8236-000000000025")
	require.ErrorIs(t, e, record.ErrNotFound)
	// Same key/different payload and stale CAS cannot add revisions.
	changed := command.Record.Input
	changed.ExpectedRevision++
	_, e = full.app.Targets.Create(ctx, formalKey, changed)
	require.ErrorIs(t, e, record.ErrConflict)
	_, e = full.app.Targets.Create(ctx, "12361236-1236-4236-8236-000000000026", baseline.Input)
	require.ErrorIs(t, e, record.ErrConflict)
	// Reconstruct read-only local module against the unchanged schema/pool;
	// normal official saves/reads do not depend on its writer or rule reader.
	p.Enabled = false
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(acquisitionCurrentPolicies{})
	resolver := workbenchcontext.NewResolver(grants, "project", "v1", nil)
	deps := routeAuthDependencies{workbenchVerifier: titleVerifier{}, organizationResolver: resolver, authorizer: permissions, auditRecorder: workbenchcontext.NewStructuredAuditRecorder(logrus.New())}
	disabled, err := buildControlledLocalDraftModule(ctx, product, stores, ControlledLocalDraftDependencies{Policy: p, AssetDB: assets}, deps, permissions, true, false, true, true, true, false)
	require.NoError(t, err)
	require.Nil(t, disabled.app.Targets)
	require.Nil(t, disabled.app.Rules)
	require.Len(t, disabled.routes, 4)
	contextView, e := disabled.app.LocalDraftContext(ctx, p.Binding.SourceID, p.Binding.CompletionDraft)
	require.NoError(t, e)
	require.False(t, contextView.WritesEnabled)
	same, e = disabled.app.ReadRecord(ctx, localBefore.ID)
	require.NoError(t, e)
	require.Equal(t, localBefore, same)
	_, e = full.app.ReadRecord(ctx, final.ID)
	require.NoError(t, e)
	grants.revoked.Store(true)
	_, e = disabled.app.LocalDraftContext(ctx, p.Binding.SourceID, p.Binding.CompletionDraft)
	require.Error(t, e)
	_, e = disabled.app.ReadRecord(ctx, localBefore.ID)
	require.Error(t, e)
	_, e = full.app.ReadRecord(ctx, final.ID)
	require.Error(t, e)
	grants.revoked.Store(false)
	var n int64
	require.NoError(t, product.Raw("SELECT count(*) FROM listing_preparation_targets WHERE source_id=?", p.Binding.SourceID).Scan(&n).Error)
	require.EqualValues(t, 2, n)
	write("copy-browser-result.json", map[string]any{"status": "PASS", "localR3Unchanged": localBefore.ID, "formalTarget": final.TargetID, "formalRevision": final.Revision, "targets": 2, "records": 6, "rows": rows, "stock": "0/unfilled/unfilled", "images": "no platform binding; original source unchanged", "originalCommands": 6, "concurrentReplays": 8, "rollback": true, "CASConflict": true, "sameKeyConflict": true, "localDisabledHistory": true, "liveRevocation": true, "officialFixtureOnly": true, "realPublication": false})
}

func coexistUTC(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.UTC()
	return &v
}
