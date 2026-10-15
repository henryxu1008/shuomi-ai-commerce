package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"task-processor/internal/commercial/billing"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	platformdatabase "task-processor/internal/platform/database"
)

var storageEnableTables = []string{"storage_capacity_accounts", "storage_capacity_reservations", "storage_capacity_objects", "storage_capacity_bindings", "storage_capacity_lots", "storage_capacity_namespaces"}

func runStorageEnableEmpty(path string) error {
	cfg, err := loadSchemaOwnerConfig(path)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	db, err := platformdatabase.OpenExistingWritableContext(ctx, cfg)
	if err != nil {
		return fmt.Errorf("open existing commercial schema owner: %w", err)
	}
	defer platformdatabase.Close(db)
	return enableEmptyStorageOwner(ctx, db)
}

func enableEmptyStorageOwner(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return fmt.Errorf("capacity enable requires the existing PostgreSQL commercial owner")
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, statement := range []string{"SET LOCAL search_path = public, pg_catalog", "SET LOCAL lock_timeout = '5s'", "SET LOCAL statement_timeout = '30s'"} {
			if err := tx.Exec(statement).Error; err != nil {
				return err
			}
		}
		tables := append(append([]string{}, storageEnableTables...), "commercial_offers", "commercial_quotes", "commercial_order_items", "commercial_orders")
		qualified := make([]string, len(tables))
		for i, table := range tables {
			qualified[i] = "public." + table
		}
		// A missing table fails here, before any reusable additive helper can
		// enter an AutoMigrate branch. Locks fence empty-state checks and grants.
		if err := tx.Exec("LOCK TABLE " + strings.Join(qualified, ",") + " IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		var owned int64
		if err := tx.Raw("SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind='r' AND c.relname IN ? AND pg_get_userbyid(c.relowner)=current_user", tables).Scan(&owned).Error; err != nil {
			return err
		}
		if owned != int64(len(tables)) {
			return fmt.Errorf("capacity enable requires the original owner of all existing tables")
		}
		var runtime bool
		if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='commercial_owner_runtime' AND NOT rolsuper AND NOT rolcreaterole AND NOT rolcreatedb AND NOT rolreplication AND NOT rolbypassrls)").Scan(&runtime).Error; err != nil {
			return err
		}
		if !runtime {
			return fmt.Errorf("capacity enable requires the original narrow runtime role")
		}
		// Check all seven columns before the shared owner validator. The helper
		// may add missing columns in its separate disabled-install mode only.
		for table, columns := range map[string][]string{
			"commercial_offers":      {"storage_unit_bytes", "storage_duration_seconds"},
			"commercial_quotes":      {"storage_unit_bytes", "storage_duration_seconds"},
			"commercial_order_items": {"storage_unit_bytes", "storage_duration_seconds"},
			"commercial_orders":      {"storage_activation_receipt"},
		} {
			for _, column := range columns {
				var exists bool
				if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=? AND column_name=?)", table, column).Scan(&exists).Error; err != nil {
					return err
				}
				if !exists {
					return fmt.Errorf("capacity enable cannot add missing public.%s.%s", table, column)
				}
			}
		}
		if err := commercialstore.AddDisabledStorageSchema(ctx, tx); err != nil {
			return err
		}
		for _, table := range storageEnableTables {
			var destructive bool
			if err := tx.Raw("SELECT has_table_privilege('commercial_owner_runtime',?,'DELETE,TRUNCATE,REFERENCES,TRIGGER') OR has_any_column_privilege('commercial_owner_runtime',?,'REFERENCES')", "public."+table, "public."+table).Scan(&destructive).Error; err != nil {
				return err
			}
			if destructive {
				return fmt.Errorf("capacity runtime has unexpected destructive privileges")
			}
		}
		if err := tx.Exec("GRANT SELECT,INSERT,UPDATE ON TABLE " + strings.Join(qualified[:len(storageEnableTables)], ",") + " TO commercial_owner_runtime").Error; err != nil {
			return err
		}
		repo, err := commercialstore.New(tx)
		if err != nil {
			return err
		}
		return repo.CreateOfferIfAbsent(ctx, billing.InitialStorageOffer())
	})
}
