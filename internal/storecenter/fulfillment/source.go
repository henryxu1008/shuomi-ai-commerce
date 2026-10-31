package fulfillment

import (
	"encoding/json"
	"time"
)

type Line struct {
	GoodsStatus int    `json:"goodsStatus,omitempty"`
	ExchangeTag int    `json:"exchangeTag,omitempty"`
	ID          string `json:"id"`
	SKU         string `json:"sku"`
	Quantity    int64  `json:"quantity"`
	Delivered   int64  `json:"delivered"`
	Received    *int64 `json:"received"`
	Stored      *int64 `json:"stored"`
	Defective   *int64 `json:"defective"`
}
type Source struct {
	Kind              string          `json:"kind"`
	ID                string          `json:"id"`
	Market            string          `json:"market"`
	Status            int             `json:"status"`
	StockMode         int             `json:"stockMode"`
	PrintStatus       int             `json:"printStatus"`
	Logistics         []int           `json:"logistics"`
	SelectedLogistics int             `json:"selectedLogistics"`
	Scene             int             `json:"scene"`
	Type              int             `json:"type"`
	CountryMarket     *int            `json:"countryMarket"`
	StorageID         string          `json:"storageId"`
	Rules             json.RawMessage `json:"rules"`
	Deadline          time.Time       `json:"deadline"`
	Lines             []Line          `json:"lines"`
}

func (s Source) DisplayMarket() string {
	if s.Kind == "consumer_order" {
		return s.Market
	}
	return "unknown"
}
func (s Source) FulfillableLines() []Line {
	result := []Line{}
	for _, line := range s.Lines {
		if line.Quantity > line.Delivered {
			result = append(result, line)
		}
	}
	return result
}
