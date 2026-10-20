package productagentruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"gorm.io/gorm"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/integration/openai"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
)

// VerifyInvocationRelation reuses the current invocation owner's native
// qualification for explicit maintenance. It performs no DDL or row writes.
func VerifyInvocationRelation(ctx context.Context, db *gorm.DB, owner string) error {
	if ctx == nil || db == nil || owner == "" {
		return fmt.Errorf("invocation owner unavailable")
	}
	spec, err := invocationContract(db)
	if err != nil {
		return err
	}
	spec.Owner = owner
	return verifyRelation(db.WithContext(ctx), spec)
}

type columnContract struct {
	Name, Type, Default string
	NotNull             bool
}
type indexContract struct {
	Name, Predicate, Constraint string
	Columns                     []string
	Unique, Primary             bool
}
type relationContract struct {
	Name, Owner         string
	Columns             []columnContract
	Indexes             []indexContract
	Checks, ForeignKeys map[string]string
}

func normalizedType(value string) string {
	value = strings.ReplaceAll(value, "varchar", "character varying")
	switch value {
	case "bigserial":
		return "bigint"
	case "serial":
		return "integer"
	case "timestamptz":
		return "timestamp with time zone"
	case "decimal":
		return "numeric"
	case "bool":
		return "boolean"
	}
	return value
}

// The GORM owners remain authoritative; this fixed composition does not copy
// credential/invocation/Catalog/source persistence rows or migrate old models.
func modelContract(db *gorm.DB, model any, owner string) (relationContract, error) {
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		return relationContract{}, err
	}
	spec := relationContract{Name: stmt.Schema.Table, Owner: owner}
	var primary []string
	for _, name := range stmt.Schema.DBNames {
		field := stmt.Schema.FieldsByDBName[name]
		if field.IgnoreMigration {
			continue
		}
		c := columnContract{Name: name, Type: normalizedType(db.Dialector.DataTypeOf(field)), NotNull: field.NotNull || field.PrimaryKey}
		if field.PrimaryKey {
			primary = append(primary, name)
		}
		if field.AutoIncrement {
			c.Default = "@owned-sequence"
		} else if field.HasDefaultValue {
			c.Default = field.DefaultValue
			if field.IndirectFieldType.Kind() == reflect.String {
				c.Default = "'" + strings.ReplaceAll(field.DefaultValue, "'", "''") + "'::" + strings.Split(c.Type, "(")[0]
			}
		}
		spec.Columns = append(spec.Columns, c)
	}
	if len(primary) > 0 {
		spec.Indexes = append(spec.Indexes, indexContract{Name: spec.Name + "_pkey", Columns: primary, Unique: true, Primary: true, Constraint: "p"})
	}
	for _, index := range stmt.Schema.ParseIndexes() {
		i := indexContract{Name: index.Name, Unique: index.Class == "UNIQUE", Predicate: index.Where}
		for _, field := range index.Fields {
			if field.Expression != "" || field.Sort != "" || field.Collate != "" {
				return spec, fmt.Errorf("unsupported current owner index definition")
			}
			i.Columns = append(i.Columns, field.DBName)
		}
		spec.Indexes = append(spec.Indexes, i)
	}
	return spec, nil
}

func runContracts(db *gorm.DB) ([]relationContract, error) {
	runs := relationContract{Name: "product_agent_runs", Owner: "product_agent_owner", Columns: []columnContract{
		{"run_id", "character varying(128)", "", true}, {"org", "character varying(128)", "", true}, {"actor", "character varying(128)", "", true}, {"context_kind", "character varying(128)", "", true}, {"context_id", "character varying(128)", "", true}, {"request_key", "character varying(128)", "", true}, {"fingerprint", "character varying(64)", "", true}, {"revision", "bigint", "", true}, {"phase", "character varying(32)", "", true}, {"payload", "bytea", "", true},
	}, Indexes: []indexContract{{Name: "product_agent_runs_pkey", Columns: []string{"run_id"}, Unique: true, Primary: true, Constraint: "p"}, {Columns: []string{"org", "actor", "context_kind", "context_id", "request_key"}, Unique: true, Constraint: "u"}}, Checks: map[string]string{
		"product_agent_runs_revision_check": "CHECK ((revision > 0))",
		"product_agent_runs_phase_check":    "CHECK (((phase)::text = ANY ((ARRAY['running'::character varying, 'interrupted'::character varying, 'human_review_required'::character varying, 'stopped'::character varying])::text[])))",
		"product_agent_runs_payload_check":  "CHECK ((octet_length(payload) <= 2097152))",
	}}
	tools := relationContract{Name: "product_agent_tool_calls", Owner: "product_agent_owner", Columns: []columnContract{{"run_id", "character varying(128)", "", true}, {"call_id", "character varying(128)", "", true}, {"payload", "bytea", "", true}}, Indexes: []indexContract{{Name: "product_agent_tool_calls_pkey", Columns: []string{"run_id", "call_id"}, Unique: true, Primary: true, Constraint: "p"}}, Checks: map[string]string{"product_agent_tool_calls_payload_check": "CHECK ((octet_length(payload) <= 8192))"}, ForeignKeys: map[string]string{"product_agent_tool_calls_run_id_fkey": "FOREIGN KEY (run_id) REFERENCES product_agent_runs(run_id)"}}
	credential, err := modelContract(db, &openai.AIClientCredential{}, "product_agent_owner")
	if err != nil {
		return nil, err
	}
	return []relationContract{runs, tools, credential}, nil
}

func invocationContract(db *gorm.DB) (relationContract, error) {
	spec, err := modelContract(db, aistore.InvocationSchemaModel(), "product_agent_owner")
	if err == nil && len(spec.Columns) != 53 {
		err = fmt.Errorf("invocation owner differs from approved 53-column contract")
	}
	return spec, err
}

func reviewContracts(db *gorm.DB) ([]relationContract, error) {
	pendingPredicate := "(state = ANY (ARRAY['pending'::text, 'accepted'::text]))"
	proposals := relationContract{Name: "product_title_proposals", Owner: "acquisition_owner", Columns: []columnContract{{"org", "text", "", true}, {"id", "text", "", true}, {"owner", "text", "", true}, {"state", "text", "", true}, {"payload", "bytea", "", true}}, Indexes: []indexContract{{Name: "product_title_proposals_pkey", Columns: []string{"org", "id"}, Unique: true, Primary: true, Constraint: "p"}, {Name: "ix_product_title_proposals_actionable_admin", Columns: []string{"org", "id"}, Predicate: pendingPredicate}, {Name: "ix_product_title_proposals_actionable_owner", Columns: []string{"org", "owner", "id"}, Predicate: pendingPredicate}}, Checks: map[string]string{
		"ck_product_title_proposals_state":         "CHECK ((state = ANY (ARRAY['pending'::text, 'accepted'::text, 'rejected'::text, 'applied'::text])))",
		"ck_product_title_proposals_payload_size":  "CHECK ((octet_length(payload) <= 65536))",
		"ck_product_title_proposals_state_payload": "CHECK ((state = ((convert_from(payload, 'UTF8'::name))::jsonb ->> 'State'::text)))",
	}}
	operations := relationContract{Name: "product_title_operations", Owner: "acquisition_owner", Columns: []columnContract{{"org", "text", "", true}, {"actor", "text", "", true}, {"operation_key", "text", "", true}, {"fingerprint", "text", "", true}, {"response", "bytea", "", true}}, Indexes: []indexContract{{Name: "product_title_operations_pkey", Columns: []string{"org", "actor", "operation_key"}, Unique: true, Primary: true, Constraint: "p"}}, Checks: map[string]string{"product_title_operations_response_check": "CHECK ((octet_length(response) <= 65536))"}}
	result := []relationContract{proposals, operations}
	models := append([]any{&catalogstore.SnapshotVersionRecord{}, &catalogstore.SnapshotHeadRecord{}}, sourcestore.SchemaModels()...)
	for _, model := range models {
		spec, err := modelContract(db, model, "acquisition_owner")
		if err != nil {
			return nil, err
		}
		result = append(result, spec)
	}
	result[4].Checks = map[string]string{
		"ck_source_publication_envelope_size":   "CHECK (((octet_length(envelope_json) >= 1) AND (octet_length(envelope_json) <= 2097152)))",
		"ck_source_publication_snapshot_size":   "CHECK (((octet_length(snapshot_json) >= 1) AND (octet_length(snapshot_json) <= 2097152)))",
		"ck_source_publication_catalog_version": "CHECK ((catalog_version > 0))",
	}
	result[5].Checks = map[string]string{"ck_source_receipt_catalog_version": "CHECK ((catalog_version > 0))"}
	return result, nil
}

func verifyRelation(db *gorm.DB, spec relationContract) error {
	var rel struct {
		Owner, Kind, Persistence  string
		Rls, Forced               bool
		Triggers, Rules, Inherits int64
	}
	err := db.Raw(`SELECT pg_catalog.pg_get_userbyid(c.relowner) AS owner,c.relkind::text AS kind,c.relpersistence::text AS persistence,c.relrowsecurity AS rls,c.relforcerowsecurity AS forced,(SELECT count(*) FROM pg_catalog.pg_trigger t WHERE t.tgrelid=c.oid AND NOT t.tgisinternal) AS triggers,(SELECT count(*) FROM pg_catalog.pg_rewrite r WHERE r.ev_class=c.oid) AS rules,(SELECT count(*) FROM pg_catalog.pg_inherits i WHERE i.inhrelid=c.oid OR i.inhparent=c.oid) AS inherits FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relname=?`, spec.Name).Scan(&rel).Error
	if err != nil {
		return err
	}
	if rel.Owner != spec.Owner || rel.Kind != "r" || rel.Persistence != "p" || rel.Rls || rel.Forced || rel.Triggers != 0 || rel.Rules != 0 || rel.Inherits != 0 {
		return fmt.Errorf("public.%s owner, durability or relation contract differs", spec.Name)
	}
	var cols []struct {
		DefaultCollation                         bool
		Name, Type, Default, Identity, Generated string
		NotNull                                  bool
		Position                                 int
	}
	err = db.Raw(`SELECT a.attname AS name,pg_catalog.format_type(a.atttypid,a.atttypmod) AS type,COALESCE(pg_catalog.pg_get_expr(d.adbin,d.adrelid),'') AS default,a.attidentity::text AS identity,a.attgenerated::text AS generated,a.attcollation=t.typcollation AS default_collation,a.attnotnull AS not_null,a.attnum AS position FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_type t ON t.oid=a.atttypid LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE a.attrelid=pg_catalog.to_regclass(?) AND a.attnum>0 AND NOT a.attisdropped ORDER BY a.attnum`, "public."+spec.Name).Scan(&cols).Error
	if err != nil {
		return err
	}
	if len(cols) != len(spec.Columns) {
		return fmt.Errorf("public.%s columns differ", spec.Name)
	}
	for i, expected := range spec.Columns {
		actual := cols[i]
		defaultOK := actual.Default == expected.Default
		if expected.Default == "@owned-sequence" {
			defaultOK = actual.Default == "nextval('ai_client_credentials_id_seq'::regclass)"
		}
		if actual.Name != expected.Name || actual.Type != expected.Type || actual.NotNull != expected.NotNull || actual.Position != i+1 || actual.Identity != "" || actual.Generated != "" || !actual.DefaultCollation || !defaultOK {
			return fmt.Errorf("public.%s.%s column contract differs", spec.Name, expected.Name)
		}
	}
	var constraints []struct {
		Name, Kind, Definition string
		Validated, Deferrable  bool
	}
	err = db.Raw(`SELECT conname AS name,contype::text AS kind,pg_catalog.pg_get_constraintdef(oid) AS definition,convalidated AS validated,condeferrable AS deferrable FROM pg_catalog.pg_constraint WHERE conrelid=pg_catalog.to_regclass(?)`, "public."+spec.Name).Scan(&constraints).Error
	if err != nil {
		return err
	}
	checks, foreign := map[string]string{}, map[string]string{}
	for _, c := range constraints {
		if !c.Validated || c.Deferrable {
			return fmt.Errorf("public.%s constraint state differs", spec.Name)
		}
		switch c.Kind {
		case "c":
			checks[c.Name] = c.Definition
		case "f":
			foreign[c.Name] = c.Definition
		case "p", "u":
		default:
			return fmt.Errorf("public.%s has unexpected constraints", spec.Name)
		}
	}
	if len(checks) != len(spec.Checks) || len(foreign) != len(spec.ForeignKeys) {
		return fmt.Errorf("public.%s constraint counts differ", spec.Name)
	}
	for name, definition := range spec.Checks {
		if checks[name] != definition {
			return fmt.Errorf("public.%s check %s differs", spec.Name, name)
		}
	}
	for name, definition := range spec.ForeignKeys {
		if foreign[name] != definition {
			return fmt.Errorf("public.%s foreign key %s differs", spec.Name, name)
		}
	}
	return verifyIndexes(db, spec)
}

func verifyIndexes(db *gorm.DB, spec relationContract) error {
	var actual []struct {
		Name, Predicate, Constraint, Method, Columns string
		Unique, Primary, Valid                       bool
	}
	err := db.Raw(`SELECT c.relname AS name,i.indisunique AS unique,i.indisprimary AS primary,i.indisvalid AND i.indisready AND i.indnatts=i.indnkeyatts AND NOT i.indisexclusion AND NOT i.indnullsnotdistinct AS valid,am.amname AS method,COALESCE(pg_catalog.pg_get_expr(i.indpred,i.indrelid),'') AS predicate,COALESCE(k.contype::text,'') AS constraint,array_to_json(ARRAY(SELECT pg_catalog.pg_get_indexdef(i.indexrelid,x,true) FROM generate_series(1,i.indnkeyatts) x))::text AS columns FROM pg_catalog.pg_index i JOIN pg_catalog.pg_class c ON c.oid=i.indexrelid JOIN pg_catalog.pg_am am ON am.oid=c.relam LEFT JOIN pg_catalog.pg_constraint k ON k.conindid=i.indexrelid AND k.conrelid=i.indrelid AND k.contype IN ('p','u') WHERE i.indrelid=pg_catalog.to_regclass(?)`, "public."+spec.Name).Scan(&actual).Error
	if err != nil {
		return err
	}
	if len(actual) != len(spec.Indexes) {
		return fmt.Errorf("public.%s index count differs", spec.Name)
	}
	used := make([]bool, len(actual))
	for _, expected := range spec.Indexes {
		found := false
		for i, index := range actual {
			if used[i] {
				continue
			}
			var columns []string
			if json.Unmarshal([]byte(index.Columns), &columns) != nil {
				return fmt.Errorf("invalid index metadata")
			}
			if (expected.Name == "" || expected.Name == index.Name) && index.Valid && index.Method == "btree" && index.Unique == expected.Unique && index.Primary == expected.Primary && index.Constraint == expected.Constraint && index.Predicate == expected.Predicate && reflect.DeepEqual(columns, expected.Columns) {
				used[i], found = true, true
				break
			}
		}
		if !found {
			return fmt.Errorf("public.%s index %s differs", spec.Name, expected.Name)
		}
	}
	return nil
}
