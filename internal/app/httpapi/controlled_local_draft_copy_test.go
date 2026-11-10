package httpapi

import (
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
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	assetstore "task-processor/internal/integration/persistence/product/asset"
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

// This bounded retained-copy test reuses the existing browser/HTTP fixtures.
// Only the external identity/grant port is synthetic; Store, Source, R1,
// assets, live authorization, controlled composition and Target use native
// owners and the explicitly restored PostgreSQL copy. No original DSN is read.
type localCopyGrants struct {
	scope   collection.Scope
	revoked atomic.Bool
}

func (g *localCopyGrants) Load(_ context.Context, source workbenchcontext.GrantSource, req workbenchcontext.GrantRequest) (workbenchcontext.GrantResult, error) {
	out := workbenchcontext.GrantResult{Source: source}
	if req.Subject == g.scope.ActorID && !g.revoked.Load() {
		out.Grants = []authidentity.OrganizationGrant{{AuthorizationID: g.scope.MemberID, OrganizationID: g.scope.OrganizationID, OrganizationName: "隔离副本 · 本地草稿", ProjectID: "project", Roles: []string{"listingkit_admin"}}}
	}
	return out, nil
}
func (*localCopyGrants) Invalidate(string, string) {}

func TestControlledLocalDraftRetainedCopy(t *testing.T) {
	dir := os.Getenv("CONTROLLED_LOCAL_DRAFT_COPY_DIR")
	if dir == "" {
		t.Skip("explicit retained-copy validation only")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "copy-config.private.json"))
	require.NoError(t, err)
	var cfg struct {
		IsolatedPg string         `json:"isolatedPg"`
		Runtime    current.Config `json:"runtime"`
		StoreID    string         `json:"storeId"`
		StoreKey   string         `json:"storeKey"`
	}
	require.NoError(t, json.Unmarshal(raw, &cfg))
	require.True(t, strings.HasPrefix(cfg.IsolatedPg, "issue1158-local-"))
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
	proposed, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: cfg.StoreID, OrganizationID: scope.OrganizationID, ActorSubject: scope.ActorID, Name: "离线测试 · #1158 本地草稿", Platform: "shein", Region: "US", CreateIdempotencyKey: cfg.StoreKey, OccurredAt: time.Now()})
	require.NoError(t, err)
	savedStore, _, err := storeRepo.CreateOrReplay(actor, scope.OrganizationID, proposed)
	require.NoError(t, err)
	require.Equal(t, cfg.StoreID, savedStore.ID())
	second, _, err := storeRepo.CreateOrReplay(actor, scope.OrganizationID, proposed)
	require.NoError(t, err)
	require.Equal(t, savedStore.ID(), second.ID())
	policy := record.LocalTrialPolicy{Scope: scope, Binding: record.LocalTrial{ContractVersion: 1, TrialID: "issue1158-local-draft-v1", SourceID: "41cb802a-fa17-523e-9444-12044f47e5dc", CompletionDraft: record.CompletionDraftSelection{RecordID: "f7853e11-353a-5214-a7c7-75a8ded6e278", Revision: 1}, StoreID: cfg.StoreID, FixtureVersion: "issue1158-local-draft-v1", FixtureHash: collection.Digest(listingapp.ControlledLocalRules())}, ProductKey: "own-27ef901c-30e0-57f9-81d9-56032b0efa61", ProductVersion: 1, Enabled: true}
	module, err := buildControlledLocalDraftModule(actor, product, stores, ControlledLocalDraftDependencies{Policy: policy, AssetDB: assets}, deps, permissions, cfg.Runtime.SupplyMarket != nil, cfg.Runtime.POD != nil, cfg.Runtime.ImageAgent != nil && cfg.Runtime.ImageAgent.WorkerConfigFile != "", cfg.Runtime.AcquisitionPlans, cfg.Runtime.SupplyCompletionDrafts, false)
	require.NoError(t, err)
	core, err := buildNativeDraftReadCore(actor, product, deps, permissions, nil, cfg.Runtime.SupplyMarket != nil, cfg.Runtime.POD != nil, cfg.Runtime.ImageAgent != nil && cfg.Runtime.ImageAgent.WorkerConfigFile != "", cfg.Runtime.AcquisitionPlans, cfg.Runtime.SupplyCompletionDrafts, false)
	require.NoError(t, err)
	sourceBefore, err := core.app.Source(actor, policy.Binding.SourceID)
	require.NoError(t, err)
	write := func(name string, value any) {
		raw, e := json.Marshal(value)
		require.NoError(t, e)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), raw, 0600))
	}
	write("source-before.private.json", sourceBefore)
	// Independent normal source materials composition: no controlled full pointer.
	materials, err := buildSourceMaterialsModule(actor, product, assets, deps, permissions, core.collections, nil)
	require.NoError(t, err)
	require.NotNil(t, materials.app.Approvals)
	completion, err := composeCompletionDrafts(actor, product, deps, permissions, core.collections, materials.app)
	require.NoError(t, err)
	registry := kernelmodule.NewRegistry()
	require.NoError(t, contextapi.NewModule(contextapi.NewHandlerWithWorkbenchAuthorizer(permissions)).Register(registry))
	routes := registry.Routes()
	routes = append(routes, module.routes...)
	for _, r := range materials.routes {
		if r.Method == http.MethodGet {
			routes = append(routes, r)
		}
	}
	for _, r := range completion.routes {
		if r.Method == http.MethodGet {
			routes = append(routes, r)
		}
	}
	routes = append(routes, httproute.Descriptor{Method: "GET", Path: "/api/v1/workbench/stores", Module: "store-center", Permission: authz.PermissionWorkbenchStoreRead, AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyCachedRead, RequestTimeout: 5 * time.Second, Handler: func(c *gin.Context) {
		c.JSON(200, gin.H{"items": []gin.H{{"id": savedStore.ID(), "name": savedStore.Name(), "platform": "shein", "region": "US", "externalStoreId": "", "recordStatus": "active", "serviceStatus": "pending_activation", "serviceStartedAt": nil, "serviceExpiresAt": nil, "connectionStatus": "disconnected", "version": savedStore.Version(), "createdAt": savedStore.CreatedAt().UTC(), "updatedAt": savedStore.UpdatedAt().UTC()}}, "pagination": gin.H{"page": 1, "pageSize": 100, "total": 1}})
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
	write("go.json", map[string]any{"goOrigin": ts.URL, "token": scope.ActorID, "organizationId": scope.OrganizationID, "userId": scope.ActorID, "storeId": savedStore.ID(), "sourceId": policy.Binding.SourceID, "preparationId": sourceBefore.Source.PreparationID, "candidateRecordId": policy.Binding.CompletionDraft.RecordID, "candidateRevision": 1, "scope": "retained PostgreSQL copy and native owners; external identity fixture only; no original instance/model/publication"})
	t.Log("CONTROLLED_COPY_READY")
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
	// Persist exact owner head and preserve immutable original source after UI.
	final, err := module.app.ReadTarget(actor, policy.Binding.SourceID, cfg.StoreID)
	require.NoError(t, err)
	write("saved-target.private.json", final)
	require.EqualValues(t, 3, final.Revision)
	require.True(t, record.ValidPlatformRecord(final))
	require.False(t, record.PublicationReady(final))
	var commands []struct {
		Key      string
		RecordID string
	}
	require.NoError(t, product.Raw("SELECT command_key AS key,record_id FROM listing_target_record_commands ORDER BY record_id").Scan(&commands).Error)
	require.Len(t, commands, 3)
	for _, command := range commands {
		historical, e := module.app.ReadTargetCommand(actor, command.Key)
		require.NoError(t, e)
		replay, e := module.app.Targets.Create(actor, command.Key, historical.Record.Input)
		require.NoError(t, e)
		require.True(t, replay.Replayed)
		require.Equal(t, historical.Record, replay.Record)
	}
	var wg sync.WaitGroup
	errorsCh := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			old, e := module.app.ReadTargetCommand(actor, commands[0].Key)
			if e == nil {
				replay, e2 := module.app.Targets.Create(actor, commands[0].Key, old.Record.Input)
				e = e2
				if e == nil && (!replay.Replayed || replay.Record.ID != old.Record.ID) {
					e = errors.New("wrong original receipt")
				}
			}
			errorsCh <- e
		}()
	}
	wg.Wait()
	close(errorsCh)
	for e := range errorsCh {
		require.NoError(t, e)
	}
	// The native transaction must reject promotion even if another trusted
	// full composition later has apparently valid official rules for this Store.
	ordinaryRules := copyQualifiedRules{binding: final.Merchant}
	ordinaryRules.binding.ApplicationID = "synthetic-official"
	ordinaryRules.binding.ApplicationRevision = "v1:self_operated"
	ordinaryRules.binding.ConnectionRevision = 1
	ordinaryRules.binding.ServiceExpiresAt = time.Now().Add(time.Hour)
	assetReader, e := assetstore.NewBoundedApprovedInventoryReader(assets, record.MaxPayloadBytes)
	require.NoError(t, e)
	ordinary, e := record.NewTargetService(record.TargetDependencies{Sources: core.app.Sources, Products: core.app.Products, Assets: assetReader, Rules: ordinaryRules, Records: module.app.Records, Images: controlledNoImages{}, Authorizer: core.app.Authorization})
	require.NoError(t, e)
	promoted := final.Input
	promoted.LocalTrial = nil
	promoted.SheinImport = nil
	promoted.SheinSalesMapping = nil
	promoted.ExpectedRevision = final.Revision
	_, e = ordinary.Create(actor, "06c0caf8-032d-4a81-8e5c-b0fb3b68c53c", promoted)
	require.ErrorIs(t, e, record.ErrForbidden)
	// Inject a command insert failure after record/head writes. It must roll
	// back the existing shared transaction, leaving exactly the three UI saves.
	sel := final.Input.SheinSalesMapping.Selection
	sel.Rows = []record.SheinSalesRow{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 1, SKUIndex: 0}}
	preview, e := module.app.Targets.PreviewSheinSalesMapping(actor, record.SheinSalesRequest{SourceID: policy.Binding.SourceID, BaseRecordID: final.ID, CompletionDraft: policy.Binding.CompletionDraft, Selection: sel})
	require.NoError(t, e)
	require.NotNil(t, preview.Input, "%v", preview.Issues)
	require.NoError(t, product.Callback().Create().Before("gorm:create").Register("issue1158-fail-command", func(tx *gorm.DB) {
		if tx.Statement.Table == "listing_target_record_commands" {
			tx.AddError(errors.New("isolated command insert fault"))
		}
	}))
	_, e = module.app.Targets.Create(actor, "33f6667c-740b-41c9-a012-e5e2bf38d489", *preview.Input)
	require.NoError(t, product.Callback().Create().Remove("issue1158-fail-command"))
	require.ErrorIs(t, e, record.ErrUnavailable)
	unchanged, e := module.app.ReadTarget(actor, policy.Binding.SourceID, cfg.StoreID)
	require.NoError(t, e)
	require.Equal(t, final, unchanged)
	after, err := core.app.Source(actor, policy.Binding.SourceID)
	require.NoError(t, err)
	require.Equal(t, sourceBefore, after)
	disabledPolicy := policy
	disabledPolicy.Enabled = false
	disabled, err := buildControlledLocalDraftModule(actor, product, stores, ControlledLocalDraftDependencies{Policy: disabledPolicy, AssetDB: assets}, deps, permissions, cfg.Runtime.SupplyMarket != nil, cfg.Runtime.POD != nil, cfg.Runtime.ImageAgent != nil && cfg.Runtime.ImageAgent.WorkerConfigFile != "", cfg.Runtime.AcquisitionPlans, cfg.Runtime.SupplyCompletionDrafts, false)
	require.NoError(t, err)
	require.Nil(t, disabled.app.Targets)
	same, err := disabled.app.ReadRecord(actor, final.ID)
	require.NoError(t, err)
	require.Equal(t, final, same)
	for _, command := range commands {
		r, e := disabled.app.ReadTargetCommand(actor, command.Key)
		require.NoError(t, e)
		require.Equal(t, command.RecordID, r.Record.ID)
	}
	for _, r := range disabled.routes {
		require.Equal(t, http.MethodGet, r.Method)
	}
	for _, r := range final.Input.Draft.Product.SKCs {
		for _, sku := range r.SKUs {
			if sku.SupplierSKU == "QA1195-BLK-M" {
				require.Len(t, sku.Stock, 1)
				require.Zero(t, sku.Stock[0].Quantity)
			} else {
				require.Empty(t, sku.Stock)
			}
			require.Empty(t, sku.Prices)
		}
	}
	require.Len(t, final.Input.Draft.Product.SKCs, 2)
	require.Equal(t, []string{"QA1195-BLK-M", "QA1195-BLK-L", "QA1195-WHT-M"}, []string{final.Input.Draft.Product.SKCs[0].SKUs[0].SupplierSKU, final.Input.Draft.Product.SKCs[0].SKUs[1].SupplierSKU, final.Input.Draft.Product.SKCs[1].SKUs[0].SupplierSKU})
	require.Empty(t, final.Input.Draft.Images)
	var counts []struct{ N int64 }
	require.NoError(t, product.Raw("SELECT count(*) AS n FROM listing_target_records WHERE target_id=?", final.TargetID).Scan(&counts).Error)
	require.EqualValues(t, 3, counts[0].N)
	grants.revoked.Store(true)
	_, e = disabled.app.ReadRecord(actor, final.ID)
	require.Error(t, e)
	grants.revoked.Store(false)
	write("copy-browser-result.json", map[string]any{"status": "PASS", "oneTarget": final.TargetID, "records": 3, "sourceUnchanged": true, "stock": "0/unfilled/unfilled", "disabledReadOnly": true, "originalKeys": 3, "concurrentReplays": 8, "promotionDeniedByTransaction": true, "commandInsertFailureRollback": true, "liveRevocationDenied": true})
}

type copyQualifiedRules struct {
	binding storecenter.ProductMerchantBinding
}

func (r copyQualifiedRules) ReadTargetRules(context.Context, collection.Scope, string, goods.OfficialDraftInput) (storecenter.ProductMerchantBinding, goods.OfficialRuleSnapshot, error) {
	v := listingapp.ControlledLocalRules()
	v.Authority = ""
	v.ApplicationMode = model.ModeSelfOperated
	return r.binding, v, nil
}
