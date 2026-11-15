package aiworkbenchpersistence

import (
	"context"
	_ "embed"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/pkg/pgschema"
)

//go:embed daily_report_schema.sql
var dailyReportSchema string

func (s *Store) VerifyDailyReports(ctx context.Context) error {
	if s == nil {
		return aiworkbench.ErrUnavailable
	}
	return VerifyDailyReportSchema(ctx, s.db)
}

// InstallDailyReportSchema is an explicit owner operation. Serving startup is
// read-only; installation never touches a plan, occurrence, task or balance.
func InstallDailyReportSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return aiworkbench.ErrUnavailable
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':workbench-daily-report-schema',0))").Error; err != nil {
			return err
		}
		return tx.Exec(dailyReportSchema).Error
	})
}

func GrantDailyReportRuntime(db *gorm.DB, role string) error {
	if db == nil || !roleName.MatchString(role) {
		return aiworkbench.ErrInvalid
	}
	if err := validateRuntimeRoleIdentity(db, role); err != nil {
		return err
	}
	if member, e := hasRoleMembership(db, role); e != nil || member {
		return aiworkbench.ErrInvalid
	}
	if cross, e := hasCrossOwnerRelationPrivilege(db, role); e != nil || cross {
		return aiworkbench.ErrInvalid
	}
	if err := validateBoundedWorkbenchPrivileges(db, role); err != nil {
		return err
	}
	quoted := `"` + role + `"`
	for _, sql := range []string{
		"REVOKE ALL ON ai_workbench.report_execution FROM " + quoted,
		"GRANT SELECT, INSERT ON ai_workbench.report_execution TO " + quoted,
		"GRANT UPDATE(capture_started,closed_no_write,report_id,report_digest,proof) ON ai_workbench.report_execution TO " + quoted,
	} {
		if err := db.Exec(sql).Error; err != nil {
			return err
		}
	}
	return nil
}

func VerifyDailyReportSchema(ctx context.Context, db *gorm.DB) error {
	if err := VerifySchema(ctx, db); err != nil {
		return err
	}
	var role string
	if db.WithContext(ctx).Raw("SELECT current_user").Scan(&role).Error != nil {
		return aiworkbench.ErrUnavailable
	}
	if err := validateWorkbenchPrivilegeBounds(db.WithContext(ctx), role, []string{"report_execution"}, map[string][]string{"report_execution": {"capture_started", "closed_no_write", "report_id", "report_digest", "proof"}}); err != nil {
		return aiworkbench.ErrUnavailable
	}
	if db.WithContext(ctx).Exec("SELECT report_spec FROM ai_workbench.scheduled_plans LIMIT 0").Error != nil {
		return aiworkbench.ErrUnavailable
	}
	return verifyDailyReportConstraints(ctx, db)
}

func verifyDailyReportConstraints(ctx context.Context, db *gorm.DB) error {
	var drift bool
	if db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM pg_attribute WHERE attnum>0 AND NOT attisdropped AND (
 (attrelid='ai_workbench.scheduled_plans'::regclass AND attnotnull=(attname=ANY(ARRAY['task_selection','report_spec','rule_start_at','once_at_utc','local_hour','local_minute','local_weekday','until_utc','active_occurrence_id']))) OR
 (attrelid='ai_workbench.report_execution'::regclass AND attnotnull=(attname=ANY(ARRAY['report_id','report_digest','proof'])))))`).Scan(&drift).Error != nil || drift {
		return aiworkbench.ErrUnavailable
	}
	var rows []struct {
		Name, Definition string
		Validated        bool
	}
	if db.WithContext(ctx).Raw(`SELECT conname AS name,pg_get_constraintdef(oid) AS definition,convalidated AS validated FROM pg_constraint WHERE conrelid IN ('ai_workbench.scheduled_plans'::regclass,'ai_workbench.report_execution'::regclass)`).Scan(&rows).Error != nil || len(rows) != len(dailyReportConstraints) {
		return aiworkbench.ErrUnavailable
	}
	for _, r := range rows {
		expected, ok := dailyReportConstraints[r.Name]
		if !ok || !r.Validated || !pgschema.SchemaConstraintEqual(r.Definition, expected) {
			return aiworkbench.ErrUnavailable
		}
	}
	return nil
}
