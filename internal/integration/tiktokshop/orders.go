package tiktokshop

import (
	"encoding/json"
	"strconv"
	obs "task-processor/internal/storecenter/observations"
	"time"
)

// DecodeOrderPage parses the public order-list protocol without retaining raw
// responses or opening a network connection.
func DecodeOrderPage(raw []byte) (obs.TikTokOrderPage, error) {
	var result obs.TikTokOrderPage
	if len(raw) == 0 || len(raw) > 2<<20 {
		return result, obs.ErrUnavailable
	}
	var wire struct {
		Code *int `json:"code"`
		Data *struct {
			Total  *int            `json:"total_count"`
			Next   json.RawMessage `json:"next_page_token"`
			Orders *[]struct {
				ID      string          `json:"id"`
				Date    json.RawMessage `json:"create_time"`
				Status  string          `json:"status"`
				Type    string          `json:"order_type"`
				Payment *struct {
					Currency string          `json:"currency"`
					Amount   json.RawMessage `json:"sub_total"`
				} `json:"payment"`
			} `json:"orders"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Code == nil || *wire.Code != 0 || wire.Data == nil || wire.Data.Total == nil || *wire.Data.Total < 0 || wire.Data.Orders == nil || len(*wire.Data.Orders) > 100 {
		return result, obs.ErrUnavailable
	}
	result.Total = *wire.Data.Total
	result.Items = []obs.TikTokOrder{}
	if len(wire.Data.Next) > 0 {
		if json.Unmarshal(wire.Data.Next, &result.NextToken) != nil || string(wire.Data.Next) == "null" || len(result.NextToken) > 8192 || !obs.Text(result.NextToken, 8192) {
			return result, obs.ErrUnavailable
		}
	}
	seen := map[string]bool{}
	for _, v := range *wire.Data.Orders {
		o := obs.TikTokOrder{ID: v.ID, Site: "tiktokshop-us", Status: v.Status, Type: v.Type}
		if len(v.Date) > 0 && string(v.Date) != "null" {
			text, e := orderScalar(v.Date)
			if e != nil {
				return result, e
			}
			seconds, e := strconv.ParseInt(text, 10, 64)
			if e != nil || seconds <= 0 || seconds > 253402300799 {
				return result, obs.ErrUnavailable
			}
			o.Date = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
		}
		if v.Payment != nil {
			amount, e := orderScalar(v.Payment.Amount)
			if e != nil {
				return result, e
			}
			o.Amount = &obs.Price{Currency: v.Payment.Currency, Value: amount}
		}
		if !o.Valid() || seen[o.ID] {
			return obs.TikTokOrderPage{}, obs.ErrUnavailable
		}
		seen[o.ID] = true
		result.Items = append(result.Items, o)
	}
	return result, nil
}
func orderScalar(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	if raw[0] == '"' {
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return "", obs.ErrUnavailable
		}
		return text, nil
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		return "", obs.ErrUnavailable
	}
	return number.String(), nil
}
