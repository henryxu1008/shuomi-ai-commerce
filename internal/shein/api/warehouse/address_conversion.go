package warehouse

func WarehousesFromStoreAddresses(info StoreAddressListInfo) *WarehouseResponse {
	resp := &WarehouseResponse{
		Data: make([]Warehouse, 0, len(info.Addresses)),
	}

	for _, address := range info.Addresses {
		if address.WarehouseCode == "" {
			continue
		}

		resp.Data = append(resp.Data, Warehouse{
			WarehouseName:   address.WarehouseName,
			WarehouseCode:   address.WarehouseCode,
			SaleCountryList: collectSaleCountries(address.StoreSiteInfos),
			WarehouseType:   address.WarehouseType,
		})
	}

	resp.Meta.Count = len(resp.Data)
	return resp
}

func collectSaleCountries(siteInfos []StoreSiteInfo) []string {
	if len(siteInfos) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(siteInfos))
	countries := make([]string, 0, len(siteInfos))
	for _, siteInfo := range siteInfos {
		for _, country := range siteInfo.SaleCountries {
			if country == "" {
				continue
			}
			if _, ok := seen[country]; ok {
				continue
			}
			seen[country] = struct{}{}
			countries = append(countries, country)
		}
	}

	return countries
}
