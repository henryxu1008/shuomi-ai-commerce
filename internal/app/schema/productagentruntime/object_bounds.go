package productagentruntime

import (
	"fmt"

	"gorm.io/gorm"
	configstore "task-processor/internal/integration/persistence/agentconfig"
	skillstore "task-processor/internal/integration/persistence/skill"
)

// Only the three fixed identities in the current target database are examined.
// Inherited/PUBLIC/column access uses PostgreSQL's effective privilege checks,
// as in the existing Workbench qualifier; no ACL is rewritten or repaired.
func verifyObjectBounds(db *gorm.DB, role string) error {
	allowed, err := ownerGrantBounds(db, role)
	if err != nil {
		return err
	}
	var schemaAuthority bool
	if err = db.Raw(`SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_namespace n WHERE n.nspname<>'information_schema' AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\' AND (pg_catalog.has_schema_privilege(?,n.oid,'CREATE') OR pg_catalog.has_schema_privilege(?,n.oid,'USAGE WITH GRANT OPTION')))`, role, role).Scan(&schemaAuthority).Error; err != nil {
		return err
	}
	if schemaAuthority {
		return fmt.Errorf("role %s has unapproved schema authority", role)
	}
	var rows []struct {
		Object, ColumnName, Privilege string
		Grantable, Sequence           bool
	}
	if err = db.Raw(effectiveObjectPrivileges, role).Scan(&rows).Error; err != nil {
		return err
	}
	for _, p := range rows {
		if p.Sequence {
			if !p.Grantable && role == "title_credential_writer" && p.Object == "public.ai_client_credentials_id_seq" && p.Privilege == "USAGE" {
				continue
			}
		} else if grant, ok := allowed[p.Object]; ok && !p.Grantable && grant.Allows(p.Privilege, p.ColumnName) {
			continue
		}
		return fmt.Errorf("role %s has unapproved %s authority on %s", role, p.Privilege, p.Object)
	}
	return nil
}

func ownerGrantBounds(db *gorm.DB, role string) (map[string]configstore.RuntimeTableGrant, error) {
	var targets []tableGrant
	switch role {
	case "product_agent_runtime", "title_credential_writer":
		ledger, err := invocationContract(db)
		if err != nil {
			return nil, err
		}
		targets = runGrants(ledger)
	case "product_review_runtime":
		targets = reviewGrants()
	default:
		return nil, fmt.Errorf("unsupported native runtime role")
	}
	result := map[string]configstore.RuntimeTableGrant{}
	for _, g := range targets {
		if g.Role == role {
			relation := "public." + g.Table
			result[relation] = configstore.RuntimeTableGrant{Relation: relation, Select: g.Select, Insert: g.Insert, UpdateColumns: g.Update}
		}
	}
	if role == "product_agent_runtime" {
		// These are the same definitions consumed by the original initializers,
		// not a second copy of the Skill/configuration permission authority.
		shared := append(configstore.RuntimeTableGrants(), configstore.SkillAdmissionRuntimeTableGrant())
		shared = append(shared, skillstore.RuntimeTableGrants()...)
		for _, g := range shared {
			result[g.Relation] = g
		}
	}
	return result, nil
}

const effectiveObjectPrivileges = `WITH target AS (SELECT oid FROM pg_catalog.pg_roles WHERE rolname=?),
relations AS (SELECT c.oid,c.relkind,c.relowner,n.nspname||'.'||c.relname AS object FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname<>'information_schema' AND n.nspname NOT LIKE 'pg\_%' ESCAPE '\')
SELECT r.object,''::text AS column_name,p.privilege,pg_catalog.has_table_privilege(t.oid,r.oid,p.privilege||' WITH GRANT OPTION') AS grantable,false AS sequence
FROM relations r CROSS JOIN target t CROSS JOIN LATERAL (SELECT privilege_type AS privilege FROM pg_catalog.aclexplode(pg_catalog.acldefault('r',r.relowner))) p
WHERE r.relkind IN ('r','p','v','m','f') AND pg_catalog.has_table_privilege(t.oid,r.oid,p.privilege)
UNION ALL
SELECT r.object,a.attname,p.privilege,pg_catalog.has_column_privilege(t.oid,r.oid,a.attnum,p.privilege||' WITH GRANT OPTION'),false
FROM relations r JOIN pg_catalog.pg_attribute a ON a.attrelid=r.oid AND a.attnum>0 AND NOT a.attisdropped CROSS JOIN target t CROSS JOIN (VALUES ('SELECT'),('INSERT'),('UPDATE'),('REFERENCES')) p(privilege)
WHERE r.relkind IN ('r','p','v','m','f') AND pg_catalog.has_column_privilege(t.oid,r.oid,a.attnum,p.privilege)
UNION ALL
SELECT r.object,''::text,p.privilege,pg_catalog.has_sequence_privilege(t.oid,r.oid,p.privilege||' WITH GRANT OPTION'),true
FROM relations r CROSS JOIN target t CROSS JOIN (VALUES ('USAGE'),('SELECT'),('UPDATE')) p(privilege)
WHERE r.relkind='S' AND pg_catalog.has_sequence_privilege(t.oid,r.oid,p.privilege)`
