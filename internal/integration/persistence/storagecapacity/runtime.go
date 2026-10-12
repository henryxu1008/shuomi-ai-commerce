package storagecapacitystore

import (
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"task-processor/internal/ledger/storagecapacity"
)

// VerifyDisabled proves that the mounted owner has never admitted capacity.
// Any durable receipt, including historical/empty rows, makes metering mandatory.
func VerifyDisabled(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Config == nil || db.Dialector == nil || db.Statement == nil || db.ConnPool == nil {
		return storagecapacity.ErrUnavailable
	}
	for _, table := range []string{"storage_capacity_namespaces", "storage_capacity_accounts", "storage_capacity_reservations", "storage_capacity_objects", "storage_capacity_bindings", "storage_capacity_lots"} {
		if db.Dialector.Name() == "postgres" {
			table = "public." + table
		}
		var exists bool
		if db.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM "+table+" LIMIT 1)").Scan(&exists).Error != nil {
			return storagecapacity.ErrUnavailable
		}
		if exists {
			return errors.New("enterprise storage cannot be disabled after capacity admission")
		}
	}
	return nil
}

func VerifyRuntime(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return storagecapacity.ErrUnavailable
	}
	for _, table := range []string{"storage_capacity_accounts", "storage_capacity_reservations", "storage_capacity_objects", "storage_capacity_bindings", "storage_capacity_lots", "storage_capacity_namespaces"} {
		if db.WithContext(ctx).Exec("SELECT 1 FROM "+table+" LIMIT 0").Error != nil {
			return storagecapacity.ErrUnavailable
		}
		if db.Dialector.Name() == "postgres" {
			var allowed bool
			e := db.WithContext(ctx).Raw("SELECT has_table_privilege(current_user, ?, 'SELECT') AND has_table_privilege(current_user, ?, 'INSERT') AND has_table_privilege(current_user, ?, 'UPDATE')", "public."+table, "public."+table, "public."+table).Scan(&allowed).Error
			if e != nil || !allowed {
				return fmt.Errorf("storage capacity runtime privileges unavailable")
			}
		}
	}
	return nil
}
