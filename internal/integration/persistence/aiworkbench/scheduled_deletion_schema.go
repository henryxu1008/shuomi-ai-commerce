package aiworkbenchpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/pkg/pgschema"
)

const scheduledDeletionConstraint = `CHECK (((NOT deleted) OR (state = 'CLOSED'::text)))`

// InstallScheduledDeletionSchema is an explicit, bounded owner operation.
// It never deletes plans, rewrites business rows, or changes running work.
func InstallScheduledDeletionSchema(db *gorm.DB, role string) error {
	if db == nil || db.Dialector.Name() != "postgres" || !roleName.MatchString(role) {
		return aiworkbench.ErrInvalid
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':workbench-scheduled-deletion-schema',0))").Error; err != nil {
			return err
		}
		if err := validateRuntimeRoleIdentity(tx, role); err != nil {
			return err
		}
		if member, err := hasRoleMembership(tx, role); err != nil || member {
			return aiworkbench.ErrInvalid
		}
		if cross, err := hasCrossOwnerRelationPrivilege(tx, role); err != nil || cross {
			return aiworkbench.ErrInvalid
		}
		if err := tx.Exec("ALTER TABLE ai_workbench.scheduled_plans ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT false").Error; err != nil {
			return err
		}
		var exists bool
		if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='ai_workbench.scheduled_plans'::regclass AND conname='scheduled_plans_deleted_closed')").Scan(&exists).Error; err != nil {
			return err
		}
		if !exists {
			if err := tx.Exec("ALTER TABLE ai_workbench.scheduled_plans ADD CONSTRAINT scheduled_plans_deleted_closed CHECK (NOT deleted OR state='CLOSED')").Error; err != nil {
				return err
			}
		}
		// Existing bounds are verified after the single-column grant; any drift
		// rejects and rolls back the entire installation transaction.
		if err := tx.Exec(`GRANT UPDATE(deleted) ON ai_workbench.scheduled_plans TO "` + role + `"`).Error; err != nil {
			return err
		}
		if err := validateBoundedWorkbenchPrivileges(tx, role); err != nil {
			return err
		}
		return verifyScheduledDeletionSchema(tx.Statement.Context, tx)
	})
}

func verifyScheduledDeletionSchema(ctx context.Context, db *gorm.DB) error {
	var valid bool
	if db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE a.attrelid='ai_workbench.scheduled_plans'::regclass AND a.attname='deleted' AND NOT a.attisdropped AND a.atttypid='boolean'::regtype AND a.attnotnull AND pg_get_expr(d.adbin,d.adrelid)='false')`).Scan(&valid).Error != nil || !valid {
		return aiworkbench.ErrUnavailable
	}
	var check struct {
		Definition string
		Validated  bool
	}
	if db.WithContext(ctx).Raw(`SELECT pg_get_constraintdef(oid) AS definition,convalidated AS validated FROM pg_constraint WHERE conrelid='ai_workbench.scheduled_plans'::regclass AND conname='scheduled_plans_deleted_closed' AND contype='c'`).Scan(&check).Error != nil || !check.Validated || !pgschema.SchemaConstraintEqual(check.Definition, scheduledDeletionConstraint) {
		return aiworkbench.ErrUnavailable
	}
	return nil
}
