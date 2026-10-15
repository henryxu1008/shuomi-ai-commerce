package agentpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/agent"
)

// EnsureRuntimeSchema is only the create hook for the fixed native preparation
// transaction. Existing groups are verified read-only by that composition.
func EnsureRuntimeSchema(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || tx == nil || tx.Statement == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return agent.ErrUnavailable
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return agent.ErrUnavailable
	}
	var admitted bool
	if err := tx.WithContext(ctx).Raw(`SELECT current_user='product_agent_owner' AND current_schema()='public' AND to_regclass('public.product_agent_runs') IS NULL AND to_regclass('public.product_agent_tool_calls') IS NULL`).Scan(&admitted).Error; err != nil || !admitted {
		return agent.ErrUnavailable
	}
	return InstallSchema(tx.WithContext(ctx))
}
