package rules

import (
	"strconv"

	"task-processor/internal/marketplace/temu/model"
)

// Units are explicit draft facts. They are deliberately outside the E17 wire
// packageInfo, which has no unit fields, and must match the exact E24 profile.
type LocalPackageUnits struct {
	Weight    string `json:"weight"`
	Dimension string `json:"dimension"`
}
type LocalNumericalIssue struct {
	Field string `json:"field"`
	Code  string `json:"code"`
}
type LocalNumericalCheck struct {
	Target LocalTarget           `json:"target"`
	Issues []LocalNumericalIssue `json:"issues"`
}

// CheckLocalNumbers is only the E17/E24 numerical check. It does not grant
// publishing readiness, validate merchant/category/asset/consent authority,
// translate currencies or units, or modify the caller's supplied facts.
func (c *Catalog) CheckLocalNumbers(partition string, regionID int64, skus []model.LocalSKU, units LocalPackageUnits) LocalNumericalCheck {
	view := LocalNumericalCheck{Target: c.LocalTarget(partition, regionID), Issues: []LocalNumericalIssue{}}
	issue := func(field, code string) { view.Issues = append(view.Issues, LocalNumericalIssue{field, code}) }
	for _, code := range view.Target.Issues {
		issue("target", string(code))
	}
	if len(skus) < 1 || len(skus) > 500 {
		issue("skuList", "sku_count_invalid")
		return view
	}
	if len(view.Target.Issues) != 0 {
		return view
	}
	spec := view.Target.Specification
	if units.Weight != spec.Weight.Unit {
		issue("packageUnits.weight", "unit_mismatch")
	}
	if units.Dimension != spec.Dimension.Unit {
		issue("packageUnits.dimension", "unit_mismatch")
	}
	for index, sku := range skus {
		path := "skuList." + strconv.Itoa(index)
		if sku.Quantity == nil || *sku.Quantity < 0 || *sku.Quantity > 1000000 {
			issue(path+".quantity", "quantity_invalid")
		}
		if sku.Price.BasePrice.Currency != spec.BaseCurrency {
			issue(path+".price.basePrice.currency", "currency_mismatch")
		}
		baseValid := spec.BasePrice.Allows(sku.Price.BasePrice.Amount)
		if !baseValid {
			issue(path+".price.basePrice.amount", "number_invalid")
		}
		if retail := sku.Price.ListPrice; retail != nil {
			if retail.Currency != spec.RetailCurrency {
				issue(path+".price.listPrice.currency", "currency_mismatch")
			}
			retailValid := spec.RetailPrice.Allows(retail.Amount)
			if !retailValid {
				issue(path+".price.listPrice.amount", "number_invalid")
			}
			if baseValid && retailValid && sku.Price.BasePrice.Currency == spec.BaseCurrency && retail.Currency == spec.RetailCurrency && spec.BaseCurrency == spec.RetailCurrency {
				base, _, _ := exactDecimal(sku.Price.BasePrice.Amount)
				retailValue, _, _ := exactDecimal(retail.Amount)
				if retailValue.Cmp(base) <= 0 {
					issue(path+".price.listPrice.amount", "retail_not_above_base")
				}
			}
		}
		if !spec.Weight.Number.Allows(sku.PackageInfo.Weight) {
			issue(path+".packageInfo.weight", "number_invalid")
		}
		for _, dimension := range []struct{ name, value string }{{"length", sku.PackageInfo.Length}, {"width", sku.PackageInfo.Width}, {"height", sku.PackageInfo.Height}} {
			if !spec.Dimension.Number.Allows(dimension.value) {
				issue(path+".packageInfo."+dimension.name, "number_invalid")
			}
		}
	}
	return view
}
