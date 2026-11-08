package walmart

import (
	"encoding/json"
	"strconv"
	"strings"
	marketplace "task-processor/internal/marketplace/walmart"
	"time"
)

// DecodeOrderPage is the bounded protocol boundary used by the offline source.
// This phase deliberately adds no network client or live reader registration.
func DecodeOrderPage(raw []byte) (marketplace.OrderPage, error) {
	var result marketplace.OrderPage
	if len(raw) == 0 || len(raw) > 2<<20 {
		return result, marketplace.ErrUnavailable
	}
	type charge struct {
		Type   string `json:"chargeType"`
		Amount *struct {
			Currency string          `json:"currency"`
			Amount   json.RawMessage `json:"amount"`
		} `json:"chargeAmount"`
		Discount     json.RawMessage `json:"isDiscount"`
		DiscountType string          `json:"discountType"`
	}
	var wire struct {
		Errors []json.RawMessage `json:"errors"`
		List   *struct {
			Meta *struct {
				Total  *int            `json:"totalCount"`
				Cursor json.RawMessage `json:"nextCursor"`
			} `json:"meta"`
			Elements *struct {
				Orders *[]struct {
					ID       string          `json:"purchaseOrderId"`
					Customer string          `json:"customerOrderId"`
					Date     json.RawMessage `json:"orderDate"`
					Type     string          `json:"orderType"`
					Lines    *struct {
						Items []struct {
							Number   string `json:"lineNumber"`
							Quantity *struct {
								Amount json.RawMessage `json:"amount"`
							} `json:"orderLineQuantity"`
							Charges *struct {
								Items []charge `json:"charge"`
							} `json:"charges"`
						} `json:"orderLine"`
					} `json:"orderLines"`
				} `json:"order"`
			} `json:"elements"`
		} `json:"list"`
	}
	if json.Unmarshal(raw, &wire) != nil || len(wire.Errors) > 0 || wire.List == nil || wire.List.Meta == nil || wire.List.Meta.Total == nil || *wire.List.Meta.Total < 0 || wire.List.Elements == nil || wire.List.Elements.Orders == nil || len(*wire.List.Elements.Orders) > 30 {
		return result, marketplace.ErrUnavailable
	}
	result.Total = *wire.List.Meta.Total
	result.Items = []marketplace.Order{}
	if v := wire.List.Meta.Cursor; len(v) > 0 && string(v) != "null" {
		if json.Unmarshal(v, &result.NextCursor) != nil || len(result.NextCursor) > 8192 || strings.ContainsAny(result.NextCursor, "\r\n\x00") {
			return marketplace.OrderPage{}, marketplace.ErrUnavailable
		}
	}
	seen := map[string]bool{}
	for _, v := range *wire.List.Elements.Orders {
		o := marketplace.Order{ID: v.ID, Site: "walmart-us", CustomerOrderID: v.Customer, Type: v.Type, Lines: []marketplace.OrderLine{}}
		if date := string(v.Date); date != "" && date != "null" {
			text, e := orderNumber(v.Date)
			if e != nil {
				return result, e
			}
			millis, e := strconv.ParseInt(text, 10, 64)
			if e != nil || millis <= 0 {
				return result, marketplace.ErrUnavailable
			}
			o.Date = time.UnixMilli(millis).UTC().Format(time.RFC3339Nano)
		}
		if v.Lines != nil {
			for _, line := range v.Lines.Items {
				saved := marketplace.OrderLine{Number: line.Number, Charges: []marketplace.OrderCharge{}}
				if line.Quantity != nil {
					var e error
					saved.Quantity, e = orderNumber(line.Quantity.Amount)
					if e != nil {
						return result, e
					}
				}
				if line.Charges != nil {
					for _, c := range line.Charges.Items {
						if c.Type == "" {
							saved.Ambiguous = true
						}
						if c.Type != "PRODUCT" {
							continue
						}
						fee := marketplace.OrderCharge{}
						if c.Amount != nil {
							var e error
							fee.Currency = c.Amount.Currency
							fee.Amount, e = orderNumber(c.Amount.Amount)
							if e != nil {
								return result, e
							}
						}
						if len(c.Discount) > 0 && string(c.Discount) != "false" && string(c.Discount) != "null" || c.DiscountType != "" {
							saved.Ambiguous = true
						}
						saved.Charges = append(saved.Charges, fee)
					}
				}
				o.Lines = append(o.Lines, saved)
			}
		}
		if !o.Valid() || seen[o.ID] {
			return marketplace.OrderPage{}, marketplace.ErrUnavailable
		}
		seen[o.ID] = true
		result.Items = append(result.Items, o)
	}
	return result, nil
}
func orderNumber(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	if raw[0] == '"' {
		var value string
		if json.Unmarshal(raw, &value) != nil {
			return "", marketplace.ErrUnavailable
		}
		return value, nil
	}
	var number json.Number
	if json.Unmarshal(raw, &number) != nil {
		return "", marketplace.ErrUnavailable
	}
	return number.String(), nil
}
