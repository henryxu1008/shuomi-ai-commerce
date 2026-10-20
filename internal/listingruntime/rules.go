package listingruntime

type PricingRule struct {
	ID                 int64
	Name               string
	RuleCode           string
	Description        *string
	StoreID            *int64
	CategoryID         *int64
	PriceMin           *float64
	PriceMax           *float64
	RuleType           string
	RuleValue          *float64
	FixedValue         *float64
	AcceptCondition    *string
	RejectCondition    *string
	Status             int
	Remark             *string
	TenantID           int64
	TargetProfitMargin float64
	MinProfitMargin    float64
	AcceptBelowTarget  bool
	ReappealBelowMin   bool
}
