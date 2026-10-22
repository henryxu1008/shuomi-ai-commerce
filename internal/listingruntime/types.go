package listingruntime

type ProductImportMapping struct {
	ID                      int64
	ImportTaskID            int64
	StoreID                 int64
	OwnerUserID             string
	Platform                string
	Region                  string
	ProductID               string
	ParentProductID         *string
	SKU                     *string
	PlatformProductID       *string
	PlatformParentProductID *string
	CostPrice               float64
	FilterRuleID            int64
	FilterRuleRange         *string
	ProfitRuleID            int64
	SalePriceMultiplier     *float64
	DiscountPriceMultiplier *float64
	Status                  int16
	Remark                  *string
	TenantID                int64
}
