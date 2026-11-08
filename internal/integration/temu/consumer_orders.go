package temu

import (
	"encoding/json"
	"fmt"
	"strconv"
	o "task-processor/internal/storecenter/observations"
	"time"
)

// DecodeConsumerOrderPage has no I/O and retains only the allowlisted facts.
func DecodeConsumerOrderPage(raw []byte, amounts map[string][]byte) (o.TemuOrderPage, error) {
	var out o.TemuOrderPage
	if len(raw) == 0 || len(raw) > 2<<20 {
		return out, o.ErrUnavailable
	}
	var w struct {
		Success *bool `json:"success"`
		Result  *struct {
			Total *int `json:"totalItemNum"`
			Items *[]struct {
				Parent *struct {
					ID     string `json:"parentOrderSn"`
					Date   *int64 `json:"parentOrderTime"`
					Region int64  `json:"regionId"`
					Status *int   `json:"parentOrderStatus"`
				} `json:"parentOrderMap"`
			} `json:"pageItems"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &w) != nil || w.Success == nil || !*w.Success || w.Result == nil || w.Result.Total == nil || *w.Result.Total < 0 || w.Result.Items == nil || len(*w.Result.Items) > 100 {
		return out, o.ErrUnavailable
	}
	out.Total = *w.Result.Total
	out.Items = []o.TemuOrder{}
	seen := map[string]bool{}
	for _, v := range *w.Result.Items {
		if v.Parent == nil {
			return out, o.ErrUnavailable
		}
		p := v.Parent
		item := o.TemuOrder{ID: p.ID, Site: "temu-order-us", RegionID: p.Region}
		if p.Date != nil {
			if *p.Date <= 0 || *p.Date > 253402300799 {
				return out, o.ErrUnavailable
			}
			item.Date = time.Unix(*p.Date, 0).UTC().Format(time.RFC3339)
		}
		if p.Status != nil {
			item.Status = strconv.Itoa(*p.Status)
		}
		var e error
		item.Amount, e = decodeConsumerAmount(amounts[p.ID], p.ID)
		if e != nil || !item.Valid() || seen[item.ID] {
			return o.TemuOrderPage{}, o.ErrUnavailable
		}
		seen[item.ID] = true
		out.Items = append(out.Items, item)
	}
	return out, nil
}
func decodeConsumerAmount(raw []byte, id string) (*o.Price, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	if len(raw) > 2<<20 {
		return nil, o.ErrUnavailable
	}
	var w struct {
		Success *bool `json:"success"`
		Result  *struct {
			Parent *struct {
				ID    string `json:"parentOrderSn"`
				Price *struct {
					Amount   *int64 `json:"amount"`
					Currency string `json:"currency"`
				} `json:"retailPriceTotal"`
			} `json:"parentOrderMap"`
			Warning []string `json:"warning"`
		} `json:"result"`
	}
	if json.Unmarshal(raw, &w) != nil || w.Success == nil {
		return nil, o.ErrUnavailable
	}
	if !*w.Success {
		return nil, nil
	}
	if w.Result == nil || w.Result.Parent == nil || w.Result.Parent.ID != id {
		return nil, o.ErrUnavailable
	}
	if len(w.Result.Warning) > 0 || w.Result.Parent.Price == nil {
		return nil, nil
	}
	p := w.Result.Parent.Price
	if p.Currency != "" && p.Currency != "USD" {
		return nil, o.ErrUnavailable
	}
	result := &o.Price{Currency: p.Currency}
	if p.Amount != nil {
		if *p.Amount < 0 {
			return nil, o.ErrUnavailable
		}
		result.Value = fmt.Sprintf("%d.%02d", *p.Amount/100, *p.Amount%100)
	}
	return result, nil
}
