package rules

import (
	"encoding/json"
	"strconv"
	"strings"

	"task-processor/internal/marketplace/temu/model"
)

type SupplierPriceIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}
type SupplierPriceCheck struct {
	Issues []SupplierPriceIssue `json:"issues"`
}

// CheckSupplierPriceLayout checks E13/E27's mode-specific wire representation.
// The caller must obtain mode from verified Store identity. This check proves
// neither current merchant currency/limits, site/warehouse eligibility, category,
// assets nor readiness, and never converts, computes or fills prices/stock.
func CheckSupplierPriceLayout(mode string, input model.SupplierGoods) SupplierPriceCheck {
	check := SupplierPriceCheck{Issues: []SupplierPriceIssue{}}
	issue := func(field, code string) { check.Issues = append(check.Issues, SupplierPriceIssue{field, code}) }
	if mode != "fully_managed" && mode != "semi_managed" {
		issue("mode", "supplier_mode_unverified")
		return check
	}
	raw, err := json.Marshal(input)
	if err != nil || len(raw) > 2<<20 {
		issue("product", "input_too_large")
		return check
	}
	semi := mode == "semi_managed"
	bound := map[int64]bool{}
	var sites []int64
	if semi {
		if input.SemiManaged == nil || len(input.SemiManaged.SiteIDs) == 0 {
			issue("productSemiManagedReq.bindSiteIds", "sites_missing")
		} else {
			sites = input.SemiManaged.SiteIDs
			for i, id := range sites {
				path := "productSemiManagedReq.bindSiteIds." + strconv.Itoa(i)
				if id < 1 {
					issue(path, "site_invalid")
				} else if bound[id] {
					issue(path, "site_duplicated")
				}
				bound[id] = true
			}
		}
	} else if input.SemiManaged != nil {
		issue("productSemiManagedReq", "mode_field_forbidden")
	}
	if len(input.SKCs) == 0 {
		issue("productSkcReqs", "skcs_missing")
	}
	for skcIndex, skc := range input.SKCs {
		base := "productSkcReqs." + strconv.Itoa(skcIndex) + ".productSkuReqs"
		if len(skc.SKUs) == 0 {
			issue(base, "skus_missing")
		}
		for skuIndex, sku := range skc.SKUs {
			path := base + "." + strconv.Itoa(skuIndex)
			if sku.Currency != "CNY" && sku.Currency != "USD" {
				issue(path+".currencyType", "currency_invalid")
			}
			if semi {
				if sku.SupplierPrice != nil {
					issue(path+".supplierPrice", "mode_field_forbidden")
				}
				seen := map[int64]bool{}
				for priceIndex, price := range sku.SitePrices {
					field := path + ".siteSupplierPrices." + strconv.Itoa(priceIndex)
					if price.SiteID == nil || *price.SiteID < 1 || !bound[*price.SiteID] {
						issue(field, "price_site_not_bound")
						continue
					}
					if seen[*price.SiteID] {
						issue(field, "price_site_duplicated")
					}
					seen[*price.SiteID] = true
					if price.Price == nil {
						issue(field+".supplierPrice", "price_missing")
					} else if *price.Price < 0 {
						issue(field+".supplierPrice", "price_invalid")
					}
				}
				for _, id := range sites {
					if !seen[id] {
						issue(path+".siteSupplierPrices", "bound_site_price_missing")
					}
				}
				if sku.Stock == nil || len(sku.Stock.Warehouses) == 0 {
					issue(path+".productSkuStockQuantityReq", "stock_missing")
				}
			} else {
				if sku.SupplierPrice == nil {
					issue(path+".supplierPrice", "price_missing")
				} else if *sku.SupplierPrice < 0 {
					issue(path+".supplierPrice", "price_invalid")
				}
				if len(sku.SitePrices) != 0 {
					issue(path+".siteSupplierPrices", "mode_field_forbidden")
				}
			}
			if sku.Stock != nil {
				seen := map[string]bool{}
				for stockIndex, stock := range sku.Stock.Warehouses {
					field := path + ".productSkuStockQuantityReq.warehouseStockQuantityReqs." + strconv.Itoa(stockIndex)
					if strings.TrimSpace(stock.WarehouseID) == "" {
						issue(field+".warehouseId", "warehouse_missing")
					} else if seen[stock.WarehouseID] {
						issue(field+".warehouseId", "warehouse_duplicated")
					}
					seen[stock.WarehouseID] = true
					if stock.Target == nil {
						issue(field+".targetStockAvailable", "stock_missing")
					} else if *stock.Target < 0 {
						issue(field+".targetStockAvailable", "stock_invalid")
					}
					if stock.Current != nil && *stock.Current < 0 {
						issue(field+".currentStockAvailable", "stock_invalid")
					}
				}
			}
		}
	}
	return check
}
