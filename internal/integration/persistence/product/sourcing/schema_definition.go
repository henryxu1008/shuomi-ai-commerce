package sourcingpersistence

// SchemaModels exposes the two current definitions for native read-only
// qualification; installation and publication remain with this owner.
func SchemaModels() []any { return []any{&publicationRecord{}, &receiptRecord{}} }
