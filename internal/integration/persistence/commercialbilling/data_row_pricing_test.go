package commercialbilling

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/commercial/billing"
	"testing"
)

func TestDataRowPricePublicationVersionReplayAndEmptyInstallation(t *testing.T) {
	ctx := context.Background()
	r := commercialRepository(t)
	cmd := billing.DataRowPriceCommand{ActorID: "platform-admin", OrganizationID: "org-a", ExpectedVersion: "initial", UnitPriceMinor: 0, IdempotencyKey: "publish-zero", Reason: "isolated debugging"}
	_, err := r.PublishDataRowPrice(ctx, cmd)
	require.ErrorIs(t, err, billing.ErrOfferUnavailable)
	initial := zeroDataOffer()
	initial.UnitPriceMinor = 5
	initial.SettlementMode = billing.SettlementWallet
	initial.PricingVersion = "initial"
	require.NoError(t, r.CreateOffer(ctx, initial))
	q, err := r.CreateQuote(ctx, billing.QuoteRequest{OrganizationID: "org-a", OfferID: initial.OfferID, Quantity: 3})
	require.NoError(t, err)
	receipt, err := r.PublishDataRowPrice(ctx, cmd)
	require.NoError(t, err)
	require.Equal(t, int64(0), receipt.Offer.UnitPriceMinor)
	require.NotEqual(t, initial.PricingVersion, receipt.Offer.PricingVersion)
	replay, err := r.PublishDataRowPrice(ctx, cmd)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	cmd.UnitPriceMinor = 7
	_, err = r.PublishDataRowPrice(ctx, cmd)
	require.ErrorIs(t, err, billing.ErrConflict)
	cmd.IdempotencyKey = "stale-version"
	_, err = r.PublishDataRowPrice(ctx, cmd)
	require.ErrorIs(t, err, billing.ErrConflict)
	current, err := r.CurrentDataRowOffer(ctx)
	require.NoError(t, err)
	require.Equal(t, receipt.Offer, current)
	frozen, err := r.ReadQuote(ctx, "org-a", q.QuoteID)
	require.NoError(t, err)
	require.Equal(t, int64(15), frozen.TotalMinor)
	cmd.ExpectedVersion = current.PricingVersion
	cmd.IdempotencyKey = "return-to-paid"
	_, err = r.PublishDataRowPrice(ctx, cmd)
	require.NoError(t, err)
	original, err := r.ReadDataRowPriceReceipt(ctx, "platform-admin", "publish-zero")
	require.NoError(t, err)
	require.Equal(t, receipt, original)
	history, err := r.DataRowPriceHistory(ctx, "", 1)
	require.NoError(t, err)
	require.Len(t, history.Items, 1)
	require.NotEmpty(t, history.NextCursor)
	page, err := r.DataRowPriceHistory(ctx, history.NextCursor, 1)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Empty(t, page.NextCursor)
}
