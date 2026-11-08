package commercialbilling

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/commercial/billing"
	"task-processor/internal/ledger/money"
	"task-processor/internal/ledger/orgresource"
)

type zeroOrderWallet struct {
	billing.WalletPort
	reserves int
}

func (w *zeroOrderWallet) ReserveCommercialPurchase(context.Context, money.ReserveWalletFundsInput) (money.WalletReservation, error) {
	w.reserves++
	return money.WalletReservation{}, errors.New("zero order must not reserve money")
}

type lostZeroGrant struct {
	billing.PurchasedResourceGrantPort
	lose bool
}

func (g *lostZeroGrant) GrantPurchasedResource(ctx context.Context, in orgresource.PurchasedResourceGrantInput) (orgresource.PurchasedResourceGrantResult, error) {
	out, err := g.PurchasedResourceGrantPort.GrantPurchasedResource(ctx, in)
	if err == nil && g.lose {
		g.lose = false
		return orgresource.PurchasedResourceGrantResult{}, errors.New("synthetic grant acknowledgement lost")
	}
	return out, err
}
func zeroDataOffer() billing.Offer {
	return billing.Offer{OfferID: "zero-data", ProductKind: billing.ProductDataRow, ResourceType: orgresource.ResourceDataRow, Currency: "CNY", PricingVersion: "zero-version", UnitPriceMinor: 0, MinQuantity: 1, MaxQuantity: 100000, Status: billing.OfferActive, SettlementMode: "ZERO_PRICE"}
}

func TestDataRowZeroPriceOrderAndOriginalGrantRecovery(t *testing.T) {
	for _, lose := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "lost-grant"}[lose], func(t *testing.T) {
			ctx := context.Background()
			repo, wallet, grants := resourceRecoveryOwners(t, false)
			old, err := repo.ReadOffer(ctx, "recovery-offer")
			require.NoError(t, err)
			old.Status = billing.OfferDisabled
			require.NoError(t, repo.SaveOffer(ctx, old))
			require.NoError(t, repo.CreateOffer(ctx, zeroDataOffer()), "explicit DATA_ROW zero offer must be usable")
			w := &zeroOrderWallet{WalletPort: wallet}
			g := &lostZeroGrant{PurchasedResourceGrantPort: grants, lose: lose}
			svc, err := billing.NewService(repo, repo, repo, repo, w, g)
			require.NoError(t, err)
			q, err := svc.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "recovery-org", OfferID: "zero-data", Quantity: 3})
			require.NoError(t, err)
			require.Zero(t, q.TotalMinor)
			require.Equal(t, billing.SettlementMode("ZERO_PRICE"), q.SettlementMode)
			req := billing.CreateResourceOrderRequest{OrganizationID: "recovery-org", ActorID: "test-admin", QuoteID: q.QuoteID, IdempotencyKey: "original-zero"}
			first, err := svc.CreateResourceOrder(ctx, req)
			if lose {
				require.ErrorIs(t, err, billing.ErrReconciliationRequired)
				_, publishErr := repo.PublishDataRowPrice(ctx, billing.DataRowPriceCommand{ActorID: "platform", OrganizationID: "recovery-org", ExpectedVersion: "zero-version", UnitPriceMinor: 7, IdempotencyKey: "back-to-paid", Reason: "restore paid price after original zero grant"})
				require.NoError(t, publishErr)
				repo.now = func() time.Time { return q.ExpiresAt.Add(time.Minute) }
				svc, err = billing.NewService(repo, repo, repo, repo, w, g)
				require.NoError(t, err)
				require.NoError(t, svc.ReconcileRecoverableResourceOrders(ctx, 50))
			} else {
				require.NoError(t, err)
			}
			got, err := svc.CreateResourceOrder(ctx, req)
			require.NoError(t, err)
			require.Equal(t, first.OrderID, got.OrderID)
			require.Equal(t, billing.OrderFulfilled, got.Status)
			require.NoError(t, got.Validate())
			require.Empty(t, got.WalletReservationID)
			require.Empty(t, got.PaymentID)
			require.Zero(t, w.reserves)
			require.Equal(t, int64(3), got.Items[0].ResourceQuantity)
			replay, err := grants.GrantPurchasedResource(ctx, orgresource.PurchasedResourceGrantInput{OrganizationID: "recovery-org", OperationID: "grant:" + got.OrderID, CommercialOrderID: got.OrderID, CommercialOrderItemID: got.Items[0].OrderItemID, ResourceType: orgresource.ResourceDataRow, Quantity: 3, Principal: orgresource.Principal{ID: "commercial-billing", Kind: orgresource.PrincipalTrustedCommercial}})
			require.NoError(t, err)
			require.True(t, replay.Replayed)
			require.Equal(t, "3", replay.Snapshot.BalanceAfter)
			req.ActorID = "other-member"
			_, err = svc.CreateResourceOrder(ctx, req)
			require.ErrorIs(t, err, billing.ErrConflict)
		})
	}
}

func TestDataRowZeroQuoteKeepsFrozenPriceAcrossChangeAndExpiry(t *testing.T) {
	ctx := context.Background()
	repo := commercialRepository(t)
	zero := zeroDataOffer()
	require.NoError(t, repo.CreateOffer(ctx, zero))
	q, err := repo.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "org-a", OfferID: zero.OfferID, Quantity: 3})
	require.NoError(t, err)
	zero.Status = billing.OfferDisabled
	require.NoError(t, repo.SaveOffer(ctx, zero))
	frozen, err := repo.ReadQuote(ctx, "org-a", q.QuoteID)
	require.NoError(t, err)
	require.Equal(t, int64(0), frozen.TotalMinor)
	_, err = repo.ReadQuote(ctx, "org-b", q.QuoteID)
	require.Error(t, err)
	repo.now = func() time.Time { return q.ExpiresAt }
	_, err = repo.ReadQuote(ctx, "org-a", q.QuoteID)
	require.ErrorIs(t, err, billing.ErrQuoteExpired)
}

func TestDataRowCurrentPriceHasDatabaseUniqueFence(t *testing.T) {
	repo := commercialRepository(t)
	ctx := context.Background()
	first := zeroDataOffer()
	first.UnitPriceMinor = 5
	first.SettlementMode = ""
	require.NoError(t, repo.CreateOffer(ctx, first))
	other := first
	other.OfferID = "competing-data"
	other.PricingVersion = "other-version"
	require.Error(t, repo.CreateOffer(ctx, other), "two current data prices must not both commit")
}
