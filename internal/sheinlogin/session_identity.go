package sheinlogin

import (
	"context"
	"strconv"

	other "task-processor/internal/shein/api/other"
	warehouse "task-processor/internal/shein/api/warehouse"
	"task-processor/internal/shein/client"
)

type sessionIdentityReader struct{}

func NewSessionIdentityReader() SessionIdentityReader { return sessionIdentityReader{} }
func explicitTransport(a Account, state map[string]any) (*client.SessionTransport, error) {
	cookies, e := client.CookiesFromBrowserState(state)
	if e != nil {
		return nil, e
	}
	return client.NewSessionTransport(a.LoginURL, a.Proxy, cookies)
}
func (sessionIdentityReader) Merchant(ctx context.Context, a Account, state map[string]any) (string, error) {
	t, e := explicitTransport(a, state)
	if e != nil {
		return "", e
	}
	defer t.Close()
	var response other.SupplierOperateInfoResponse
	if e = t.Read(ctx, client.GetSupplierOperateInfoEndpoint(), nil, &response); e != nil || response.Code != "0" || response.Info.StoreID <= 0 {
		return "", ErrUnavailable
	}
	return strconv.FormatInt(response.Info.StoreID, 10), nil
}
func (sessionIdentityReader) Warehouses(ctx context.Context, a Account, state map[string]any) ([]WarehouseOption, error) {
	t, e := explicitTransport(a, state)
	if e != nil {
		return nil, e
	}
	defer t.Close()
	var response struct {
		Code string                      `json:"code"`
		Info warehouse.WarehouseResponse `json:"info"`
	}
	e = t.Read(ctx, client.GetWarehousesEndpoint(), nil, &response)
	if e != nil || response.Code != "0" || len(response.Info.Data) == 0 {
		var addresses struct {
			Code string                         `json:"code"`
			Info warehouse.StoreAddressListInfo `json:"info"`
		}
		if e = t.Read(ctx, client.GetStoreAddressListEndpoint(), warehouse.StoreAddressListRequest{AddressType: 2}, &addresses); e != nil || addresses.Code != "0" {
			return nil, ErrUnavailable
		}
		converted := warehouse.WarehousesFromStoreAddresses(addresses.Info)
		if len(converted.Data) == 0 {
			return nil, ErrUnavailable
		}
		return warehouseOptions(*converted), nil
	}
	return warehouseOptions(response.Info), nil
}
func warehouseOptions(response warehouse.WarehouseResponse) []WarehouseOption {
	var result []WarehouseOption
	for _, w := range response.Data {
		result = append(result, WarehouseOption{w.WarehouseCode, w.WarehouseName, append([]string(nil), w.SaleCountryList...), w.WarehouseType})
	}
	return result
}
