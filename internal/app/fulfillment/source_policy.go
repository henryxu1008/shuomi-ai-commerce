package fulfillmentapp

import (
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/storecenter/fulfillment"
	"time"
)

func eligibleSourceLines(s sf.Source) []sf.Line {
	result := []sf.Line{}
	for _, line := range s.FulfillableLines() {
		if s.Kind != "consumer_order" || ((line.GoodsStatus == 1 || line.GoodsStatus == 2 || line.GoodsStatus == 7) && (line.ExchangeTag == 1 || line.ExchangeTag == 3)) {
			result = append(result, line)
		}
	}
	return result
}

func eligibleSource(s sf.Source, now time.Time) error {
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
	if len(eligibleSourceLines(s)) == 0 {
		return f.ErrConflict
	}
	switch s.Kind {
	case "consumer_order":
		if s.Market != "shein-us" || s.StockMode != 3 || s.PrintStatus != 1 || s.Status != 1 && s.Status != 2 && s.Status != 4 && s.Status != 7 || s.SelectedLogistics == 1 || s.Type != 1 && s.Type != 2 && s.Type != 4 {
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
			if line.Quantity != 1 || line.GoodsStatus < 1 || line.GoodsStatus > 7 || line.ExchangeTag < 1 || line.ExchangeTag > 3 {
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
