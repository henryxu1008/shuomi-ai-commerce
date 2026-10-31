package shein

import (
	"context"
	"encoding/json"
	"net/http"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"testing"
	"time"
)

func TestSupplyCreationRequiresExactOriginalReadback(t *testing.T) {
	for _, wrong := range []bool{true, false} {
		calls := 0
		c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return observationResponse(`{"code":"0","info":{"deliveryCode":"shipment"}}`), nil
			}
			if r.Method != "GET" || r.URL.Path != "/open-api/shipping/delivery" || r.URL.Query().Get("deliveryCode") != "shipment" {
				t.Fatal("not original shipment readback", r.URL)
			}
			order := "original"
			if wrong {
				order = "other-order"
			}
			return observationResponse(`{"code":"0","info":{"count":1,"list":[{"deliveryCode":"shipment","deliveryType":4,"supplierWarehouseId":55,"sendPackage":1,"deliveryOrderDataList":[{"orderNo":"` + order + `","skuCode":"sku","deliveryQuantity":2}]}]}}`), nil
		}))
		plan := sf.Plan{Purpose: "supply_shipping", Body: json.RawMessage(`{"deliveryType":4,"supplierWarehouseId":55,"list":[{"orderNo":"original","skuCode":"sku","tempDeliveryQuantity":2}]}`)}
		result, err := c.ExecuteFulfillmentPlan(context.Background(), goodsCredential(), plan)
		if wrong {
			if err != f.ErrUnknown {
				t.Fatal("unproven original effect accepted", result, err)
			}
		} else if err != nil || result.Result["shipping_id"] != "shipment" {
			t.Fatal(result, err)
		}
		if calls != 2 {
			t.Fatal("creation receipt alone treated as success", calls)
		}
	}
}

func TestAcceptAddressIsExplicitAndTrackingUsesCurrentChannel(t *testing.T) {
	source := sf.Source{Kind: "consumer_order", ID: "order", Market: "shein-us", Type: 1, StockMode: 3, Status: 1, PrintStatus: 1, SelectedLogistics: 2, Logistics: []int{2}, Deadline: time.Now().Add(time.Hour), Lines: []sf.Line{{ID: "123", SKU: "sku", Quantity: 1, GoodsStatus: 1, ExchangeTag: 1}}}
	reads := 0
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		reads++
		if r.Method != "POST" || r.URL.Path != "/open-api/order/express-channel" {
			t.Fatal("unexpected provider call", r.Method, r.URL.Path)
		}
		return observationResponse(`{"code":"0","info":{"expressChannels":[{"site":"shein-us","expressIdCode":"CurrentCarrier","expressChannelCode":"current-channel"}]}}`), nil
	}))
	plan, err := c.BuildFulfillmentPlan(context.Background(), goodsCredential(), source, sf.Delivery{}, sf.Selection{Purpose: "accept_address"})
	if err != nil {
		t.Fatal(err)
	}
	var address struct {
		HandleType int    `json:"handleType"`
		OrderNo    string `json:"orderNo"`
	}
	_ = json.Unmarshal(plan.Body, &address)
	if address.HandleType != 2 || address.OrderNo != "order" || reads != 0 {
		t.Fatal("acceptance read/dispatched during preparation")
	}
	source.SelectedLogistics = 0
	if _, err = c.BuildFulfillmentPlan(context.Background(), goodsCredential(), source, sf.Delivery{}, sf.Selection{Purpose: "accept_address"}); err == nil {
		t.Fatal("silently chose self shipping")
	}
	source.SelectedLogistics, source.Status = 2, 2
	source.Lines[0].GoodsStatus = 2
	delivery := sf.Delivery{Name: "draft-dn", PackageID: "package"}
	delivery.Lines = append(delivery.Lines, struct {
		SourceLine string `json:"source_line"`
		Qty        int64  `json:"qty"`
	}{"123", 1})
	_, err = c.BuildFulfillmentPlan(context.Background(), goodsCredential(), source, delivery, sf.Selection{Purpose: "consumer_tracking", Channel: "old-channel", Tracking: "waybill"})
	if err == nil {
		t.Fatal("stale carrier accepted")
	}
	plan, err = c.BuildFulfillmentPlan(context.Background(), goodsCredential(), source, delivery, sf.Selection{Purpose: "consumer_tracking", Channel: "current-channel", Tracking: "waybill"})
	if err != nil || len(plan.GoodsIDs) != 1 || reads != 2 {
		t.Fatal(plan, err, reads)
	}
}
