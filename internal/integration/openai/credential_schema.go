package openai

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

// EnsureCredentialSchema creates only the current credential model in the
// admitted owner transaction. It never migrates or repairs an existing table.
func EnsureCredentialSchema(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || tx == nil || tx.Statement == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return errors.New("credential schema preparation unavailable")
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return errors.New("credential schema requires owner transaction")
	}
	var admitted bool
	if err := tx.WithContext(ctx).Raw(`SELECT current_user='product_agent_owner' AND current_schema()='public' AND to_regclass('public.ai_client_credentials') IS NULL`).Scan(&admitted).Error; err != nil || !admitted {
		return errors.New("credential schema requires an absent current target")
	}
	return tx.WithContext(ctx).Migrator().CreateTable(&AIClientCredential{})
}
