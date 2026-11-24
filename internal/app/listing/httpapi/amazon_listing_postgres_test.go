package listinghttp

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	listingapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/listing"
	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	amazonapi "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/credentialprotection"
	officialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/official"
	prepstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/preparation"
	recordstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record"
	effectsstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	assetstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/asset"
	catalogstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/catalog"
	collectionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazonus"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestAmazonListingWorkbenchReviewConfirmAndCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("isolated_amazon_listing"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("synthetic-amazon"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	for _, install := range []func(*gorm.DB) error{
		catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema, recordstore.InstallSchema,
		effectsstore.InstallSchema, officialstore.InstallOfficialSchema, assetstore.AutoMigrate,
	} {
		require.NoError(t, install(db))
	}
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
	require.NoError(t, officialstore.InstallIsolatedAmazonSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())

	scope := collection.Scope{OrganizationID: "org-amazon", ActorID: "actor-amazon", MemberID: "member-amazon"}
	authority := &amazonFixtureAuthority{scope: scope}
	application := storecenter.AmazonAuthorizationApplication{
		ID: "amzn1.sellerapps.app.00000000-0000-4000-8000-000000000001", Revision: "isolated-1",
		RedirectURI: "https://workbench.example.org/workbench/stores/amazon/redirect",
	}
	protector, err := credentialprotection.NewAESGCM("synthetic", bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	stores, err := storecenter.NewMemberScopedStoreRepository(db, authority)
	require.NoError(t, err)
	storeID := seedAmazonListingStore(t, ctx, db, stores, scope, application, protector, "SYNTHETICONE")
	otherStoreID := seedAmazonListingStore(t, ctx, db, stores, scope, application, protector, "SYNTHETICTWO")
	revokedStoreID := seedAmazonListingStore(t, ctx, db, stores, scope, application, protector, "SYNTHETICTHREE")
	rejectStoreID := seedAmazonListingStore(t, ctx, db, stores, scope, application, protector, "SYNTHETICFOUR")
	sourceID, snapshot := seedAmazonListingSource(t, db, scope, "synthetic-amazon-product")
	duplicateSourceID, duplicateSnapshot := seedAmazonListingSource(t, db, scope, "synthetic-amazon-second-product")

	prepRepo, err := prepstore.NewRepository(ctx, db)
	require.NoError(t, err)
	collections, err := collection.NewService(struct{ collection.Repository }{}, authority, struct {
		sourcing.PublishedAcquisitionReader
	}{})
	require.NoError(t, err)
	preps, err := preparation.NewService(prepRepo, collections, authority)
	require.NoError(t, err)
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, sourcing.MaxEncodedSnapshotBytes)
	require.NoError(t, err)
	sources, err := preparation.NewSourceSelector(preps, collections, prepRepo, snapshots)
	require.NoError(t, err)
	sources, err = sources.WithExecution(authority, collection.ExecutionOwnerAuthority{Authorization: authority})
	require.NoError(t, err)
	assets, err := assetstore.NewRepository(db)
	require.NoError(t, err)
	for _, product := range []catalog.PublishedSnapshot{snapshot, duplicateSnapshot} {
		_, err = assets.CommitApproval(ctx, asset.ApprovalCommit{
			TenantID: scope.OrganizationID, ProductKey: product.Identity.ProductKey, TargetPlatform: "product",
			ActionID: "approval-" + product.Identity.ProductKey, SourceSnapshotVersion: 1,
			Assets: []asset.ApprovedAsset{{ID: amazonFixtureAssetID(product.Identity.ProductKey), RunID: "fixture-run", PlanRevision: 1, SlotID: "main", Attempt: 1,
				Role: asset.RoleMain, URL: "https://images.example.org/main.jpg", Width: 900, Height: 900}},
		})
		require.NoError(t, err)
	}
	provider := &syntheticAmazonListings{}
	client, err := amazonapi.NewClient(amazonapi.Options{ClientID: "fixture", ClientSecret: "fixture", RedirectURI: application.RedirectURI,
		UserAgent: "Shuomi/1.0", HTTPClient: &http.Client{Transport: provider}})
	require.NoError(t, err)
	access, err := storeapp.NewAmazonProductAccess(stores, client, protector, application)
	require.NoError(t, err)
	preparer := listingapp.AmazonTargetPreparer{Access: access, Images: amazonFixtureImages{}}
	products := record.OriginalTargetProduct{}
	records, err := recordstore.NewRepository(ctx, db)
	require.NoError(t, err)
	targets, err := record.NewTargetService(record.TargetDependencies{
		Sources: sources, Products: products, Assets: assets, Rules: struct{ record.TargetRuleReader }{},
		Amazon: preparer, Records: records, Images: amazonFixtureImages{}, Authorizer: authority,
	})
	require.NoError(t, err)
	official, err := officialstore.NewOfficialRepository(ctx, db)
	require.NoError(t, err)
	effects, err := effectsstore.NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(effects)
	require.NoError(t, err)
	app := &listingapp.Application{Authorization: authority, Targets: targets, Records: records, Sources: sources,
		Amazon: &listingapp.AmazonListingService{Preparer: preparer, Records: records, Sources: sources, Products: products,
			Assets: assets, Authorization: authority, Repository: official, Kernel: kernel}}
	router := gin.New()
	for _, route := range ListingRoutes(app, func(ctx context.Context, _ string) (context.Context, error) {
		return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
			UserID: scope.ActorID, TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID,
			EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute),
		}), nil
	}) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	requestRoute := func(action, key string, input any) *httptest.ResponseRecorder {
		t.Helper()
		wire, e := json.Marshal(input)
		require.NoError(t, e)
		request := httptest.NewRequest(http.MethodPost, ListingBasePath+"/amazon-listings/"+action, bytes.NewReader(wire))
		request.Header.Set("Content-Type", "application/json")
		if key != "" {
			request.Header.Set("Idempotency-Key", key)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	invoke := func(action, key string, input any, into any) {
		t.Helper()
		response := requestRoute(action, key, input)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), into))
	}
	quantity := 0
	in := amazonReviewInput{SourceID: sourceID, StoreID: storeID, EffectiveVersion: 1, Details: amazonmarket.DraftInput{
		ProductType: "LUGGAGE", Title: "Synthetic pouch", Brand: "Synthetic", GTIN: "036000291452", PriceCents: 1230, Quantity: &quantity, MainImageID: amazonFixtureAssetID(snapshot.Identity.ProductKey),
	}}
	var review amazonReviewView
	invoke("review", uuid.NewString(), in, &review)
	require.Equal(t, int32(0), provider.writes.Load(), "review cannot publish")
	require.Equal(t, "Synthetic pouch", review.Details.Title)
	require.Equal(t, 0, *review.Details.Quantity)
	saved, err := records.ReadTargetRecord(ctx, scope, review.RecordID)
	require.NoError(t, err)
	require.True(t, record.PublicationReady(saved))
	var result listingapp.AmazonListingResult
	key := uuid.NewString()
	confirmed := map[string]string{"recordId": review.RecordID, "wireHash": review.WireHash}
	invoke("confirm", key, confirmed, &result)
	require.Equal(t, "accepted", result.Submission)
	require.Equal(t, "SYNTHETIC-RECEIPT", result.Receipt.SubmissionID)
	require.Nil(t, result.Observation)
	attemptID := result.AttemptID
	invoke("confirm", key, confirmed, &result)
	invoke("confirm", uuid.NewString(), confirmed, &result)
	require.Equal(t, attemptID, result.AttemptID)
	require.Equal(t, int32(1), provider.writes.Load(), "same review and SKU must send once")
	query := map[string]string{"sourceId": sourceID, "storeId": storeID}
	invoke("check", "", query, &result)
	require.Equal(t, "not_found", result.Check)
	provider.visible.Store(true)
	provider.noASIN.Store(true)
	result = listingapp.AmazonListingResult{}
	invoke("check", "", query, &result)
	require.NotNil(t, result.Observation)
	require.Empty(t, result.Observation.ASIN)
	require.Equal(t, "90220", result.Observation.Issues[0].Code)
	provider.noASIN.Store(false)
	result = listingapp.AmazonListingResult{}
	invoke("check", "", query, &result)
	require.Equal(t, "B000TE0001", result.Observation.ASIN)
	require.True(t, result.Observation.Buyable)

	duplicate := in
	duplicate.SourceID = duplicateSourceID
	duplicate.Details.GTIN = "00036000291452"
	duplicate.Details.MainImageID = amazonFixtureAssetID(duplicateSnapshot.Identity.ProductKey)
	var duplicateReview amazonReviewView
	invoke("review", uuid.NewString(), duplicate, &duplicateReview)
	response := requestRoute("confirm", uuid.NewString(), map[string]string{"recordId": duplicateReview.RecordID, "wireHash": duplicateReview.WireHash})
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Equal(t, int32(1), provider.writes.Load(), "equivalent GTIN belonging to another saved product must not send")

	in.StoreID = otherStoreID
	provider.loseResponse.Store(true)
	invoke("review", uuid.NewString(), in, &review)
	confirmed = map[string]string{"recordId": review.RecordID, "wireHash": review.WireHash}
	result = listingapp.AmazonListingResult{}
	invoke("confirm", uuid.NewString(), confirmed, &result)
	require.Equal(t, "unknown", result.Submission)
	require.Nil(t, result.Receipt)
	invoke("confirm", uuid.NewString(), confirmed, &result)
	require.Equal(t, int32(2), provider.writes.Load(), "lost response must never trigger a second PUT")
	invoke("check", "", map[string]string{"sourceId": sourceID, "storeId": otherStoreID}, &result)
	require.Equal(t, "unknown", result.Submission)
	require.NotNil(t, result.Observation)
	require.Nil(t, result.Receipt, "a GET observation cannot make the original PUT look accepted")

	in.StoreID = revokedStoreID
	invoke("review", uuid.NewString(), in, &review)
	provider.loseResponse.Store(false)
	authority.submitGate = func(ctx context.Context) bool {
		var claimed bool
		e := db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM public.listing_submission_execution_attempts WHERE store_id=? AND status='claimed')", revokedStoreID).Scan(&claimed).Error
		return e == nil && !claimed
	}
	result = listingapp.AmazonListingResult{}
	invoke("confirm", uuid.NewString(), map[string]string{"recordId": review.RecordID, "wireHash": review.WireHash}, &result)
	require.Equal(t, int32(2), provider.writes.Load(), "authorization revoked after acquisition must prevent the provider PUT")
	require.Nil(t, result.Receipt)

	authority.submitGate = nil
	provider.reject.Store(true)
	in.StoreID = rejectStoreID
	invoke("review", uuid.NewString(), in, &review)
	result = listingapp.AmazonListingResult{}
	invoke("confirm", uuid.NewString(), map[string]string{"recordId": review.RecordID, "wireHash": review.WireHash}, &result)
	require.Equal(t, "rejected", result.Submission)
	require.False(t, result.Receipt.Accepted)
	require.Equal(t, "90220", result.Receipt.Issues[0].Code)
	require.Equal(t, int32(3), provider.writes.Load())
}

type amazonFixtureAuthority struct {
	scope      collection.Scope
	submitGate func(context.Context) bool
}

func (a amazonFixtureAuthority) Authorize(context.Context, string) (collection.Scope, error) {
	return a.scope, nil
}
func (a amazonFixtureAuthority) AuthorizeExecution(ctx context.Context, scope collection.Scope, permission string) error {
	if scope != a.scope || permission == preparation.PermissionSubmit && a.submitGate != nil && !a.submitGate(ctx) {
		return collection.ErrForbidden
	}
	return nil
}

func amazonFixtureAssetID(product string) string { return "main-" + product }
func (a amazonFixtureAuthority) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if org != a.scope.OrganizationID {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{
		OrganizationID: org, ActorID: a.scope.ActorID, MemberID: a.scope.MemberID, Administrator: true, CanWrite: true,
	}, nil
}

type amazonFixtureImages struct{ record.TargetImageReader }

func (amazonFixtureImages) Probe(_ context.Context, image asset.ApprovedAsset, slot int) (goods.OfficialImageObservation, error) {
	return goods.OfficialImageObservation{AssetID: image.ID, SourceURL: image.URL, Type: slot, ContentHash: collection.Digest(image.ID),
		Width: 900, Height: 900, Bytes: 1000, MediaType: "image/jpeg"}, nil
}

func seedAmazonListingStore(t *testing.T, ctx context.Context, db *gorm.DB, stores *storecenter.MemberScopedStoreRepository, scope collection.Scope,
	app storecenter.AmazonAuthorizationApplication, protector *credentialprotection.AESGCM, seller string) string {
	t.Helper()
	id, attemptID := uuid.NewString(), uuid.NewString()
	fixture, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: id, OrganizationID: scope.OrganizationID, ActorSubject: scope.ActorID,
		Name: "Synthetic Amazon Store", Platform: "shein", Region: "US", ExternalStoreID: id, CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	_, _, err = stores.CreateOrReplay(ctx, scope.OrganizationID, fixture)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE public.workbench_stores SET platform='amazon' WHERE id=?", id).Error)
	proof := storecenter.AmazonAuthorizationClaim{Scope: storecenter.AmazonBrowserScope{OrganizationID: scope.OrganizationID},
		Application: app, StoreID: id, AttemptID: attemptID, SellerID: seller, Generation: 1}
	token, err := protector.Seal([]byte("synthetic-refresh"), proof.CredentialAAD())
	require.NoError(t, err)
	require.NoError(t, db.Exec(`INSERT INTO public.workbench_store_amazon_grants
		(organization_id,store_id,attempt_id,app_id,app_revision,beta,seller_id,generation,key_id,ciphertext,bound_at)
		VALUES (?,?,?,?,?,false,?,1,?,?,?)`, scope.OrganizationID, id, attemptID, app.ID, app.Revision, seller, protector.KeyID(), token, time.Now().UTC().Add(-time.Minute)).Error)
	return id
}

func seedAmazonListingSource(t *testing.T, db *gorm.DB, scope collection.Scope, product string) (string, catalog.PublishedSnapshot) {
	t.Helper()
	source, prep, batch, item, publication := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	now := time.Now().UTC().Truncate(time.Microsecond)
	snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: product}, Version: 1,
		PublicationID: publication, Snapshot: catalog.ProductSnapshot{Title: "Synthetic pouch"}}
	raw, err := json.Marshal(snapshot.Snapshot)
	require.NoError(t, err)
	hash := sha256.Sum256(raw)
	require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: product, Version: 1, PublicationID: publication,
		PayloadHash: hex.EncodeToString(hash[:]), SnapshotJSON: raw}).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES(?,?,?,?,?,'manual',1,?)",
		scope.OrganizationID, scope.ActorID, scope.MemberID, batch, "Synthetic Amazon batch", now).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES(?,?,?,?,?,?,?,1,'own','',1,?)",
		scope.OrganizationID, scope.ActorID, scope.MemberID, item, batch, product, publication, now).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparations(organization_id,actor_id,member_id,id,command_key,input_hash,input_json,source_batch_id,source_revision,name,item_count,revision,created_at) VALUES(?,?,?,?,?,?,'{}',?,1,'Fixture',1,1,?)",
		scope.OrganizationID, scope.ActorID, scope.MemberID, prep, uuid.NewString(), collection.Digest("amazon-prep"), batch, now).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparation_sources(organization_id,actor_id,member_id,id,preparation_id,collection_item_id,collection_revision,product_key,publication_id,original_version,source_kind,source_operation_id) VALUES(?,?,?,?,?,?,1,?,?,1,'own','')",
		scope.OrganizationID, scope.ActorID, scope.MemberID, source, prep, item, product, publication).Error)
	return source, snapshot
}

type syntheticAmazonListings struct {
	writes       atomic.Int32
	visible      atomic.Bool
	loseResponse atomic.Bool
	noASIN       atomic.Bool
	reject       atomic.Bool
}

func (a *syntheticAmazonListings) RoundTrip(request *http.Request) (*http.Response, error) {
	const schema = `{"$schema":"https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1","type":"object","required":["brand","fulfillment_availability","purchasable_offer"]}`
	reply := func(status int, body string) (*http.Response, error) {
		header := make(http.Header)
		header.Set("X-Amzn-RequestId", "synthetic")
		return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
	}
	path := request.URL.Path
	switch {
	case request.URL.Host == "api.amazon.com" && path == "/auth/o2/token":
		return reply(200, `{"access_token":"synthetic-access","token_type":"bearer","expires_in":3600}`)
	case request.URL.Host == "selling-partner-definitions-prod-iad.s3.amazonaws.com":
		return reply(200, schema)
	case request.URL.Host != "sellingpartnerapi-na.amazon.com" || request.Header.Get("X-Amz-Access-Token") != "synthetic-access":
		return nil, errors.New("fixture rejected an unexpected network request")
	case strings.HasPrefix(path, "/definitions/"):
		checksum := md5.Sum([]byte(schema))
		return reply(200, `{"productType":"LUGGAGE","locale":"en_US","requirements":"LISTING","requirementsEnforced":"ENFORCED","marketplaceIds":["ATVPDKIKX0DER"],"productTypeVersion":{"version":"fixture","latest":true},"schema":{"link":{"verb":"GET","resource":"https://selling-partner-definitions-prod-iad.s3.amazonaws.com/schema/LUGGAGE.json/fixture?X-Amz-Signature=`+strings.Repeat("a", 64)+`"},"checksum":"`+base64.StdEncoding.EncodeToString(checksum[:])+`"}}`)
	case path == "/catalog/2022-04-01/items" || request.URL.Query().Get("identifiersType") == "SKU":
		return reply(200, `{"numberOfResults":0,"items":[]}`)
	case strings.HasPrefix(path, "/listings/") && request.Method == http.MethodPut:
		sku := path[strings.LastIndexByte(path, '/')+1:]
		if request.URL.Query().Get("mode") == "VALIDATION_PREVIEW" {
			return reply(200, `{"sku":"`+sku+`","status":"VALID","issues":[]}`)
		}
		a.writes.Add(1)
		if a.loseResponse.Load() {
			return nil, errors.New("synthetic connection lost after Amazon receives the PUT")
		}
		if a.reject.Load() {
			return reply(200, `{"sku":"`+sku+`","status":"INVALID","submissionId":"SYNTHETIC-REJECTION","issues":[{"code":"90220","severity":"ERROR","attributeNames":["brand"]}]}`)
		}
		return reply(200, `{"sku":"`+sku+`","status":"ACCEPTED","submissionId":"SYNTHETIC-RECEIPT","issues":[]}`)
	case strings.HasPrefix(path, "/listings/") && request.Method == http.MethodGet:
		if !a.visible.Load() {
			return reply(404, `{"errors":[{"code":"NOT_FOUND"}]}`)
		}
		sku := path[strings.LastIndexByte(path, '/')+1:]
		if a.noASIN.Load() {
			return reply(200, `{"sku":"`+sku+`","summaries":[{"marketplaceId":"ATVPDKIKX0DER","status":[]}],"issues":[{"code":"90220","severity":"ERROR","attributeNames":["brand"]}]}`)
		}
		return reply(200, `{"sku":"`+sku+`","summaries":[{"marketplaceId":"ATVPDKIKX0DER","asin":"B000TE0001","status":["BUYABLE","DISCOVERABLE"]}],"issues":[]}`)
	}
	return nil, errors.New("unimplemented synthetic Amazon operation")
}
