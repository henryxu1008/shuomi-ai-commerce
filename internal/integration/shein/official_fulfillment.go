package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"time"
)

func trackingReceipt(raw []byte, expected []string) ([]string, error) {
	var result struct {
		Code json.RawMessage `json:"code"`
		Info json.RawMessage `json:"info"`
	}
	if json.Unmarshal(raw, &result) != nil {
		return nil, f.ErrUnknown
	}
	code, err := scalar(result.Code)
	if err != nil || code != "0" || string(result.Info) != "[]" {
		return nil, f.ErrUnknown
	}
	// SHEIN explicitly documents an empty error list as complete acceptance of this request.
	// Acceptance does not assert pickup, shipment or delivery.
	return append([]string(nil), expected...), nil
}

func (c *OfficialClient) QueryFulfillmentSource(ctx context.Context, k storecenter.OfficialMerchantCredential, kind, id string) (sf.Source, error) {
	if !boundedParameter(id, 140) {
		return sf.Source{}, f.ErrInvalid
	}
	if kind == "consumer_order" {
		body, _ := json.Marshal(struct {
			IDs []string `json:"orderNoList"`
		}{[]string{id}})
		raw, err := c.goodsPost(ctx, "/open-api/order/order-detail", k, body)
		if err != nil {
			return sf.Source{}, f.ErrUnavailable
		}
		var envelope struct {
			Code json.RawMessage `json:"code"`
			Info []struct {
				observedRawDetail
				Print     *int  `json:"printOrderStatus"`
				Logistics []int `json:"optionalLogisticsList"`
				Selected  int   `json:"orderLogisticsType"`
				Goods     []struct {
					ID       json.RawMessage `json:"goodsId"`
					SKU      string          `json:"skuCode"`
					Status   json.RawMessage `json:"newGoodsStatus"`
					Exchange json.RawMessage `json:"goodsExchangeTag"`
					Timeouts []struct {
						Type int    `json:"timeOutType"`
						Time string `json:"timeOutTime"`
					} `json:"timeOutList"`
				} `json:"orderGoodsInfoList"`
			} `json:"info"`
		}
		if json.Unmarshal(raw, &envelope) != nil || len(envelope.Info) != 1 {
			return sf.Source{}, f.ErrUnavailable
		}
		code, _ := scalar(envelope.Code)
		row := envelope.Info[0]
		status, e1 := observedInt(row.Status)
		stock, e2 := observedInt(row.Stock)
		if code != "0" || row.ID != id || status == nil || stock == nil || row.Print == nil || e1 != nil || e2 != nil {
			return sf.Source{}, f.ErrUnavailable
		}
		orderType, typeErr := observedInt(row.Type)
		if typeErr != nil || orderType == nil {
			return sf.Source{}, f.ErrUnavailable
		}
		source := sf.Source{Kind: kind, ID: id, Market: row.Site, Type: *orderType, Status: *status, StockMode: *stock, PrintStatus: *row.Print, Logistics: row.Logistics, SelectedLogistics: row.Selected, Lines: []sf.Line{}}
		for _, goods := range row.Goods {
			identity, err := scalar(goods.ID)
			if err != nil || !observationDigits.MatchString(identity) {
				return sf.Source{}, f.ErrUnavailable
			}
			status, e1 := observedInt(goods.Status)
			exchange, e2 := observedInt(goods.Exchange)
			if e1 != nil || e2 != nil || status == nil || exchange == nil {
				return sf.Source{}, f.ErrUnavailable
			}
			source.Lines = append(source.Lines, sf.Line{ID: identity, SKU: goods.SKU, Quantity: 1, GoodsStatus: *status, ExchangeTag: *exchange})
			found := false
			for _, timeout := range goods.Timeouts {
				if timeout.Type == 2 || timeout.Type == 35 || timeout.Type == 1 && *status == 1 {
					timestamp, err := time.Parse("2006-01-02T15:04:05.999Z0700", timeout.Time)
					if err != nil {
						return sf.Source{}, f.ErrUnavailable
					}
					if source.Deadline.IsZero() || timestamp.Before(source.Deadline) {
						source.Deadline = timestamp.UTC()
					}
					found = true
				}
			}
			if !found {
				return sf.Source{}, f.ErrUnavailable
			}
		}
		return source, nil
	}
	if kind != "platform_supply" {
		return sf.Source{}, f.ErrInvalid
	}
	raw, err := c.goodsRequest(ctx, http.MethodGet, "/open-api/order/purchase-order-infos", url.Values{"orderNos": {id}, "pageNo": {"1"}, "pageSize": {"20"}}.Encode(), k, nil)
	if err != nil {
		return sf.Source{}, f.ErrUnavailable
	}
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Info *struct {
			Count int `json:"count"`
			List  []struct {
				ID       string          `json:"orderNo"`
				Status   int             `json:"status"`
				Scene    int             `json:"sceneType"`
				Type     int             `json:"type"`
				Market   int             `json:"countryMarket"`
				Storage  json.RawMessage `json:"storageId"`
				Deadline string          `json:"requestReceiptTime"`
				Rules    json.RawMessage `json:"deliveryCheckRuleFields"`
				Lines    []struct {
					SKU       string `json:"skuCode"`
					Qty       int64  `json:"orderQuantity"`
					Delivered int64  `json:"deliveryQuantity"`
					Received  int64  `json:"receiptQuantity"`
					Stored    int64  `json:"storageQuantity"`
					Defective int64  `json:"defectiveQuantity"`
				} `json:"orderExtends"`
			} `json:"list"`
		} `json:"info"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.Info == nil || envelope.Info.Count != 1 || len(envelope.Info.List) != 1 {
		return sf.Source{}, f.ErrUnavailable
	}
	code, _ := scalar(envelope.Code)
	row := envelope.Info.List[0]
	if code != "0" || row.ID != id {
		return sf.Source{}, f.ErrUnavailable
	}
	deadline, err := time.ParseInLocation("2006-01-02 15:04:05", row.Deadline, time.FixedZone("SHEIN", 8*3600))
	if err != nil {
		return sf.Source{}, f.ErrUnavailable
	}
	storage, err := scalar(row.Storage)
	if err != nil {
		return sf.Source{}, f.ErrUnavailable
	}
	source := sf.Source{Kind: kind, ID: id, Market: "unknown", Status: row.Status, Scene: row.Scene, Type: row.Type, CountryMarket: row.Market, StorageID: storage, Deadline: deadline.UTC(), Rules: row.Rules, Lines: []sf.Line{}}
	for _, line := range row.Lines {
		source.Lines = append(source.Lines, sf.Line{ID: line.SKU, SKU: line.SKU, Quantity: line.Qty, Delivered: line.Delivered, Received: line.Received, Stored: line.Stored, Defective: line.Defective})
	}
	return source, nil
}
