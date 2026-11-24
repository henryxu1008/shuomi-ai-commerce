package preparation

import "context"

// Read projection of existing owners; none of these values is a new durable state.
type SourceStageFacts struct {
	SourceID                string
	ProductKey              string
	Title                   string
	SourceKind              string
	SKUs                    string
	RecordID                string
	RecordRevision          int64
	EffectiveVersion        uint64
	ApplyReceiptID          string
	Ready                   bool
	PublishedRecordID       string
	OptimizationOperationID string
	Optimization            OperationItem
}
type StageFactsReader interface {
	ListStageFacts(ctx context.Context, scope Scope, preparationID, storeID, shopifyShopKey, supplierHash, after string) ([]SourceStageFacts, error)
	LatestOptimization(context.Context, Scope, string, string) (OperationItem, error)
}
