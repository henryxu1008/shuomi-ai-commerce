package productagentruntime

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

// PrepareProviderRejectionProof is an explicit current-system capability
// expansion. It adds one nullable column and one precise existing-role grant.
// Serving startup and ordinary PrepareRun never invoke this operation.
func PrepareProviderRejectionProof(ctx context.Context, db *gorm.DB) (Result, error) {
	return prepare(ctx, db, "invocation-provider-rejection", "product_agent_owner", func(tx *gorm.DB, result *Result) error {
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
		// The original ledger is shared with qualified audit readers. Match
		// PrepareRun's relation and runtime grants without closing other owners' ACL.
		if tx.Migrator().HasColumn("ai_invocations", "provider_rejection_proof_json") {
			if err = verifyRelation(tx, current); err != nil {
				return err
			}
			ready, err := verifyGrants(tx, runGrants(current)[:1])
			if err != nil || !ready {
				return errors.New("current invocation capability grants differ")
			}
			result.Outcome = "ALREADY_READY"
			return nil
		}
		base := current
		base.Columns = nil
		for _, column := range current.Columns {
			if column.Name != "provider_rejection_proof_json" {
				base.Columns = append(base.Columns, column)
			}
		}
		if err = verifyRelation(tx, base); err != nil {
			return err
		}
		ready, err := verifyGrants(tx, runGrants(base)[:1])
		if err != nil || !ready {
			return errors.New("original invocation capability grants differ")
		}
		if err = tx.Exec("ALTER TABLE public.ai_invocations ADD COLUMN provider_rejection_proof_json text").Error; err != nil {
			return err
		}
		if err = tx.Exec("GRANT UPDATE(provider_rejection_proof_json) ON public.ai_invocations TO product_agent_runtime").Error; err != nil {
			return err
		}
		if err = verifyRelation(tx, current); err != nil {
			return err
		}
		ready, err = verifyGrants(tx, runGrants(current)[:1])
		if err != nil || !ready {
			return errors.New("expanded invocation capability grants differ")
		}
		result.Outcome = "CREATED"
		return nil
	})
}
