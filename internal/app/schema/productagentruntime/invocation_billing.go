package productagentruntime

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// PrepareInvocationBillingMode expands the existing invocation ledger in
// place. Original facts and other owners' access stay intact.
func PrepareInvocationBillingMode(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "invocation-billing-mode", "product_agent_owner", func(tx *gorm.DB, result *Result) error {
		result.Tables = []string{"ai_invocations"}
		result.Roles = []string{"product_agent_runtime"}
		if err := verifyRole(tx, "product_agent_runtime"); err != nil {
			return err
		}
		current, err := invocationContract(tx)
		if err != nil {
			return err
		}
		if err = tx.Exec("LOCK TABLE public.ai_invocations IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		verify := func(spec relationContract) error {
			if err := verifyRelation(tx, spec); err != nil {
				return err
			}
			ready, err := verifyGrants(tx, runGrants(spec)[:1])
			if err != nil {
				return err
			}
			if !ready {
				return errors.New("invocation runtime grants differ")
			}
			return nil
		}
		if tx.Migrator().HasColumn("ai_invocations", "billing_mode") {
			result.Outcome = "ALREADY_READY"
			return verify(current)
		}
		if err := verify(invocationContractBeforeBilling(current)); err != nil {
			return err
		}
		if err := tx.Exec("ALTER TABLE public.ai_invocations ADD COLUMN billing_mode text CONSTRAINT ai_invocations_billing_mode_check CHECK (billing_mode IN ('platform_points','usage_only'))").Error; err != nil {
			return err
		}
		result.Outcome = "CREATED"
		return verify(current)
	})
}
