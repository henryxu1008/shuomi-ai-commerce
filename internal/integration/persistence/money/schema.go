package money

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/money"
	"gorm.io/gorm"
)

// Install is the explicit greenfield schema entry point for the canonical
// money owner. It is never called by a serving handler.
func Install(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil {
		return money.ErrInvalid
	}
	return AutoMigrate(db)
}
