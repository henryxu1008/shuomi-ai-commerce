package storeobservations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"reflect"
	"task-processor/internal/pkg/pgschema"
	"task-processor/internal/storecenter"
)

var alignmentChecks = []struct{ table, name string }{
	{"heads", "heads_kind_check"}, {"records", "records_kind_check"},
	{"records", "store_observation_record_variant"}, {"syncs", "syncs_kind_check"},
}

// One fixed observed precondition. It is never used by serving or as a fallback.
var alignmentBaselineChecks = map[string]string{
	"heads.heads_kind_check":                   `CHECK (((kind)::text = ANY ((ARRAY['products'::character varying, 'orders'::character varying])::text[])))`,
	"records.records_kind_check":               `CHECK (((kind)::text = ANY ((ARRAY['products'::character varying, 'orders'::character varying])::text[])))`,
	"records.store_observation_record_variant": `CHECK (((((kind)::text = 'products'::text) AND (NOT (payload ? 'order'::text)) AND (((payload ? 'product'::text) AND (NOT (payload ? 'walmartProduct'::text)) AND (NOT (payload ? 'temuProduct'::text)) AND (((payload -> 'product'::text) ->> 'id'::text) = (platform_id)::text)) OR ((payload ? 'temuProduct'::text) AND (NOT (payload ? 'walmartProduct'::text)) AND (NOT (payload ? 'product'::text)) AND (((payload -> 'temuProduct'::text) ->> 'id'::text) = (platform_id)::text) AND (((payload -> 'temuProduct'::text) ->> 'entity'::text) = ANY (ARRAY['local_goods'::text, 'supplier_skc'::text]))) OR ((payload ? 'walmartProduct'::text) AND (NOT (payload ? 'product'::text)) AND (NOT (payload ? 'temuProduct'::text)) AND (jsonb_typeof((payload -> 'walmartProduct'::text)) = 'object'::text) AND (COALESCE(((payload -> 'walmartProduct'::text) ->> 'sku'::text), ''::text) <> ''::text) AND (((payload -> 'walmartProduct'::text) ->> 'sku'::text) = (platform_id)::text)))) OR (((kind)::text = 'orders'::text) AND (payload ? 'order'::text) AND (((payload -> 'order'::text) ->> 'id'::text) = (platform_id)::text) AND (((payload -> 'order'::text) ->> 'site'::text) = 'shein-us'::text) AND (NOT (payload ? 'product'::text)) AND (NOT (payload ? 'temuProduct'::text)) AND (NOT (payload ? 'walmartProduct'::text)))))`,
	"syncs.syncs_kind_check":                   `CHECK (((kind)::text = ANY ((ARRAY['products'::character varying, 'orders'::character varying])::text[])))`,
}

// InspectEmptyContractTx proves the whole native A or B contract, not keywords.
func InspectEmptyContractTx(ctx context.Context, tx *sql.Tx) (bool, error) {
	if ctx == nil || tx == nil {
		return false, errors.New("pinned observation transaction required")
	}
	var empty, ordinary bool
	if err := tx.QueryRowContext(ctx, `SELECT NOT(EXISTS(SELECT 1 FROM store_observations.commands) OR EXISTS(SELECT 1 FROM store_observations.heads) OR EXISTS(SELECT 1 FROM store_observations.records) OR EXISTS(SELECT 1 FROM store_observations.syncs))`).Scan(&empty); err != nil || !empty {
		return false, errors.New("empty observation maintenance refuses nonempty tables")
	}
	if err := tx.QueryRowContext(ctx, `SELECT NOT EXISTS(SELECT 1 FROM pg_rewrite r JOIN pg_class c ON c.oid=r.ev_class WHERE c.relnamespace='store_observations'::regnamespace) AND NOT EXISTS(SELECT 1 FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid WHERE c.relnamespace='store_observations'::regnamespace AND (NOT k.conislocal OR k.coninhcount<>0))`).Scan(&ordinary); err != nil || !ordinary {
		return false, errors.New("observation rules or inheritance differ")
	}
	actual, expected, err := storecenter.ReadObservationsAlignmentCatalog(ctx, tx)
	if err != nil {
		return false, err
	}
	if alignmentCatalogEqual(actual, expected, false) {
		return true, nil
	}
	if alignmentCatalogEqual(actual, expected, true) {
		return false, nil
	}
	return false, errors.New("observation contract is mixed, unknown or differs from the exact baseline")
}

func alignmentCatalogEqual(actual, expected []byte, baseline bool) bool {
	var got, want map[string]any
	if json.Unmarshal(actual, &got) != nil || json.Unmarshal(expected, &want) != nil {
		return false
	}
	if baseline {
		for _, value := range want["constraints"].([]any) {
			r := value.([]any)
			if def, ok := alignmentBaselineChecks[r[0].(string)+"."+r[1].(string)]; ok {
				r[3] = def
			}
		}
	}
	for _, section := range []string{"constraints", "indexes"} {
		a, okA := got[section].([]any)
		b, okB := want[section].([]any)
		if !okA || !okB || len(a) != len(b) {
			return false
		}
		for i := range a {
			x, okX := a[i].([]any)
			y, okY := b[i].([]any)
			position := 3
			if section == "indexes" {
				position = 1
			}
			if !okX || !okY || len(x) != len(y) || len(x) <= position {
				return false
			}
			defA, okA := x[position].(string)
			defB, okB := y[position].(string)
			if !okA || !okB {
				return false
			}
			same := pgschema.SchemaConstraintEqual(defA, defB)
			if section == "indexes" {
				same = pgschema.SchemaIndexEqual(defA, defB)
			}
			if !same {
				return false
			}
			x[position] = defB
		}
	}
	return reflect.DeepEqual(got, want)
}

// AlignEmptyContractTx only replaces the four admitted CHECKs. Its caller owns
// the native identity/permission checks, fixed table locks and atomic commit.
func AlignEmptyContractTx(ctx context.Context, tx *sql.Tx) error {
	current, err := InspectEmptyContractTx(ctx, tx)
	if err != nil || current {
		return err
	}
	_, expected, err := storecenter.ReadObservationsAlignmentCatalog(ctx, tx)
	if err != nil {
		return err
	}
	var contract struct {
		Constraints [][]any `json:"constraints"`
	}
	if err = json.Unmarshal(expected, &contract); err != nil {
		return err
	}
	for _, check := range alignmentChecks {
		var definition string
		for _, row := range contract.Constraints {
			if row[0] == check.table && row[1] == check.name {
				definition, _ = row[3].(string)
			}
		}
		if definition == "" {
			return errors.New("canonical observation CHECK unavailable")
		}
		// Identifiers and definitions come only from fixed native owner contracts.
		if _, err = tx.ExecContext(ctx, "ALTER TABLE store_observations."+check.table+" DROP CONSTRAINT "+check.name+", ADD CONSTRAINT "+check.name+" "+definition); err != nil {
			return err
		}
	}
	current, err = InspectEmptyContractTx(ctx, tx)
	if err != nil {
		return err
	}
	if !current {
		return errors.New("observation postcondition differs")
	}
	return nil
}
