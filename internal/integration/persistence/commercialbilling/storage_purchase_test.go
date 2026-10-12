package commercialbilling

import (
	"context"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"task-processor/internal/commercial/billing"
	moneystore "task-processor/internal/integration/persistence/money"
	storage "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/ledger/money"
	"task-processor/internal/ledger/storagecapacity"
	"testing"
	"time"
)

func TestStorageQuoteFreezesTermsAndActivationIsAtomicWithOrder(t *testing.T) {
	r := commercialRepository(t)
	// SQLite compares timestamp text; use the fixture's UTC clock for GORM's
	// automatic updated_at too, matching PostgreSQL's timestamp semantics.
	r.db.Config.NowFunc = r.now
	ctx := context.Background()
	require.NoError(t, storage.AutoMigrate(r.db))
	capacity, err := storage.New(r.db)
	require.NoError(t, err)
	require.NoError(t, r.CreateOffer(ctx, billing.InitialStorageOffer()))
	quote, err := r.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "org", OfferID: billing.InitialStorageOffer().OfferID, Quantity: 2})
	require.NoError(t, err)
	require.Equal(t, int64(3980), quote.TotalMinor)
	require.Equal(t, storagecapacity.PackBytes, quote.StorageUnitBytes)
	order, err := r.CreatePendingResourceOrder(ctx, billing.CreateResourceOrderRequest{OrganizationID: "org", ActorID: "admin", QuoteID: quote.QuoteID, IdempotencyKey: "purchase"}, quote)
	require.NoError(t, err)
	require.Equal(t, billing.OrderStorageCapacityPurchase, order.Kind)
	require.NoError(t, r.PrepareStorageOrder(ctx, order))
	summary, err := capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "5368709120", summary.QuotaBytes)
	_, err = r.ActivateStorageOrder(ctx, order)
	require.Error(t, err, "no COMMITTED wallet proof may activate capacity")
	order.Status = billing.OrderFulfilling
	order.WalletReservationID = "wallet-receipt"
	order.WalletReservationState = "RESERVED"
	require.NoError(t, r.UpdateOrder(ctx, order))
	order.Version++
	order.WalletReservationState = "COMMITTED"
	active, err := r.ActivateStorageOrder(ctx, order)
	require.NoError(t, err)
	require.Equal(t, billing.OrderFulfilled, active.Status)
	summary, err = capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "112742891520", summary.QuotaBytes)
	replay, err := r.ActivateStorageOrder(ctx, order)
	require.NoError(t, err)
	require.Equal(t, active.OrderID, replay.OrderID)
	summary, err = capacity.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Len(t, summary.ActiveLots, 1)
	spend, err := r.ReadOrderSummary(ctx, "org", active.UpdatedAt.Add(-time.Minute), active.UpdatedAt.Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, int64(3980), spend.SpendMinor)
	require.Equal(t, int64(3980), spend.OtherSpendMinor)
	kind, product := billing.OrderStorageCapacityPurchase, billing.ProductStorageCapacityPack
	page, err := r.ListOrders(ctx, "org", billing.OrderFilter{Kind: &kind, ProductKind: &product, Limit: 10})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	other := order
	other.OrganizationID = "other"
	_, err = r.ActivateStorageOrder(ctx, other)
	require.Error(t, err)
}

func TestStorageLostCommitAcknowledgementRecoversOriginalLot(t *testing.T) {
	r := commercialRepository(t)
	ctx := context.Background()
	require.NoError(t, storage.AutoMigrate(r.db))
	cap, err := storage.New(r.db)
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open("file:storage-wallet-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	pool, _ := db.DB()
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, moneystore.AutoMigrate(db))
	wallet, err := moneystore.New(db)
	require.NoError(t, err)
	now := time.Now().UTC()
	require.NoError(t, wallet.RecordPaymentSettlement(ctx, money.PaymentSettlement{PaymentID: "payment", PayerUserID: "admin", Currency: "CNY", GrossAmountMinor: 10000, CommissionableAmountMinor: 10000, Status: money.PaymentSettled, SettledAt: now, ProviderReference: "provider", Version: 1}))
	_, err = wallet.CreditSettledTopUp(ctx, "", money.OrganizationTopUpSettlement{PaymentID: "payment", CommercialOrderID: "topup", OrganizationID: "org", Currency: "CNY", AmountMinor: 10000, SettledAt: now, ProviderReference: "provider", Version: 1})
	require.NoError(t, err)
	lost := &loseFirstCommitAcknowledgement{OrganizationWalletReader: wallet, OrganizationWalletCommander: wallet, lose: true}
	svc, err := billing.NewService(r, r, r, r, lost, nil)
	require.NoError(t, err)
	svc.ConfigureStorage(r)
	require.NoError(t, r.CreateOffer(ctx, billing.InitialStorageOffer()))
	quote, err := svc.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "org", OfferID: billing.InitialStorageOffer().OfferID, Quantity: 1})
	require.NoError(t, err)
	req := billing.CreateResourceOrderRequest{OrganizationID: "org", ActorID: "admin", QuoteID: quote.QuoteID, IdempotencyKey: "buy"}
	order, err := svc.CreateResourceOrder(ctx, req)
	require.ErrorIs(t, err, billing.ErrReconciliationRequired)
	summary, err := cap.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "5368709120", summary.QuotaBytes)
	require.NoError(t, svc.ReconcileRecoverableResourceOrders(ctx, 50))
	replayed, err := svc.CreateResourceOrder(ctx, req)
	require.NoError(t, err)
	require.Equal(t, order.OrderID, replayed.OrderID)
	require.Equal(t, billing.OrderFulfilled, replayed.Status)
	summary, err = cap.ReadSummary(ctx, "org")
	require.NoError(t, err)
	require.Equal(t, "59055800320", summary.QuotaBytes)
	require.Len(t, summary.ActiveLots, 1)
	balance, err := wallet.ReadOrganizationWallet(ctx, "org", "CNY")
	require.NoError(t, err)
	require.Equal(t, int64(8010), balance.AvailableMinor)
}
