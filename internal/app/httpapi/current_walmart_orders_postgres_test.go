//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/stretchr/testify/require"
	testcontainers "github.com/testcontainers/testcontainers-go"
	tcpg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	storeapp "task-processor/internal/app/storecenter"
	obshttp "task-processor/internal/app/storeobservations/httpapi"
	obspg "task-processor/internal/integration/persistence/storeobservations"
	walmart "task-processor/internal/integration/walmart"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/walmart"
)

type walmartOrderReceipt struct {
	id, customer, amount string
	date                 time.Time
}

// All requests terminate in this transport. There is no network fallback.
type walmartOrderFixture struct {
	scope     o.Scope
	materials map[string]storecenter.ProductExecutionMaterial
	rows      map[string][]walmartOrderReceipt
	access    *storeapp.OfficialObservationAccess
	mu        sync.Mutex
	calls     int
}

func newWalmartOrderFixture(t *testing.T, scope o.Scope, stores []string, today time.Time) *walmartOrderFixture {
	t.Helper()
	f := &walmartOrderFixture{scope: scope, materials: map[string]storecenter.ProductExecutionMaterial{}, rows: map[string][]walmartOrderReceipt{}}
	app := auth.Application{ClientID: "offline-walmart-app", Revision: "offline-v1", Environment: "sandbox", CallbackURL: "https://example.invalid/walmart-callback"}
	provider, e := walmart.NewOfficialClient(walmart.Options{Application: app, ClientSecret: "synthetic-app-secret", HTTPClient: &http.Client{Transport: f}})
	require.NoError(t, e)
	protection, e := walmart.NewCredentialProtection("offline-walmart-key", bytes.Repeat([]byte{8}, 32))
	require.NoError(t, e)
	for _, store := range stores {
		g := storecenter.WalmartGrant{OrganizationID: scope.OrganizationID, StoreID: store, ID: uuid.NewString(), AppID: app.ClientID, AppRevision: app.Revision, Environment: app.Environment, SellerID: store, MerchantIdentityHash: storecenter.WalmartMerchantIdentity(app.Environment, store), ScopesHash: marketplace.Hash(map[string]string{"orders": "view_only"}), Generation: 1, Version: 1, Status: "active", AccessExpiresAt: time.Now().Add(24 * time.Hour), RefreshExpiresAt: time.Now().Add(24 * time.Hour)}
		g.KeyID, g.Ciphertext, e = protection.Seal(g.CredentialBinding(), auth.TokenSet{AccessToken: "synthetic-access", RefreshToken: "synthetic-refresh", SellerID: store, AccessExpiresAt: g.AccessExpiresAt, RefreshExpiresAt: g.RefreshExpiresAt})
		require.NoError(t, e)
		f.materials[store] = storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformWalmart, StoreVersion: 1, ServiceExpiresAt: time.Now().Add(24 * time.Hour), Walmart: &g, Connection: storecenter.OfficialConnectionView{AttemptID: uuid.NewString(), Version: 1, Status: storecenter.ConnectionStatusConnected}}
	}
	registry, e := storeapp.WithWalmartApplication(nil, provider, protection)
	require.NoError(t, e)
	f.access, e = storeapp.NewOfficialObservationAccess(f, f, registry)
	require.NoError(t, e)
	if len(stores) >= 2 {
		recent := today.AddDate(0, 0, -1).Add(12 * time.Hour)
		f.rows[stores[0]] = []walmartOrderReceipt{{"PO-A-1", "CUSTOMER-A", "10.00", recent}, {"PO-A-2", "CUSTOMER-A", "20.00", today.Add(-time.Millisecond)}, {"PO-A-OLD", "CUSTOMER-OLD", "5.00", today.AddDate(0, 0, -10).Add(12 * time.Hour)}}
		f.rows[stores[1]] = []walmartOrderReceipt{{"PO-B-1", "CUSTOMER-B", "7.25", recent}, {"PO-B-MISSING", "CUSTOMER-MISSING", "", today.AddDate(0, 0, -3).Add(12 * time.Hour)}}
	}
	return f
}
func (f *walmartOrderFixture) Authorize(_ context.Context, s o.Scope, k o.Kind, _ bool) error {
	if s != f.scope || k != o.Orders {
		return o.ErrForbidden
	}
	return nil
}
func (f *walmartOrderFixture) AuthorizeObservation(_ context.Context, s storecenter.ObservationSubject) (storecenter.ObservationAuthorization, error) {
	if s.OrganizationID != f.scope.OrganizationID || s.ActorID != f.scope.ActorID || s.MemberID != f.scope.MemberID || s.Purpose != storecenter.ObservationPurposeOrders {
		return storecenter.ObservationAuthorization{}, o.ErrForbidden
	}
	return storecenter.ObservationAuthorization{}, nil
}
func (f *walmartOrderFixture) ReadObservation(ctx context.Context, s storecenter.ObservationSubject, store string, a storecenter.ObservationAuthorizer, _ time.Time) (storecenter.ProductExecutionMaterial, error) {
	if _, e := a.AuthorizeObservation(ctx, s); e != nil {
		return storecenter.ProductExecutionMaterial{}, e
	}
	m, ok := f.materials[store]
	if !ok {
		return m, storecenter.ErrNotFound
	}
	return m, nil
}
func (f *walmartOrderFixture) ListStores(_ context.Context, s o.Scope, k o.Kind, platform string) ([]string, error) {
	if s != f.scope || k != o.Orders {
		return nil, o.ErrForbidden
	}
	ids := []string{}
	if platform == "" || platform == "walmart" {
		for id := range f.materials {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (f *walmartOrderFixture) Open(ctx context.Context, s o.Scope, id string, k o.Kind, sync bool, b *o.Binding) (o.Merchant, error) {
	if e := f.Authorize(ctx, s, k, sync); e != nil {
		return nil, e
	}
	return f.access.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: sync}, id, b)
}
func (f *walmartOrderFixture) ReadSavedWalmartOrderBinding(ctx context.Context, s o.Scope, id string) (o.Binding, error) {
	if e := f.Authorize(ctx, s, o.Orders, false); e != nil {
		return o.Binding{}, e
	}
	return f.access.ReadSavedWalmartOrderBinding(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders}, id)
}
func (f *walmartOrderFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	store := r.Header.Get("WM_PARTNER.ID")
	if r.Method != "GET" || r.URL.Host != "sandbox.walmartapis.com" || f.materials[store].Walmart == nil || r.Header.Get("WM_SEC.ACCESS_TOKEN") != "synthetic-access" {
		return nil, fmt.Errorf("forbidden offline Walmart request")
	}
	var payload any
	switch r.URL.Path {
	case "/v3/token/detail":
		payload = map[string]any{"is_valid": true, "expire_at": time.Now().Add(time.Hour).UnixMilli(), "scopes": map[string]string{"orders": "view_only"}}
	case "/v3/orders":
		q := r.URL.Query()
		start, e := time.Parse(time.RFC3339, q.Get("createdStartDate"))
		if e != nil {
			return nil, e
		}
		end, e := time.Parse(time.RFC3339, q.Get("createdEndDate"))
		if e != nil {
			return nil, e
		}
		if q.Get("limit") != "30" || q.Get("replacementInfo") != "true" || q.Has("status") {
			return nil, fmt.Errorf("incorrect Walmart query")
		}
		rows := []map[string]any{}
		if q.Get("shipNodeType") == "SellerFulfilled" {
			for _, row := range f.rows[store] {
				if !row.date.After(start) || !row.date.Before(end) {
					continue
				}
				charges := []map[string]any{}
				if row.amount != "" {
					charges = append(charges, map[string]any{"chargeType": "PRODUCT", "chargeAmount": map[string]any{"currency": "USD", "amount": row.amount}})
				}
				rows = append(rows, map[string]any{"purchaseOrderId": row.id, "customerOrderId": row.customer, "orderDate": row.date.UnixMilli(), "customerEmailId": "private@example.invalid", "orderLines": map[string]any{"orderLine": []map[string]any{{"lineNumber": "1", "orderLineQuantity": map[string]any{"amount": "3"}, "charges": map[string]any{"charge": charges}}}}})
			}
		}
		payload = map[string]any{"list": map[string]any{"meta": map[string]any{"totalCount": len(rows)}, "elements": map[string]any{"order": rows}}}
	default:
		return nil, fmt.Errorf("forbidden offline Walmart path")
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(raw)), Request: r}, nil
}
func TestWalmartManualOrdersToSavedAmountsPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, e := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("walmart_orders"), tcpg.WithUsername("installer"), tcpg.WithPassword(uuid.NewString()), tcpg.BasicWaitStrategies(), testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
		c.PortBindings = network.PortMap{network.MustParsePort("5432/tcp"): []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"}}}
	}))
	require.NoError(t, e)
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })
	dsn, e := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	open := func(dsn string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	owner := open(dsn)
	require.NoError(t, obspg.Install(ctx, owner))
	password := uuid.NewString()
	require.NoError(t, owner.Exec("CREATE ROLE walmart_runtime LOGIN PASSWORD '"+password+"' NOSUPERUSER NOCREATEDB NOCREATEROLE").Error)
	require.NoError(t, obspg.GrantRuntime(ctx, owner, "walmart_runtime"))
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.User = url.UserPassword("walmart_runtime", password)
	repo, e := obspg.NewRepository(ctx, open(u.String()))
	require.NoError(t, e)
	zone := time.FixedZone("UTC+8", 8*3600)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	scope := o.Scope{OrganizationID: "walmart-org", ActorID: "walmart-actor", MemberID: "walmart-member"}
	a, b := uuid.NewString(), uuid.NewString()
	f := newWalmartOrderFixture(t, scope, []string{a, b}, today)
	service := &o.Service{Repository: repo, Access: f, Directory: f, WalmartOrders: f.access}
	service.Starter = semiOrderWorker{service}
	router := sheinOrderFixtureHTTP(service, scope)
	begin := func(store string, days int) {
		start, end := today.AddDate(0, 0, -days), today
		in := o.BeginInput{Kind: o.Orders, Stores: []string{store}, Start: &start, End: &end}
		raw, e := json.Marshal(in)
		require.NoError(t, e)
		key := uuid.NewString()
		var original string
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", obshttp.BasePath+"/orders/syncs", bytes.NewReader(raw))
			req.Header.Set("Idempotency-Key", key)
			req.Header.Set("Content-Type", "application/json")
			out := httptest.NewRecorder()
			router.ServeHTTP(out, req)
			require.Equal(t, 200, out.Code, out.Body.String())
			var envelope struct {
				Data o.Command `json:"data"`
			}
			require.NoError(t, json.Unmarshal(out.Body.Bytes(), &envelope))
			require.Len(t, envelope.Data.Syncs, 1)
			status, err := service.Status(ctx, scope, envelope.Data.Syncs[0].ID)
			require.NoError(t, err)
			require.Equal(t, "completed", status.Status)
			if i == 0 {
				original = envelope.Data.Syncs[0].ID
			} else {
				require.Equal(t, original, envelope.Data.Syncs[0].ID)
			}
		}
	}
	begin(a, 30)
	begin(b, 7)
	calls := f.calls
	for _, test := range []struct {
		store          string
		days           int
		status, amount string
		count          int
	}{{a, 7, "complete", "30.00", 2}, {a, 30, "complete", "35.00", 3}, {a, 1, "complete", "30.00", 2}, {b, 7, "missing_fields", "", 2}, {b, 30, "coverage_incomplete", "", 0}, {b, 1, "complete", "7.25", 1}} {
		value, e := service.SavedWalmartOrderAmounts(ctx, scope, o.OrderAmountQuery{StoreID: test.store, Start: today.AddDate(0, 0, -test.days), End: today})
		require.NoError(t, e)
		require.Equal(t, test.status, value.Status)
		if test.amount != "" {
			require.NotNil(t, value.Amounts)
			require.Equal(t, test.amount, (*value.Amounts)[0].Amount)
		} else {
			require.Nil(t, value.Amounts)
		}
		if test.status != "coverage_incomplete" {
			require.Equal(t, test.count, *value.MatchedOrderCount)
		}
	}
	require.Equal(t, calls, f.calls, "saved dashboard reads must remain local-only")
	var count int64
	require.NoError(t, owner.Raw("SELECT count(*) FROM store_observations.records").Scan(&count).Error)
	require.Equal(t, int64(5), count)
}
