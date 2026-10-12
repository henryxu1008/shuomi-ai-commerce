package commercialbilling

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"task-processor/internal/commercial/billing"
	capacitystore "task-processor/internal/integration/persistence/storagecapacity"
)

// AddDisabledStorageSchema materializes only current owner fields. The caller
// must keep schema changes and the limited runtime grant in the same transaction.
func AddDisabledStorageSchema(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || tx == nil || tx.Statement == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return billing.ErrFeatureUnavailable
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return billing.ErrFeatureUnavailable
	}
	tx = tx.WithContext(ctx)
	var public bool
	if err := tx.Raw("SELECT current_schema()='public'").Scan(&public).Error; err != nil {
		return err
	}
	if !public {
		return fmt.Errorf("storage additive installation requires the public commercial schema")
	}
	plans := []struct {
		model  any
		fields []string
	}{
		{&offerRow{}, []string{"StorageUnitBytes", "StorageDurationSeconds"}},
		{&quoteRow{}, []string{"StorageUnitBytes", "StorageDurationSeconds"}},
		{&orderItemRow{}, []string{"StorageUnitBytes", "StorageDurationSeconds"}},
		{&orderRow{}, []string{"StorageActivationReceipt"}},
	}
	for _, plan := range plans {
		if !tx.Migrator().HasTable(plan.model) {
			return fmt.Errorf("retained commercial table is missing")
		}
	}
	var purchased bool
	if err := tx.WithContext(ctx).Raw("SELECT EXISTS (SELECT 1 FROM public.commercial_orders WHERE kind=? LIMIT 1)", billing.OrderStorageCapacityPurchase).Scan(&purchased).Error; err != nil {
		return err
	}
	if purchased {
		return fmt.Errorf("storage additive installation requires no capacity purchase orders")
	}
	if err := capacitystore.EnsureDisabledSchema(ctx, tx); err != nil {
		return err
	}
	for _, plan := range plans {
		statement := &gorm.Statement{DB: tx}
		if err := statement.Parse(plan.model); err != nil {
			return err
		}
		var fieldNames []string
		for _, name := range plan.fields {
			fieldNames = append(fieldNames, statement.Schema.FieldsByName[name].DBName)
		}
		// The additive fields define NOT NULL/defaults only. Leave unrelated
		// retained Commercial constraints/indexes intact; refuse dependencies
		// on these fields, including standalone and expression indexes.
		var constrained bool
		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum>0 AND NOT a.attisdropped WHERE n.nspname='public' AND t.relname=? AND a.attname IN ? AND (a.attnum=ANY(c.conkey) OR EXISTS (SELECT 1 FROM pg_depend d WHERE d.refclassid='pg_class'::regclass AND d.refobjid=t.oid AND d.refobjsubid=a.attnum AND ((d.classid='pg_constraint'::regclass AND d.objid=c.oid) OR (d.classid='pg_class'::regclass AND d.objid=c.conindid)))))`, statement.Table, fieldNames).Scan(&constrained).Error; err != nil {
			return err
		}
		if constrained {
			return fmt.Errorf("retained additive fields on public.%s have an unexpected constraint", statement.Table)
		}
		var indexed bool
		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_namespace n ON n.oid=t.relnamespace JOIN pg_depend d ON d.classid='pg_class'::regclass AND d.objid=i.indexrelid AND d.refclassid='pg_class'::regclass AND d.refobjid=t.oid JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=d.refobjsubid AND NOT a.attisdropped WHERE n.nspname='public' AND t.relname=? AND a.attname IN ?)`, statement.Table, fieldNames).Scan(&indexed).Error; err != nil {
			return err
		}
		if indexed {
			return fmt.Errorf("retained additive fields on public.%s have an unexpected index", statement.Table)
		}
		columns, err := tx.Migrator().ColumnTypes(plan.model)
		if err != nil {
			return err
		}
		for _, name := range plan.fields {
			field := statement.Schema.FieldsByName[name]
			var existing gorm.ColumnType
			for _, column := range columns {
				if column.Name() == field.DBName {
					existing = column
					break
				}
			}
			if existing == nil {
				if err := tx.Migrator().AddColumn(plan.model, name); err != nil {
					return err
				}
				continue
			}
			base := strings.Split(tx.Dialector.DataTypeOf(field), "(")[0]
			aliases := append(tx.Migrator().GetTypeAliases(existing.DatabaseTypeName()), existing.DatabaseTypeName())
			typeMatches := false
			for _, alias := range aliases {
				if alias == base {
					typeMatches = true
				}
			}
			nullable, nullKnown := existing.Nullable()
			defaultValue, defaultKnown := existing.DefaultValue()
			size, sizeKnown := existing.Length()
			if !typeMatches || !nullKnown || nullable || !defaultKnown || defaultValue != field.DefaultValue || field.Size > 0 && base == "varchar" && (!sizeKnown || size != int64(field.Size)) {
				return fmt.Errorf("retained column public.%s.%s differs from its current owner definition", statement.Table, field.DBName)
			}
		}
	}
	return nil
}
