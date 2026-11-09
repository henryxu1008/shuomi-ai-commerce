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
	temu "task-processor/internal/integration/temu"

	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

type temuOrderReceipt struct {
	id, customer, amount string
	date                 time.Time
}

// All requests terminate in this transport. There is no network fallback.
type temuOrderFixture struct {
	scope     o.Scope
	materials map[string]storecenter.ProductExecutionMaterial
	rows      map[string][]temuOrderReceipt
	access    *storeapp.OfficialObservationAccess
	mu        sync.Mutex
	calls     int
}

func newTemuOrderFixture(t *testing.T, scope o.Scope, stores []string, today time.Time) *temuOrderFixture {
	t.Helper()
	f := &temuOrderFixture{scope: scope, materials: map[string]storecenter.ProductExecutionMaterial{}, rows: map[string][]temuOrderReceipt{}}
	app := storecenter.OfficialApplication{AppID: "offline-temu-app", Version: "offline:temu:US:manual_token:3:r2", Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionUS, AuthorizationMethod: storecenter.AuthorizationManualToken, SupportedModes: storecenter.ModesLocal | storecenter.ModesSemiManaged}
	provider, e := temu.NewOfficialClient(temu.OfficialClientOptions{Application: app, AppSecret: "synthetic-temu-secret", HTTPClient: &http.Client{Transport: f}})
	require.NoError(t, e)
	protection, e := temu.NewCredentialProtection("offline-temu-key", bytes.Repeat([]byte{9}, 32))
	require.NoError(t, e)
	for i, store := range stores {
		now := time.Now().UTC().Truncate(time.Second)
		mode := storecenter.MerchantLocal
		namespace := storecenter.MerchantLocalNamespace
		semi := ""
		if i%2 == 1 {
			mode = storecenter.MerchantSemiManaged
			namespace = storecenter.MerchantRegionalSemiNamespace
			semi = "semi-" + store
		}
		facts := storecenter.OfficialVerificationFacts{Platform: app.Platform, IdentityPartition: app.IdentityPartition, Mode: mode, Namespace: namespace, MerchantID: strconv.Itoa(i + 100), SemiUniqueID: semi, RegionID: 211, ExpiresAt: now.Add(24 * time.Hour), Scopes: []string{"bg.order.list.v2.get", "bg.order.amount.query"}, ContractRevision: "temu-regional-order-identity-v2", VerifiedAt: now}
		raw, e := json.Marshal(facts)
		require.NoError(t, e)
		identity, e := facts.MerchantIdentityHash()
		require.NoError(t, e)
		attempt := storecenter.OfficialConnectionAttempt{OrganizationID: scope.OrganizationID, StoreID: store, AttemptID: uuid.NewString(), ActorID: scope.ActorID, MemberID: scope.MemberID, AppID: app.AppID, AppVersion: app.Version, Platform: app.Platform, IdentityPartition: app.IdentityPartition, AuthorizationMethod: app.AuthorizationMethod, StoreVersion: 1, ConnectionVersion: 1, State: "verified", StateHash: "original-offline-state", VerificationJSON: string(raw)}
		attempt.KeyID, attempt.Ciphertext, e = protection.Seal(attempt, storecenter.OfficialMerchantCredential{AppID: app.AppID, Temu: &storecenter.TemuMerchantCredential{AccessToken: "synthetic-" + store}})
		require.NoError(t, e)
		f.materials[store] = storecenter.ProductExecutionMaterial{Platform: app.Platform, StoreVersion: 1, ServiceExpiresAt: now.Add(24 * time.Hour), Attempt: attempt, Connection: storecenter.OfficialConnectionView{AttemptID: attempt.AttemptID, AppID: app.AppID, AppRevision: app.Version, Version: 1, Status: storecenter.ConnectionStatusConnected, State: "verified", Platform: app.Platform, IdentityPartition: app.IdentityPartition, AuthorizationMethod: app.AuthorizationMethod, Mode: mode, RegionID: 211, MerchantIdentityHash: identity, CredentialExpiresAt: &facts.ExpiresAt}}
	}
	registry, e := storeapp.NewOfficialApplicationRegistry([]storeapp.OfficialApplicationRegistration{{Provider: provider, Protection: protection}})
	require.NoError(t, e)
	f.access, e = storeapp.NewOfficialObservationAccess(f, f, registry)
	require.NoError(t, e)
	if len(stores) >= 2 {
		recent := today.AddDate(0, 0, -1).Add(12 * time.Hour)
		f.rows[stores[0]] = []temuOrderReceipt{{"ORDER-A-1", "CUSTOMER-A", "10.00", recent}, {"ORDER-A-2", "CUSTOMER-A", "20.00", today.Add(-time.Second)}, {"ORDER-A-OLD", "CUSTOMER-OLD", "5.00", today.AddDate(0, 0, -10).Add(12 * time.Hour)}}
		f.rows[stores[1]] = []temuOrderReceipt{{"ORDER-B-1", "CUSTOMER-B", "7.25", recent}, {"ORDER-B-MISSING", "CUSTOMER-MISSING", "", today.AddDate(0, 0, -3).Add(12 * time.Hour)}}
	}
	return f
}
func (f *temuOrderFixture) Authorize(_ context.Context, s o.Scope, k o.Kind, _ bool) error {
	if s != f.scope || k != o.Orders {
		return o.ErrForbidden
	}
	return nil
}
func (f *temuOrderFixture) AuthorizeObservation(_ context.Context, s storecenter.ObservationSubject) (storecenter.ObservationAuthorization, error) {
	if s.OrganizationID != f.scope.OrganizationID || s.ActorID != f.scope.ActorID || s.MemberID != f.scope.MemberID || s.Purpose != storecenter.ObservationPurposeOrders {
		return storecenter.ObservationAuthorization{}, o.ErrForbidden
	}
	return storecenter.ObservationAuthorization{}, nil
}
func (f *temuOrderFixture) ReadObservation(ctx context.Context, s storecenter.ObservationSubject, store string, a storecenter.ObservationAuthorizer, _ time.Time) (storecenter.ProductExecutionMaterial, error) {
	if _, e := a.AuthorizeObservation(ctx, s); e != nil {
		return storecenter.ProductExecutionMaterial{}, e
	}
	m, ok := f.materials[store]
	if !ok {
		return m, storecenter.ErrNotFound
	}
	return m, nil
}
func (f *temuOrderFixture) ListStores(_ context.Context, s o.Scope, k o.Kind, platform string) ([]string, error) {
	if s != f.scope || k != o.Orders {
		return nil, o.ErrForbidden
	}
	ids := []string{}
	if platform == "" || platform == "temu" {
		for id := range f.materials {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
func (f *temuOrderFixture) Open(ctx context.Context, s o.Scope, id string, k o.Kind, sync bool, b *o.Binding) (o.Merchant, error) {
	if e := f.Authorize(ctx, s, k, sync); e != nil {
		return nil, e
	}
	return f.access.Authorize(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders, Sync: sync}, id, b)
}
func (f *temuOrderFixture) ReadSavedTemuOrderBinding(ctx context.Context, s o.Scope, id string) (o.Binding, error) {
	if e := f.Authorize(ctx, s, o.Orders, false); e != nil {
		return o.Binding{}, e
	}
	return f.access.ReadSavedTemuOrderBinding(ctx, storecenter.ObservationSubject{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: s.MemberID, Purpose: storecenter.ObservationPurposeOrders}, id)
}
func (f *temuOrderFixture) RoundTrip(r *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if r.Method != "POST" || r.URL.Host != "openapi-b-us.temu.com" || r.URL.Path != "/openapi/router" {
		return nil, fmt.Errorf("forbidden offline Temu request")
	}
	var fields map[string]json.RawMessage
	if json.NewDecoder(r.Body).Decode(&fields) != nil {
		return nil, fmt.Errorf("invalid offline request")
	}
	var api, token string
	_ = json.Unmarshal(fields["type"], &api)
	_ = json.Unmarshal(fields["access_token"], &token)
	store := strings.TrimPrefix(token, "synthetic-")
	material, ok := f.materials[store]
	if !ok {
		return nil, fmt.Errorf("wrong offline credential")
	}
	var facts storecenter.OfficialVerificationFacts
	if json.Unmarshal([]byte(material.Attempt.VerificationJSON), &facts) != nil {
		return nil, fmt.Errorf("invalid original facts")
	}
	var payload any
	switch api {
	case "bg.open.accesstoken.info.get":
		mode := 100
		if facts.Mode == storecenter.MerchantSemiManaged {
			mode = 1
		}
		mall, _ := strconv.Atoi(facts.MerchantID)
		payload = map[string]any{"success": true, "result": map[string]any{"mallId": mall, "mallType": mode, "semiUniqueId": facts.SemiUniqueID, "regionId": 211, "expiredTime": facts.ExpiresAt.Unix(), "apiScopeList": facts.Scopes}}
	case "bg.order.list.v2.get":
		var start, end int64
		var page, size int
		_ = json.Unmarshal(fields["createAfter"], &start)
		_ = json.Unmarshal(fields["createBefore"], &end)
		_ = json.Unmarshal(fields["pageNumber"], &page)
		_ = json.Unmarshal(fields["pageSize"], &size)
		if size != 100 || page != 1 {
			return nil, fmt.Errorf("incorrect original page")
		}
		rows := []map[string]any{}
		for _, row := range f.rows[store] {
			if row.date.Unix() < start || row.date.Unix() > end {
				continue
			}
			rows = append(rows, map[string]any{"parentOrderMap": map[string]any{"parentOrderSn": row.id, "parentOrderTime": row.date.Unix(), "regionId": 211, "parentOrderStatus": 2}, "orderList": []map[string]any{{"orderSn": "child-1", "quantity": 9}, {"orderSn": "child-2", "quantity": 4}}, "sites": []int{1, 2}, "packages": []string{"p1", "p2"}, "buyerEmail": "PRIVATE@example.invalid"})
		}
		payload = map[string]any{"success": true, "result": map[string]any{"totalItemNum": len(rows), "pageItems": rows}}
	case "bg.order.amount.query":
		var id string
		_ = json.Unmarshal(fields["parentOrderSn"], &id)
		found := false
		for _, row := range f.rows[store] {
			if row.id != id {
				continue
			}
			found = true
			parent := map[string]any{"parentOrderSn": id}
			if row.amount != "" {
				cents, e := strconv.ParseInt(strings.ReplaceAll(row.amount, ".", ""), 10, 64)
				if e != nil {
					return nil, e
				}
				parent["retailPriceTotal"] = map[string]any{"amount": cents, "currency": "USD"}
				parent["basePriceTotal"] = map[string]any{"amount": 999999, "currency": "USD"}
			}
			payload = map[string]any{"success": true, "result": map[string]any{"parentOrderMap": parent, "orderList": []map[string]any{{"quantity": 99, "basePrice": 999999}}}}
		}
		if !found {
			return nil, fmt.Errorf("unknown parent order")
		}
	default:
		return nil, fmt.Errorf("forbidden offline Temu API")
	}
	raw, e := json.Marshal(payload)
	if e != nil {
		return nil, e
	}
	return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(bytes.NewReader(raw)), Request: r}, nil
}
func TestTemuManualOrdersToSavedAmountsPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, e := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("temu_orders"), tcpg.WithUsername("installer"), tcpg.WithPassword(uuid.NewString()), tcpg.BasicWaitStrategies(), testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
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
	require.NoError(t, owner.Exec("CREATE ROLE temu_runtime LOGIN PASSWORD '"+password+"' NOSUPERUSER NOCREATEDB NOCREATEROLE").Error)
	require.NoError(t, obspg.GrantRuntime(ctx, owner, "temu_runtime"))
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.User = url.UserPassword("temu_runtime", password)
	repo, e := obspg.NewRepository(ctx, open(u.String()))
	require.NoError(t, e)
	zone := time.FixedZone("UTC+8", 8*3600)
	now := time.Now().In(zone)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, zone)
	scope := o.Scope{OrganizationID: "temu-org", ActorID: "temu-actor", MemberID: "temu-member"}
	a, b := uuid.NewString(), uuid.NewString()
	f := newTemuOrderFixture(t, scope, []string{a, b}, today)
	service := &o.Service{Repository: repo, Access: f, Directory: f, TemuOrders: f.access}
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
		value, e := service.SavedTemuOrderAmounts(ctx, scope, o.OrderAmountQuery{StoreID: test.store, Start: today.AddDate(0, 0, -test.days), End: today})
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
