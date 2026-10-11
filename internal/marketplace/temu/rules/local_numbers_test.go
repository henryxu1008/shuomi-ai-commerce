package rules_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
)

func localNumberSKU() model.LocalSKU {
	quantity := int64(0)
	return model.LocalSKU{
		Price:       model.LocalSKUPrice{BasePrice: model.LocalMoney{Amount: "2.00", Currency: "USD"}, ListPrice: &model.LocalMoney{Amount: "2.01", Currency: "USD"}},
		Quantity:    &quantity,
		PackageInfo: model.LocalPackageInfo{Weight: "0.1", Length: "0.1", Width: "0.1", Height: "0.1"},
	}
}

func TestLocalNumericalChecksUseOriginalRegionCurrencyAndExplicitUnits(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	sku := localNumberSKU()
	units := rules.LocalPackageUnits{Weight: "lb", Dimension: "in"}
	view := c.CheckLocalNumbers("US", 211, []model.LocalSKU{sku}, units)
	require.Empty(t, view.Issues, "explicit inventory zero is a supplied fact")
	require.Equal(t, int64(100), view.Target.Region.SiteID)
	require.Equal(t, c.Revision(), view.Target.SnapshotRevision)
	for _, tc := range []struct {
		name, partition string
		region          int64
		units           rules.LocalPackageUnits
		change          func(*model.LocalSKU)
		field, code     string
	}{
		{"wrong currency", "US", 211, units, func(s *model.LocalSKU) { s.Price.BasePrice.Currency = "EUR" }, "skuList.0.price.basePrice.currency", "currency_mismatch"},
		{"no currency default", "US", 211, units, func(s *model.LocalSKU) { s.Price.BasePrice.Currency = "" }, "skuList.0.price.basePrice.currency", "currency_mismatch"},
		{"no unit conversion", "US", 211, rules.LocalPackageUnits{Weight: "g", Dimension: "cm"}, nil, "packageUnits.weight", "unit_mismatch"},
		{"no unit default", "US", 211, rules.LocalPackageUnits{}, nil, "packageUnits.dimension", "unit_mismatch"},
		{"no routing from currency", "GLOBAL", 211, units, nil, "target", string(rules.PartitionMismatch)},
		{"Georgia missing specification", "EU", 75, units, nil, "target", string(rules.SpecificationsMissing)},
		{"Armenia missing specification", "EU", 10, units, nil, "target", string(rules.SpecificationsMissing)},
		{"blank opening", "EU", 174, units, nil, "target", string(rules.LocalOpeningUnverified)},
		{"Canada minimum", "GLOBAL", 37, rules.LocalPackageUnits{Weight: "g", Dimension: "cm"}, func(s *model.LocalSKU) {
			s.Price.BasePrice = model.LocalMoney{Amount: "1.99", Currency: "CAD"}
			s.Price.ListPrice = nil
		}, "skuList.0.price.basePrice.amount", "number_invalid"},
		{"Japan integer", "GLOBAL", 100, rules.LocalPackageUnits{Weight: "g", Dimension: "cm"}, func(s *model.LocalSKU) {
			s.Price.BasePrice = model.LocalMoney{Amount: "2.01", Currency: "JPY"}
			s.Price.ListPrice = nil
		}, "skuList.0.price.basePrice.amount", "number_invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := localNumberSKU()
			if tc.change != nil {
				tc.change(&item)
			}
			check := c.CheckLocalNumbers(tc.partition, tc.region, []model.LocalSKU{item}, tc.units)
			require.Contains(t, check.Issues, rules.LocalNumericalIssue{Field: tc.field, Code: tc.code})
		})
	}
}

func TestLocalNumericalChecksRejectRoundingEqualRetailAndMissingQuantity(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	for _, tc := range []struct {
		name, field, code string
		change            func(*model.LocalSKU)
	}{
		{"below lower bound", "price.basePrice.amount", "number_invalid", func(s *model.LocalSKU) { s.Price.BasePrice.Amount = "0.01" }},
		{"no float rounding", "price.basePrice.amount", "number_invalid", func(s *model.LocalSKU) { s.Price.BasePrice.Amount = "1.999" }},
		{"no exponent", "price.basePrice.amount", "number_invalid", func(s *model.LocalSKU) { s.Price.BasePrice.Amount = "2e0" }},
		{"no fraction", "price.basePrice.amount", "number_invalid", func(s *model.LocalSKU) { s.Price.BasePrice.Amount = "2/1" }},
		{"retail equal exact", "price.listPrice.amount", "retail_not_above_base", func(s *model.LocalSKU) { s.Price.ListPrice.Amount = "2.0" }},
		{"retail below exact", "price.listPrice.amount", "retail_not_above_base", func(s *model.LocalSKU) {
			s.Price.BasePrice.Amount = "99999999.99"
			s.Price.ListPrice.Amount = "99999999.98"
		}},
		{"wrong retail currency", "price.listPrice.currency", "currency_mismatch", func(s *model.LocalSKU) { s.Price.ListPrice.Currency = "CAD" }},
		{"missing weight", "packageInfo.weight", "number_invalid", func(s *model.LocalSKU) { s.PackageInfo.Weight = "" }},
		{"wrong dimension precision", "packageInfo.height", "number_invalid", func(s *model.LocalSKU) { s.PackageInfo.Height = "0.12" }},
		{"missing inventory is not zero", "quantity", "quantity_invalid", func(s *model.LocalSKU) { s.Quantity = nil }},
		{"negative inventory", "quantity", "quantity_invalid", func(s *model.LocalSKU) { q := int64(-1); s.Quantity = &q }},
		{"inventory above upper bound", "quantity", "quantity_invalid", func(s *model.LocalSKU) { q := int64(1000001); s.Quantity = &q }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sku := localNumberSKU()
			tc.change(&sku)
			before, err := json.Marshal(sku)
			require.NoError(t, err)
			check := c.CheckLocalNumbers("US", 211, []model.LocalSKU{sku}, rules.LocalPackageUnits{Weight: "lb", Dimension: "in"})
			require.Contains(t, check.Issues, rules.LocalNumericalIssue{Field: "skuList.0." + tc.field, Code: tc.code})
			after, err := json.Marshal(sku)
			require.NoError(t, err)
			require.Equal(t, before, after, "checks must not clamp, fill or mutate facts")
		})
	}
	sku := localNumberSKU()
	sku.Price.ListPrice = nil
	q := int64(1000000)
	sku.Quantity = &q
	require.Empty(t, c.CheckLocalNumbers("US", 211, []model.LocalSKU{sku}, rules.LocalPackageUnits{Weight: "lb", Dimension: "in"}).Issues)
}

func TestLocalNumericalChecksAreBoundedAndDoNotGrantPublishReadiness(t *testing.T) {
	c, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	units := rules.LocalPackageUnits{Weight: "lb", Dimension: "in"}
	for _, skus := range [][]model.LocalSKU{nil, make([]model.LocalSKU, 501)} {
		require.Contains(t, c.CheckLocalNumbers("US", 211, skus, units).Issues, rules.LocalNumericalIssue{Field: "skuList", Code: "sku_count_invalid"})
	}
	check := c.CheckLocalNumbers("US", 211, []model.LocalSKU{localNumberSKU()}, units)
	raw, err := json.Marshal(check)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "ready", "numeric checks do not prove category, assets, consent or merchant authorization")
}
