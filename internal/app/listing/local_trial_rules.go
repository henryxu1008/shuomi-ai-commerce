package listingapp

import (
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
)

// ControlledLocalRules is the bounded, compiled fixture used by the admitted
// trial. IDs are synthetic. Each call returns fresh data, never a mutable cache.
func ControlledLocalRules() goods.OfficialRuleSnapshot {
	p := func(v int) *int { return &v }
	leaf := true
	return goods.OfficialRuleSnapshot{
		Authority: "local_trial", ApplicationMode: model.ModeSelfOperated,
		Categories: []model.Category{{ID: 123, ProductTypeID: 456, Leaf: &leaf, Name: "本地测试服装"}},
		Sites:      []model.MainSite{{ID: "shein", Sites: []model.Site{{Abbreviation: "shein-us", Status: p(1), StoreType: p(2), Currency: "USD"}}}},
		Fill:       model.FillStandards{DefaultLanguage: "en", DefaultTitleMaximum: p(150), Fields: []model.FillRule{}, Pictures: []model.PictureRule{}},
		Attributes: model.AttributeTemplate{ProductTypeID: 456, MainAttributeStatus: p(3), Attributes: []model.Attribute{
			{ID: 12, Name: "Color (本地测试)", Type: p(1), MainLabel: p(1), Show: p(1), Status: p(3), Mode: p(2), Options: []model.AttributeOption{{ID: 34, Name: "黑", Show: p(1)}, {ID: 35, Name: "白", Show: p(1)}}},
			{ID: 13, Name: "Size (本地测试)", Type: p(1), MainLabel: p(0), Show: p(1), Status: p(3), Mode: p(2), Options: []model.AttributeOption{{ID: 36, Name: "M", Show: p(1)}, {ID: 37, Name: "L", Show: p(1)}}},
		}},
		Linked: []model.LinkedRules{}, Brands: []model.Brand{},
		Warehouses: []model.Warehouse{{Code: "trial-us", Name: "本地测试 US 仓", Type: 1, SaleCountries: []string{"US"}}},
	}
}
