package storecenter

import (
	"context"
	"errors"
	"gorm.io/gorm"
	observationschema "task-processor/internal/integration/persistence/sheinobservations"
)

// InstallObservations is explicit fresh-instance setup, never runtime startup.
// Install rejects an existing observation schema and does not migrate facts.
func InstallObservations(ctx context.Context, records *gorm.DB) error {
	if records == nil {
		return errors.New("Store observation schema-owner unavailable")
	}
	return records.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("public.workbench_stores").Count(&count).Error; err != nil || count != 0 {
			return errors.New("observation initialization requires a fresh empty Store instance")
		}
		if err := observationschema.Install(ctx, tx); err != nil {
			return err
		}
		return observationschema.GrantRuntime(ctx, tx, "store_center_runtime")
	})
}

// RestoreObservationsGrants is an explicit owner operation after an authorized
// rollback. It never installs schema, migrates facts or changes base Store grants.
func RestoreObservationsGrants(ctx context.Context, records *gorm.DB) error {
	if ctx == nil || records == nil || records.Dialector.Name() != "postgres" {
		return errors.New("Store observation schema-owner unavailable")
	}
	return records.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owned bool
		err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname='shein_observations' AND nspowner=(SELECT oid FROM pg_roles WHERE rolname=current_user))
		 AND (SELECT count(*)=4 AND bool_and(c.relname IN('commands','syncs','records','heads') AND c.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user)) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='shein_observations' AND c.relkind IN('r','p'))`).Scan(&owned).Error
		if err != nil || !owned {
			return errors.New("original owner-controlled observation schema required; no installation or migration")
		}
		return observationschema.GrantRuntime(ctx, tx, "store_center_runtime")
	})
}
