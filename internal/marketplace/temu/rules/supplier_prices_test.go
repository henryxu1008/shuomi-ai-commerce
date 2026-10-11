package rules_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
)

func supplierPriceFixture(semi bool) model.SupplierGoods {
	sku := model.SupplierSKU{Currency: "CNY", SupplierPrice: variationPointer(int64(9007199254740993))}
	input := model.SupplierGoods{SKCs: []model.SupplierSKC{{SKUs: []model.SupplierSKU{sku}}}}
	if semi {
		input.SemiManaged = &model.SupplierSemiManaged{SiteIDs: []int64{103, 104}}
		input.SKCs[0].SKUs[0].SupplierPrice = nil
		input.SKCs[0].SKUs[0].SitePrices = []model.SupplierSitePrice{{SiteID: variationPointer(int64(104)), Price: variationPointer(int64(3))}, {SiteID: variationPointer(int64(103)), Price: variationPointer(int64(2))}}
		input.SKCs[0].SKUs[0].Stock = &model.SupplierStock{Warehouses: []model.SupplierWarehouseStock{{WarehouseID: "official-wh", Target: variationPointer(int64(0))}}}
	}
	return input
}

func TestSupplierPriceLayoutSeparatesVerifiedModesWithoutUnitConversion(t *testing.T) {
	for _, mode := range []string{"fully_managed", "semi_managed"} {
		input := supplierPriceFixture(mode == "semi_managed")
		before, err := json.Marshal(input)
		require.NoError(t, err)
		require.Empty(t, rules.CheckSupplierPriceLayout(mode, input).Issues)
		after, err := json.Marshal(input)
		require.NoError(t, err)
		require.Equal(t, before, after)
		if mode == "fully_managed" {
			require.Contains(t, string(after), `"supplierPrice":9007199254740993`, "wire integers must not pass through float64")
			require.NotContains(t, string(after), "siteSupplierPrices")
		} else {
			require.Contains(t, string(after), `"targetStockAvailable":0`, "supplied zero is not missing inventory")
		}
		result, err := json.Marshal(rules.CheckSupplierPriceLayout(mode, input))
		require.NoError(t, err)
		require.NotContains(t, string(result), "ready", "layout does not prove merchant currency, price limits, warehouse or authorization")
	}
}

func TestSupplierPriceLayoutRejectsWrongModeFieldsAndUnpricedSites(t *testing.T) {
	for _, tc := range []struct {
		name, mode, field, code string
		change                  func(*model.SupplierGoods)
	}{
		{"unknown mode", "local", "mode", "supplier_mode_unverified", nil},
		{"no currency default", "fully_managed", "productSkcReqs.0.productSkuReqs.0.currencyType", "currency_invalid", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].Currency = "" }},
		{"local currency is not supplier currency", "fully_managed", "productSkcReqs.0.productSkuReqs.0.currencyType", "currency_invalid", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].Currency = "CAD" }},
		{"missing full price", "fully_managed", "productSkcReqs.0.productSkuReqs.0.supplierPrice", "price_missing", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].SupplierPrice = nil }},
		{"full must not use site prices", "fully_managed", "productSkcReqs.0.productSkuReqs.0.siteSupplierPrices", "mode_field_forbidden", func(p *model.SupplierGoods) {
			p.SKCs[0].SKUs[0].SitePrices = []model.SupplierSitePrice{{SiteID: variationPointer(int64(100)), Price: variationPointer(int64(2))}}
		}},
		{"full must not fabricate targets", "fully_managed", "productSemiManagedReq", "mode_field_forbidden", func(p *model.SupplierGoods) { p.SemiManaged = &model.SupplierSemiManaged{SiteIDs: []int64{100}} }},
		{"semi requires site set", "semi_managed", "productSemiManagedReq.bindSiteIds", "sites_missing", func(p *model.SupplierGoods) { p.SemiManaged = nil }},
		{"semi must not use full price", "semi_managed", "productSkcReqs.0.productSkuReqs.0.supplierPrice", "mode_field_forbidden", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].SupplierPrice = variationPointer(int64(2)) }},
		{"unpriced bound site", "semi_managed", "productSkcReqs.0.productSkuReqs.0.siteSupplierPrices", "bound_site_price_missing", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].SitePrices = p.SKCs[0].SKUs[0].SitePrices[:1] }},
		{"duplicate site price", "semi_managed", "productSkcReqs.0.productSkuReqs.0.siteSupplierPrices.1", "price_site_duplicated", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].SitePrices[1].SiteID = variationPointer(int64(104)) }},
		{"foreign site price", "semi_managed", "productSkcReqs.0.productSkuReqs.0.siteSupplierPrices.1", "price_site_not_bound", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].SitePrices[1].SiteID = variationPointer(int64(100)) }},
		{"missing initial stock", "semi_managed", "productSkcReqs.0.productSkuReqs.0.productSkuStockQuantityReq", "stock_missing", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].Stock = nil }},
		{"missing stock not zero", "semi_managed", "productSkcReqs.0.productSkuReqs.0.productSkuStockQuantityReq.warehouseStockQuantityReqs.0.targetStockAvailable", "stock_missing", func(p *model.SupplierGoods) { p.SKCs[0].SKUs[0].Stock.Warehouses[0].Target = nil }},
		{"negative stock", "semi_managed", "productSkcReqs.0.productSkuReqs.0.productSkuStockQuantityReq.warehouseStockQuantityReqs.0.targetStockAvailable", "stock_invalid", func(p *model.SupplierGoods) {
			p.SKCs[0].SKUs[0].Stock.Warehouses[0].Target = variationPointer(int64(-1))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := supplierPriceFixture(tc.mode == "semi_managed")
			if tc.change != nil {
				tc.change(&input)
			}
			require.Contains(t, rules.CheckSupplierPriceLayout(tc.mode, input).Issues, rules.SupplierPriceIssue{Field: tc.field, Code: tc.code})
		})
	}
}
