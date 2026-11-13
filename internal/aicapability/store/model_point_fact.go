package store

import (
	"context"
	"database/sql"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/aicapability"
)

// HasInvocationID is an authoritative, unfiltered existence check on the
// owner primary. It returns no cross-tenant data and never locks or writes.
func (r *GormInvocationRecorder) HasInvocationID(ctx context.Context, id string) (bool, error) {
	if r == nil || r.db == nil || ctx == nil || strings.TrimSpace(id) == "" {
		return false, fmt.Errorf("invalid invocation identity")
	}
	var count int64
	if r.db.Dialector.Name() != "postgres" {
		if err := r.db.WithContext(ctx).Model(&invocationRow{}).Where("invocation_id = ?", id).Count(&count).Error; err != nil {
			return false, err
		}
		return count != 0, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var authoritative bool
		if err := tx.Raw(`SELECT NOT pg_is_in_recovery() AND current_schema()='public' AND NOT c.relrowsecurity AND c.relkind='r' AND has_table_privilege(current_user,c.oid,'SELECT') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname='ai_invocations'`).Scan(&authoritative).Error; err != nil {
			return err
		}
		if !authoritative {
			return fmt.Errorf("invocation primary visibility is not authoritative")
		}
		return tx.Model(&invocationRow{}).Where("invocation_id = ?", id).Count(&count).Error
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true})
	if err != nil {
		return false, err
	}
	return count != 0, nil
}

// ReadModelInvocation reads admitted priced-text operations from the
// existing AI invocation owner, never caller-supplied usage.
func (r *GormInvocationRecorder) ReadModelInvocation(ctx context.Context, org, id string) (aicapability.InvocationRecord, error) {
	if r == nil || r.db == nil || strings.TrimSpace(org) == "" || strings.TrimSpace(id) == "" {
		return aicapability.InvocationRecord{}, fmt.Errorf("invalid model fact identity")
	}
	var row invocationRow
	if err := r.db.WithContext(ctx).Where("tenant_id = ? AND invocation_id = ? AND operation IN ?", org, id,
		[]aicapability.Operation{aicapability.OperationProductAgentDecision, aicapability.OperationProductContentDecision, aicapability.OperationListingMaterialsDecision, aicapability.OperationProductResearchDecision, aicapability.OperationProductSelectionDecision, aicapability.OperationShopDiagnosisDecision, aicapability.OperationProductImageAnalysis, aicapability.OperationOwnProductIntake, aicapability.OperationAIWorkbenchChatPlan, aicapability.OperationAIWorkbenchStoreRoute}).Take(&row).Error; err != nil {
		return aicapability.InvocationRecord{}, err
	}
	fact, err := invocationRecordFromRow(row)
	if err != nil {
		return aicapability.InvocationRecord{}, err
	}
	if !fact.OfficialTariff.Valid() || !fact.OfficialTariff.MatchesWireModel(fact.ModelID) || fact.MaximumPromptTokens <= 0 || fact.MaximumCompletionTokens <= 0 || fact.StartedAt.IsZero() || fact.UserID == "" || fact.MemberID == "" || fact.InputHash == "" || !validPricedTextOperationIdentity(fact) {
		return aicapability.InvocationRecord{}, fmt.Errorf("invalid priced model fact")
	}
	if err := validateUsage(fact); err != nil {
		return aicapability.InvocationRecord{}, err
	}
	return fact, nil
}

// Startup verifies a freshly installed schema. It never alters an old ledger.
func VerifyModelPointInvocationSchema(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("model invocation database is required")
	}
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(&invocationRow{}); err != nil {
		return err
	}
	columns, err := db.WithContext(ctx).Migrator().ColumnTypes(&invocationRow{})
	if err != nil {
		return err
	}
	if len(columns) != len(stmt.Schema.DBNames) {
		return fmt.Errorf("current model invocation schema has unexpected columns; install on a fresh database")
	}
	present := make(map[string]bool, len(columns))
	for _, column := range columns {
		present[column.Name()] = true
	}
	for _, name := range stmt.Schema.DBNames {
		if !present[name] {
			return fmt.Errorf("current model invocation schema is missing %s; install on a fresh database", name)
		}
	}
	return nil
}
