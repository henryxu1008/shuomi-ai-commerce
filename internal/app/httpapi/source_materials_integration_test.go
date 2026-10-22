package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	supplyhttp "task-processor/internal/app/supplychain/httpapi"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	planstore "task-processor/internal/integration/persistence/product/acquisitionplan"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/listing/readiness/tools/readinessinspect"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
)

type sourceMaterialSelectionWire struct {
	asset.SourceSelectionRequest
	OriginalSnapshotVersion string `json:"originalSnapshotVersion"`
	EffectiveCatalogVersion string `json:"effectiveCatalogVersion"`
}
type sourceMaterialApprovalWire struct {
	asset.SourceApprovalCommand
	Selection sourceMaterialSelectionWire `json:"selection"`
}

// Only Resource accounting is synthetic. Product intents/proofs, SRC/Catalog,
// Collection, Preparation and Asset use the actual independently owned PG stores.
type sourceMaterialChargeFixture struct {
	mu       sync.Mutex
	port     testHTTPResourceCharges
	receipts map[orgresource.ConsumerChargeIdentity]orgresource.ConsumerChargeReceipt
}

func (f *sourceMaterialChargeFixture) Reserve(ctx context.Context, in orgresource.ConsumerChargeIntent) (orgresource.ConsumerChargeReceipt, error) {
	r, err := f.port.Reserve(ctx, in.Identity)
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		f.receipts[in.Identity] = r
	}
	return r, err
}
func (f *sourceMaterialChargeFixture) Read(_ context.Context, id orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r, ok := f.receipts[id]
	if !ok {
		return r, orgresource.ErrConsumerChargeUnknown
	}
	return r, nil
}
func (f *sourceMaterialChargeFixture) Settle(ctx context.Context, r orgresource.ConsumerChargeReceipt, _ orgresource.ConsumerChargeProof) (orgresource.ConsumerChargeReceipt, error) {
	r, err := f.port.Reconcile(ctx, r.Intent.Identity)
	f.mu.Lock()
	defer f.mu.Unlock()
	if err == nil {
		f.receipts[r.Intent.Identity] = r
	}
	return r, err
}
func (*sourceMaterialChargeFixture) ClaimDue(context.Context, []orgresource.ResourceConsumer) ([]orgresource.ConsumerChargeIdentity, error) {
	return nil, nil
}

func sourceMaterialAssetFixture(t *testing.T, f *acquisitionHTTPFixture) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("ISSUE398_TEST_DSN") // newAcquisitionHTTPFixture already enforces task ownership.
	name := "issue859_assets_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, f.owner.Exec("CREATE DATABASE "+name).Error)
	owner, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	ownerPool, err := owner.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, ownerPool.Close())
		require.NoError(t, f.owner.Exec("DROP DATABASE "+name+" WITH (FORCE)").Error)
	})
	require.NoError(t, assetstore.InstallSourceApprovalSchema(owner))
	require.NoError(t, owner.Exec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='supply_asset_runtime') THEN CREATE ROLE supply_asset_runtime LOGIN PASSWORD 'issue859-synthetic' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$`).Error)
	require.NoError(t, assetstore.GrantSourceRuntimePermissions(context.Background(), owner))
	db, err := gorm.Open(postgres.Open(dsn+" dbname="+name+" user=supply_asset_runtime password=issue859-synthetic"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(4)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func TestSourceMaterialsMountedClosureUsesOriginalSourceAndIndependentAssetOwner(t *testing.T) {
	f := newAcquisitionHTTPFixture(t)
	ctx := context.Background()
	for _, install := range []func(*gorm.DB) error{collectionstore.InstallSchema, prepstore.InstallSchema, planstore.InstallSchema} {
		require.NoError(t, install(f.owner))
	}
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, f.owner, acquisitionstore.RuntimeCapabilities{Collections: true, AcquisitionPlans: true}))
	assets := sourceMaterialAssetFixture(t, f)
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		source, err := sourcing.Canonical1688Source("981645030344")
		require.NoError(t, err)
		evidence, err := f.provider.Acquire(r.Context(), source)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(evidence))
	}))
	defer collector.Close()
	deps := newRouteAuthDependencies()
	deps.workbenchVerifier = titleVerifier{}
	deps.organizationResolver = workbenchcontext.NewResolver(acquisitionCurrentGrants{f.grants}, "project", "v1", nil)
	resource := &sourceMaterialChargeFixture{port: testHTTPResourceCharges{f.db}, receipts: map[orgresource.ConsumerChargeIdentity]orgresource.ConsumerChargeReceipt{}}
	factories := currentApplicationFactories{
		buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
			return workbenchContextBuildResult{module: currentApplicationTestModule{name: "workbench", routes: currentWorkbenchApplicationRoutes[:4]}, authDependencies: &deps}, nil
		},
		buildSourceAccount: func(_ *gorm.DB, a *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
			a.SetRolePolicyReader(acquisitionCurrentPolicies{})
			return currentApplicationTestModule{name: "source", routes: currentWorkbenchApplicationRoutes[5:]}, nil
		},
		buildCommercial: func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) {
			return currentApplicationTestModule{name: "commercial", routes: currentWorkbenchApplicationRoutes[4:5]}, nil
		},
		buildResourceCharges: func(ctx context.Context, db, _, _ *gorm.DB, _ *authz.ListingKitAuthorizer, _ storecenter.RuntimeCapabilities) (*orgresource.ConsumerChargeService, error) {
			owner, err := buildProductChargeOwner(ctx, db)
			if err != nil {
				return nil, err
			}
			return orgresource.NewConsumerChargeService(resource, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{orgresource.ConsumerProductAcquisition: owner})
		},
	}
	cfg := currentApplicationTestConfig()
	cfg.ListingKit.Zitadel.TenantDirectoryToken = "synthetic"
	cfg.BrowserCollector.Endpoint, cfg.BrowserCollector.Credential = collector.URL, "synthetic"
	open := func() *httptest.Server {
		server, err := buildCurrentApplication(ctx, &gorm.DB{}, cfg, logrus.New(), factories, WithProductAcquisition(f.db), WithCommercialOwnerDatabase(emptyStorageCommercialOwner(t)), WithProductCollections(), WithAcquisitionPlans(), WithSourceMaterials(assets))
		require.NoError(t, err)
		s := httptest.NewServer(server.Handler)
		t.Cleanup(s.Close)
		return s
	}
	server := open()
	call := func(method, path, actor, org, key string, body any, status int) []byte {
		var raw []byte
		if body != nil {
			var err error
			raw, err = json.Marshal(body)
			require.NoError(t, err)
		}
		got, output, err := acquisitionHTTPRequest(server, method, path, actor, org, key, string(raw))
		require.NoError(t, err)
		require.Equal(t, status, got, string(output))
		return output
	}
	var saved acquisitionResultDTO
	require.NoError(t, json.Unmarshal(call("POST", productAcquisitionBase, "operator", "B", uuid.NewString(), map[string]string{"source": "981645030344"}, 200), &saved))
	var item struct {
		ID, BatchID string
		Revision    int64
	}
	require.NoError(t, f.owner.Raw("SELECT i.id,i.batch_id,b.revision FROM product_collection_items i JOIN product_collection_batches b ON i.organization_id=b.organization_id AND i.actor_id=b.actor_id AND i.batch_id=b.id WHERE i.publication_id=?", saved.PublicationID).Scan(&item).Error)
	require.NotEmpty(t, item.ID)
	for i := 0; i < 2; i++ {
		var detail collection.ItemDetail
		require.NoError(t, json.Unmarshal(call("GET", "/api/v1/workbench/collections/items/"+item.ID, "operator", "B", "", nil, 200), &detail))
		require.Equal(t, saved.PublicationID, detail.Item.Source.PublicationID)
		require.NotEmpty(t, detail.Product.Sources)
	}
	var transfer preparation.TransferReceipt
	require.NoError(t, json.Unmarshal(call("POST", supplyhttp.SupplyBasePath+"/transfer", "operator", "B", uuid.NewString(), collection.BatchSelectionInput{BatchID: item.BatchID, ExpectedRevision: item.Revision, ItemIDs: []string{item.ID}}, 200), &transfer))
	var sources collection.Page[preparation.SourceItem]
	require.NoError(t, json.Unmarshal(call("GET", supplyhttp.SupplyBasePath+"/"+transfer.Preparation.ID+"/sources", "operator", "B", "", nil, 200), &sources))
	require.Len(t, sources.Items, 1)
	id := sources.Items[0].ID
	var detail struct {
		Images []struct {
			ID string `json:"id"`
		} `json:"images"`
	}
	require.NoError(t, json.Unmarshal(call("GET", supplyhttp.SupplyBasePath+"/sources/"+id, "operator", "B", "", nil, 200), &detail))
	require.NotEmpty(t, detail.Images)
	path := supplyhttp.SupplyBasePath + "/sources/" + id + "/product-inputs"
	var before struct {
		Inputs readinessinspect.Output `json:"inputs"`
	}
	require.NoError(t, json.Unmarshal(call("GET", path, "operator", "B", "", nil, 200), &before))
	require.Equal(t, "blocked", before.Inputs.Status)
	require.Equal(t, "missing", before.Inputs.AssetBinding.Status)
	require.Equal(t, saved.PublicationID, before.Inputs.PublicationID)
	selection := sourceMaterialSelectionWire{SourceSelectionRequest: asset.SourceSelectionRequest{ItemID: id, OriginalPublicationID: saved.PublicationID, TargetPlatform: "product"}, OriginalSnapshotVersion: saved.CatalogVersion, EffectiveCatalogVersion: saved.CatalogVersion}
	command := sourceMaterialApprovalWire{SourceApprovalCommand: asset.SourceApprovalCommand{Images: []asset.SourceImageChoice{{ID: detail.Images[0].ID, Role: asset.RoleMain}}, Approved: []asset.ApprovedImageChoice{}}, Selection: selection}
	key := uuid.NewString()
	first := call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", key, command, 200)
	commits, err := assetstore.NewBoundedApprovalCommitReader(assets, 2<<20)
	require.NoError(t, err)
	stored, err := commits.ReadApprovalCommit(ctx, "B", key)
	require.NoError(t, err, "the original human-source receipt must remain readable")
	require.Equal(t, key, stored.ActionID)
	server.Close()
	server = open() // original action replays across HTTP application restart.
	var receipt, replay asset.ApprovalReceipt
	require.NoError(t, json.Unmarshal(first, &receipt))
	require.NoError(t, json.Unmarshal(call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", key, command, 200), &replay))
	require.Equal(t, receipt, replay)
	var after struct {
		Inputs readinessinspect.Output `json:"inputs"`
	}
	require.NoError(t, json.Unmarshal(call("GET", path, "operator", "B", "", nil, 200), &after))
	require.Equal(t, "exact", after.Inputs.AssetBinding.Status)
	require.Equal(t, 1, after.Inputs.AssetBinding.Count)
	require.Equal(t, "not_evaluated", after.Inputs.Marketplace.Status)
	require.Equal(t, saved.PublicationID, after.Inputs.PublicationID)
	if after.Inputs.Diagnostics.NeedsReview {
		require.Equal(t, "blocked", after.Inputs.Status)
		require.Contains(t, after.Inputs.Reasons, "source_review_required")
	}
	command.Images[0].Role = asset.RoleGallery
	call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", key, command, 409)
	call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", uuid.NewString(), command, 200)
	command.Images[0].Role = asset.RoleMain
	call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", key, command, 200)
	var current struct {
		Assets []asset.ApprovedAsset `json:"assets"`
	}
	require.NoError(t, json.Unmarshal(call("POST", supplyhttp.SupplyBasePath+"/images/inventory", "operator", "B", "", selection, 200), &current))
	require.Len(t, current.Assets, 1)
	require.Equal(t, asset.RoleGallery, current.Assets[0].Role, "old action replay must not rewind the later inventory head")
	command.Approved = []asset.ApprovedImageChoice{{ActionID: "other", AssetID: "borrowed"}}
	call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", uuid.NewString(), command, 400)
	command.Approved = nil
	command.Selection.EffectiveCatalogVersion = "2"
	call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", uuid.NewString(), command, 409)
	command.Selection.EffectiveCatalogVersion = selection.EffectiveCatalogVersion
	for _, actorOrg := range [][2]string{{"operator2", "B"}, {"operator", "A"}} {
		call("GET", path, actorOrg[0], actorOrg[1], "", nil, 404)
	}
	for _, query := range []string{"?version=2", "?target=temu", "?org=A"} {
		call("GET", path+query, "operator", "B", "", nil, 400)
	}
	call("GET", path, "operator", "B", "", map[string]string{"productKey": saved.ProductKey}, 400)
	for _, target := range []string{"shein", "temu"} {
		command.Selection.TargetPlatform = target
		call("POST", supplyhttp.SupplyBasePath+"/images/approve", "operator", "B", uuid.NewString(), command, 409)
	}
	for _, endpoint := range []string{"targets", "operations", "uploads/resolve"} {
		call("POST", supplyhttp.SupplyBasePath+"/"+endpoint, "operator", "B", uuid.NewString(), map[string]string{}, 404)
	}
	f.grants.failed.Store(true)
	call("GET", path, "operator", "B", "", nil, 503)
	f.grants.failed.Store(false)
	f.grants.revoked.Store(true)
	call("GET", path, "operator", "B", "", nil, 403)
	require.EqualValues(t, 1, f.fetches.Load(), "read/approval/replay never reacquires")
}
