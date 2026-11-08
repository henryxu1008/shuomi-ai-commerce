package productagentruntime

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// VerifyInvocationRecoveryRuntime verifies existing schema and exact DML grants.
// It performs no repairs and is valid on a read-only inspect connection.
func VerifyInvocationRecoveryRuntime(ctx context.Context, db *gorm.DB) error {
	var role string
	if err := db.WithContext(ctx).Raw("SELECT current_user").Scan(&role).Error; err != nil || role != "product_agent_runtime" {
		return errors.New("invocation recovery runtime role unavailable")
	}
	if err := verifyRole(db.WithContext(ctx), role); err != nil {
		return err
	}
	ledger, err := invocationContract(db)
	if err != nil {
		return err
	}
	if err = verifyRelation(db.WithContext(ctx), ledger); err != nil {
		return err
	}
	if err = verifyPrivateACL(db.WithContext(ctx), ledger.Name); err != nil {
		return err
	}
	ready, err := verifyGrants(db.WithContext(ctx), runGrants(ledger)[:1])
	if err != nil || !ready {
		return errors.New("invocation recovery runtime grants unavailable")
	}
	return nil
}
