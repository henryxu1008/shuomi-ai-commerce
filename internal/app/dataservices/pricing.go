package dataservicesapp

import (
	"context"
	"task-processor/internal/commercial/billing"
	"task-processor/internal/product/dataacquisition"
)

// CommercialDataPrice only maps the canonical Commercial offer into a Product
// value. It neither stores a second price nor supplies a default on failure.
type CommercialDataPrice struct{ Reader billing.DataRowPriceReader }

func (p CommercialDataPrice) CurrentDataRowPrice(ctx context.Context) (dataacquisition.PriceSnapshot, error) {
	if p.Reader == nil {
		return dataacquisition.PriceSnapshot{}, dataacquisition.ErrUnavailable
	}
	o, e := p.Reader.CurrentDataRowOffer(ctx)
	if e != nil || o.Validate() != nil {
		return dataacquisition.PriceSnapshot{}, dataacquisition.ErrUnavailable
	}
	v := dataacquisition.PriceSnapshot{OfferID: o.OfferID, Version: o.PricingVersion, Currency: o.Currency, UnitPriceFen: o.UnitPriceMinor}
	if v.Validate() != nil {
		return v, dataacquisition.ErrUnavailable
	}
	return v, nil
}
