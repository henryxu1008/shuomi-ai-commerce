package model

// RuleSelection selects named, read-only metadata for the original verified
// store. SupplierSiteID is only the optional supplier category query field;
// local metadata has no client-selected site or mall. This is not a payload,
// authorization, price profile or evidence that a merchant can publish.
type RuleSelection struct {
	ParentCatID          int64     `json:"parentCatId"`
	CatID                int64     `json:"catId"`
	Language             string    `json:"language,omitempty"`
	CostTemplateID       string    `json:"costTemplateId,omitempty"`
	SupplierSiteID       int64     `json:"supplierSiteId,omitempty"`
	SupplierCategoryPath [10]int64 `json:"supplierCategoryPath,omitempty"`
	SupplierSizeOffset   int64     `json:"supplierSizeOffset,omitempty"`
	TrademarkPage        int64     `json:"trademarkPage,omitempty"`
	TrademarkSize        int64     `json:"trademarkSize,omitempty"`
}
