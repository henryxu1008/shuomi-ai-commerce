//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
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
	tiktok "task-processor/internal/integration/tiktokshop"

	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/tiktokshop"
)

type tiktokOrderReceipt struct {
	id, customer, amount string
	date                 time.Time
}

// All requests terminate in this transport. There is no network fallback.
type tiktokOrderFixture struct {
	scope     o.Scope
	materials map[string]storecenter.ProductExecutionMaterial
	rows      map[string][]tiktokOrderReceipt
	access    *storeapp.OfficialObservationAccess
	mu        sync.Mutex
	calls     int
}

func newTikTokOrderFixture(t *testing.T, scope o.Scope, stores []string, today time.Time) *tiktokOrderFixture {
	t.Helper()
	f := &tiktokOrderFixture{scope: scope, materials: map[string]storecenter.ProductExecutionMaterial{}, rows: map[string][]tiktokOrderReceipt{}}
	app := auth.Application{AppKey: "offline-tiktok-app", ServiceID: "offline-service", Revision: "offline-v1", CallbackURL: "https://example.invalid/tiktok-callback"}
	provider, e := tiktok.NewOfficialClient(tiktok.OfficialClientOptions{Application: app, AppSecret: "synthetic-app-secret", HTTPClient: &http.Client{Transport: f}})
	require.NoError(t, e)
	protection, e := tiktok.NewCredentialProtection("offline-tiktok-key", bytes.Repeat([]byte{9}, 32))
	require.NoError(t, e)
	for _, store := range stores {
		now := time.Now().UTC()
		identity := sha256.Sum256([]byte("seller-" + store))
		g := auth.GrantSnapshot{OrganizationID: scope.OrganizationID, ID: uuid.NewString(), AppKey: app.AppKey, AppRevision: app.Revision, SellerIdentityHash: hex.EncodeToString(identity[:]), Generation: 1, Revision: 1, Status: auth.GrantActive, AccessExpiresAt: now.Add(24 * time.Hour), RefreshExpiresAt: now.Add(48 * time.Hour), CreatedAt: now, UpdatedAt: now}
		g.KeyID, g.Ciphertext, e = protection.Seal(auth.CredentialBinding{OrganizationID: g.OrganizationID, AppKey: g.AppKey, AppRevision: g.AppRevision, ID: g.ID, Purpose: "grant", Generation: g.Generation}, auth.TokenSet{AccessToken: "synthetic-" + store, RefreshToken: "synthetic-refresh", OpenID: "seller-" + store, UserType: 0, GrantedScopes: []string{"seller.authorization.info", "seller.order.info"}, AccessExpiresAt: g.AccessExpiresAt, RefreshExpiresAt: g.RefreshExpiresAt})
		require.NoError(t, e)
		f.materials[store] = storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformTikTokShop, StoreVersion: 1, ServiceExpiresAt: now.Add(24 * time.Hour), Connection: storecenter.OfficialConnectionView{AttemptID: uuid.NewString(), Version: 1, Status: storecenter.ConnectionStatusConnected}, TikTok: &storecenter.TikTokProductExecutionMaterial{Grant: g, Binding: storecenter.TikTokShopBinding{OrganizationID: scope.OrganizationID, StoreID: store, GrantID: g.ID, AppKey: app.AppKey, Shop: auth.Shop{ID: "shop-" + store, Code: "CODE-" + store, Cipher: "old-" + store, Region: "US", SellerType: "CROSS_BORDER"}, Status: "active", Version: 1}}}
	}
	registry, e := storeapp.WithTikTokApplication(nil, provider, protection)
	require.NoError(t, e)
	f.access, e = storeapp.NewOfficialObservationAccess(f, f, registry)
	require.NoError(t, e)
	if len(stores) >= 2 {
		recent := today.AddDate(0, 0, -1).Add(12 * time.Hour)
		f.rows[stores[0]] = []tiktokOrderReceipt{{"ORDER-A-1", "CUSTOMER-A", "10.00", recent}, {"ORDER-A-2", "CUSTOMER-A", "20.00", today.Add(-time.Second)}, {"ORDER-A-OLD", "CUSTOMER-OLD", "5.00", today.AddDate(0, 0, -10).Add(12 * time.Hour)}}
		f.rows[stores[1]] = []tiktokOrderReceipt{{"ORDER-B-1", "CUSTOMER-B", "7.25", recent}, {"ORDER-B-MISSING", "CUSTOMER-MISSING", "", today.AddDate(0, 0, -3).Add(12 * time.Hour)}}
	}
	return f
}
func (f *tiktokOrderFixture) Authorize(_ context.Context, s o.Scope, k o.Kind, _ bool) error {
	if s != f.scope || k != o.Orders {
		return o.ErrForbidden
	}
	return nil
}
func (f *tiktokOrderFixture) AuthorizeObservation(_ context.Context, s storecenter.ObservationSubject) (storecenter.ObservationAuthorization, error) {
	if s.OrganizationID != f.scope.OrganizationID || s.ActorID != f.scope.ActorID || s.MemberID != f.scope.MemberID || s.Purpose != storecenter.ObservationPurposeOrders {
		return storecenter.ObservationAuthorization{}, o.ErrForbidden
	}
	return storecenter.ObservationAuthorization{}, nil
}
func (f *tiktokOrderFixture) ReadObservation(ctx context.Context, s storecenter.ObservationSubject, store string, a storecenter.ObservationAuthorizer, _ time.Time) (storecenter.ProductExecutionMaterial, error) {
	if _, e := a.AuthorizeObservation(ctx, s); e != nil {
		return storecenter.ProductExecutionMaterial{}, e
	}
	m, ok := f.materials[store]
	if !ok {
		return m, storecenter.ErrNotFound
	}
	return m, nil
}
func (f *tiktokOrderFixture) ListStores(_ context.Context, s o.Scope, k o.Kind, platform string) ([]string, error) {
	if s != f.scope || k != o.Orders {
		return nil, o.ErrForbidden
	}
	ids := []string{}
	if platform == "" || platform == "tiktokshop" {
		for id := range f.materials {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (f *tiktokOrderFixture) Open(ctx context.Context, s o.Scope, id string, k o.Kind, sync bool, b *o.Binding) (o.Merchant, error) {
	if e := f.Authorize(ctx, s, k, sync); e != nil {
		return nil, e
	}
	return f.access.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: sync}, id, b)
}
func (f *tiktokOrderFixture) ReadSavedTikTokOrderBinding(ctx context.Context, s o.Scope, id string) (o.Binding, error) {
	if e := f.Authorize(ctx, s, o.Orders, false); e != nil {
		return o.Binding{}, e
	}
	return f.access.ReadSavedTikTokOrderBinding(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders}, id)
}
func (f *tiktokOrderFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	store := strings.TrimPrefix(r.Header.Get("x-tts-access-token"), "synthetic-")
	if r.URL.Host != "open-api.tiktokglobalshop.com" || f.materials[store].TikTok == nil {
		return nil, fmt.Errorf("forbidden offline TikTok request")
	}
	var payload any
	switch r.URL.Path {
	case "/authorization/202309/shops":
		if r.Method != "GET" {
			return nil, fmt.Errorf("unexpected shop method")
		}
		payload = map[string]any{"code": 0, "data": map[string]any{"shops": []map[string]any{{"id": "shop-" + store, "code": "CODE-" + store, "region": "US", "seller_type": "CROSS_BORDER", "cipher": "fresh-" + store}}}}
	case "/order/202309/orders/search":
		q := r.URL.Query()
		if r.Method != "POST" || q.Get("shop_cipher") != "fresh-"+store || q.Get("page_size") != "100" || q.Get("sort_field") != "create_time" || q.Get("sort_order") != "ASC" {
			return nil, fmt.Errorf("incorrect TikTok query")
		}
		var query map[string]int64
		if e := json.NewDecoder(r.Body).Decode(&query); e != nil || len(query) != 2 {
			return nil, fmt.Errorf("incorrect TikTok window body")
		}
		rows := []map[string]any{}
		for _, row := range f.rows[store] {
			if row.date.Unix() < query["create_time_ge"] || row.date.Unix() >= query["create_time_lt"] {
				continue
			}
			payment := map[string]any{}
			if row.amount != "" {
				payment = map[string]any{"currency": "USD", "sub_total": row.amount, "total_amount": "99999.99"}
			}
			rows = append(rows, map[string]any{"id": row.id, "create_time": row.date.Unix(), "status": "COMPLETED", "payment": payment, "buyer_email": "private@example.invalid", "line_items": []map[string]any{{"id": "line-1", "quantity": 3}, {"id": "line-2", "quantity": 2}}, "packages": []map[string]any{{"id": "package-1"}, {"id": "package-2"}}})
		}
		total := len(rows)
		next := ""
		expected := "next?&=" + strconv.FormatInt(query["create_time_ge"], 10)
		if q.Get("page_token") == "" {
			if total > 1 {
				next = expected
				rows = rows[:1]
			}
		} else {
			if q.Get("page_token") != expected || total < 2 {
				return nil, fmt.Errorf("wrong original page token")
			}
			rows = rows[1:]
		}
		payload = map[string]any{"code": 0, "data": map[string]any{"orders": rows, "total_count": total, "next_page_token": next}}
	default:
		return nil, fmt.Errorf("forbidden offline TikTok path")
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(raw)), Request: r}, nil
}
func TestTikTokManualOrdersToSavedAmountsPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, e := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("tiktok_orders"), tcpg.WithUsername("installer"), tcpg.WithPassword(uuid.NewString()), tcpg.BasicWaitStrategies(), testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
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
	require.NoError(t, owner.Exec("CREATE ROLE tiktok_runtime LOGIN PASSWORD '"+password+"' NOSUPERUSER NOCREATEDB NOCREATEROLE").Error)
	require.NoError(t, obspg.GrantRuntime(ctx, owner, "tiktok_runtime"))
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.User = url.UserPassword("tiktok_runtime", password)
	repo, e := obspg.NewRepository(ctx, open(u.String()))
	require.NoError(t, e)
	zone := time.FixedZone("UTC+8", 8*3600)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	scope := o.Scope{OrganizationID: "tiktok-org", ActorID: "tiktok-actor", MemberID: "tiktok-member"}
	a, b := uuid.NewString(), uuid.NewString()
	f := newTikTokOrderFixture(t, scope, []string{a, b}, today)
	service := &o.Service{Repository: repo, Access: f, Directory: f, TikTokOrders: f.access}
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
		value, e := service.SavedTikTokOrderAmounts(ctx, scope, o.OrderAmountQuery{StoreID: test.store, Start: today.AddDate(0, 0, -test.days), End: today})
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
