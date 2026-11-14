package reportcenterpersistence

import (
	"context"
	_ "embed"
	"gorm.io/gorm"
	"task-processor/internal/pkg/pgschema"
	rc "task-processor/internal/reportcenter"
)

//go:embed daily_schema.sql
var dailySchema string

func InstallDailyReportSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return rc.ErrUnavailable
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':report-center-schema-init',0))").Error; e != nil {
			return e
		}
		if e := tx.Exec("SET LOCAL ROLE report_center_owner").Error; e != nil {
			return e
		}
		if e := tx.Exec(dailySchema).Error; e != nil {
			return e
		}
		return tx.Exec("RESET ROLE").Error
	})
}
func VerifyDailyReportSchema(ctx context.Context, db *gorm.DB) error {
	if e := VerifySchema(ctx, db); e != nil {
		return e
	}
	if db.WithContext(ctx).Exec("SELECT scheduled_origin FROM report_center.saved_reports LIMIT 0").Error != nil {
		return rc.ErrUnavailable
	}
	var drift bool
	if db.WithContext(ctx).Raw(`SELECT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='report_center.saved_reports'::regclass AND attnum>0 AND NOT attisdropped AND attnotnull=(attname=ANY(ARRAY['source_at','scheduled_origin'])))`).Scan(&drift).Error != nil || drift {
		return rc.ErrUnavailable
	}
	var rows []struct {
		Name, Definition string
		Validated        bool
	}
	if db.WithContext(ctx).Raw(`SELECT conname AS name,pg_get_constraintdef(oid) AS definition,convalidated AS validated FROM pg_constraint WHERE conrelid='report_center.saved_reports'::regclass`).Scan(&rows).Error != nil || len(rows) != len(dailyReportConstraints) {
		return rc.ErrUnavailable
	}
	for _, r := range rows {
		expected, ok := dailyReportConstraints[r.Name]
		if !ok || !r.Validated || !pgschema.SchemaConstraintEqual(r.Definition, expected) {
			return rc.ErrUnavailable
		}
	}
	return nil
}
