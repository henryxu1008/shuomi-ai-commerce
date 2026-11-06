//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	target "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/shein/goods"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	rc "task-processor/internal/reportcenter"
	reporthttp "task-processor/internal/reportcenter/httpapi"
	"task-processor/internal/storecenter"
	"task-processor/internal/workbenchcontext"
)

type reportTargetPolicies struct{ sources bool }

func (p *reportTargetPolicies) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	result := map[string][]string{}
	for _, key := range keys {
		if key == authz.EnterpriseRoleKey(org, 1) {
			result[key] = []string{"reports"}
			if p.sources {
				result[key] = append(result[key], "acquisition", "data-mine", "supply-mine")
			}
		}
	}
	return result, nil
}

func TestReportTargetMountedNativeReaderAndOriginalRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("reports"), tcpostgres.WithUsername("issue398_owner"), tcpostgres.WithPassword("fixture-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	// Reuse the existing synthetic acquisition fixture on this task-owned PG.
	t.Setenv("ISSUE398_TEST_DSN", fmt.Sprintf("host=127.0.0.1 port=%s user=issue398_owner password=fixture-only dbname=reports sslmode=disable", u.Port()))
	f := newAcquisitionHTTPFixture(t)
	for _, install := range []func(*gorm.DB) error{collectionstore.InstallSchema, prepstore.InstallSchema, recordstore.InstallSchema} {
		require.NoError(t, install(f.owner))
	}
	require.NoError(t, acquisitionstore.GrantRuntimePermissions(ctx, f.owner, acquisitionstore.RuntimeCapabilities{Collections: true}))
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	policy := &reportTargetPolicies{sources: true}
	permissions.SetRolePolicyReader(policy)
	resolver := workbenchcontext.NewResolver(acquisitionCurrentGrants{f.grants}, "project", "v1", nil)
	deps := routeAuthDependencies{organizationResolver: resolver}
	identity, err := (titleVerifier{}).Verify(ctx, "operator")
	require.NoError(t, err)
	identity, err = resolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{Identity: identity, BearerToken: "operator", RequestedOrganizationID: "B"})
	require.NoError(t, err)
	bound, err := (productReviewCapabilityBinder{}).Bind(authidentity.WithAuthenticatedIdentity(ctx, identity), "Bearer operator")
	require.NoError(t, err)
	_, options, err := collectionAcquisitionOptions([]bool{true})
	require.NoError(t, err)
	acquisition, err := productsourcing.NewPublicAcquisition(ctx, f.db, &productReviewLiveOrganizationAccess{resolver: resolver, now: time.Now}, permissions, f.provider, testHTTPResourceCharges{db: f.db}, options...)
	require.NoError(t, err)
	saved, err := acquisition.Acquire(bound, uuid.NewString(), "981645030344")
	require.NoError(t, err)
	require.NotNil(t, saved.Publication)
	collections, err := buildProductCollectionService(ctx, f.db, deps, permissions)
	require.NoError(t, err)
	// Only this disposable fixture grants native Preparation/record read access.
	require.NoError(t, f.owner.Exec(`GRANT SELECT,INSERT,UPDATE ON listing_preparations,listing_preparation_sources TO source_acquisition_runtime; GRANT SELECT ON listing_target_records,listing_preparation_targets,listing_target_record_commands TO source_acquisition_runtime`).Error)
	core, err := composePreparationCore(ctx, f.db, deps, permissions, collections)
	require.NoError(t, err)
	var item struct {
		ID, BatchID string
		Revision    int64
	}
	require.NoError(t, f.owner.Raw(`SELECT i.id,i.batch_id,b.revision FROM product_collection_items i JOIN product_collection_batches b ON i.organization_id=b.organization_id AND i.actor_id=b.actor_id AND i.batch_id=b.id WHERE i.publication_id=?`, saved.Publication.Receipt.PublicationID).Scan(&item).Error)
	require.NotEmpty(t, item.ID)
	transfer, err := core.app.Preparations.Transfer(bound, uuid.NewString(), collection.BatchSelectionInput{BatchID: item.BatchID, ExpectedRevision: item.Revision, ItemIDs: []string{item.ID}})
	require.NoError(t, err)
	rows, err := core.app.Preparations.ListSources(bound, transfer.Preparation.ID, collection.Query{Limit: 20})
	require.NoError(t, err)
	require.Len(t, rows.Items, 1)
	storeID := uuid.NewString()
	value := target.TargetRecord{ID: uuid.NewString(), Revision: 1, Source: rows.Items[0], EffectiveVersion: saved.Publication.Receipt.CatalogVersion, ProductHash: strings.Repeat("a", 64), InventoryHash: strings.Repeat("b", 64), RulesHash: strings.Repeat("c", 64), Merchant: storecenter.ProductMerchantBinding{OrganizationID: "B", StoreID: storeID, Site: "shein-us"}, Input: target.TargetInput{SourceID: rows.Items[0].ID, StoreID: storeID, EffectiveVersion: saved.Publication.Receipt.CatalogVersion, Draft: goods.OfficialDraftInput{Product: model.PublishProduct{Names: []model.LanguageContent{{Language: "en", Name: "exact historical material"}}}}}, Result: goods.OfficialDraft{Issues: []goods.OfficialIssue{{Field: "images", Code: "missing", Message: "saved construction issue"}}}, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	value.TargetID = target.TargetIdentity(collection.Scope{OrganizationID: "B", ActorID: "operator"}, value.Source.ID, storeID)
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	// A synthetic native business record, never a pre-seeded report. The real
	// native repository must verify org/actor/member, body hash and source proof.
	require.NoError(t, f.owner.Exec(`INSERT INTO listing_target_records(organization_id,actor_id,member_id,id,target_id,source_id,store_id,site,revision,input_hash,body_hash,body_json,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, "B", "operator", identity.EffectiveMemberID, value.ID, value.TargetID, value.Source.ID, storeID, "shein-us", 1, collection.Digest(value.Input), collection.Digest(value), string(raw), value.CreatedAt).Error)
	core.app.Records, err = recordstore.NewRepository(ctx, f.db)
	require.NoError(t, err)
	owner, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	ownerPool, _ := owner.DB()
	t.Cleanup(func() { _ = ownerPool.Close() })
	require.NoError(t, owner.Exec(`REVOKE CREATE ON DATABASE reports FROM PUBLIC; REVOKE CREATE ON SCHEMA public FROM PUBLIC; CREATE ROLE report_center_runtime LOGIN PASSWORD 'fixture-report' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`).Error)
	require.NoError(t, reportstore.InstallSchema(owner))
	require.NoError(t, reportstore.GrantRuntime(owner, "report_center_runtime"))
	u.User = url.UserPassword("report_center_runtime", "fixture-report")
	reports, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, _ := reports.DB()
	t.Cleanup(func() { _ = pool.Close() })
	module, err := buildReportCenter(ctx, reports, deps, permissions, nil, core.app)
	require.NoError(t, err)
	router := gin.New()
	currentIdentity := identity
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), currentIdentity))
	})
	for _, route := range module.(reportCenterModule).routes {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	request := func(method, suffix, key, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, reporthttp.Base+suffix, strings.NewReader(body))
		r.Header.Set("Authorization", "Bearer "+currentIdentity.UserID)
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	path := "/sources/SHEIN_TARGET_RECORD/" + value.ID
	response := request("GET", path, "", "")
	require.Equal(t, 200, response.Code, response.Body.String())
	var source rc.SourceInfo
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &source))
	body, _ := json.Marshal(map[string]any{"source": source.Ref})
	key := uuid.NewString()
	response = request("POST", "", key, string(body))
	require.Equal(t, 200, response.Code, response.Body.String())
	var first rc.Result
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &first))
	require.Contains(t, response.Body.String(), "exact historical material")
	policy.sources = false
	require.Equal(t, 403, request("GET", path, "", "").Code, "report permissions cannot grant the original source")
	require.Equal(t, 200, request("GET", "/"+first.Report.ID, "", "").Code, "existing snapshots do not re-read the source")
	policy.sources = true
	for _, change := range []func(){func() { currentIdentity.UserID = "other" }, func() { currentIdentity.TenantID = "A"; currentIdentity.EffectiveOrganizationID = "A" }} {
		currentIdentity = identity
		change()
		response = request("GET", path, "", "")
		require.NotEqual(t, 200, response.Code, response.Body.String())
	}
	currentIdentity = identity
	require.NoError(t, f.owner.Exec(`UPDATE listing_target_records SET member_id='different-member' WHERE id=?`, value.ID).Error)
	require.Equal(t, 404, request("GET", path, "", "").Code)
	// Reconstruct the normal module with new capture disabled. Original report
	// recovery, read, favorite and JSON/text data do not recapture the source.
	off, err := buildReportCenter(ctx, reports, deps, permissions, nil, nil)
	require.NoError(t, err)
	router = gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authidentity.WithAuthenticatedIdentity(c.Request.Context(), identity))
	})
	for _, route := range off.(reportCenterModule).routes {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	response = request("POST", "", key, string(body))
	require.Equal(t, 200, response.Code, response.Body.String())
	var replay rc.Result
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &replay))
	require.True(t, replay.Replayed)
	require.Equal(t, first.Report, replay.Report)
	require.Equal(t, 503, request("GET", path, "", "").Code)
	require.Equal(t, 200, request("GET", "/"+first.Report.ID, "", "").Code)
	require.Equal(t, 200, request("POST", "/"+first.Report.ID+"/favorite", uuid.NewString(), `{"favorite":true}`).Code)
	require.EqualValues(t, 1, f.fetches.Load(), "report capture must not call the synthetic acquisition provider again")
}
