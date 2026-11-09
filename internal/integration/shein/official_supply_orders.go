package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
)

// Supply reads use the existing signed GET transport and never a fulfillment mutation.
func (c *OfficialClient) QueryObservedSupplyOrders(ctx context.Context, k storecenter.OfficialMerchantCredential, ids []string) (o.SupplyPage, error) {
	requested, e := o.NormalizeSupplyOrderIDs(ids)
	if e != nil || len(requested) > o.SupplyChunkSize {
		return o.SupplyPage{}, o.ErrInvalid
	}
	query := url.Values{"orderNos": {strings.Join(requested, ",")}, "pageNumber": {"1"}, "pageSize": {"200"}, "selectJitMother": {"1"}}
	raw, e := c.goodsRequest(ctx, http.MethodGet, "/open-api/order/purchase-order-infos", query.Encode(), k, nil)
	if e != nil {
		return o.SupplyPage{}, o.ErrUnavailable
	}
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Info *struct {
			Count *json.Number `json:"count"`
			List  []struct {
				ID      string          `json:"orderNo"`
				Type    json.RawMessage `json:"type"`
				Status  json.RawMessage `json:"status"`
				Scene   json.RawMessage `json:"sceneType"`
				Country json.RawMessage `json:"countryMarket"`
				Mother  string          `json:"isJitMotherName"`
				Lines   []struct {
					SKU       string       `json:"skuCode"`
					Ordered   *json.Number `json:"orderQuantity"`
					Delivered *json.Number `json:"deliveryQuantity"`
					Received  *json.Number `json:"receiptQuantity"`
					Stored    *json.Number `json:"storageQuantity"`
					Defective *json.Number `json:"defectiveQuantity"`
				} `json:"orderExtends"`
			} `json:"list"`
		} `json:"info"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Info == nil {
		return o.SupplyPage{}, o.ErrUnavailable
	}
	code, e := scalar(envelope.Code)
	if e != nil || code != "0" || len(envelope.Info.List) > 200 {
		return o.SupplyPage{}, o.ErrUnavailable
	}
	page := o.SupplyPage{Items: []o.SupplyOrder{}, Notes: []string{}}
	count, e := exactFulfillmentQuantity(envelope.Info.Count)
	if e != nil || count == nil || *count != int64(len(envelope.Info.List)) {
		page.Notes = append(page.Notes, "supply_count_unverified")
	}
	allowed := map[string]bool{}
	for _, id := range requested {
		allowed[id] = true
	}
	seen := map[string]o.SupplyOrder{}
	conflict := map[string]bool{}
	for _, row := range envelope.Info.List {
		if !allowed[row.ID] {
			return o.SupplyPage{}, o.ErrConflict
		}
		order := o.SupplyOrder{ID: row.ID, MotherRaw: row.Mother, Mother: o.SupplyMother(row.Mother), Lines: []o.SupplyLine{}}
		for _, pair := range []struct {
			raw  json.RawMessage
			dest **int
		}{{row.Type, &order.Type}, {row.Status, &order.Status}, {row.Scene, &order.Scene}, {row.Country, &order.CountryMarket}} {
			value, err := observedInt(pair.raw)
			if err != nil {
				return o.SupplyPage{}, o.ErrUnavailable
			}
			*pair.dest = value
		}
		for _, rowLine := range row.Lines {
			line := o.SupplyLine{SKU: rowLine.SKU}
			for _, pair := range []struct {
				raw  *json.Number
				dest **int64
			}{{rowLine.Ordered, &line.Ordered}, {rowLine.Delivered, &line.Delivered}, {rowLine.Received, &line.Received}, {rowLine.Stored, &line.Stored}, {rowLine.Defective, &line.Defective}} {
				value, err := exactFulfillmentQuantity(pair.raw)
				if err != nil {
					return o.SupplyPage{}, o.ErrUnavailable
				}
				*pair.dest = value
			}
			order.Lines = append(order.Lines, line)
		}
		if !order.Valid() {
			return o.SupplyPage{}, o.ErrUnavailable
		}
		if old, ok := seen[order.ID]; ok && !reflect.DeepEqual(old, order) {
			conflict[order.ID] = true
		}
		seen[order.ID] = order
	}
	for _, id := range requested {
		if conflict[id] {
			page.Notes = append(page.Notes, "supply_identity_conflict")
			continue
		}
		if order, ok := seen[id]; ok {
			page.Items = append(page.Items, order)
		}
	}
	return page, nil
}
