//go:build integration

package storeobservations

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	app "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storeobservations"
	routes "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storeobservations/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authruntime/zitadel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type orderFixtureMember struct{}

func (orderFixtureMember) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	scope, err := app.Scope(ctx)
	if err != nil || scope.OrganizationID != org {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: scope.ActorID, MemberID: scope.MemberID, Administrator: true, CanWrite: true}, nil
}

type orderFixtureTransport func(*http.Request) (*http.Response, error)

func (f orderFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type orderFixtureIAM struct{}

func (orderFixtureIAM) ReadExactServiceProjectAuthorization(_ context.Context, token, actor, project, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	if token != "fixture-service" || project != "fixture-project" || (org != "org-a" && org != "org-b") {
		return zitadel.ExactServiceProjectAuthorization{}, nil
	}
	member, role := "member-"+org, authz.EnterpriseRoleKey(org, 1)
	switch {
	case actor == "order-member-org-a" && org == "org-a":
		member = actor
	case actor == "store-member-org-a" && org == "org-a":
		member, role = actor, authz.EnterpriseRoleKey(org, 2)
	case actor != "actor-"+org:
		return zitadel.ExactServiceProjectAuthorization{}, nil
	}
	return zitadel.ExactServiceProjectAuthorization{
		Found: true, State: "STATE_ACTIVE", AuthorizationID: member, Roles: []string{role},
	}, nil
}

type orderFixtureRolePolicy struct{}

func (orderFixtureRolePolicy) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	policies := make(map[string][]string, len(keys))
	for _, key := range keys {
		switch key {
		case authz.EnterpriseRoleKey(org, 1):
			policies[key] = []string{"store-orders"}
		case authz.EnterpriseRoleKey(org, 2):
			policies[key] = []string{"stores"}
		}
	}
	return policies, nil
}

type shopifyFixtureAuthorization struct {
	app.Authorization
	db       *gorm.DB
	bindings map[string]o.Binding
	reader   *wire.SyntheticOrderReader
}

func (a *shopifyFixtureAuthorization) Authorize(ctx context.Context, scope o.Scope, kind o.Kind, sync bool) error {
	// The current enterprise order module bundles read and sync. Exercise the
	// existing permission split with a synthetic read-only order grant.
	if scope == (o.Scope{OrganizationID: "org-a", ActorID: "reader-org-a", MemberID: "reader-org-a"}) {
		if kind != o.Orders || sync {
			return o.ErrForbidden
		}
		return nil
	}
	return a.Authorization.Authorize(ctx, scope, kind, sync)
}

func (a *shopifyFixtureAuthorization) Open(ctx context.Context, scope o.Scope, storeID string, kind o.Kind, sync bool, expected *o.Binding) (o.Merchant, error) {
	if err := a.Authorize(ctx, scope, kind, sync); err != nil {
		return nil, err
	}
	binding, known := a.bindings[storeID]
	if !known || binding.OrganizationID != scope.OrganizationID {
		return nil, o.ErrForbidden
	}
	if !binding.ValidShopifyOrderApplication() {
		return nil, o.ErrUnsupported
	}
	if expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	merchant := shopifyFixtureMerchant{binding: binding, db: a.db}
	if err := merchant.Check(ctx); err != nil {
		return nil, err
	}
	return merchant, nil
}

func (a *shopifyFixtureAuthorization) ReadShopifyOrders(ctx context.Context, scope o.Scope, binding o.Binding, query o.ShopifyOrderQuery) (o.ShopifyOrderPage, error) {
	m, err := a.Open(ctx, scope, binding.StoreID, o.Orders, true, &binding)
	if err != nil {
		return o.ShopifyOrderPage{}, err
	}
	return a.reader.Orders(ctx, "fixture.myshopify.com", binding.OrganizationID+"-order-fixture", query, m.Check)
}

type shopifyFixtureMerchant struct {
	o.Merchant // Only the Shopify order reader may read provider data in this fixture.
	binding    o.Binding
	db         *gorm.DB
}

func (m shopifyFixtureMerchant) Binding() o.Binding { return m.binding }
func (m shopifyFixtureMerchant) Check(ctx context.Context) error {
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return storecenter.LockShopifyOrderObservation(tx, m.binding)
	})
	if errors.Is(err, storecenter.ErrNotFound) {
		return o.ErrForbidden
	}
	if err != nil {
		return o.ErrUnavailable
	}
	return nil
}

type orderFixtureStarter struct{}

func (orderFixtureStarter) Ensure(context.Context, string, string) error { return nil }

func orderFixtureIdentity(ctx context.Context, account string, expires time.Time) context.Context {
	org, actor, member := account, "actor-"+account, "member-"+account
	if account == "reader-org-a" || account == "store-member-org-a" || account == "order-member-org-a" {
		org, actor, member = "org-a", account, account
	}
	return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{
		TenantID: org, EffectiveOrganizationID: org, UserID: actor,
		EffectiveMemberID: member, TokenExpiresAt: expires,
	})
}

func installSyntheticOrderRecords(t *testing.T, owner *gorm.DB, key string) {
	t.Helper()
	// Extend only the freshly installed test projection. Regular Store
	// schema initialization and all application configurations stay disabled.
	const start = "CONSTRAINT store_observation_record_variant CHECK("
	_, definition, found := strings.Cut(strings.ReplaceAll(schemaSQL, "\r\n", "\n"), start)
	require.True(t, found)
	baseline, _, found := strings.Cut(definition, "\n )\n);")
	require.True(t, found)
	err := owner.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE store_observations.records IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var count int
		if err := tx.Raw("SELECT count(*) FROM store_observations.records").Row().Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return errors.New("order fixtures require a fresh observation schema")
		}
		shape := `ALTER TABLE store_observations.records DROP CONSTRAINT store_observation_record_variant,
			ADD CONSTRAINT store_observation_record_variant CHECK ((NOT (payload ? 'shopifyOrder') AND (` + baseline + `))
			OR (kind='orders' AND jsonb_typeof(payload->'shopifyOrder')='object'
				AND payload= jsonb_build_object('shopifyOrder',payload->'shopifyOrder')
				AND coalesce(payload->'shopifyOrder'->>'id','')=platform_id))`
		return tx.Exec(strings.ReplaceAll(shape, "shopifyOrder", key)).Error
	})
	require.NoError(t, err)
}

func TestSyntheticShopifyOrdersHTTPAndOriginalObservationAuthority(t *testing.T) {
	ctx, owner, repo := observationDatabase(t)
	installSyntheticOrderRecords(t, owner, "shopifyOrder")
	pool, err := owner.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallShopifySchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallIsolatedShopifyOrderConsentTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, owner.Exec(`GRANT SELECT,UPDATE ON workbench_stores,workbench_store_shopify_shops,
 workbench_store_shopify_authorizations,workbench_store_shopify_order_consents TO observation_runtime`).Error)

	now := time.Now().UTC().Truncate(time.Second)
	stores, err := storecenter.NewMemberScopedStoreRepository(owner, orderFixtureMember{})
	require.NoError(t, err)
	newBinding := func(org, shopID string) o.Binding {
		auth := orderFixtureIdentity(ctx, org, now.Add(time.Hour))
		store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: org, ActorSubject: "actor-" + org,
			Name: "Shopify " + org, Platform: "shopify", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: now})
		require.NoError(t, err)
		store, _, err = stores.CreateOrReplay(auth, org, store)
		require.NoError(t, err)
		command := storecenter.OfficialConnectionCommand{OrganizationID: org, StoreID: store.ID(), AttemptID: uuid.NewString(), ExpectedStoreVersion: store.Version()}
		provider := storecenter.ShopifyInstallation{AppID: "gid://shopify/App/4", ShopID: "gid://shopify/Shop/" + shopID,
			InstallationID: "gid://shopify/AppInstallation/" + shopID, Domain: "fixture.myshopify.com", Currency: "USD"}
		browser, state := strings.Repeat("a", 64), strings.Repeat("b", 64)
		pair, err := stores.BeginShopifyPairing(auth, command, provider.AppID, browser, state, now)
		require.NoError(t, err)
		_, err = stores.ClaimShopifyPairing(auth, pair.Key(), browser, state, now)
		require.NoError(t, err)
		_, err = stores.StageShopifyPairing(auth, pair.Key(), browser, provider, 0, "fixture-key", "fixture-ciphertext", now)
		require.NoError(t, err)
		bound, err := stores.ConfirmShopifyPairing(auth, pair.Key(), browser, provider, now)
		require.NoError(t, err)
		expires := now.Add(24 * time.Hour)
		require.NoError(t, owner.Exec(`UPDATE workbench_stores SET service_status='active',service_started_at=?,service_expires_at=?
			WHERE organization_id=? AND id=?`, now.Add(-time.Hour), expires, org, store.ID()).Error)
		current, err := stores.Get(auth, org, store.ID())
		require.NoError(t, err)
		return o.Binding{Platform: storecenter.PlatformShopify, OrganizationID: org, StoreID: store.ID(), Site: "shopify",
			StoreVersion: current.Version(), ConnectionRevision: bound.Generation, ApplicationID: provider.AppID,
			ApplicationRevision: "shopify-admin-2026-10", SupplierIdentityHash: strings.Repeat("a", 64),
			Shopify: storecenter.ShopifyMerchantBinding{ShopKey: bound.ShopKey, Currency: "USD"}, ServiceExpiresAt: expires}
	}
	first, second := newBinding("org-a", "18"), newBinding("org-b", "19")
	grant := func(b o.Binding, generation int64, active bool) o.Binding {
		require.NoError(t, owner.Exec(`INSERT INTO workbench_store_shopify_order_consents VALUES(?,?,?,?,?) ON CONFLICT(organization_id,shop_key)
			DO UPDATE SET order_generation=EXCLUDED.order_generation,granted=EXCLUDED.granted`,
			b.OrganizationID, b.Shopify.ShopKey, b.ConnectionRevision, generation, active).Error)
		b.Shopify.OrderGeneration = generation
		return b
	}
	calls := 0
	partial := false
	reader, err := wire.NewSyntheticOrderReader(orderFixtureTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		token := req.Header.Get("X-Shopify-Access-Token")
		require.Contains(t, []string{"org-a-order-fixture", "org-b-order-fixture"}, token)
		var request struct {
			Query     string
			Variables struct{ After *string }
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&request))
		require.Contains(t, request.Query, "OrderObservations")
		order := map[string]any{"id": "gid://shopify/Order/17", "name": "#1001", "createdAt": now.Add(-time.Hour), "updatedAt": now.Add(-time.Minute), "cancelledAt": nil, "displayFulfillmentStatus": "PARTIALLY_FULFILLED",
			"lineItems": map[string]any{"pageInfo": map[string]any{"hasNextPage": false}, "nodes": []map[string]any{{"id": "gid://shopify/LineItem/51", "title": "Item " + token, "sku": nil, "quantity": 2}}}}
		next, cursor := request.Variables.After == nil, ""
		if next {
			cursor = "second-page"
		}
		if partial {
			order["displayFulfillmentStatus"] = nil
		}
		raw, err := json.Marshal(map[string]any{"data": map[string]any{"orders": map[string]any{"pageInfo": map[string]any{"hasNextPage": next, "endCursor": cursor}, "nodes": []any{order}}}})
		require.NoError(t, err)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	}))
	require.NoError(t, err)
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(orderFixtureRolePolicy{})
	access := &shopifyFixtureAuthorization{
		Authorization: app.Authorization{
			Client: orderFixtureIAM{}, ProjectID: "fixture-project", Permissions: permissions,
			ServiceToken: func(context.Context) (string, error) { return "fixture-service", nil },
		},
		db: repo.db, bindings: map[string]o.Binding{first.StoreID: first, second.StoreID: second}, reader: reader,
	}
	service := &o.Service{Repository: repo, Access: access, ShopifyOrders: access, Directory: app.Directory{Stores: stores, ShopifyOrders: true},
		Starter: orderFixtureStarter{}, Now: func() time.Time { return now }}
	router := gin.New()
	for _, route := range routes.Routes(&app.Application{Service: service, Ready: func() bool { return true }}, func(ctx context.Context, bearer string) (context.Context, error) {
		switch bearer {
		case "Bearer org-a", "Bearer org-b", "Bearer reader-org-a", "Bearer store-member-org-a":
			return orderFixtureIdentity(ctx, strings.TrimPrefix(bearer, "Bearer "), now.Add(time.Hour)), nil
		default:
			return nil, o.ErrForbidden
		}
	}) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	httpCall := func(org, method, path, key, body string) *httptest.ResponseRecorder {
		var requestBody io.Reader
		if body != "" {
			requestBody = strings.NewReader(body)
		}
		req := httptest.NewRequest(method, routes.BasePath+"/orders"+path, requestBody).WithContext(ctx)
		req.Header.Set("Authorization", "Bearer "+org)
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		return res
	}
	begin := func(b o.Binding, key string) o.Sync {
		result := httpCall(b.OrganizationID, http.MethodPost, "/syncs", key, `{"kind":"orders","stores":["`+b.StoreID+`"]}`)
		require.Equal(t, 200, result.Code, result.Body.String())
		var response struct{ Data o.Command }
		require.NoError(t, json.Unmarshal(result.Body.Bytes(), &response))
		require.Len(t, response.Data.Syncs, 1, result.Body.String())
		return response.Data.Syncs[0]
	}
	complete := func(sync o.Sync, org string) o.Sync {
		var err error
		for !sync.Terminal() {
			sync, err = service.Step(ctx, org, sync.ID)
			require.NoError(t, err)
		}
		return sync
	}
	missing := begin(first, uuid.NewString())
	require.Equal(t, "suspended", missing.Status)
	require.Equal(t, "unsupported_application", missing.ErrorCode)
	require.Zero(t, calls)
	first, second = grant(first, 1, true), grant(second, 1, true)
	access.bindings[first.StoreID], access.bindings[second.StoreID] = first, second
	key := uuid.NewString()
	launched := begin(first, key)
	require.Equal(t, "pending", launched.Status)
	require.Equal(t, now.Add(-30*24*time.Hour), launched.Range.Start)
	require.Equal(t, launched.ID, begin(first, key).ID)
	done := complete(launched, "org-a")
	require.Equal(t, "completed", done.Status)
	require.Equal(t, 2, calls)
	require.Equal(t, done.ID, begin(first, key).ID)
	require.Equal(t, 2, calls)

	query := "?platform=shopify&storeId=" + first.StoreID
	statusPath := "/stores/" + first.StoreID + "/syncs/" + done.ID + "/records/17"
	list := httpCall("org-a", http.MethodGet, query, "", "")
	require.Equal(t, 200, list.Code, list.Body.String())
	var envelope struct{ Data o.Result }
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &envelope))
	result := envelope.Data
	require.True(t, result.Complete)
	require.Len(t, result.Items, 1)
	require.Equal(t, "#1001", result.Items[0].ShopifyOrder.Number)
	require.NotContains(t, list.Body.String(), "second-page")
	require.NotContains(t, list.Body.String(), `"windows":null`)
	detail := httpCall("org-a", http.MethodGet, statusPath, "", "")
	require.Equal(t, 200, detail.Code, detail.Body.String())
	require.Contains(t, detail.Body.String(), `"quantity":2`)
	require.NotContains(t, detail.Body.String(), `"amount"`)

	readOnlyList := httpCall("reader-org-a", http.MethodGet, query, "", "")
	require.Equal(t, http.StatusOK, readOnlyList.Code, readOnlyList.Body.String())
	readOnlyDetail := httpCall("reader-org-a", http.MethodGet, statusPath, "", "")
	require.Equal(t, http.StatusOK, readOnlyDetail.Code, readOnlyDetail.Body.String())
	syncBody := `{"kind":"orders","stores":["` + first.StoreID + `"]}`
	require.Equal(t, http.StatusForbidden, httpCall("store-member-org-a", http.MethodGet, query, "", "").Code)
	require.Equal(t, http.StatusForbidden, httpCall("store-member-org-a", http.MethodGet, statusPath, "", "").Code)
	require.Equal(t, http.StatusForbidden, httpCall("store-member-org-a", http.MethodPost, "/syncs", uuid.NewString(), syncBody).Code)
	for _, syncKey := range []string{key, uuid.NewString()} {
		syncDenied := httpCall("reader-org-a", http.MethodPost, "/syncs", syncKey, syncBody)
		require.Equal(t, http.StatusForbidden, syncDenied.Code, syncDenied.Body.String())
	}
	require.Equal(t, 2, calls)

	track := httpCall("org-a", http.MethodGet, statusPath+"/packages/fixture/track", "", "")
	require.Equal(t, 409, track.Code, track.Body.String())
	require.Contains(t, track.Body.String(), "UNSUPPORTED_APPLICATION")
	require.NotEqual(t, 200, httpCall("org-b", http.MethodGet, statusPath, "", "").Code)
	require.NotEqual(t, 200, httpCall("org-b", http.MethodGet, query, "", "").Code)
	other := complete(begin(second, uuid.NewString()), "org-b")
	require.Equal(t, "completed", other.Status)
	require.Contains(t, httpCall("org-b", http.MethodGet, "?platform=shopify&storeId="+second.StoreID, "", "").Body.String(), "Item org-b-order-fixture")

	partial = true
	failed := complete(begin(first, uuid.NewString()), "org-a")
	require.Equal(t, "partial", failed.Status)
	heads, _, err := repo.Heads(ctx, "org-a", o.Orders, []string{first.StoreID})
	require.NoError(t, err)
	require.Len(t, heads, 1)
	require.Equal(t, done.ID, heads[0].ID)
	partial = false

	late := begin(first, uuid.NewString())
	late, err = repo.ReadSync(ctx, "org-a", late.ID)
	require.NoError(t, err)
	grant(first, 2, false)
	_, err = repo.CommitPage(ctx, late, o.Checkpoint{Page: 1, ShopifyOrders: &o.ShopifyOrderProgress{Complete: true}}, nil, "completed", now)
	require.ErrorIs(t, err, o.ErrConflict)
	revoked, err := service.Step(ctx, "org-a", late.ID)
	require.NoError(t, err)
	require.Equal(t, "suspended", revoked.Status)
	require.NotEqual(t, 200, httpCall("org-a", http.MethodGet, query, "", "").Code)
	require.NotEqual(t, 200, httpCall("org-a", http.MethodGet, statusPath, "", "").Code)
	first = grant(first, 3, true)
	access.bindings[first.StoreID] = first
	require.NotEqual(t, 200, httpCall("org-a", http.MethodGet, statusPath, "", "").Code)
}
