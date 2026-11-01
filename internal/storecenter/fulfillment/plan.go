package fulfillment

import "encoding/json"

type Delivery struct {
	Name      string `json:"name"`
	PackageID string `json:"package_id"`
	Lines     []struct {
		SourceLine string `json:"source_line"`
		Qty        int64  `json:"qty"`
	} `json:"lines"`
}
type Selection struct {
	Purpose       string `json:"purpose"`
	Demand        string `json:"demand"`
	Delivery      string `json:"delivery,omitempty"`
	Channel       string `json:"channel,omitempty"`
	Tracking      string `json:"tracking,omitempty"`
	AddressID     int64  `json:"address_id,omitempty"`
	WarehouseID   int64  `json:"warehouse_id,omitempty"`
	DeliveryType  int    `json:"delivery_type,omitempty"`
	ShippingRoute int    `json:"shipping_route"`
	ArrivalTime   string `json:"arrival_time,omitempty"`
	ReserveTime   string `json:"reserve_time,omitempty"`
	Motorcade     string `json:"motorcade,omitempty"`
}
type Plan struct {
	Purpose  string          `json:"purpose"`
	Body     json.RawMessage `json:"body"`
	GoodsIDs []string        `json:"goods_ids,omitempty"`
}
type MutationResult struct {
	Result  map[string]any
	Private map[string]any
}
type SupplyShipment struct {
	ID              string  `json:"deliveryCode"`
	DeliveryType    int     `json:"deliveryType"`
	Warehouse       int64   `json:"supplierWarehouseId"`
	Packages        *int64  `json:"sendPackage"`
	Tracking        string  `json:"expressCode"`
	Carrier         string  `json:"expressCompanyName"`
	Created         *string `json:"addTime"`
	Pickup          *string `json:"takeParcelTime"`
	ExpectedArrival *string `json:"preReceiptTime"`
	Arrival         *string `json:"receiptTime"`
	Lines           []struct {
		Order string `json:"orderNo"`
		SKU   string `json:"skuCode"`
		Qty   *int64 `json:"deliveryQuantity"`
	} `json:"deliveryOrderDataList"`
}
type Channel struct {
	Site string `json:"site"`
	Name string `json:"expressIdCode"`
	Code string `json:"expressChannelCode"`
}
type ShippingOptions struct {
	Channels  []Channel `json:"channels,omitempty"`
	Addresses []struct {
		ID   int64  `json:"addressId"`
		Name string `json:"addressName"`
	} `json:"addressList,omitempty"`
	DeliveryTypes []struct {
		Name  string `json:"deliveryTypeName"`
		Value int    `json:"deliveryTypeValue"`
	} `json:"deliveryTypeList,omitempty"`
	Warehouses []struct {
		ID   int64  `json:"supplierWarehouseId"`
		Name string `json:"warehouseName"`
	} `json:"supplierWarehouseList,omitempty"`
	Motorcades []struct {
		ID   string `json:"motorcadeId"`
		Name string `json:"motorcadeName"`
	} `json:"motorcadeList,omitempty"`
	PackageType int    `json:"packageType,omitempty"`
	Routes      string `json:"shippingRouteList,omitempty"`
}
