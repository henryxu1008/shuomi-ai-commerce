package warehouse

import (
	"reflect"
	"testing"
)

func TestWarehousesFromStoreAddressesPreservesCurrentValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		info StoreAddressListInfo
		want []Warehouse
	}{
		{name: "empty input keeps nonnil data", want: []Warehouse{}},
		{name: "empty warehouse code is omitted", info: StoreAddressListInfo{Addresses: []StoreAddress{{WarehouseName: "ignored"}}}, want: []Warehouse{}},
		{
			name: "values order duplicates and country order",
			info: StoreAddressListInfo{Addresses: []StoreAddress{
				{WarehouseName: "first", WarehouseCode: "WH-1", WarehouseType: 3, StoreSiteInfos: []StoreSiteInfo{{SaleCountries: []string{"US", "", "CA", "US"}}, {SaleCountries: []string{"CA", "MX"}}}},
				{WarehouseName: "ignored"},
				{WarehouseName: "second", WarehouseCode: "WH-2", WarehouseType: 2},
				{WarehouseName: "duplicate code retained", WarehouseCode: "WH-1", WarehouseType: 1, StoreSiteInfos: []StoreSiteInfo{{}}},
			}},
			want: []Warehouse{
				{WarehouseName: "first", WarehouseCode: "WH-1", WarehouseType: 3, SaleCountryList: []string{"US", "CA", "MX"}},
				{WarehouseName: "second", WarehouseCode: "WH-2", WarehouseType: 2},
				{WarehouseName: "duplicate code retained", WarehouseCode: "WH-1", WarehouseType: 1, SaleCountryList: []string{}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := WarehousesFromStoreAddresses(tc.info)
			if got == nil || !reflect.DeepEqual(got.Data, tc.want) || got.Meta.Count != len(tc.want) || got.Meta.CustomObj != nil {
				t.Fatalf("warehouse conversion = %#v, want data %#v and count %d", got, tc.want, len(tc.want))
			}
		})
	}
}

func TestCollectSaleCountriesPreservesCurrentZeroValuesAndOrder(t *testing.T) {
	for _, tc := range []struct {
		name string
		info []StoreSiteInfo
		want []string
	}{
		{name: "no sites returns nil"},
		{name: "empty sites returns nil", info: []StoreSiteInfo{}},
		{name: "present site with no countries returns empty", info: []StoreSiteInfo{{}}, want: []string{}},
		{name: "empty countries are ignored", info: []StoreSiteInfo{{SaleCountries: []string{"", ""}}}, want: []string{}},
		{name: "first occurrence order", info: []StoreSiteInfo{{SaleCountries: []string{"CA", "US", "CA"}}, {SaleCountries: []string{"", "MX", "US"}}}, want: []string{"CA", "US", "MX"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := collectSaleCountries(tc.info); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("country conversion = %#v, want %#v", got, tc.want)
			}
		})
	}
}
