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
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
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
	obsapp "task-processor/internal/app/storeobservations"
	obshttp "task-processor/internal/app/storeobservations/httpapi"
	"task-processor/internal/authidentity"
	obspg "task-processor/internal/integration/persistence/storeobservations"
	"task-processor/internal/integration/shein"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

// Only IAM/Store material and the platform transport are synthetic. The real
// application registry, credential protection, read-only handle, parser, sync
// service and PostgreSQL transaction/head/amount reader run without substitutes.
type semiOrderFixture struct {
	scope     o.Scope
	access    *storeapp.OfficialObservationAccess
	materials map[string]storecenter.ProductExecutionMaterial
	mu        sync.Mutex
	rows      map[string][]semiOrderReceipt
	calls     int
}

type semiOrderReceipt struct {
	id, amount, currency string
	issued               time.Time
}

func newSemiOrderFixture(t *testing.T, scope o.Scope, stores []string) *semiOrderFixture {
	t.Helper()
	f := &semiOrderFixture{scope: scope, materials: map[string]storecenter.ProductExecutionMaterial{}, rows: map[string][]semiOrderReceipt{}}
	app := storecenter.OfficialApplication{Platform: storecenter.PlatformShein, AuthorizationMethod: storecenter.AuthorizationCallback, AppID: "offline-semi-app", Version: storeapp.BoundOfficialRevision("offline-v1", storecenter.ApplicationSemiManaged), CallbackURL: "https://example.invalid/shein-callback"}
	provider, err := shein.NewOfficialClient(shein.OfficialClientOptions{Application: app, AppSecret: "synthetic-app-secret-never-sent", APIOrigin: "https://openapi.sheincorp.com", HTTPClient: &http.Client{Transport: f}})
	require.NoError(t, err)
	protection, err := shein.NewCredentialProtection("offline-semi-key", bytes.Repeat([]byte{7}, 32))
	require.NoError(t, err)
	for _, store := range stores {
		attempt := storecenter.OfficialConnectionAttempt{OrganizationID: scope.OrganizationID, StoreID: store, AttemptID: uuid.NewString(), ActorID: scope.ActorID, MemberID: scope.MemberID, AppID: app.AppID, AppVersion: app.Version, ConnectionVersion: 1, State: "verified"}
		attempt.KeyID, attempt.Ciphertext, err = protection.Seal(attempt, storecenter.OfficialMerchantCredential{AppID: app.AppID, SupplierID: "123", OpenKeyID: store, SecretKey: "synthetic-merchant-secret"})
		require.NoError(t, err)
		f.materials[store] = storecenter.ProductExecutionMaterial{Platform: storecenter.PlatformShein, StoreVersion: 1, ServiceExpiresAt: time.Now().Add(24 * time.Hour), Attempt: attempt, Connection: storecenter.OfficialConnectionView{AttemptID: attempt.AttemptID, Version: 1, Status: storecenter.ConnectionStatusConnected}}
	}
	registry, err := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{{Provider: provider, Protection: protection, Type: storecenter.ApplicationSemiManaged}})
	require.NoError(t, err)
	f.access, err = storeapp.NewOfficialObservationAccess(f, f, registry)
	require.NoError(t, err)
	return f
}

func (f *semiOrderFixture) Authorize(_ context.Context, s o.Scope, k o.Kind, _ bool) error {
	if s != f.scope || k != o.Orders {
		return o.ErrForbidden
	}
	return nil
}
func (f *semiOrderFixture) AuthorizeObservation(_ context.Context, s storecenter.ObservationSubject) (storecenter.ObservationAuthorization, error) {
	if s.OrganizationID != f.scope.OrganizationID || s.ActorID != f.scope.ActorID || s.MemberID != f.scope.MemberID || s.Purpose != storecenter.ObservationPurposeOrders {
		return storecenter.ObservationAuthorization{}, o.ErrForbidden
	}
	return storecenter.ObservationAuthorization{}, nil
}
func (f *semiOrderFixture) ReadObservation(ctx context.Context, s storecenter.ObservationSubject, store string, a storecenter.ObservationAuthorizer, _ time.Time) (storecenter.ProductExecutionMaterial, error) {
	if _, err := a.AuthorizeObservation(ctx, s); err != nil {
		return storecenter.ProductExecutionMaterial{}, err
	}
	m, ok := f.materials[store]
	if !ok {
		return m, storecenter.ErrNotFound
	}
	return m, nil
}
func (f *semiOrderFixture) Open(ctx context.Context, s o.Scope, store string, k o.Kind, write bool, expected *o.Binding) (o.Merchant, error) {
	if err := f.Authorize(ctx, s, k, write); err != nil {
		return nil, err
	}
	return f.access.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: write}, store, expected)
}
func (f *semiOrderFixture) ListStores(ctx context.Context, s o.Scope, k o.Kind, platform string) ([]string, error) {
	if err := f.Authorize(ctx, s, k, false); err != nil {
		return nil, err
	}
	if platform != "" && platform != "shein" {
		return nil, o.ErrInvalid
	}
	out := []string{}
	for id := range f.materials {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}
func (f *semiOrderFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	// There is deliberately no fallback transport, DNS lookup or socket dial.
	if r.URL.Host != "openapi.sheincorp.com" || r.Method != http.MethodPost || f.materials[r.Header.Get("x-lt-openKeyId")].Attempt.AppID == "" {
		return nil, fmt.Errorf("unexpected offline SHEIN request")
	}
	rows := f.rows[r.Header.Get("x-lt-openKeyId")]
	zone := time.FixedZone("UTC+8", 8*3600)
	var info any
	switch r.URL.Path {
	case "/open-api/order/order-list":
		var q struct {
			Type      int    `json:"queryType"`
			Warehouse int    `json:"queryOrderType"`
			Start     string `json:"startTime"`
			End       string `json:"endTime"`
			Page      int    `json:"page"`
			Size      int    `json:"pageSize"`
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		start, e1 := time.ParseInLocation("2006-01-02 15:04:05", q.Start, zone)
		end, e2 := time.ParseInLocation("2006-01-02 15:04:05", q.End, zone)
		if e1 != nil || e2 != nil || q.Type != 1 || q.Warehouse != 4 || q.Page != 1 || q.Size != 30 || end.Sub(start) > 48*time.Hour {
			return nil, fmt.Errorf("wrong SHEIN sync window or query")
		}
		list := []map[string]any{}
		for _, row := range rows {
			if !row.issued.Before(start) && !row.issued.After(end) {
				stamp := row.issued.In(zone).Format("2006-01-02 15:04:05")
				list = append(list, map[string]any{"orderNo": row.id, "orderStatus": 2, "orderCreateTime": stamp, "orderUpdateTime": stamp})
			}
		}
		info = map[string]any{"count": len(list), "orderList": list}
	case "/open-api/order/order-detail":
		var q struct {
			IDs []string `json:"orderNoList"`
		}
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			return nil, err
		}
		list := []map[string]any{}
		for _, id := range q.IDs {
			for _, row := range rows {
				if row.id != id {
					continue
				}
				detail := map[string]any{"orderNo": id, "salesSite": "shein-us", "orderStatus": 2, "stockMode": 1, "orderType": 1, "orderTime": row.issued.AddDate(0, 0, -20).Format("2006-01-02T15:04:05.000-0700"), "orderCurrency": row.currency, "saleCurrency": "USD", "totalCostPrice": "999.00", "orderGoodsInfoList": []map[string]any{{"goodsId": "101", "skuCode": "sku-a"}, {"goodsId": "102", "skuCode": "sku-b"}}, "packageWaybillList": []map[string]any{{"packageNo": "package-a"}, {"packageNo": "package-b"}}}
				if row.amount != "" {
					detail["productTotalPrice"] = row.amount
				}
				list = append(list, detail)
			}
		}
		info = list
	default:
		return nil, fmt.Errorf("forbidden offline platform path %s", r.URL.Path)
	}
	raw, err := json.Marshal(map[string]any{"code": "0", "info": info})
	if err != nil {
		return nil, err
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(raw)), Request: r}, nil
}

type semiOrderWorker struct{ service *o.Service }

func (w semiOrderWorker) Ensure(ctx context.Context, org, id string) error {
	for i := 0; i < 64; i++ {
		result, err := w.service.Step(ctx, org, id)
		if err != nil {
			return err
		}
		if result.Terminal() {
			return nil
		}
	}
	return fmt.Errorf("offline sync did not complete")
}

func seedSemiOrderReceipts(t *testing.T, repo *obspg.Repository, scope o.Scope, shopA, shopB string, today time.Time) (*o.Service, *semiOrderFixture) {
	t.Helper()
	f := newSemiOrderFixture(t, scope, []string{shopA, shopB})
	service := &o.Service{Repository: repo, Access: f, Directory: f}
	service.Starter = semiOrderWorker{service}
	ctx := context.Background()
	app := &obsapp.Application{Service: service, Ready: func() bool { return true }}
	router := gin.New()
	for _, route := range obshttp.Routes(app, func(ctx context.Context, _ string) (context.Context, error) {
		return authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, UserID: scope.ActorID, EffectiveOrganizationID: scope.OrganizationID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Hour)}), nil
	}) {
		router.Handle(route.Method, route.Path, route.Handler)
	}
	recent := today.AddDate(0, 0, -1).Add(12 * time.Hour)
	older := today.AddDate(0, 0, -10).Add(12 * time.Hour)
	begin := func(store string, days int) {
		start, end := today.AddDate(0, 0, -days), today.Add(-time.Second)
		in := o.BeginInput{Kind: o.Orders, Stores: []string{store}, Start: &start, End: &end}
		key := uuid.NewString()
		request := func() o.Command {
			raw, e := json.Marshal(in)
			require.NoError(t, e)
			req := httptest.NewRequest(http.MethodPost, obshttp.BasePath+"/orders/syncs", bytes.NewReader(raw))
			req.Header.Set("Idempotency-Key", key)
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			var envelope struct {
				Data o.Command `json:"data"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
			return envelope.Data
		}
		cmd := request()
		require.Len(t, cmd.Syncs, 1)
		status, err := service.Status(ctx, scope, cmd.Syncs[0].ID)
		require.NoError(t, err)
		require.Equal(t, "completed", status.Status)
		before := f.calls
		again := request()
		require.Equal(t, cmd.ID, again.ID)
		require.Equal(t, cmd.Syncs[0].ID, again.Syncs[0].ID)
		require.Equal(t, before, f.calls, "same-key replay must not resynchronize")
	}
	f.rows[shopA] = []semiOrderReceipt{{"order-a-1", "999.00", "USD", recent}}
	begin(shopA, 30)
	// The boundary order is returned by two adjacent closed 48h windows.
	f.rows[shopA] = []semiOrderReceipt{{"order-a-1", "19.90", "USD", recent}, {"order-a-2", "0.10", "USD", recent.Add(time.Second)}, {"order-a-boundary", "5.25", "USD", today.AddDate(0, 0, -2)}, {"order-a-old", "30.01", "USD", older}, {"order-a-cny", "3.25", "CNY", older}}
	begin(shopA, 30)
	f.rows[shopB] = []semiOrderReceipt{{"order-b-1", "7.25", "USD", recent}, {"order-b-missing", "", "USD", today.AddDate(0, 0, -3).Add(12 * time.Hour)}, {"order-b-currency", "2.01", "", today.AddDate(0, 0, -3).Add(12 * time.Hour)}}
	begin(shopB, 7)
	return service, f
}

func TestSheinSemiOrdersSyncToSavedAmountsPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("semi_orders"), tcpg.WithUsername("installer"), tcpg.WithPassword(uuid.NewString()), tcpg.BasicWaitStrategies(), testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
		c.PortBindings = network.PortMap{network.MustParsePort("5432/tcp"): []network.PortBinding{{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "0"}}}
	}))
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(context.Background()) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
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
	require.NoError(t, owner.Exec("CREATE ROLE semi_runtime LOGIN PASSWORD '"+password+"' NOSUPERUSER NOCREATEDB NOCREATEROLE").Error)
	require.NoError(t, obspg.GrantRuntime(ctx, owner, "semi_runtime"))
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword("semi_runtime", password)
	repo, err := obspg.NewRepository(ctx, open(u.String()))
	require.NoError(t, err)
	zone := time.FixedZone("UTC+8", 8*3600)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	scope := o.Scope{OrganizationID: "semi-org", ActorID: "semi-actor", MemberID: "semi-member"}
	a, b := uuid.NewString(), uuid.NewString()
	service, fixture := seedSemiOrderReceipts(t, repo, scope, a, b, today)
	before := fixture.calls
	for _, test := range []struct {
		name, store string
		days        int
		status      string
		count       int
		amounts     []o.CurrencyOrderAmount
	}{
		{"A seven days", a, 7, "complete", 3, []o.CurrencyOrderAmount{{Currency: "USD", Amount: "25.25", OrderCount: 3}}},
		{"A thirty days", a, 30, "complete", 5, []o.CurrencyOrderAmount{{Currency: "CNY", Amount: "3.25", OrderCount: 1}, {Currency: "USD", Amount: "55.26", OrderCount: 4}}},
		{"B missing fields", b, 7, "missing_fields", 3, nil},
		{"B insufficient coverage", b, 30, "coverage_incomplete", 0, nil},
		{"B yesterday", b, 1, "complete", 1, []o.CurrencyOrderAmount{{Currency: "USD", Amount: "7.25", OrderCount: 1}}},
		{"A yesterday", a, 1, "complete", 2, []o.CurrencyOrderAmount{{Currency: "USD", Amount: "20.00", OrderCount: 2}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, e := service.SavedOrderAmounts(ctx, scope, o.OrderAmountQuery{StoreID: test.store, Start: today.AddDate(0, 0, -test.days), End: today})
			require.NoError(t, e)
			require.Equal(t, test.status, value.Status)
			if test.amounts != nil {
				require.NotNil(t, value.Amounts)
				require.Equal(t, test.amounts, *value.Amounts)
			} else {
				require.Nil(t, value.Amounts)
			}
			if value.MatchedOrderCount != nil {
				require.Equal(t, test.count, *value.MatchedOrderCount)
			}
			if test.status == "missing_fields" {
				require.Equal(t, &o.OrderAmountMissing{AmountCount: 1, CurrencyCount: 1}, value.Missing)
			}
			if test.status == "coverage_incomplete" {
				require.Nil(t, value.MatchedOrderCount)
			}
		})
	}
	require.Equal(t, before, fixture.calls, "saved metrics must not invoke any platform transport")
	heads, _, err := repo.Heads(ctx, scope.OrganizationID, o.Orders, []string{a})
	require.NoError(t, err)
	require.Len(t, heads, 1)
	record, err := repo.Record(ctx, scope.OrganizationID, a, o.Orders, heads[0].ID, "order-a-1")
	require.NoError(t, err)
	require.Equal(t, storecenter.ApplicationSemiManaged, heads[0].Binding.ApplicationType)
	require.Equal(t, "19.90", record.Order.Amount.Value)
	require.Equal(t, "999.00", record.Order.SupplyCost.Value)
	require.NotEqual(t, record.Order.CreatedAt, record.Order.IssuedAt, "detail time must not replace list cohort time")
}
