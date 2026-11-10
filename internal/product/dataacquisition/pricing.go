package dataacquisition

import (
	"context"
	"math"
	"strconv"
	"strings"

	sigjson "sigs.k8s.io/json"
)

// PriceSnapshot is an immutable Commercial value, not another price source.
type PriceSnapshot struct {
	OfferID      string `json:"offerId"`
	Version      string `json:"pricingVersion"`
	Currency     string `json:"currency"`
	UnitPriceFen int64  `json:"unitPriceFen,string"`
}

// UnmarshalJSON requires an explicit canonical amount: Go's int64 zero value
// alone cannot distinguish a free Commercial price from missing evidence.
func (p *PriceSnapshot) UnmarshalJSON(raw []byte) error {
	var value struct {
		OfferID  string  `json:"offerId"`
		Version  string  `json:"pricingVersion"`
		Currency string  `json:"currency"`
		Amount   *string `json:"unitPriceFen"`
	}
	strict, err := sigjson.UnmarshalStrict(raw, &value, sigjson.DisallowUnknownFields, sigjson.DisallowDuplicateFields)
	if err != nil || len(strict) != 0 || value.Amount == nil {
		return ErrInvalid
	}
	amount, err := strconv.ParseInt(*value.Amount, 10, 64)
	if err != nil || strconv.FormatInt(amount, 10) != *value.Amount {
		return ErrInvalid
	}
	next := PriceSnapshot{OfferID: value.OfferID, Version: value.Version, Currency: value.Currency, UnitPriceFen: amount}
	if next.Validate() != nil {
		return ErrInvalid
	}
	*p = next
	return nil
}

func (p PriceSnapshot) Validate() error {
	if p.OfferID == "" || len(p.OfferID) > 128 || strings.TrimSpace(p.OfferID) != p.OfferID || p.Version == "" || len(p.Version) > 128 || strings.TrimSpace(p.Version) != p.Version || p.Currency != "CNY" || p.UnitPriceFen < 0 {
		return ErrInvalid
	}
	return nil
}
func (p PriceSnapshot) Cost(rows int) (int64, error) {
	if p.Validate() != nil || rows < 1 || rows > 200 || p.UnitPriceFen > math.MaxInt64/int64(rows) {
		return 0, ErrInvalid
	}
	return p.UnitPriceFen * int64(rows), nil
}

type PriceReader interface {
	CurrentDataRowPrice(context.Context) (PriceSnapshot, error)
}
type PriceConfirmation struct {
	Version        string
	MaximumCostFen int64
}

func (c PriceConfirmation) Validate() error {
	if c.MaximumCostFen < 0 || c.Version == "" || len(c.Version) > 128 || strings.TrimSpace(c.Version) != c.Version {
		return ErrInvalid
	}
	return nil
}
