package shein

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	f "task-processor/internal/fulfillment"
	sf "task-processor/internal/marketplace/shein/fulfillment"
	"task-processor/internal/storecenter"
	"time"
)

func (c *OfficialClient) QueryFulfillmentOptions(ctx context.Context, k storecenter.OfficialMerchantCredential, s sf.Source, selection sf.Selection) (sf.ShippingOptions, error) {
	if s.Kind == "consumer_order" {
		raw, err := c.goodsPost(ctx, "/open-api/order/express-channel", k, []byte(`{}`))
		if err != nil {
			return sf.ShippingOptions{}, f.ErrUnavailable
		}
		info, err := observationInfo[struct {
			Channels []sf.Channel `json:"expressChannels"`
		}](raw, false)
		if err != nil || len(info.Channels) > 500 {
			return sf.ShippingOptions{}, f.ErrUnavailable
		}
		result := sf.ShippingOptions{Channels: []sf.Channel{}}
		for _, channel := range info.Channels {
			if channel.Site == "shein-us" && boundedParameter(channel.Code, 140) && boundedParameter(channel.Name, 140) {
				result.Channels = append(result.Channels, channel)
			}
		}
		return result, nil
	}
	if s.Kind != "platform_supply" || s.Type != 1 && s.Type != 2 {
		return sf.ShippingOptions{}, f.ErrInvalid
	}
	query := url.Values{"orderType": {strconv.Itoa(s.Type)}, "orderNoList": {s.ID}, "includeSharedAddr": {"2"}}
	if selection.AddressID > 0 {
		query.Set("addressId", strconv.FormatInt(selection.AddressID, 10))
	}
	raw, err := c.goodsRequest(ctx, http.MethodGet, "/open-api/shipping/basic", query.Encode(), k, nil)
	if err != nil {
		return sf.ShippingOptions{}, f.ErrUnavailable
	}
	options, err := observationInfo[sf.ShippingOptions](raw, false)
	if err != nil || len(options.Addresses) > 500 || len(options.Warehouses) > 500 || len(options.DeliveryTypes) > 30 {
		return sf.ShippingOptions{}, f.ErrUnavailable
	}
	return options, nil
}

func (c *OfficialClient) BuildFulfillmentPlan(ctx context.Context, k storecenter.OfficialMerchantCredential, s sf.Source, delivery sf.Delivery, selection sf.Selection) (sf.Plan, error) {
	empty := sf.Plan{}
	if err := s.Eligible(time.Now()); err != nil {
		return empty, err
	}
	if selection.Purpose == "accept_address" || selection.Purpose == "read_address" {
		if s.Kind != "consumer_order" || s.SelectedLogistics != 2 || selection.Purpose == "accept_address" && s.Status != 1 {
			return empty, f.ErrConflict
		}
		handle := 1
		if selection.Purpose == "accept_address" {
			handle = 2
		}
		raw, _ := json.Marshal(map[string]any{"orderNo": s.ID, "handleType": handle})
		return sf.Plan{Purpose: selection.Purpose, Body: raw}, nil
	}
	if delivery.Name == "" || delivery.PackageID == "" || len(delivery.Lines) == 0 || len(delivery.Lines) > 20 {
		return empty, f.ErrInvalid
	}
	original := map[string]sf.Line{}
	for _, line := range s.FulfillableLines() {
		original[line.ID] = line
	}
	seen := map[string]bool{}
	for _, line := range delivery.Lines {
		source, ok := original[line.SourceLine]
		if !ok || seen[line.SourceLine] || line.Qty < 1 || line.Qty > source.Quantity-source.Delivered {
			return empty, f.ErrConflict
		}
		seen[line.SourceLine] = true
	}
	options, err := c.QueryFulfillmentOptions(ctx, k, s, selection)
	if err != nil {
		return empty, err
	}
	if selection.Purpose == "consumer_tracking" {
		if s.Kind != "consumer_order" || s.SelectedLogistics != 2 || s.Status != 2 && s.Status != 7 || !boundedParameter(selection.Tracking, 100) {
			return empty, f.ErrConflict
		}
		var channel sf.Channel
		for _, entry := range options.Channels {
			if entry.Code == selection.Channel {
				channel = entry
			}
		}
		if channel.Code == "" {
			return empty, f.ErrConflict
		}
		rows := []map[string]any{}
		ids := []string{}
		for _, line := range delivery.Lines {
			goods := original[line.SourceLine]
			if line.Qty != 1 || !observationDigits.MatchString(line.SourceLine) || goods.GoodsStatus != 2 && goods.GoodsStatus != 7 {
				return empty, f.ErrInvalid
			}
			rows = append(rows, map[string]any{"goodsId": json.Number(line.SourceLine), "expressCode": selection.Tracking, "expressIdCode": channel.Name, "expressChannelCode": channel.Code, "status": 2})
			ids = append(ids, line.SourceLine)
		}
		raw, _ := json.Marshal(map[string]any{"orderNo": s.ID, "infoList": rows})
		return sf.Plan{Purpose: selection.Purpose, Body: raw, GoodsIDs: ids}, nil
	}
	if selection.Purpose != "supply_shipping" || s.Kind != "platform_supply" {
		return empty, f.ErrInvalid
	}
	// A complete original order avoids SHEIN partial-shipment cancellation/short-delivery semantics.
	if len(delivery.Lines) != len(original) {
		return empty, f.ErrConflict
	}
	for _, line := range delivery.Lines {
		if line.Qty != original[line.SourceLine].Quantity-original[line.SourceLine].Delivered {
			return empty, f.ErrConflict
		}
	}
	if selection.ShippingRoute != 0 || options.Routes == "" || !strings.Contains(","+strings.Trim(options.Routes, "[] ")+",", ",0,") || options.PackageType != 1 && options.PackageType != 2 {
		return empty, f.ErrUnavailable
	}
	allowedType, address, warehouse, motorcade := false, false, false, false
	for _, entry := range options.DeliveryTypes {
		if entry.Value == selection.DeliveryType {
			allowedType = true
		}
	}
	for _, entry := range options.Addresses {
		if entry.ID == selection.AddressID {
			address = true
		}
	}
	for _, entry := range options.Warehouses {
		if entry.ID == selection.WarehouseID {
			warehouse = true
		}
	}
	for _, entry := range options.Motorcades {
		if entry.ID == selection.Motorcade {
			motorcade = true
		}
	}
	if !allowedType || !address || !warehouse {
		return empty, f.ErrConflict
	}
	// This first transport supports the native direct-delivery choices. Other choices stay closed.
	if selection.DeliveryType != 4 && selection.DeliveryType != 21 && (selection.DeliveryType != 2 || !motorcade) {
		return empty, f.ErrUnavailable
	}
	arrival, err := time.ParseInLocation("2006-01-02 15:04:05", selection.ArrivalTime, time.FixedZone("SHEIN", 8*3600))
	if err != nil || !time.Now().Before(arrival) || arrival.After(s.Deadline) {
		return empty, f.ErrConflict
	}
	if selection.DeliveryType == 2 {
		reserve, err := time.ParseInLocation("2006-01-02 15:04:05", selection.ReserveTime, time.FixedZone("SHEIN", 8*3600))
		if err != nil || !time.Now().Before(reserve) || reserve.After(arrival) {
			return empty, f.ErrConflict
		}
	}
	checkBody, _ := json.Marshal(map[string]any{"sellerOrderNos": []string{s.ID}})
	checked, err := c.goodsPost(ctx, "/open-api/purchase/intelligent-packing-result", k, checkBody)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	grouping, err := observationInfo[struct {
		Groups []struct {
			Orders []struct {
				ID    string `json:"sellerOrderNo"`
				State int    `json:"state"`
			} `json:"orders"`
		} `json:"groups"`
		Undeliverable []json.RawMessage `json:"undeliverableOrders"`
	}](checked, false)
	if err != nil || len(grouping.Undeliverable) != 0 || len(grouping.Groups) != 1 || len(grouping.Groups[0].Orders) != 1 || grouping.Groups[0].Orders[0].ID != s.ID || grouping.Groups[0].Orders[0].State != 2 {
		return empty, f.ErrConflict
	}
	query := url.Values{"addressId": {strconv.FormatInt(selection.AddressID, 10)}, "orderType": {strconv.Itoa(s.Type)}, "sendType": {strconv.Itoa(selection.DeliveryType)}, "orderNoList": {s.ID}, "shippingRoute": {"0"}}
	if selection.DeliveryType == 2 {
		query.Set("coding", selection.Motorcade)
	}
	raw, err := c.goodsRequest(ctx, http.MethodGet, "/open-api/shipping/warehouse", query.Encode(), k, nil)
	if err != nil {
		return empty, f.ErrUnavailable
	}
	target, err := observationInfo[struct {
		ID int64 `json:"subWareHouseId"`
	}](raw, false)
	if err != nil || target.ID < 1 {
		return empty, f.ErrConflict
	}
	rows := []map[string]any{}
	quantities := map[string]int64{}
	for _, line := range delivery.Lines {
		quantities[line.SourceLine] = line.Qty
	}
	for _, line := range s.Lines {
		if line.Quantity > 0 {
			rows = append(rows, map[string]any{"orderNo": s.ID, "skuCode": line.SKU, "orderQuantity": line.Quantity, "tempDeliveryQuantity": quantities[line.ID], "packageNum": 1})
		}
	}
	info := map[string]any{"addrId": selection.AddressID, "packageNumber": 1}
	if selection.ReserveTime != "" {
		info["reserveParcelTime"] = selection.ReserveTime
	}
	body := map[string]any{"adminName": "Shuomi ERP", "list": rows, "orderType": s.Type, "deliveryType": selection.DeliveryType, "arrivalTime": selection.ArrivalTime, "shippingRoute": 0, "expressInfo": []any{info}, "supplierWarehouseId": selection.WarehouseID, "subWarehouseId": target.ID}
	if selection.DeliveryType == 2 {
		body["coding"] = selection.Motorcade
	}
	raw, _ = json.Marshal(body)
	return sf.Plan{Purpose: selection.Purpose, Body: raw}, nil
}

func (c *OfficialClient) ExecuteFulfillmentPlan(ctx context.Context, k storecenter.OfficialMerchantCredential, plan sf.Plan) (sf.MutationResult, error) {
	result := sf.MutationResult{Result: map[string]any{}}
	path := ""
	switch plan.Purpose {
	case "accept_address", "read_address":
		path = "/open-api/order/export-address"
	case "consumer_tracking":
		path = "/open-api/order/import-batch-multiple-express"
	case "supply_shipping":
		path = "/open-api/shipping/orderToShipping"
	default:
		return result, f.ErrInvalid
	}
	if !json.Valid(plan.Body) || len(plan.Body) > 256*1024 {
		return result, f.ErrInvalid
	}
	raw, err := c.goodsPost(ctx, path, k, plan.Body)
	if err != nil {
		return result, f.ErrUnknown
	}
	if plan.Purpose == "consumer_tracking" {
		ids, err := trackingReceipt(raw, plan.GoodsIDs)
		if err != nil {
			return result, err
		}
		result.Result["accepted_goods_ids"] = ids
		return result, nil
	}
	if plan.Purpose == "supply_shipping" {
		info, err := observationInfo[struct {
			ID string `json:"deliveryCode"`
		}](raw, false)
		if err != nil || !boundedParameter(info.ID, 140) {
			return result, f.ErrUnknown
		}
		shipment, err := c.QueryFulfillmentShipment(ctx, k, info.ID)
		if err != nil || validateSupplyShipment(shipment, plan.Body) != nil {
			return result, f.ErrUnknown
		}
		result.Result["shipping_id"] = info.ID
		return result, nil
	}
	var requested struct {
		Order string `json:"orderNo"`
	}
	_ = json.Unmarshal(plan.Body, &requested)
	info, err := observationInfo[struct {
		Addresses []struct {
			Order    string `json:"orderNo"`
			First    string `json:"firstName"`
			Middle   string `json:"middleName"`
			Last     string `json:"lastName"`
			Country  string `json:"country"`
			Province string `json:"province"`
			City     string `json:"city"`
			District string `json:"district"`
			Street   string `json:"street"`
			Address  string `json:"address"`
			Extra    string `json:"addressExt"`
			Phone    string `json:"phone"`
			Postcode string `json:"postCode"`
		} `json:"receiveMsgList"`
	}](raw, false)
	if err != nil || len(info.Addresses) != 1 || info.Addresses[0].Order != requested.Order || info.Addresses[0].Country != "United States" || info.Addresses[0].Address == "" {
		return result, f.ErrUnknown
	}
	private, _ := json.Marshal(info.Addresses[0])
	_ = json.Unmarshal(private, &result.Private)
	return result, nil
}
