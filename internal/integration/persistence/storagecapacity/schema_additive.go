package storagecapacitystore

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"gorm.io/gorm"
	"task-processor/internal/ledger/storagecapacity"
)

func schemaModels() []any {
	return []any{&accountRow{}, &reservationRow{}, &objectRow{}, &bindingRow{}, &lotRow{}, &namespaceRow{}}
}

// EnsureDisabledSchema creates the current six tables only when all are absent.
// A complete empty installation is checked read-only; partial or changed schema
// is never repaired by this retained-install entry.
func EnsureDisabledSchema(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || tx == nil || tx.Statement == nil || tx.Dialector == nil || tx.Dialector.Name() != "postgres" {
		return storagecapacity.ErrUnavailable
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return storagecapacity.ErrUnavailable
	}
	tx = tx.WithContext(ctx)
	var public bool
	if err := tx.Raw("SELECT current_schema()='public'").Scan(&public).Error; err != nil {
		return err
	}
	if !public {
		return fmt.Errorf("capacity schema installation requires public")
	}
	models := schemaModels()
	existing := 0
	for _, model := range models {
		if tx.Migrator().HasTable(model) {
			existing++
		}
	}
	if existing == 0 {
		return AutoMigrate(tx.WithContext(ctx))
	}
	if existing != len(models) {
		return fmt.Errorf("partial capacity schema cannot be changed by retained additive installation")
	}
	if err := VerifyDisabled(ctx, tx); err != nil {
		return err
	}
	for _, model := range models {
		statement := &gorm.Statement{DB: tx}
		if err := statement.Parse(model); err != nil {
			return err
		}
		// These current models define primary keys and named indexes, but no
		// CHECK, foreign-key, exclusion or other additional table constraints.
		var constrained bool
		if err := tx.Raw(`SELECT EXISTS (SELECT 1 FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname=? AND c.contype<>'p')`, statement.Table).Scan(&constrained).Error; err != nil {
			return err
		}
		if constrained {
			return fmt.Errorf("capacity table public.%s has an unexpected constraint", statement.Table)
		}
		columns, err := tx.Migrator().ColumnTypes(model)
		if err != nil {
			return err
		}
		if len(columns) != len(statement.Schema.DBNames) {
			return fmt.Errorf("capacity table public.%s has a different shape", statement.Table)
		}
		for _, column := range columns {
			field := statement.Schema.FieldsByDBName[column.Name()]
			if field == nil {
				return fmt.Errorf("unexpected capacity column")
			}
			base := strings.Split(tx.Dialector.DataTypeOf(field), "(")[0]
			aliases := append(tx.Migrator().GetTypeAliases(column.DatabaseTypeName()), column.DatabaseTypeName())
			nullable, nullKnown := column.Nullable()
			primary, primaryKnown := column.PrimaryKey()
			size, sizeKnown := column.Length()
			defaultValue, defaultKnown := column.DefaultValue()
			if !slices.Contains(aliases, base) || !nullKnown || nullable != (!field.NotNull && !field.PrimaryKey) || !primaryKnown || primary != field.PrimaryKey || defaultKnown != field.HasDefaultValue || defaultKnown && defaultValue != field.DefaultValue || base == "varchar" && (!sizeKnown || size != int64(field.Size)) {
				return fmt.Errorf("capacity column public.%s.%s differs from its owner definition", statement.Table, column.Name())
			}
		}
		indexes := statement.Schema.ParseIndexes()
		var indexCount int64
		if err := tx.Raw(`SELECT count(*) FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname='public' AND t.relname=? AND NOT i.indisprimary`, statement.Table).Scan(&indexCount).Error; err != nil {
			return err
		}
		if indexCount != int64(len(indexes)) {
			return fmt.Errorf("capacity table public.%s has a different index set", statement.Table)
		}
		for _, index := range indexes {
			var observed struct {
				Unique  bool
				Columns string
			}
			result := tx.Raw(`SELECT i.indisunique AS unique, (SELECT json_agg(a.attname ORDER BY k.ordinality)::text FROM unnest(i.indkey) WITH ORDINALITY k(attnum,ordinality) JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=k.attnum) AS columns FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_namespace n ON n.oid=t.relnamespace JOIN pg_class x ON x.oid=i.indexrelid WHERE n.nspname='public' AND t.relname=? AND x.relname=? AND i.indpred IS NULL AND i.indexprs IS NULL AND i.indisvalid AND i.indisready`, statement.Table, index.Name).Scan(&observed)
			if result.Error != nil {
				return result.Error
			}
			var actual []string
			if result.RowsAffected != 1 || json.Unmarshal([]byte(observed.Columns), &actual) != nil {
				return fmt.Errorf("capacity index is missing")
			}
			var expected []string
			for _, field := range index.Fields {
				expected = append(expected, field.DBName)
			}
			if observed.Unique != (index.Class == "UNIQUE") || !slices.Equal(actual, expected) {
				return fmt.Errorf("capacity index differs from its owner definition")
			}
		}
	}
	return nil
}
