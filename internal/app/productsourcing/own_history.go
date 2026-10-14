package productsourcing

import (
	"gorm.io/gorm"
	sourcingstore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/product/sourcing"
)

func NewOwnHistoryReader(db *gorm.DB) (sourcing.OwnHistoryReadStore, error) {
	return sourcingstore.NewOwnHistoryReader(db, newCatalogBridge)
}
