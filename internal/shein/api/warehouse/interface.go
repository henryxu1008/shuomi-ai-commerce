package warehouse

// Warehouse 仓库信息
type Warehouse struct {
	WarehouseName   string   `json:"warehouse_name"`
	WarehouseCode   string   `json:"warehouse_code"`
	SaleCountryList []string `json:"sale_country_list"`
	WarehouseType   int      `json:"warehouse_type"`
}

// WarehouseResponse 仓库信息响应
type WarehouseResponse struct {
	Data []Warehouse `json:"data"`
	Meta struct {
		Count     int `json:"count"`
		CustomObj any `json:"customObj"`
	} `json:"meta"`
}

type StoreAddressListRequest struct {
	AddressType int `json:"addressType"`
}

type StoreAddress struct {
	ID                    int64           `json:"id"`
	Country               string          `json:"country"`
	CountryID             int             `json:"countryId"`
	State                 string          `json:"state"`
	StateID               int             `json:"stateId"`
	City                  string          `json:"city"`
	CityID                int             `json:"cityId"`
	District              string          `json:"district"`
	DistrictID            int             `json:"districtId"`
	PostCode              string          `json:"postCode"`
	Address1              string          `json:"address1"`
	Address2              string          `json:"address2"`
	FirstName             string          `json:"firstName"`
	LastName              string          `json:"lastName"`
	FullName              string          `json:"fullName"`
	Phone                 string          `json:"phone"`
	FullAddress           string          `json:"fullAddress"`
	AddressType           int             `json:"addressType"`
	CollectionPatternType int             `json:"collectionPatternType"`
	SellerEmail           string          `json:"sellerEmail"`
	WarehouseName         string          `json:"warehouseName"`
	WarehouseCode         string          `json:"warehouseCode"`
	WarehouseType         int             `json:"warehouseType"`
	CollectionMark        int             `json:"collectionMark"`
	StoreSiteInfos        []StoreSiteInfo `json:"storeSiteInfos"`
}

type StoreSiteInfo struct {
	Site             string   `json:"site"`
	SiteStatus       int      `json:"siteStatus"`
	DefaultWarehouse int      `json:"defaultWarehouse"`
	SaleCountries    []string `json:"saleCountries"`
}

type StoreAddressListInfo struct {
	Addresses []StoreAddress `json:"addresses"`
}
