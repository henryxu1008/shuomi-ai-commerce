package dataacquisitionpersistence

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"

	"gorm.io/gorm"
	"task-processor/internal/pkg/pgschema"
	"task-processor/internal/product/dataacquisition"
)

// Fixed native physical contract, independently checked against the fresh owner
// installer in PostgreSQL tests. It contains no business rows or role secrets.
//
//go:embed native_price_contract.json
var nativePriceContract []byte

const priceCatalogSQL = `WITH rels AS (SELECT c.* FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname IN('data_acquisition_jobs','data_acquisition_items'))
SELECT jsonb_build_object(
 'tables',(SELECT jsonb_agg(jsonb_build_array(relname,relkind,relpersistence,relrowsecurity,relforcerowsecurity,relispartition,relreplident) ORDER BY relname) FROM rels),
 'columns',(SELECT jsonb_agg(jsonb_build_array(r.relname,a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,a.attidentity,a.attgenerated,a.attndims,pg_get_expr(d.adbin,d.adrelid),co.collname) ORDER BY r.relname,a.attname) FROM rels r JOIN pg_attribute a ON a.attrelid=r.oid LEFT JOIN pg_attrdef d ON d.adrelid=r.oid AND d.adnum=a.attnum LEFT JOIN pg_collation co ON co.oid=a.attcollation WHERE a.attnum>0 AND NOT a.attisdropped),
 'constraints',(SELECT jsonb_agg(jsonb_build_array(r.relname,k.conname,k.contype,k.convalidated,k.condeferrable,k.condeferred,k.connoinherit,pg_get_constraintdef(k.oid)) ORDER BY r.relname,k.conname) FROM rels r JOIN pg_constraint k ON k.conrelid=r.oid),
 'indexes',(SELECT jsonb_agg(jsonb_build_array(r.relname,c.relname,i.indisvalid,i.indisready,i.indisunique,i.indisprimary,pg_get_indexdef(i.indexrelid)) ORDER BY r.relname,c.relname) FROM rels r JOIN pg_index i ON i.indrelid=r.oid JOIN pg_class c ON c.oid=i.indexrelid),
 'extras',jsonb_build_array((SELECT count(*) FROM pg_trigger WHERE tgrelid IN(SELECT oid FROM rels) AND NOT tgisinternal),(SELECT count(*) FROM pg_rewrite WHERE ev_class IN(SELECT oid FROM rels)),(SELECT count(*) FROM pg_inherits WHERE inhrelid IN(SELECT oid FROM rels) OR inhparent IN(SELECT oid FROM rels)),(SELECT count(*) FROM pg_policy WHERE polrelid IN(SELECT oid FROM rels)))
)::text`

func ReadPriceContract(ctx context.Context, db *gorm.DB) ([]byte, error) {
	var raw string
	err := db.WithContext(ctx).Raw(priceCatalogSQL).Scan(&raw).Error
	return []byte(raw), err
}

// PriceContractState accepts only the exact current B or fixed pre-price A.
// The latter is only consumed by explicit maintenance, never by serving.
func PriceContractState(ctx context.Context, db *gorm.DB) (bool, error) {
	raw, err := ReadPriceContract(ctx, db)
	if err != nil {
		return false, err
	}
	if equalPriceContract(raw, false) {
		return true, nil
	}
	if equalPriceContract(raw, true) {
		return false, nil
	}
	return false, fmt.Errorf("%w: acquisition schema differs from complete native A/B contract", dataacquisition.ErrUnavailable)
}

func equalPriceContract(raw []byte, baseline bool) bool {
	var got, want map[string]any
	if json.Unmarshal(raw, &got) != nil || json.Unmarshal(nativePriceContract, &want) != nil {
		return false
	}
	if baseline {
		for _, section := range []string{"columns", "constraints"} {
			var filtered []any
			for _, v := range want[section].([]any) {
				r := v.([]any)
				if r[0] == "data_acquisition_jobs" && (r[1] == "price_snapshot" || r[1] == "data_acquisition_price_snapshot") {
					continue
				}
				filtered = append(filtered, r)
			}
			want[section] = filtered
		}
	}
	for _, section := range []string{"constraints", "indexes"} {
		a, okA := got[section].([]any)
		b, okB := want[section].([]any)
		if !okA || !okB || len(a) != len(b) {
			return false
		}
		position := 7
		if section == "indexes" {
			position = 6
		}
		for i := range a {
			x, okX := a[i].([]any)
			y, okY := b[i].([]any)
			if !okX || !okY || len(x) != len(y) || len(x) <= position {
				return false
			}
			sx, okX := x[position].(string)
			sy, okY := y[position].(string)
			if !okX || !okY {
				return false
			}
			equal := pgschema.SchemaConstraintEqual(sx, sy)
			if section == "indexes" {
				equal = pgschema.SchemaIndexEqual(sx, sy)
			}
			if !equal {
				return false
			}
			x[position] = sy
		}
	}
	return reflect.DeepEqual(got, want)
}
