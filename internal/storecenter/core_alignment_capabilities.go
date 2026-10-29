package storecenter

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

// Catalog contracts are read-only snapshots of the existing capability owners'
// native schema.sql, verified against those installers in the PostgreSQL fixture.
// No business values, ACLs, OIDs or generated timestamps enter these contracts.
//
//go:embed core_alignment_observations.json
var retainedObservationsContract []byte

//go:embed core_alignment_cockpit.json
var retainedCockpitContract []byte

const retainedCapabilityCatalogQuery = `SELECT jsonb_build_object(
 'relations',(SELECT jsonb_agg(jsonb_build_array(c.relname,c.relkind,c.relpersistence,c.relispartition,c.relrowsecurity,c.relforcerowsecurity,pg_get_userbyid(c.relowner)) ORDER BY c.relname) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relkind NOT IN ('i','I','t')),
 'columns',(SELECT jsonb_agg(jsonb_build_array(c.relname,a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,pg_get_expr(d.adbin,d.adrelid),a.attidentity,a.attgenerated,a.attcollation::regcollation::text) ORDER BY c.relname,a.attnum) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE n.nspname=$1 AND c.relkind IN ('r','p','v','m','f') AND a.attnum>0 AND NOT a.attisdropped),
 'constraints',(SELECT jsonb_agg(jsonb_build_array(c.relname,k.conname,k.contype,pg_get_constraintdef(k.oid),k.convalidated,k.connoinherit,k.condeferrable,k.condeferred) ORDER BY c.relname,k.conname) FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1),
 'indexes',(SELECT jsonb_agg(jsonb_build_array(i.relname,pg_get_indexdef(i.oid),x.indisvalid,x.indisready) ORDER BY i.relname) FROM pg_index x JOIN pg_class i ON i.oid=x.indexrelid JOIN pg_namespace n ON n.oid=i.relnamespace WHERE n.nspname=$1),
 'triggers',(SELECT count(*) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND NOT t.tgisinternal),
 'functions',(SELECT count(*) FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname=$1),
 'types',(SELECT count(*) FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE n.nspname=$1 AND t.typrelid=0 AND t.typelem=0)
)::text`

func verifyRetainedCapabilityContracts(ctx context.Context, q coreSchemaQueryer, caps RuntimeCapabilities) error {
	for _, capability := range []struct {
		schema   string
		enabled  bool
		contract []byte
	}{
		{"store_observations", caps.Observations, retainedObservationsContract},
		{"operations_cockpit", caps.OperationsCockpit, retainedCockpitContract},
	} {
		if !capability.enabled {
			continue
		}
		var raw string
		if err := q.QueryRowContext(ctx, retainedCapabilityCatalogQuery, capability.schema).Scan(&raw); err != nil {
			return errors.New("retained Store capability catalog unavailable")
		}
		var got, want any
		if json.Unmarshal([]byte(raw), &got) != nil || json.Unmarshal(capability.contract, &want) != nil || !reflect.DeepEqual(got, want) {
			return fmt.Errorf("retained Store capability structure mismatch: %s", capability.schema)
		}
		// The common allowlist rejects excess effective privileges. Explicitly
		// require every native capability grant too, including cockpit's grants.
		var complete bool
		query := `SELECT coalesce(has_schema_privilege('store_center_runtime',to_regnamespace($1),'USAGE'),false) AND NOT EXISTS (
 SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname=$1 AND c.relkind='r' AND NOT (
 has_table_privilege('store_center_runtime',c.oid,'SELECT') AND has_table_privilege('store_center_runtime',c.oid,'INSERT') AND
 ((n.nspname='store_observations' AND c.relname='commands') OR (n.nspname='operations_cockpit' AND c.relname IN ('fact_versions','goal_versions','commands')) OR has_table_privilege('store_center_runtime',c.oid,'UPDATE'))))`
		if err := q.QueryRowContext(ctx, query, capability.schema).Scan(&complete); err != nil || !complete {
			return errors.New("retained Store capability runtime grants incomplete")
		}
	}
	return nil
}
