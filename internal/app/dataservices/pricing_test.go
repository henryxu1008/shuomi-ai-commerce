package dataservicesapp

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/commercial/billing"
	"task-processor/internal/product/dataacquisition"
	"testing"
)

type testCommercialPrice struct{}

func (testCommercialPrice) CurrentDataRowPrice(context.Context) (dataacquisition.PriceSnapshot, error) {
	return dataacquisition.PriceSnapshot{OfferID: "test-offer", Version: "test-price", Currency: "CNY", UnitPriceFen: 5}, nil
}

type catalogPrice struct {
	offer billing.Offer
	err   error
}

func (c catalogPrice) CurrentDataRowOffer(context.Context) (billing.Offer, error) {
	return c.offer, c.err
}
func TestCommercialDataPriceNeverDefaultsMissingPriceToFree(t *testing.T) {
	_, e := (CommercialDataPrice{}).CurrentDataRowPrice(context.Background())
	require.Error(t, e)
	_, e = (CommercialDataPrice{Reader: catalogPrice{err: billing.ErrOfferUnavailable}}).CurrentDataRowPrice(context.Background())
	require.Error(t, e)
}
