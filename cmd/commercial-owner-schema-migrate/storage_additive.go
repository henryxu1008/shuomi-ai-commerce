package main

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	platformdatabase "task-processor/internal/platform/database"
)

func runStorageDisabledAdditive(manifestPath string) error {
	config, err := loadSchemaOwnerConfig(manifestPath)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := platformdatabase.OpenExistingWritableContext(ctx, config)
	if err != nil {
		return fmt.Errorf("open retained commercial schema owner: %w", err)
	}
	defer platformdatabase.Close(db)
	return migrateDisabledStorageOwner(ctx, db)
}

func migrateDisabledStorageOwner(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return fmt.Errorf("storage additive installation requires the existing PostgreSQL commercial owner")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{"SET LOCAL search_path = public, pg_catalog", "SET LOCAL lock_timeout = '5s'", "SET LOCAL statement_timeout = '30s'"} {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		var owner bool
		if err := tx.Raw("SELECT current_schema()='public' AND current_user <> 'commercial_owner_runtime' AND has_schema_privilege(current_user,'public','CREATE') AND EXISTS (SELECT 1 FROM pg_roles WHERE rolname='commercial_owner_runtime')").Scan(&owner).Error; err != nil {
			return err
		}
		if !owner {
			return fmt.Errorf("retained installation requires the schema owner and existing commercial_owner_runtime role")
		}
		if err := commercialstore.AddDisabledStorageSchema(ctx, tx); err != nil {
			return err
		}
		if err := tx.Exec(`GRANT SELECT ON TABLE public.storage_capacity_accounts, public.storage_capacity_reservations, public.storage_capacity_objects, public.storage_capacity_bindings, public.storage_capacity_lots, public.storage_capacity_namespaces TO commercial_owner_runtime`).Error; err != nil {
			return err
		}
		for _, table := range []string{"storage_capacity_accounts", "storage_capacity_reservations", "storage_capacity_objects", "storage_capacity_bindings", "storage_capacity_lots", "storage_capacity_namespaces"} {
			var read, write bool
			if err := tx.Raw("SELECT has_table_privilege('commercial_owner_runtime',?,'SELECT'), has_table_privilege('commercial_owner_runtime',?,'INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege('commercial_owner_runtime',?,'INSERT,UPDATE,REFERENCES')", "public."+table, "public."+table, "public."+table).Row().Scan(&read, &write); err != nil {
				return err
			}
			if !read || write {
				return fmt.Errorf("storage additive runtime must have SELECT only on public.%s", table)
			}
		}
		return commercialstore.VerifyStorageCapacityDisabled(ctx, tx)
	})
}
