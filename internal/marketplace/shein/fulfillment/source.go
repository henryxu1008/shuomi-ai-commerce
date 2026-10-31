package fulfillment

import (
	"encoding/json"
	f "task-processor/internal/fulfillment"
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
func (s Source) Eligible(now time.Time) error {
	if s.ID == "" || len(s.ID) > 140 || s.Deadline.IsZero() || !now.Before(s.Deadline) || len(s.Lines) < 1 || len(s.Lines) > 20 {
		return f.ErrConflict
	}
	seen := map[string]bool{}
	for _, line := range s.Lines {
		if line.ID == "" || line.SKU == "" || seen[line.ID] || line.Quantity < 0 || line.Quantity > 100000000 || line.Delivered < 0 || line.Delivered > line.Quantity {
			return f.ErrConflict
		}
		seen[line.ID] = true
	}
	if len(s.FulfillableLines()) == 0 {
		return f.ErrConflict
	}
	switch s.Kind {
	case "consumer_order":
		if s.Market != "shein-us" || s.StockMode != 3 || s.PrintStatus != 1 || s.Status != 1 && s.Status != 2 && s.Status != 7 || s.SelectedLogistics == 1 || s.Type != 1 && s.Type != 2 && s.Type != 4 {
			return f.ErrConflict
		}
		allowed := false
		for _, mode := range s.Logistics {
			if mode != 1 && mode != 2 {
				return f.ErrConflict
			}
			if mode == 2 {
				allowed = true
			}
		}
		if !allowed {
			return f.ErrConflict
		}
		for _, line := range s.Lines {
			if line.Quantity != 1 || line.GoodsStatus != 1 && line.GoodsStatus != 2 && line.GoodsStatus != 7 || line.ExchangeTag != 1 && line.ExchangeTag != 3 {
				return f.ErrConflict
			}
		}
	case "platform_supply":
		if s.Status != 2 && s.Status != 3 || s.Scene != 1 && s.Scene != 2 || s.Type != 1 && s.Type != 2 {
			return f.ErrConflict
		}
	default:
		return f.ErrInvalid
	}
	return nil
}
