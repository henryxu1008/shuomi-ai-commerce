package commercialbilling

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"task-processor/internal/commercial/billing"
	resource "task-processor/internal/integration/orgresource"
	moneystore "task-processor/internal/integration/persistence/money"
	"task-processor/internal/ledger/orgresource"
	"testing"
	"time"
)

func TestPostgresDataRowPublicationCASAndZeroOrderConcurrentRecovery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, e := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("issue1178"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("issue1178-isolated-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, e := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, e := db.DB()
	require.NoError(t, e)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, AutoMigrate(db))
	require.NoError(t, resource.AutoMigrate(db))
	require.NoError(t, moneystore.AutoMigrate(db))
	r, e := New(db)
	require.NoError(t, e)
	initial := zeroDataOffer()
	initial.UnitPriceMinor = 5
	initial.SettlementMode = billing.SettlementWallet
	initial.PricingVersion = "initial"
	require.NoError(t, r.CreateOffer(ctx, initial))
	outcomes := make(chan error, 2)
	var wg sync.WaitGroup
	for _, actor := range []string{"platform-a", "platform-b"} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			_, err := r.PublishDataRowPrice(ctx, billing.DataRowPriceCommand{ActorID: actor, OrganizationID: "test-org", ExpectedVersion: "initial", UnitPriceMinor: 0, IdempotencyKey: "cas", Reason: "isolated test"})
			outcomes <- err
		}(actor)
	}
	wg.Wait()
	close(outcomes)
	successes, conflicts := 0, 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else if errors.Is(err, billing.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)
	current, e := r.CurrentDataRowOffer(ctx)
	require.NoError(t, e)
	cmd := billing.DataRowPriceCommand{ActorID: "platform-a", OrganizationID: "test-org", ExpectedVersion: current.PricingVersion, UnitPriceMinor: 0, IdempotencyKey: "same-request", Reason: "same original publication"}
	receipts := make(chan billing.DataRowPriceReceipt, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); v, err := r.PublishDataRowPrice(ctx, cmd); receipts <- v; errs <- err }()
	}
	wg.Wait()
	close(receipts)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	id := ""
	for v := range receipts {
		if id == "" {
			id = v.ID
		}
		require.Equal(t, id, v.ID)
	}
	current, e = r.CurrentDataRowOffer(ctx)
	require.NoError(t, e)
	rr, e := resource.NewGormRepository(db, resource.TransactionConfig{})
	require.NoError(t, e)
	grants, e := orgresource.NewPurchasedResourceGrantService(rr, resource.TrustedCommercialGrantAuthorizer{})
	require.NoError(t, e)
	wallet, e := moneystore.New(db)
	require.NoError(t, e)
	lost := &lostZeroGrant{PurchasedResourceGrantPort: grants, lose: true}
	svc, e := billing.NewService(r, r, r, r, wallet, lost)
	require.NoError(t, e)
	q, e := svc.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "zero-org", OfferID: current.OfferID, Quantity: 3})
	require.NoError(t, e)
	req := billing.CreateResourceOrderRequest{OrganizationID: "zero-org", ActorID: "test-admin", QuoteID: q.QuoteID, IdempotencyKey: "zero-original"}
	first, e := svc.CreateResourceOrder(ctx, req)
	require.ErrorIs(t, e, billing.ErrReconciliationRequired)
	// Multiple recovery callers must converge on one grant and original order.
	errs = make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := svc.CreateResourceOrder(ctx, req); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.True(t, err == nil || errors.Is(err, billing.ErrReconciliationRequired), "unexpected recovery error: %v", err)
	}
	require.NoError(t, svc.ReconcileRecoverableResourceOrders(ctx, 50))
	done, e := svc.CreateResourceOrder(ctx, req)
	require.NoError(t, e)
	require.Equal(t, first.OrderID, done.OrderID)
	require.Equal(t, billing.OrderFulfilled, done.Status)
	var available, orders, grantsCount, moneyEntries int64
	require.NoError(t, db.Raw("SELECT available FROM saas_organization_resource_buckets WHERE organization_id='zero-org' AND resource_type='data_row'").Scan(&available).Error)
	require.Equal(t, int64(3), available)
	require.NoError(t, db.Model(&orderRow{}).Count(&orders).Error)
	require.Equal(t, int64(1), orders)
	require.NoError(t, db.Raw("SELECT count(*) FROM saas_organization_resource_events WHERE organization_id='zero-org'").Scan(&grantsCount).Error)
	require.Equal(t, int64(1), grantsCount)
	require.NoError(t, db.Raw("SELECT count(*) FROM ledger_organization_wallet_entries").Scan(&moneyEntries).Error)
	require.Zero(t, moneyEntries)
}
