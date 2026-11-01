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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	recordstore "task-processor/internal/app/listingrecordstore"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	"task-processor/internal/listing/record"
	"task-processor/internal/product/collection"
	rc "task-processor/internal/reportcenter"
	reporthttp "task-processor/internal/reportcenter/httpapi"
	"task-processor/internal/storecenter"
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
	published := publishRecordProductOnly(t, f.owner, "org-a", "synthetic-offline")
	prepareApprovedAssets(t, assetsOwner, "org-a", published.Identity.ProductKey, published.Version)
	itemID := uuid.NewString()
	source := &offlineSourceFixture{detail: collection.ItemDetail{Item: collection.Item{ID: itemID, Source: collection.Source{ProductKey: published.Identity.ProductKey, Version: published.Version}}}}
	identity := authidentity.AuthenticatedIdentity{UserID: "actor-a", TenantID: "org-a", EffectiveOrganizationID: "org-a", EffectiveMemberID: "member-a", Roles: []string{"platform_admin"}, OrganizationGrants: []authidentity.OrganizationGrant{{OrganizationID: "org-a", AuthorizationID: "member-a", Roles: []string{"platform_admin"}}}, TokenExpiresAt: time.Now().Add(time.Hour)}
	resolver := &productReviewResolverStub{value: identity}
	deps := routeAuthDependencies{organizationResolver: resolver}
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
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
	input := record.Input{ProductKey: published.Identity.ProductKey, SnapshotVersion: published.Version, StoreID: recordStoreID, Country: "US", Language: "en", Action: "save_draft"}
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
	before := source.calls
	response = request("GET", offlineRecordsPath+"/commands/"+key, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	require.Equal(t, before, source.calls, "lookup must not recreate or read fresh source facts")
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
	for _, phase := range []int{1, 2, 3, 4} {
		source.calls = 0
		source.failAt = phase
		response = request("POST", offlineRecordsPath, uuid.NewString(), string(body))
		require.Equal(t, 403, response.Code, response.Body.String())
		requireDraftS1RowCounts(t, f.owner, 1)
	}
	source.failAt = 0
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
	// Original size and FK/schema admission are checked before listen.
	require.NoError(t, f.owner.Exec("ALTER TABLE listing_shein_record_operations DROP CONSTRAINT listing_shein_record_operations_record_fk").Error)
	require.Error(t, recordstore.VerifySchema(ctx, f.db))
}
