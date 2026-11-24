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
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type amazonFixtureAccess struct {
	app.Authorization
	db       *gorm.DB
	bindings map[string]o.Binding
	reader   *wire.SyntheticOrderReader
}

func (a *amazonFixtureAccess) Open(ctx context.Context, scope o.Scope, storeID string, kind o.Kind, sync bool, expected *o.Binding) (o.Merchant, error) {
	if err := a.Authorize(ctx, scope, kind, sync); err != nil {
		return nil, err
	}
	binding, known := a.bindings[storeID]
	if !known || binding.OrganizationID != scope.OrganizationID {
		return nil, o.ErrForbidden
	}
	if !binding.ValidAmazonOrderApplication() {
		return nil, o.ErrUnsupported
	}
	if expected != nil && *expected != binding {
		return nil, o.ErrConflict
	}
	merchant := amazonFixtureMerchant{binding: binding, db: a.db, authorization: a.Authorization,
		subject: storecenter.ObservationSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID,
			MemberID: scope.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: sync}}
	if err := merchant.Check(ctx); err != nil {
		return nil, err
	}
	return merchant, nil
}

func (a *amazonFixtureAccess) ReadAmazonOrders(ctx context.Context, scope o.Scope, binding o.Binding, query o.AmazonOrderQuery) (o.AmazonOrderPage, error) {
	if _, err := a.Open(ctx, scope, binding.StoreID, o.Orders, true, &binding); err != nil {
		return o.AmazonOrderPage{}, err
	}
	return a.reader.Orders(ctx, binding.OrganizationID+"-fixture", query)
}

type amazonFixtureMerchant struct {
	o.Merchant
	binding       o.Binding
	db            *gorm.DB
	authorization app.Authorization
	subject       storecenter.ObservationSubject
}

func (m amazonFixtureMerchant) Binding() o.Binding { return m.binding }
func (m amazonFixtureMerchant) Check(ctx context.Context) error {
	proof, err := m.authorization.AuthorizeObservation(ctx, m.subject)
	if errors.Is(err, storecenter.ErrNotFound) {
		return o.ErrForbidden
	}
	if err != nil {
		return o.ErrUnavailable
	}
	if !proof.Allowed {
		return o.ErrForbidden
	}
	err = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		stores, err := storecenter.NewMemberScopedStoreRepository(tx, amazonFixtureStoreMember{access: proof.Access})
		if err != nil {
			return err
		}
		if err := stores.LockRead(ctx, m.binding.OrganizationID, []string{m.binding.StoreID}); err != nil {
			return err
		}
		return storecenter.LockAmazonOrderObservation(tx, m.binding)
	})
	if errors.Is(err, storecenter.ErrNotFound) {
		return o.ErrForbidden
	}
	if err != nil {
		return o.ErrUnavailable
	}
	return nil
}

type amazonFixtureStoreMember struct{ access storecenter.StoreMemberAccess }

func (a amazonFixtureStoreMember) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if a.access.OrganizationID != org {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return a.access, nil
}

type amazonFixtureDirectoryMember struct{}

func (amazonFixtureDirectoryMember) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	access, err := (orderFixtureMember{}).AuthorizeStoreMember(ctx, org)
	if access.MemberID == "order-member-org-a" {
		access.Administrator, access.CanWrite = false, false
	}
	return access, err
}

func TestSyntheticAmazonOrdersHTTPAndCurrentConsent(t *testing.T) {
	ctx, owner, repo := observationDatabase(t)
	installSyntheticOrderRecords(t, owner, "amazonOrder")
	pool, err := owner.DB()
	require.NoError(t, err)
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallAmazonAuthorizationSchemaTx(ctx, tx))
	require.NoError(t, storecenter.InstallIsolatedAmazonOrderConsentTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, owner.Exec(`GRANT SELECT,UPDATE ON workbench_stores,workbench_store_amazon_grants,
 workbench_store_amazon_order_consents,workbench_store_member_grants TO observation_runtime`).Error)

	now := time.Now().UTC().Truncate(time.Second)
	stores, err := storecenter.NewMemberScopedStoreRepository(owner, amazonFixtureDirectoryMember{})
	require.NoError(t, err)
	setMemberGrant := func(org, storeID, member string, active bool, version int64) {
		ctx := orderFixtureIdentity(ctx, org, now.Add(time.Hour))
		require.NoError(t, stores.SetMemberGrant(ctx, storecenter.MemberStoreGrantCommand{OrganizationID: org,
			StoreID: storeID, MemberID: member, OperationID: uuid.NewString(), Active: active, ExpectedVersion: version}))
	}
	newBinding := func(org string) o.Binding {
		subject := orderFixtureIdentity(ctx, org, now.Add(time.Hour))
		boundAt := now.Add(-time.Hour)
		id := uuid.NewString()
		store, err := storecenter.NewStore(storecenter.CreateStoreInput{ID: id, OrganizationID: org, ActorSubject: "actor-" + org,
			Name: "Amazon " + org, Platform: "shein", Region: "US", ExternalStoreID: id, CreateIdempotencyKey: uuid.NewString(), OccurredAt: now})
		require.NoError(t, err)
		store, _, err = stores.CreateOrReplay(subject, org, store)
		require.NoError(t, err)
		require.NoError(t, owner.Exec(`UPDATE workbench_stores SET platform='amazon',service_status='active',service_started_at=?,service_expires_at=?
 WHERE organization_id=? AND id=?`, now.Add(-time.Hour), now.Add(24*time.Hour), org, store.ID()).Error)
		require.NoError(t, owner.Exec(`INSERT INTO workbench_store_amazon_grants
 (organization_id,store_id,attempt_id,app_id,app_revision,beta,seller_id,generation,key_id,ciphertext,bound_at)
 VALUES(?,?,?,?,?,TRUE,?,1,'fixture-key','fixture-ciphertext',?)`, org, store.ID(), uuid.NewString(), "fixture-app", "fixture-v1", org+"-seller", boundAt).Error)
		setMemberGrant(org, store.ID(), "member-"+org, true, 0)
		return o.Binding{Platform: storecenter.PlatformAmazon, OrganizationID: org, StoreID: store.ID(), Site: "amazon-us",
			StoreVersion: store.Version(), ConnectionRevision: 1, ApplicationID: "fixture-app", ApplicationRevision: "fixture-v1",
			AmazonBeta: true, CredentialExpiresAt: boundAt.Add(365 * 24 * time.Hour)}
	}
	first, second := newBinding("org-a"), newBinding("org-b")
	grant := func(b o.Binding, generation int64, active bool) o.Binding {
		require.NoError(t, owner.Exec(`INSERT INTO workbench_store_amazon_order_consents VALUES(?,?,?,?,?) ON CONFLICT(organization_id,store_id)
 DO UPDATE SET authorization_generation=EXCLUDED.authorization_generation,order_generation=EXCLUDED.order_generation,granted=EXCLUDED.granted`,
			b.OrganizationID, b.StoreID, b.ConnectionRevision, generation, active).Error)
		b.AmazonOrderGeneration = generation
		return b
	}

	calls, repeatCursor := 0, false
	var afterProviderRequest func()
	const orderID = "111-1111111-1111111"
	reader, err := wire.NewSyntheticOrderReader(orderFixtureTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		if afterProviderRequest != nil {
			afterProviderRequest()
		}
		token := req.Header.Get("X-Amz-Access-Token")
		require.Contains(t, []string{"org-a-fixture", "org-b-fixture"}, token)
		q := req.URL.Query()
		require.Equal(t, now.Add(-30*24*time.Hour).Format(time.RFC3339), q.Get("createdAfter"))
		require.Equal(t, now.Add(-2*time.Minute).Format(time.RFC3339), q.Get("createdBefore"))
		require.Equal(t, wire.USMarketplace, q.Get("marketplaceIds"))
		order := map[string]any{"orderId": orderID, "createdTime": now.Add(-time.Hour), "lastUpdatedTime": now.Add(-time.Minute),
			"salesChannel": map[string]string{"marketplaceId": wire.USMarketplace},
			"orderItems": []map[string]any{{"orderItemId": "12345678", "quantityOrdered": 2,
				"product": map[string]string{"title": "Item " + token, "sellerSku": "US-CUP"}}}}
		page := map[string]any{"orders": []any{order}}
		if q.Get("paginationToken") == "" || repeatCursor {
			page["pagination"] = map[string]string{"nextToken": "second-page"}
		}
		raw, err := json.Marshal(page)
		require.NoError(t, err)
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(raw)))}, nil
	}))
	require.NoError(t, err)
	permissions, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	permissions.SetRolePolicyReader(orderFixtureRolePolicy{})
	access := &amazonFixtureAccess{
		Authorization: app.Authorization{Client: orderFixtureIAM{}, ProjectID: "fixture-project", Permissions: permissions,
			ServiceToken: func(context.Context) (string, error) { return "fixture-service", nil }},
		db: repo.db, bindings: map[string]o.Binding{first.StoreID: first, second.StoreID: second}, reader: reader,
	}
	service := &o.Service{Repository: repo, Access: access, AmazonOrders: access, Directory: app.Directory{Stores: stores, AmazonOrders: true},
		Starter: orderFixtureStarter{}, Now: func() time.Time { return now }}
	router := gin.New()
	for _, route := range routes.Routes(&app.Application{Service: service, Ready: func() bool { return true }}, func(ctx context.Context, bearer string) (context.Context, error) {
		switch bearer {
		case "Bearer org-a", "Bearer org-b", "Bearer store-member-org-a", "Bearer order-member-org-a":
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
	beginAs := func(account string, b o.Binding, key string) o.Sync {
		result := httpCall(account, http.MethodPost, "/syncs", key, `{"kind":"orders","stores":["`+b.StoreID+`"]}`)
		require.Equal(t, 200, result.Code, result.Body.String())
		var response struct{ Data o.Command }
		require.NoError(t, json.Unmarshal(result.Body.Bytes(), &response))
		require.Len(t, response.Data.Syncs, 1)
		return response.Data.Syncs[0]
	}
	begin := func(b o.Binding, key string) o.Sync {
		return beginAs(b.OrganizationID, b, key)
	}
	complete := func(sync o.Sync, org string) o.Sync {
		for i := 0; i < 3 && !sync.Terminal(); i++ {
			var err error
			sync, err = service.Step(ctx, org, sync.ID)
			require.NoError(t, err)
		}
		require.True(t, sync.Terminal())
		return sync
	}

	require.Equal(t, "suspended", begin(first, uuid.NewString()).Status)
	require.Zero(t, calls)
	first, second = grant(first, 1, true), grant(second, 1, true)
	access.bindings[first.StoreID], access.bindings[second.StoreID] = first, second
	key := uuid.NewString()
	sync := begin(first, key)
	require.Equal(t, now.Add(-2*time.Minute), sync.Range.End)
	require.Equal(t, sync.ID, begin(first, key).ID)
	done := complete(sync, "org-a")
	require.Equal(t, "completed", done.Status)
	require.Equal(t, done.ID, begin(first, key).ID)
	require.Equal(t, 2, calls)

	query := "?platform=amazon&storeId=" + first.StoreID
	detailPath := "/stores/" + first.StoreID + "/syncs/" + done.ID + "/records/" + orderID
	list := httpCall("org-a", http.MethodGet, query, "", "")
	require.Equal(t, 200, list.Code, list.Body.String())
	var response struct{ Data o.Result }
	require.NoError(t, json.Unmarshal(list.Body.Bytes(), &response))
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, orderID, response.Data.Items[0].AmazonOrder.ID)
	require.Empty(t, response.Data.Items[0].AmazonOrder.Status)
	require.NotContains(t, list.Body.String(), "second-page")
	filtered := httpCall("org-a", http.MethodGet, query+"&status=unknown", "", "")
	require.Equal(t, 200, filtered.Code, filtered.Body.String())
	require.Contains(t, filtered.Body.String(), orderID)
	detail := httpCall("org-a", http.MethodGet, detailPath, "", "")
	require.Equal(t, 200, detail.Code, detail.Body.String())
	require.Contains(t, detail.Body.String(), `"quantity":2`)
	require.NotEqual(t, 200, httpCall("org-b", http.MethodGet, query, "", "").Code)
	require.NotEqual(t, 200, httpCall("org-b", http.MethodGet, detailPath, "", "").Code)
	require.Equal(t, 403, httpCall("store-member-org-a", http.MethodGet, query, "", "").Code)
	require.NotEqual(t, 200, httpCall("order-member-org-a", http.MethodGet, query, "", "").Code)
	require.NotEqual(t, 200, httpCall("order-member-org-a", http.MethodGet, detailPath, "", "").Code)
	require.NotEqual(t, 200, httpCall("order-member-org-a", http.MethodGet, "/syncs/"+done.ID, "", "").Code)
	require.NotEqual(t, 200, httpCall("order-member-org-a", http.MethodPost, "/syncs", uuid.NewString(),
		`{"kind":"orders","stores":["`+first.StoreID+`"]}`).Code)
	const orderMember = "order-member-org-a"
	setMemberGrant("org-a", first.StoreID, orderMember, true, 0)
	require.Equal(t, 200, httpCall(orderMember, http.MethodGet, detailPath, "", "").Code)
	memberSync := beginAs(orderMember, first, uuid.NewString())
	memberSync, err = service.Step(ctx, "org-a", memberSync.ID)
	require.NoError(t, err)
	require.Equal(t, "running", memberSync.Status)
	setMemberGrant("org-a", first.StoreID, orderMember, false, 1)
	beforeRevoked := calls
	memberSync, err = service.Step(ctx, "org-a", memberSync.ID)
	require.NoError(t, err)
	require.Equal(t, "suspended", memberSync.Status)
	require.Equal(t, beforeRevoked, calls)
	require.NotEqual(t, 200, httpCall(orderMember, http.MethodGet, detailPath, "", "").Code)

	setMemberGrant("org-a", first.StoreID, orderMember, true, 2)
	inFlight := beginAs(orderMember, first, uuid.NewString())
	afterProviderRequest = func() {
		setMemberGrant("org-a", first.StoreID, orderMember, false, 3)
	}
	inFlight, err = service.Step(ctx, "org-a", inFlight.ID)
	afterProviderRequest = nil
	require.NoError(t, err)
	require.Equal(t, "suspended", inFlight.Status)
	require.Zero(t, inFlight.Progress.Pages)

	other := complete(begin(second, uuid.NewString()), "org-b")
	require.Equal(t, "completed", other.Status)
	require.Contains(t, httpCall("org-b", http.MethodGet, "?platform=amazon&storeId="+second.StoreID, "", "").Body.String(), "Item org-b-fixture")

	repeatCursor = true
	partial := complete(begin(first, uuid.NewString()), "org-a")
	require.Equal(t, "partial", partial.Status)
	require.Contains(t, partial.Progress.Notes, "provider_cursor_cycle")
	heads, _, err := repo.Heads(ctx, "org-a", o.Orders, []string{first.StoreID})
	require.NoError(t, err)
	require.Len(t, heads, 1)
	require.Equal(t, done.ID, heads[0].ID)

	repeatCursor = false
	expired := begin(first, uuid.NewString())
	expired, err = service.Step(ctx, "org-a", expired.ID)
	require.NoError(t, err)
	require.Equal(t, "running", expired.Status)
	beforeExpired := calls
	service.Now = func() time.Time { return now.Add(24 * time.Hour) }
	expired, err = service.Step(ctx, "org-a", expired.ID)
	service.Now = func() time.Time { return now }
	require.NoError(t, err)
	require.Equal(t, "partial", expired.Status)
	require.Contains(t, expired.Progress.Notes, "duration_limit")
	require.Equal(t, beforeExpired, calls)

	late := begin(first, uuid.NewString())
	late, err = repo.ReadSync(ctx, "org-a", late.ID)
	require.NoError(t, err)
	grant(first, 2, false)
	_, err = repo.CommitPage(ctx, late, o.Checkpoint{Page: 1, AmazonOrders: &o.AmazonOrderProgress{Complete: true}}, nil, "completed", now)
	require.ErrorIs(t, err, o.ErrConflict)
	revoked, err := service.Step(ctx, "org-a", late.ID)
	require.NoError(t, err)
	require.Equal(t, "suspended", revoked.Status)
	require.NotEqual(t, 200, httpCall("org-a", http.MethodGet, detailPath, "", "").Code)
	first = grant(first, 3, true)
	access.bindings[first.StoreID] = first
	require.NotEqual(t, 200, httpCall("org-a", http.MethodGet, detailPath, "", "").Code)

	// The current Amazon grant can advance in place while the old order consent
	// remains present. The old consent can never authorize the reconnected source.
	renewal := begin(first, uuid.NewString())
	renewal, err = repo.ReadSync(ctx, "org-a", renewal.ID)
	require.NoError(t, err)
	require.NoError(t, owner.Exec(`UPDATE workbench_store_amazon_grants SET generation=2 WHERE organization_id=? AND store_id=?`, first.OrganizationID, first.StoreID).Error)
	_, err = repo.CommitPage(ctx, renewal, o.Checkpoint{Page: 1, AmazonOrders: &o.AmazonOrderProgress{Complete: true}}, nil, "completed", now)
	require.ErrorIs(t, err, o.ErrConflict)
}
