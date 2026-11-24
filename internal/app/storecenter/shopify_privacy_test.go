package storecenterapp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	storehttp "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/httpapi"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const shopifyPrivacyFixtureSecret = "fixture-privacy-secret"

func TestIsolatedShopifyPrivacyReceiptPersistsBeforeAcknowledging(t *testing.T) {
	db := newShopifyChargeStoreDatabase(t)
	now := time.Date(2026, 11, 24, 8, 0, 0, 0, time.UTC)
	newRouter := func() *gin.Engine {
		return isolatedShopifyPrivacyTestRouter(t, db, func() time.Time { return now })
	}
	router := newRouter()
	const event = "fixture-event-token"
	const shopBody = `{"shop_id":19,"shop_domain":"fixture.myshopify.com"}`
	const buyerRequest = `{"shop_id":19,"shop_domain":"fixture.myshopify.com","orders_requested":[1234567891],"customer":{"id":222233333,"email":"fixture-buyer@example.invalid","phone":"+15555550123"},"data_request":{"id":44445555}}`
	const buyerRedact = `{"shop_id":19,"shop_domain":"fixture.myshopify.com","orders_to_redact":[1234567891],"customer":{"email":"fixture-buyer@example.invalid"}}`
	shopPath, shopTopic := storehttp.ShopifyShopRedactPath, wire.TopicShopRedact
	require.Equal(t, http.StatusUnauthorized, deliverShopifyPrivacyNotice(router, shopPath, shopTopic, event, shopBody, false).Code)
	require.Equal(t, http.StatusServiceUnavailable, deliverShopifyPrivacyNotice(router, shopPath, shopTopic, event, shopBody, true).Code)
	installShopifyPrivacyTestSchema(t, db)

	for _, test := range []struct{ path, topic, body string }{
		{storehttp.ShopifyCustomerDataRequestPath, wire.TopicCustomerDataRequest, buyerRequest},
		{storehttp.ShopifyCustomerRedactPath, wire.TopicCustomerRedact, buyerRedact},
		{shopPath, shopTopic, shopBody},
	} {
		response := deliverShopifyPrivacyNotice(router, test.path, test.topic, event, test.body, true)
		require.Equal(t, http.StatusOK, response.Code, test.topic)
		require.Empty(t, response.Body.String())
	}
	require.Equal(t, http.StatusBadRequest, deliverShopifyPrivacyNotice(router, shopPath, shopTopic, "wrong-shape", buyerRedact, true).Code)
	require.Equal(t, http.StatusBadRequest, deliverShopifyPrivacyNotice(router, shopPath, shopTopic, "duplicate-key",
		`{"shop_id":19,"shop_id":20,"shop_domain":"fixture.myshopify.com"}`, true).Code)

	firstReceipt := now
	now = now.Add(48 * time.Hour)
	restarted := newRouter()
	require.Equal(t, http.StatusOK, deliverShopifyPrivacyNotice(restarted, shopPath, shopTopic, event, shopBody, true).Code)
	require.Equal(t, http.StatusBadRequest, deliverShopifyPrivacyNotice(restarted, shopPath, shopTopic, event,
		`{"shop_id":20,"shop_domain":"another.myshopify.com"}`, true).Code)

	var count int
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.workbench_store_shopify_privacy_receipts`).Row().Scan(&count))
	require.Equal(t, 3, count)
	var validBuyerRows int
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.workbench_store_shopify_privacy_receipts
		WHERE topic IN ('customers/data_request','customers/redact') AND status='no_customer_data' AND shop_id IS NULL AND due_at IS NULL
		  AND to_jsonb(workbench_store_shopify_privacy_receipts) - ARRAY['app_id','topic','event_digest','shop_id','status','received_at','due_at'] = '{}'::jsonb`).Row().Scan(&validBuyerRows))
	require.Equal(t, 2, validBuyerRows)
	var buyerSnapshot string
	require.NoError(t, db.Raw(`SELECT jsonb_agg(to_jsonb(receipt))::text FROM public.workbench_store_shopify_privacy_receipts receipt WHERE shop_id IS NULL`).
		Row().Scan(&buyerSnapshot))
	for _, value := range []string{event, "fixture.myshopify.com", "gid://shopify/Shop/", "fixture-buyer", "+15555550123", "1234567891", "222233333", "44445555"} {
		require.NotContains(t, buyerSnapshot, value)
	}
	var shop, status string
	var receivedAt, dueAt time.Time
	require.NoError(t, db.Raw(`SELECT shop_id,status,received_at,due_at FROM public.workbench_store_shopify_privacy_receipts
		WHERE topic='shop/redact'`).Row().Scan(&shop, &status, &receivedAt, &dueAt))
	require.Equal(t, "gid://shopify/Shop/19", shop)
	require.Equal(t, "pending_verification", status)
	require.True(t, firstReceipt.Equal(receivedAt))
	require.True(t, firstReceipt.Add(30*24*time.Hour).Equal(dueAt))
}

func TestIsolatedShopifyPrivacyNoticePreservesBothGenerationsAfterReinstall(t *testing.T) {
	ctx := context.Background()
	db := newShopifyChargeStoreDatabase(t)
	installShopifyPrivacyTestSchema(t, db)
	repo, err := storecenter.NewMemberScopedStoreRepository(db, serviceContextAccess{})
	require.NoError(t, err)
	repo = repo.WithShopifyLifecycle(true)
	member := serviceContext("operator", "member-a", authz.EnterpriseRoleKey("org-a", 1))
	store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator",
		Name: "Privacy fixture", Platform: "shopify", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	require.NoError(t, err)
	store, _, err = repo.CreateOrReplay(member, "org-a", store)
	require.NoError(t, err)
	installation := storecenter.ShopifyInstallation{AppID: "gid://shopify/App/812", ShopID: "gid://shopify/Shop/19",
		InstallationID: "gid://shopify/AppInstallation/27", Domain: "fixture.myshopify.com", Currency: "USD"}
	first := pairShopifyChargeStore(t, repo, member, store, installation)
	store, err = repo.Get(member, "org-a", store.ID())
	require.NoError(t, err)
	installation.InstallationID = "gid://shopify/AppInstallation/28"
	current := pairShopifyChargeStore(t, repo, member, store, installation)
	require.Greater(t, current.Generation, first.Generation)
	require.Equal(t, first.ShopKey, current.ShopKey)

	var activeCount, revokedCount int
	findAuthorizations := func() string {
		var snapshot string
		require.NoError(t, db.Raw(`SELECT jsonb_agg(to_jsonb(auth) ORDER BY generation)::text,
			count(*) FILTER (WHERE revoked_at IS NULL), count(*) FILTER (WHERE revoked_at IS NOT NULL)
			FROM public.workbench_store_shopify_authorizations auth WHERE shop_key=?`, first.ShopKey).
			Row().Scan(&snapshot, &activeCount, &revokedCount))
		return snapshot
	}
	before := findAuthorizations()
	require.Equal(t, 1, activeCount)
	require.Equal(t, 1, revokedCount, "normal replacement already sets the old generation's revoked_at")

	router := isolatedShopifyPrivacyTestRouter(t, db, time.Now)
	const delayed = `{"shop_id":19,"shop_domain":"fixture.myshopify.com"}`
	for range 2 {
		require.Equal(t, http.StatusOK, deliverShopifyPrivacyNotice(router, storehttp.ShopifyShopRedactPath,
			wire.TopicShopRedact, "delayed-event", delayed, true).Code)
	}
	require.Equal(t, before, findAuthorizations())
	revision, err := repo.ReadShopifyLifecycleRevision(ctx, installation)
	require.NoError(t, err)
	require.Zero(t, revision)
	var pending int
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.workbench_store_shopify_privacy_receipts WHERE status='pending_verification'`).Row().Scan(&pending))
	require.Equal(t, 1, pending)
}

func installShopifyPrivacyTestSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	ctx := context.Background()
	pool, err := db.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallIsolatedShopifyPrivacySchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
}

func isolatedShopifyPrivacyTestRouter(t *testing.T, db *gorm.DB, now func() time.Time) *gin.Engine {
	t.Helper()
	repo, err := storecenter.NewMemberScopedStoreRepository(db, serviceContextAccess{})
	require.NoError(t, err)
	transport := &http.Client{Transport: startedShopifyTransport(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("privacy receipt made a provider call: %s", r.URL.Path)
		return nil, nil
	})}
	oauth, err := wire.NewOAuthClient(transport, "fixture-client", shopifyPrivacyFixtureSecret, "https://fixture.invalid/callback")
	require.NoError(t, err)
	router := gin.New()
	for _, route := range storehttp.IsolatedShopifyPrivacyReceiptRoutes(IsolatedShopifyPrivacyReceiver(repo, "gid://shopify/App/812", oauth, now)) {
		router.POST(route.Path, route.Handler)
	}
	return router
}

func deliverShopifyPrivacyNotice(router *gin.Engine, path, topic, event, payload string, sign bool) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Shopify-Topic", topic)
	request.Header.Set("X-Shopify-Event-Id", event)
	mac := hmac.New(sha256.New, []byte(shopifyPrivacyFixtureSecret))
	if sign {
		mac.Write([]byte(payload))
	}
	request.Header.Set("X-Shopify-Hmac-Sha256", base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}
