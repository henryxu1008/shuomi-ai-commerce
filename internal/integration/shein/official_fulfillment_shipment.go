package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/storecenter/fulfillment"
	"task-processor/internal/storecenter"
)

// Query only the shipment ID returned to this exact original durable operation.
// The projection contains no warehouse contacts, signed URLs or private address.
func (c *OfficialClient) QueryFulfillmentShipment(ctx context.Context, k storecenter.OfficialMerchantCredential, id string) (sf.SupplyShipment, error) {
	var empty sf.SupplyShipment
	if !boundedParameter(id, 140) {
		return empty, f.ErrInvalid
	}
	raw, err := c.goodsRequest(ctx, http.MethodGet, "/open-api/shipping/delivery", url.Values{"deliveryCode": {id}, "page": {"1"}, "perPage": {"20"}}.Encode(), k, nil)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	info, err := observationInfo[struct {
		Count *int                `json:"count"`
		List  []sf.SupplyShipment `json:"list"`
	}](raw, false)
	if err != nil || info.Count == nil || *info.Count != 1 || len(info.List) != 1 || info.List[0].ID != id || len(info.List[0].Lines) == 0 || len(info.List[0].Lines) > 20 {
		return empty, f.ErrUnknown
	}
	return info.List[0], nil
}
func validateSupplyShipment(shipment sf.SupplyShipment, body json.RawMessage) error {
	var original struct {
		DeliveryType int   `json:"deliveryType"`
		Warehouse    int64 `json:"supplierWarehouseId"`
		Lines        []struct {
			Order string `json:"orderNo"`
			SKU   string `json:"skuCode"`
			Qty   int64  `json:"tempDeliveryQuantity"`
		} `json:"list"`
	}
	if json.Unmarshal(body, &original) != nil || shipment.DeliveryType != original.DeliveryType || shipment.Warehouse != original.Warehouse || shipment.Packages == nil || *shipment.Packages != 1 {
		return f.ErrUnknown
	}
	expected := map[string]int64{}
	for _, line := range original.Lines {
		key := line.Order + "\x00" + line.SKU
		if _, ok := expected[key]; ok {
			return f.ErrUnknown
		}
		expected[key] = line.Qty
	}
	for _, line := range shipment.Lines {
		key := line.Order + "\x00" + line.SKU
		qty, ok := expected[key]
		if !ok || line.Qty == nil || *line.Qty != qty {
			return f.ErrUnknown
		}
		delete(expected, key)
	}
	for _, qty := range expected {
		if qty != 0 {
			return f.ErrUnknown
		}
	}
	return nil
}
