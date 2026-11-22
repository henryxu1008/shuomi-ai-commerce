package aiworkbenchpersistence

import (
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"gorm.io/gorm"
)

var scheduledEditColumns = []string{"task_selection", "goal_summary", "execution_spec", "max_occurrences", "per_run_upper_bound_micros", "total_upper_bound_micros"}

func scheduledEditColumnsFor(db *gorm.DB) ([]string, error) {
	var capability struct{ Spec, Execution bool }
	if err := db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='ai_workbench.scheduled_plans'::regclass AND attname='report_spec' AND NOT attisdropped) AS spec,to_regclass('ai_workbench.report_execution') IS NOT NULL AS execution`).Scan(&capability).Error; err != nil {
		return nil, err
	}
	columns := append([]string(nil), scheduledEditColumns...)
	if !capability.Spec && !capability.Execution {
		return columns, nil
	}
	if !capability.Spec || !capability.Execution {
		return nil, aiworkbench.ErrInvalid
	}
	if err := verifyDailyReportConstraints(db.Statement.Context, db); err != nil {
		return nil, err
	}
	return append(columns, "report_spec"), nil
}

// SetScheduledEditPermissions is an explicit owner-only operation. It changes
// only the installed editing column grants, never schema/business rows or other owner privileges.
func SetScheduledEditPermissions(db *gorm.DB, role string, enable bool) error {
	if db == nil || db.Dialector.Name() != "postgres" || !roleName.MatchString(role) {
		return aiworkbench.ErrInvalid
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':workbench-scheduled-edit-permissions',0))").Error; err != nil {
			return err
		}
		if err := validateRuntimeRoleIdentity(tx, role); err != nil {
			return err
		}
		if member, e := hasRoleMembership(tx, role); e != nil || member {
			return aiworkbench.ErrInvalid
		}
		if cross, e := hasCrossOwnerRelationPrivilege(tx, role); e != nil || cross {
			return aiworkbench.ErrInvalid
		}
		editable, err := scheduledEditColumnsFor(tx)
		if err != nil {
			return err
		}
		count := 0
		for _, column := range editable {
			var granted bool
			if err := tx.Raw("SELECT has_column_privilege(?, 'ai_workbench.scheduled_plans', ?, 'UPDATE')", role, column).Scan(&granted).Error; err != nil {
				return err
			}
			if granted {
				count++
			}
		}
		if count != 0 && count != len(editable) {
			return aiworkbench.ErrInvalid
		}
		current := count == len(editable)
		if err := validateWorkbenchPrivilegesForEdit(tx, role, current); err != nil {
			return err
		}
		if current != enable {
			columns := strings.Join(editable, ",")
			statement := `REVOKE UPDATE (` + columns + `) ON ai_workbench.scheduled_plans FROM "` + role + `"`
			if enable {
				statement = `GRANT UPDATE (` + columns + `) ON ai_workbench.scheduled_plans TO "` + role + `"`
			}
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		return validateWorkbenchPrivilegesForEdit(tx, role, enable)
	})
}
