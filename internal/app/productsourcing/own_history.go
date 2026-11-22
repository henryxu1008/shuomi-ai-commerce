package productsourcing

import (
	sourcingstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/sourcing"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/sourcing"
	"gorm.io/gorm"
)

func NewOwnHistoryReader(db *gorm.DB) (sourcing.OwnHistoryReadStore, error) {
	return sourcingstore.NewOwnHistoryReader(db, newCatalogBridge)
}
