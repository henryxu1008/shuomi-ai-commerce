package walmart

import (
	"encoding/json"
	"time"
)

type Product struct {
	SKU               string `json:"sku"`
	WPID              string `json:"wpid,omitempty"`
	GTIN              string `json:"gtin,omitempty"`
	ProductName       string `json:"productName,omitempty"`
	Price             *Price `json:"price,omitempty"`
	PublicationStatus string `json:"publishedStatus,omitempty"`
	LifecycleStatus   string `json:"lifecycleStatus,omitempty"`
	Availability      string `json:"availability,omitempty"`
}
type Price struct {
	Currency string      `json:"currency"`
	Amount   json.Number `json:"amount"`
}

func (p Product) Valid() bool {
	return len(p.SKU) > 0 && len(p.SKU) <= 128 && len(p.WPID) <= 128 && len(p.GTIN) <= 128 && len(p.ProductName) <= 2000
}

type ProductPage struct {
	Total      int       `json:"totalItems"`
	Items      []Product `json:"ItemResponse"`
	NextCursor *string   `json:"nextCursor"`
	ObservedAt time.Time `json:"observedAt"`
}
type AcceptedFeed struct {
	FeedID        string `json:"feedId"`
	CorrelationID string `json:"correlationId"`
}
type RuleCatalog struct {
	FeedType     string   `json:"feedType"`
	Version      string   `json:"version"`
	ProductTypes []string `json:"productTypes"`
}
