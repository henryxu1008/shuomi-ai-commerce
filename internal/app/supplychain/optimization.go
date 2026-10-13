package supplychainapp

type TitleOptimizationChoice struct {
	AgentID           string `json:"agentId"`
	ModeRevision      string `json:"modeRevision"`
	ExecutionMode     string `json:"executionMode"`
	TemplateID        string `json:"templateId"`
	Revision          string `json:"revision"`
	Name              string `json:"name"`
	QuoteHash         string `json:"quoteHash"`
	MaximumCostMicros int64  `json:"maximumCostMicros"`
	MaximumPoints     int64  `json:"maximumPoints"`
	PriceVersion      string `json:"priceVersion"`
	Currency          string `json:"currency"`
}
type OptimizationOptions struct {
	Titles      []TitleOptimizationChoice `json:"titles"`
	Reason      string                    `json:"reason,omitempty"`
	ImageReason string                    `json:"imageReason,omitempty"`
	NextCursor  string                    `json:"nextCursor,omitempty"`
}
