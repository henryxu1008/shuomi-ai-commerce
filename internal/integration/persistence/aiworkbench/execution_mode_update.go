package aiworkbenchpersistence

import (
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
)

// InstallObservedExecutionMode adds nullable, constrained proposal metadata.
// Historical rows keep NULL and their original digest; NULL never authorizes
// a new execution. Existing table grants automatically cover the new column.
func InstallObservedExecutionMode(db *gorm.DB, role string) error {
	if db == nil || db.Dialector.Name() != "postgres" || !roleName.MatchString(role) {
		return aiworkbench.ErrInvalid
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := validateRuntimeRoleIdentity(tx, role); err != nil {
			return err
		}
		if err := validateBoundedWorkbenchPrivileges(tx, role); err != nil {
			return err
		}
		return tx.Exec(`ALTER TABLE ai_workbench.execution_proposals ADD COLUMN IF NOT EXISTS observed_mode_revision text;
DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='ai_workbench.execution_proposals'::regclass AND conname='proposal_execution_mode_revision') THEN
ALTER TABLE ai_workbench.execution_proposals ADD CONSTRAINT proposal_execution_mode_revision CHECK(observed_mode_revision IS NULL OR (observed_mode_revision ~ '^(0|[1-9][0-9]{0,18})$' AND observed_mode_revision::numeric <= 9223372036854775807)); END IF; END $$;`).Error
	})
}
