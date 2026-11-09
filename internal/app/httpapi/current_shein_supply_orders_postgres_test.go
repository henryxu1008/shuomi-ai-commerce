//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
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
	obspg "task-processor/internal/integration/persistence/storeobservations"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

// This receipt setup is also used by the bounded, offline local trial.
func seedSupplyOrderReceipts(t *testing.T, repo *obspg.Repository, scope o.Scope, a, b string) (*o.Service, *semiOrderFixture) {
	t.Helper()
	f := newSheinOrderFixture(t, scope, []string{a, b}, storecenter.ApplicationFullyManaged, nil)
	row := func(id, mother string, n int) map[string]any {
		return map[string]any{"orderNo": id, "isJitMotherName": mother, "type": 1, "status": 2, "sceneType": 1, "orderExtends": []map[string]any{{"skuCode": "sku-" + id, "orderQuantity": n, "deliveryQuantity": n, "receiptQuantity": n, "storageQuantity": n, "defectiveQuantity": 0}}}
	}
	original := row("PO-A", "否", 5)
	missingQty := row("PO-B-1", "No", 2)
	delete(missingQty["orderExtends"].([]map[string]any)[0], "receiptQuantity")
	f.supplyRows = map[string][]map[string]any{a: {original, row("JIT-MOTHER-A", "是", 100), row("JIT-CHILD-A", "否", 7), original}, b: {row("PO-B-OLD", "否", 99), missingQty, row("PO-B-UNKNOWN", "", 50)}}
	service := &o.Service{Repository: repo, Access: f, Directory: f}
	service.Starter = semiOrderWorker{service}
	return service, f
}

func TestSheinSupplyOrdersBatchPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpg.Run(ctx, "postgres:16-alpine", tcpg.WithDatabase("supply_orders"), tcpg.WithUsername("installer"), tcpg.WithPassword(uuid.NewString()), tcpg.BasicWaitStrategies(), testcontainers.WithHostConfigModifier(func(c *container.HostConfig) {
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
	require.NoError(t, owner.Exec("CREATE ROLE supply_runtime LOGIN PASSWORD '"+password+"' NOSUPERUSER NOCREATEDB NOCREATEROLE").Error)
	require.NoError(t, obspg.GrantRuntime(ctx, owner, "supply_runtime"))
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword("supply_runtime", password)
	repo, err := obspg.NewRepository(ctx, open(u.String()))
	require.NoError(t, err)
	scope := o.Scope{OrganizationID: "supply-org", ActorID: "supply-actor", MemberID: "supply-member"}
	a, b := uuid.NewString(), uuid.NewString()
	service, f := seedSupplyOrderReceipts(t, repo, scope, a, b)
	router := sheinOrderFixtureHTTP(service, scope)
	begin := func(store, key string, ids []string) o.Command {
		input := o.BeginInput{Kind: o.SupplyOrders, Stores: []string{store}, SupplyOrderIDs: ids}
		raw, _ := json.Marshal(input)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/workbench/store-observations/supply-orders/syncs", bytes.NewReader(raw))
		request.Header.Set("Authorization", "Bearer offline")
		request.Header.Set("X-Requested-Organization-ID", scope.OrganizationID)
		request.Header.Set("Idempotency-Key", key)
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, 200, response.Code, response.Body.String())
		var envelope struct {
			Data o.Command `json:"data"`
		}
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
		return envelope.Data
	}
	key := uuid.NewString()
	first := begin(a, key, []string{"PO-A", "JIT-MOTHER-A", "PO-A", "JIT-CHILD-A"})
	before := f.calls
	replay := begin(a, key, []string{"JIT-CHILD-A", "PO-A", "JIT-MOTHER-A"})
	require.Equal(t, first.ID, replay.ID)
	require.Equal(t, before, f.calls)
	read := func(store string) o.SupplyObservation {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workbench/store-observations/supply-orders/observation?storeId="+store, nil)
		req.Header.Set("Authorization", "Bearer offline")
		req.Header.Set("X-Requested-Organization-ID", scope.OrganizationID)
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		require.Equal(t, 200, res.Code, res.Body.String())
		var out struct {
			Data o.SupplyObservation `json:"data"`
		}
		require.NoError(t, json.Unmarshal(res.Body.Bytes(), &out))
		return out.Data
	}
	saved := read(a)
	require.Equal(t, "saved", saved.Status)
	require.Len(t, saved.SavedOrderIDs, 3)
	require.Equal(t, 2, saved.Summary.OrderCount)
	require.Equal(t, 1, saved.Summary.MotherCount)
	require.Equal(t, "12", *saved.Summary.Quantities.Ordered.Value)
	require.Equal(t, "0", *saved.Summary.Quantities.Defective.Value)
	begin(b, uuid.NewString(), []string{"PO-B-OLD"})
	second := begin(b, uuid.NewString(), []string{"PO-B-1", "PO-B-MISSING", "PO-B-UNKNOWN"})
	partial := read(b)
	require.Equal(t, "incomplete", partial.Status)
	require.Equal(t, second.Syncs[0].ID, partial.Source.SyncID)
	require.Equal(t, []string{"PO-B-MISSING"}, partial.Source.MissingIDs)
	require.Len(t, partial.SavedOrderIDs, 2)
	require.Equal(t, 1, partial.Summary.OrderCount)
	require.Equal(t, 1, partial.Summary.UnknownTypeCount)
	require.Equal(t, "2", *partial.Summary.Quantities.Ordered.Value)
	require.Nil(t, partial.Summary.Quantities.Received.Value)
	require.Equal(t, saved, read(a))
	require.Equal(t, before+2, f.calls) // observation reads never call the provider
	var count int64
	require.NoError(t, owner.Raw("SELECT count(*) FROM store_observations.records WHERE kind='supply-orders'").Scan(&count).Error)
	require.EqualValues(t, 6, count)
	// A queued multi-chunk batch hides the older completed result. Only a
	// successful original CAS page may advance its immutable ID checkpoint.
	ids := []string{}
	for i := 0; i < 21; i++ {
		id := fmt.Sprintf("BATCH-%02d", i)
		ids = append(ids, id)
		f.supplyRows[a] = append(f.supplyRows[a], map[string]any{"orderNo": id, "isJitMotherName": "No", "orderExtends": []map[string]any{{"skuCode": id, "orderQuantity": 1}}})
	}
	manual := &o.Service{Repository: repo, Access: f, Directory: f}
	queued, err := manual.Begin(ctx, scope, uuid.NewString(), o.BeginInput{Kind: o.SupplyOrders, Stores: []string{a}, SupplyOrderIDs: ids})
	require.NoError(t, err)
	pending, err := manual.SupplyObservation(ctx, scope, a)
	require.NoError(t, err)
	require.Equal(t, "running", pending.Status)
	require.Nil(t, pending.Summary)
	require.Empty(t, pending.SavedOrderIDs)
	old := queued.Syncs[0]
	badNext := old.Progress
	progress := *old.Progress.SupplyOrders
	badNext.SupplyOrders = &progress
	badNext.SupplyOrders.Next = 20
	badNext.Seen = 20
	badNext.Pages = 1
	badNext.Page = 2
	badRows := []o.Record{}
	for _, id := range ids[:20] {
		order := o.SupplyOrder{ID: id, Lines: []o.SupplyLine{}}
		badRows = append(badRows, o.Record{StoreID: a, SyncID: old.ID, ID: id, SupplyOrder: &order})
	}
	badRows[1].StoreID = b // after the first insert, shared transaction must roll back everything
	_, err = repo.CommitPage(ctx, old, badNext, badRows, "running", time.Now())
	require.ErrorIs(t, err, o.ErrInvalid)
	var rolledBack int64
	require.NoError(t, owner.Raw("SELECT count(*) FROM store_observations.records WHERE sync_id=?", old.ID).Scan(&rolledBack).Error)
	require.Zero(t, rolledBack)
	unchanged, err := repo.ReadSync(ctx, scope.OrganizationID, old.ID)
	require.NoError(t, err)
	require.Equal(t, old.Revision, unchanged.Revision)
	require.Zero(t, unchanged.Progress.SupplyOrders.Next)
	firstChunk, err := manual.Step(ctx, scope.OrganizationID, old.ID)
	require.NoError(t, err)
	require.Equal(t, 20, firstChunk.Progress.SupplyOrders.Next)
	_, err = repo.CommitPage(ctx, old, old.Progress, nil, "partial", time.Now())
	require.ErrorIs(t, err, o.ErrConflict)
	last, err := manual.Step(ctx, scope.OrganizationID, old.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", last.Status)
	require.Equal(t, 21, last.Progress.Seen)
	all, err := manual.SupplyObservation(ctx, scope, a)
	require.NoError(t, err)
	require.Equal(t, "21", *all.Summary.Quantities.Ordered.Value)
	require.Len(t, all.SavedOrderIDs, 21)
	// Worst accepted multibyte IDs may all be absent. The final JSONB progress,
	// including both identity arrays and notes, still fits the original limit.
	longIDs := []string{}
	for i := 0; i < 78; i++ {
		longIDs = append(longIDs, strings.Repeat("采", 125)+fmt.Sprintf("%03d", i))
	}
	longBatch, err := service.Begin(ctx, scope, uuid.NewString(), o.BeginInput{Kind: o.SupplyOrders, Stores: []string{a}, SupplyOrderIDs: longIDs})
	require.NoError(t, err)
	longResult, err := repo.ReadSync(ctx, scope.OrganizationID, longBatch.Syncs[0].ID)
	require.NoError(t, err)
	require.Equal(t, "partial", longResult.Status)
	require.Len(t, longResult.Progress.SupplyOrders.MissingIDs, 78)
	var checkpointBytes int
	require.NoError(t, owner.Raw("SELECT octet_length(progress::text) FROM store_observations.syncs WHERE id=?", longResult.ID).Scan(&checkpointBytes).Error)
	require.LessOrEqual(t, checkpointBytes, 65536)
	// A current-boundary change invalidates the old result instead of returning stale facts.
	changed := f.materials[a]
	changed.StoreVersion++
	f.materials[a] = changed
	_, err = service.SupplyObservation(ctx, scope, a)
	require.ErrorIs(t, err, o.ErrConflict)
	for _, mode := range []storecenter.OfficialApplicationType{storecenter.ApplicationSelfOperated, storecenter.ApplicationSemiManaged} {
		other := newSheinOrderFixture(t, scope, []string{a}, mode, nil)
		_, err = other.Open(ctx, scope, a, o.SupplyOrders, false, nil)
		require.ErrorIs(t, err, o.ErrUnsupported)
	}
}
