package publishing

type Price struct {
	Currency  string  `json:"currency,omitempty"`
	Amount    float64 `json:"amount,omitempty"`
	CostPrice float64 `json:"cost_price,omitempty"`
}
