//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	recordstore "task-processor/internal/app/listingrecordstore"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	"task-processor/internal/listing/record"
	"task-processor/internal/marketplace/shein/draft"
	sheinvalidator "task-processor/internal/marketplace/shein/validator"
	"task-processor/internal/product/collection"
	collectionhttp "task-processor/internal/product/collection/httpapi"
	rc "task-processor/internal/reportcenter"
	reporthttp "task-processor/internal/reportcenter/httpapi"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
	"testing"
	"time"
)

type offlineSourceFixture struct {
	detail        collection.ItemDetail
	calls, failAt int
}

func (f *offlineSourceFixture) ReadItem(context.Context, string) (collection.ItemDetail, error) {
	f.calls++
	if f.failAt > 0 && f.calls >= f.failAt {
		return collection.ItemDetail{}, collection.ErrForbidden
	}
	return f.detail, nil
}
func TestOfflineNormalSavedDiagnosticReportAndRevocation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("root"), tcpostgres.WithUsername("issue398_owner"), tcpostgres.WithPassword("fixture-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	t.Setenv("ISSUE398_TEST_DSN", fmt.Sprintf("host=127.0.0.1 port=%s user=issue398_owner password=fixture-only dbname=root sslmode=disable", u.Port()))
	f := newAcquisitionHTTPFixture(t)
	open := func(database, user, password string) *gorm.DB {
		v := *u
		v.Path = "/" + database
		v.User = url.UserPassword(user, password)
		db, e := gorm.Open(postgres.Open(v.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		pool.SetMaxOpenConns(2)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	root := open("root", "issue398_owner", "fixture-only")
	for _, name := range []string{"assets", "stores", "reports"} {
		require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	}
	require.NoError(t, root.Exec("CREATE ROLE supply_asset_runtime LOGIN PASSWORD 'asset-fixture' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; CREATE ROLE report_center_runtime LOGIN PASSWORD 'report-fixture' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	assetsOwner := open("assets", "issue398_owner", "fixture-only")
	stores := open("stores", "issue398_owner", "fixture-only")
	reportsOwner := open("reports", "issue398_owner", "fixture-only")
	require.NoError(t, assetstore.InstallSourceApprovalSchema(assetsOwner))
	require.NoError(t, assetstore.GrantSourceRuntimePermissions(ctx, assetsOwner))
	assets := open("assets", "supply_asset_runtime", "asset-fixture")
	require.NoError(t, storecenter.AutoMigrateStoreRepository(stores))
	prepareRecordStore(t, stores, "org-a")
	require.NoError(t, reportstore.InstallSchema(reportsOwner))
	require.NoError(t, reportstore.GrantRuntime(reportsOwner, "report_center_runtime"))
	reports := open("reports", "report_center_runtime", "report-fixture")
	require.NoError(t, collectionstore.InstallSchema(f.owner))
	require.NoError(t, recordstore.InstallSchema(f.owner))
	capability := acquisitionstore.RuntimeCapabilities{Collections: true, OfflineRecords: true}
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, f.owner, capability))
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, f.db, capability))
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, f.db, acquisitionstore.RuntimeCapabilities{Collections: true}), "undeclared offline grants must fail")
	require.NoError(t, recordstore.VerifySchema(ctx, f.db))
	identity := authidentity.AuthenticatedIdentity{UserID: "actor-a", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member-a", Roles: []string{"platform_admin"}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "org-a", AuthorizationID: "member-a", Roles: []string{"platform_admin"}}}, TokenExpiresAt: time.Now().Add(time.Hour)}
	resolver := &productReviewResolverStub{value: identity}
	deps := routeAuthDependencies{organizationResolver: resolver}
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	source, err := buildProductCollectionService(ctx, f.db, deps, permissions, false, false, false, false, false, false, true)
	require.NoError(t, err)
	boundSource, err := (productReviewCapabilityBinder{}).Bind(authidentity.WithAuthenticatedIdentity(ctx, identity), "Bearer synthetic")
	require.NoError(t, err)
	ownSource, err := source.Mutate(boundSource, uuid.NewString(), collection.Mutation{Action: "create_product", Product: &collection.OwnProduct{Title: "Synthetic offline source"}})
	require.NoError(t, err)
	sourceDetail, err := source.ReadItem(boundSource, ownSource.ItemID)
	require.NoError(t, err)
	itemID := ownSource.ItemID
	prepareApprovedAssets(t, assetsOwner, "org-a", sourceDetail.Item.Source.ProductKey, sourceDetail.Item.Source.Version)
	module, err := buildOfflineRecords(ctx, f.db, assets, stores, deps, permissions, source)
	require.NoError(t, err)
	reportModule, err := buildReportCenter(ctx, reports, deps, permissions, nil, nil, module.app)
	require.NoError(t, err)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), resolver.value))
	})
	for _, route := range append(module.routes, reportModule.(reportCenterModule).routes...) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	request := func(method, path, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer synthetic")
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	input := record.Input{ProductKey: sourceDetail.Item.Source.ProductKey, SnapshotVersion: sourceDetail.Item.Source.Version, StoreID: recordStoreID, Country: "US", Language: "en", Action: "save_draft"}
	body, err := json.Marshal(map[string]any{"collectionItemId": itemID, "input": offlineInput(input)})
	require.NoError(t, err)
	key := uuid.NewString()
	response := request("POST", offlineRecordsPath, key, string(body))
	require.Equal(t, 201, response.Code, response.Body.String())
	var receipt record.Receipt
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &receipt))
	response = request("GET", offlineRecordsPath+"/"+receipt.RecordID, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	var saved offlineSavedDTO
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &saved))
	require.Equal(t, "saved", saved.Origin)
	require.Equal(t, receipt.InputHash, saved.Version)
	require.Equal(t, key, saved.OperationID)
	require.NotContains(t, response.Body.String(), "https://fixtures.invalid/")
	tripwire := &offlineSourceFixture{failAt: 1}
	module.app.collections = tripwire
	response = request("GET", offlineRecordsPath+"/commands/"+key, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Zero(t, tripwire.calls, "lookup must not recreate or read fresh source facts")
	module.app.collections = source
	response = request("GET", offlineRecordsPath, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), receipt.RecordID)
	path := reporthttp.Base + "/sources/SHEIN_RECORD/" + receipt.RecordID
	response = request("GET", path, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	ref := rc.SourceRef{Kind: "SHEIN_RECORD", ID: receipt.RecordID, Version: receipt.InputHash}
	saveBody, err := json.Marshal(map[string]any{"source": ref})
	require.NoError(t, err)
	saveKey := uuid.NewString()
	response = request("POST", reporthttp.Base, saveKey, string(saveBody))
	require.Equal(t, 200, response.Code, response.Body.String())
	var result rc.Result
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
	response = request("GET", reporthttp.Base+"/"+result.Report.ID, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, 0, int(f.fetches.Load()), "no provider request")
	// Admin role never bypasses normal private ownership, even for commands.
	resolver.value.UserID = "other"
	response = request("GET", offlineRecordsPath+"/"+receipt.RecordID, "", "")
	require.Equal(t, 404, response.Code, response.Body.String())
	response = request("GET", offlineRecordsPath+"/commands/"+key, "", "")
	require.Equal(t, 404, response.Code, response.Body.String())
	resolver.value = identity
	// A live source denial at each write/commit guard rolls both rows back.
	// Only the guard is wrapped: all other ports use the real current SQL owners.
	originalCreate := module.app.create
	catalogReader, err := catalogstore.NewBoundedSnapshotReader(f.db, record.MaxPayloadBytes)
	require.NoError(t, err)
	approvedReader, err := assetstore.NewBoundedApprovedInventoryReader(assets, record.MaxPayloadBytes)
	require.NoError(t, err)
	for _, phase := range []int{1, 2, 3, 4} {
		calls := 0
		module.app.create, err = record.NewService(record.ServiceDependencies{Products: catalogReader, Assets: approvedReader, Stores: module.app.stores, Records: module.app.records, Builder: draft.Builder{}, Evaluator: sheinvalidator.ExactApprovedAssetValidator{}, Authorizer: permissions, Now: time.Now, RuleRevision: sheinvalidator.DiagnosticRuleVersion, PolicyRevision: sheinvalidator.BindingVersion, CreationGuard: func(ctx context.Context, actor record.Actor, input record.Input) error {
			calls++
			if calls == phase {
				return record.ErrForbidden
			}
			return module.app.guard(ctx, actor, input)
		}})
		require.NoError(t, err)
		response = request("POST", offlineRecordsPath, uuid.NewString(), string(body))
		require.Equal(t, 403, response.Code, response.Body.String())
		require.Equal(t, phase, calls)
		requireDraftS1RowCounts(t, f.owner, 1)
	}
	module.app.create = originalCreate
	resolver.value.Roles = []string{"admin"}
	response = request("POST", offlineRecordsPath, uuid.NewString(), string(body))
	require.Equal(t, 403, response.Code, response.Body.String())
	resolver.value = identity
	// Off preserves existing Report recovery; no source is called.
	off, err := buildReportCenter(ctx, reports, deps, permissions, nil, nil)
	require.NoError(t, err)
	router = gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), identity))
	})
	for _, route := range off.(reportCenterModule).routes {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	response = request("POST", reporthttp.Base, saveKey, string(saveBody))
	require.Equal(t, 200, response.Code, response.Body.String())

	// New-empty initializer admits the original two tables; a second run refuses
	// the now nonempty database instead of performing a retained feature upgrade.
	require.NoError(t, root.Exec("CREATE DATABASE offline_empty").Error)
	manifest := filepath.Join(t.TempDir(), "offline-init.json")
	port, e := strconv.Atoi(u.Port())
	require.NoError(t, e)
	initJSON, e := json.Marshal(map[string]any{"schemaVersion": 1, "collections": true, "sheinOfflineRecords": true, "database": map[string]any{"host": "127.0.0.1", "port": port, "user": "issue398_owner", "password": "fixture-only", "database": "offline_empty"}})
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(manifest, initJSON, 0600))
	require.NoError(t, productsourcing.InitializeAcquisitionDatabase(ctx, manifest, "offline_empty"))
	require.Error(t, productsourcing.InitializeAcquisitionDatabase(ctx, manifest, "offline_empty"))
	initialized := open("offline_empty", "issue398_owner", "fixture-only")
	require.NoError(t, recordstore.VerifySchema(ctx, initialized))
	require.NoError(t, f.owner.Exec("GRANT UPDATE ON listing_shein_records TO source_acquisition_runtime").Error)
	require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, f.db, capability))
	require.NoError(t, f.owner.Exec("REVOKE UPDATE ON listing_shein_records FROM source_acquisition_runtime").Error)

	// Real Collection/Source/Catalog SQL, including its original own-product UoW.
	// The smallest admitted Product pool must not wait on its own transaction.
	pool, e := f.db.DB()
	require.NoError(t, e)
	pool.SetMaxOpenConns(1)
	realDeps := routeAuthDependencies{organizationResolver: offlineFixedResolver{identity}}
	realCollections, e := buildProductCollectionService(ctx, f.db, realDeps, permissions, false, false, false, false, false, false, true)
	require.NoError(t, e)
	binder := productReviewCapabilityBinder{}
	bound, e := binder.Bind(authidentity.WithAuthenticatedIdentity(ctx, identity), "Bearer synthetic")
	require.NoError(t, e)
	collectionRouter := gin.New()
	for _, route := range collectionhttp.Routes(realCollections, binder.Bind) {
		collectionRouter.Handle(route.Method, route.Path, route.Handler)
	}
	ownKey := uuid.NewString()
	ownBody := `{"action":"create_product","product":{"title":"Synthetic real Collection","images":[],"variants":[
        {"sourceId":"zero","title":"Zero","sku":"ZERO","currency":"USD","price":0,"stock":0,"attributes":{}},
        {"sourceId":"positive","title":"Positive","sku":"POSITIVE","currency":"USD","price":7,"stock":7,"attributes":{}}
    ]}}`
	sendCollection := func(key, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", collectionhttp.BasePath+"/commands", strings.NewReader(body)).
			WithContext(authidentity.WithAuthenticatedIdentity(ctx, identity))
		req.Header.Set("Authorization", "Bearer synthetic")
		req.Header.Set("Idempotency-Key", key)
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		collectionRouter.ServeHTTP(response, req)
		return response
	}
	require.Equal(t, 400, sendCollection(ownKey, strings.Replace(ownBody, `,"stock":0`, "", 1)).Code)
	require.Equal(t, 400, sendCollection(ownKey, strings.Replace(ownBody, `"stock":0`, `"stock":null`, 1)).Code)
	require.Equal(t, 400, sendCollection(ownKey, strings.Replace(ownBody, `"stock":0`, `"stock":0,"ownStockDeclared":true`, 1)).Code)
	created := sendCollection(ownKey, ownBody)
	require.Less(t, created.Code, 300, created.Body.String())
	var own collection.Receipt
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &own))
	replayed := sendCollection(ownKey, ownBody)
	require.Less(t, replayed.Code, 300, replayed.Body.String())
	var replay collection.Receipt
	require.NoError(t, json.Unmarshal(replayed.Body.Bytes(), &replay))
	require.Equal(t, own.ItemID, replay.ItemID)
	detail, e := realCollections.ReadItem(bound, own.ItemID)
	require.NoError(t, e)
	require.Len(t, detail.Product.Variants, 2)
	for i, want := range []int{0, 7} {
		require.True(t, detail.Product.Variants[i].OwnStockDeclared)
		require.NotNil(t, detail.Product.Variants[i].Stock)
		require.Equal(t, want, *detail.Product.Variants[i].Stock)
	}
	prepareApprovedAssets(t, assetsOwner, "org-a", detail.Item.Source.ProductKey, detail.Item.Source.Version)
	actual, e := buildOfflineRecords(ctx, f.db, assets, stores, realDeps, permissions, realCollections)
	require.NoError(t, e)
	actualRouter := gin.New()
	actualRouter.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), identity))
	})
	for _, route := range actual.routes {
		actualRouter.Handle(route.Method, route.Path, route.Handler)
	}
	actualBody, e := json.Marshal(map[string]any{"collectionItemId": own.ItemID, "input": offlineInput(record.Input{ProductKey: detail.Item.Source.ProductKey, SnapshotVersion: detail.Item.Source.Version, StoreID: recordStoreID, Country: "US", Language: "en", Action: "save_draft"})})
	require.NoError(t, e)
	createActual := func() *httptest.ResponseRecorder {
		limited, done := context.WithTimeout(ctx, 5*time.Second)
		defer done()
		req := httptest.NewRequest("POST", offlineRecordsPath, strings.NewReader(string(actualBody))).WithContext(limited)
		req.Header.Set("Authorization", "Bearer synthetic")
		req.Header.Set("Idempotency-Key", uuid.NewString())
		req.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		actualRouter.ServeHTTP(response, req)
		return response
	}
	one := createActual()
	require.Equal(t, 201, one.Code, one.Body.String())
	var savedReceipt record.Receipt
	require.NoError(t, json.Unmarshal(one.Body.Bytes(), &savedReceipt))
	get := httptest.NewRequest("GET", offlineRecordsPath+"/"+savedReceipt.RecordID, nil).WithContext(ctx)
	get.Header.Set("Authorization", "Bearer synthetic")
	savedResponse := httptest.NewRecorder()
	actualRouter.ServeHTTP(savedResponse, get)
	require.Equal(t, 200, savedResponse.Code, savedResponse.Body.String())
	require.Contains(t, savedResponse.Body.String(), "ZERO")
	require.Contains(t, savedResponse.Body.String(), "库存 0")
	require.Contains(t, savedResponse.Body.String(), "库存 7")
	require.Contains(t, savedResponse.Body.String(), "重量待补")
	// Merely raising the minimum pool size would still deadlock two transactions.
	pool.SetMaxOpenConns(2)
	results := make(chan *httptest.ResponseRecorder, 2)
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() { defer group.Done(); results <- createActual() }()
	}
	group.Wait()
	close(results)
	for result := range results {
		require.Equal(t, 201, result.Code, result.Body.String())
	}
	workbook := excelize.NewFile()
	defer workbook.Close()
	require.NoError(t, workbook.SetSheetRow("Sheet1", "A1", &collection.ImportHeaders))
	require.NoError(t, workbook.SetSheetRow("Sheet1", "A2", &[]any{"Imported", "", "", "", "IMPORTED", "USD", 0, 0}))
	sheet, e := workbook.WriteToBuffer()
	require.NoError(t, e)
	imported, e := realCollections.PreviewImport(bound, sheet.Bytes())
	require.NoError(t, e)
	importBody, e := json.Marshal(collection.Mutation{Action: "import_products", Name: "Stock import", Products: imported.Products})
	require.NoError(t, e)
	importResponse := sendCollection(uuid.NewString(), string(importBody))
	require.Equal(t, 200, importResponse.Code, importResponse.Body.String())
	var importReceipt collection.Receipt
	require.NoError(t, json.Unmarshal(importResponse.Body.Bytes(), &importReceipt))
	page, e := realCollections.ListItems(bound, importReceipt.BatchID, collection.Query{Limit: 10})
	require.NoError(t, e)
	require.Len(t, page.Items, 1)
	importedItem, e := realCollections.ReadItem(bound, page.Items[0].ID)
	require.NoError(t, e)
	require.True(t, importedItem.Product.Variants[0].OwnStockDeclared)
	require.Equal(t, new(0), importedItem.Product.Variants[0].Stock)

	revision := imported.Products[0]
	revision.Variants[0].Stock = 7
	revisionBody, e := json.Marshal(collection.Mutation{
		Action: "revise_product", ItemID: importedItem.Item.ID, ExpectedRevision: importedItem.Item.Revision,
		ExpectedSourceVersion: importedItem.Item.Source.Version, ExpectedPublicationID: importedItem.Item.Source.PublicationID, Product: &revision,
	})
	require.NoError(t, e)
	revisionResponse := sendCollection(uuid.NewString(), string(revisionBody))
	require.Equal(t, 200, revisionResponse.Code, revisionResponse.Body.String())
	revisedItem, e := realCollections.ReadItem(bound, importedItem.Item.ID)
	require.NoError(t, e)
	require.True(t, revisedItem.Product.Variants[0].OwnStockDeclared)
	require.Equal(t, new(7), revisedItem.Product.Variants[0].Stock)
	originalVersion, e := realCollections.ReadOwnVersion(bound, importedItem.Item.ID, importedItem.Item.Source.Version)
	require.NoError(t, e)
	require.Equal(t, new(0), originalVersion.Product.Variants[0].Stock)

	// Original size and FK/schema admission are checked before listen.
	require.NoError(t, f.owner.Exec("ALTER TABLE listing_shein_record_operations DROP CONSTRAINT listing_shein_record_operations_record_fk").Error)
	require.Error(t, recordstore.VerifySchema(ctx, f.db))
}

type offlineFixedResolver struct {
	value authidentity.AuthenticatedIdentity
}

func (r offlineFixedResolver) Resolve(context.Context, httproute.OrganizationAccessPolicy, workbenchcontext.ResolveInput) (authidentity.AuthenticatedIdentity, error) {
	return r.value, nil
}
