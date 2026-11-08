package billing

import (
	"context"
	"time"
)

type DataRowPriceCommand struct {
	ActorID         string `json:"-"`
	OrganizationID  string `json:"-"`
	ExpectedVersion string `json:"expected_version"`
	UnitPriceMinor  int64  `json:"unit_price_minor"`
	IdempotencyKey  string `json:"-"`
	Reason          string `json:"reason"`
}
type DataRowPriceReceipt struct {
	ID              string    `json:"id"`
	ActorID         string    `json:"actor_id"`
	OrganizationID  string    `json:"organization_id"`
	PreviousOfferID string    `json:"previous_offer_id"`
	PreviousVersion string    `json:"previous_version"`
	Offer           Offer     `json:"offer"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
}
type DataRowPriceHistory struct {
	Items      []DataRowPriceReceipt `json:"items"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

// Only the Commercial owner persists prices; consumers receive a read-only port.
type DataRowPriceReader interface {
	CurrentDataRowOffer(context.Context) (Offer, error)
}
type DataRowPriceStore interface {
	DataRowPriceReader
	PublishDataRowPrice(context.Context, DataRowPriceCommand) (DataRowPriceReceipt, error)
	ReadDataRowPriceReceipt(context.Context, string, string) (DataRowPriceReceipt, error)
	DataRowPriceHistory(context.Context, string, int) (DataRowPriceHistory, error)
}
