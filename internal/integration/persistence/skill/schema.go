package skillpersistence

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"gorm.io/gorm"
	"regexp"
	"strings"
	d "task-processor/internal/skill"
)

//go:embed schema.sql
var schemaSQL string

func InstallSchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return d.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, sql := range []string{"DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='skill_capability_owner') THEN CREATE ROLE skill_capability_owner NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS; END IF; END $$;", "CREATE SCHEMA IF NOT EXISTS skill_capability AUTHORIZATION skill_capability_owner"} {
			if e := tx.Exec(sql).Error; e != nil {
				return e
			}
		}
		var safe bool
		if e := tx.Raw(`SELECT NOT(rolcanlogin OR rolsuper OR rolcreaterole OR rolcreatedb OR rolbypassrls) AND (SELECT nspowner=pg_roles.oid FROM pg_namespace WHERE nspname='skill_capability') FROM pg_roles WHERE rolname='skill_capability_owner'`).Scan(&safe).Error; e != nil || !safe {
			return d.ErrUnavailable
		}
		for _, sql := range []string{"SET LOCAL ROLE skill_capability_owner", schemaSQL, "REVOKE ALL ON SCHEMA skill_capability FROM PUBLIC", "REVOKE ALL ON ALL TABLES IN SCHEMA skill_capability FROM PUBLIC", "RESET ROLE"} {
			if e := tx.Exec(sql).Error; e != nil {
				return e
			}
		}
		var configOwner string
		if e := tx.Raw("SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname='agent_configuration'").Scan(&configOwner).Error; e != nil || configOwner != "agent_configuration_owner" {
			return d.ErrUnavailable
		}
		for _, sql := range []string{"SET LOCAL ROLE agent_configuration_owner", `CREATE TABLE IF NOT EXISTS agent_configuration.skill_admissions(organization_id varchar(128) NOT NULL,agent_id varchar(128) NOT NULL CHECK(agent_id='product.content.agent'),allow_personal boolean NOT NULL,revision bigint NOT NULL CHECK(revision>0),updated_by varchar(128) NOT NULL,updated_at timestamptz NOT NULL,PRIMARY KEY(organization_id,agent_id),FOREIGN KEY(organization_id,agent_id) REFERENCES agent_configuration.organization_agents(organization_id,agent_id))`, "REVOKE ALL ON agent_configuration.skill_admissions FROM PUBLIC", "RESET ROLE"} {
			if e := tx.Exec(sql).Error; e != nil {
				return e
			}
		}
		return VerifySchema(ctx, tx)
	})
}

// Verification never installs schema or changes grants.
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return d.ErrUnavailable
	}
	expected := map[string]string{
		"skill_capability.skills":              "created_at:timestamp with time zone:false|creator:character varying(128):false|head_version:bigint:false|id:uuid:false|owner_id:character varying(128):false|owner_kind:character varying(16):false|revision:bigint:false|updated_at:timestamp with time zone:false",
		"skill_capability.versions":            "allow_copy:boolean:false|allow_export:boolean:false|created_at:timestamp with time zone:false|digest:character varying(64):false|files:bytea:false|skill_id:uuid:false|source_ref:bytea:true|state:character varying(8):false|version:bigint:false|visibility:character varying(8):false",
		"skill_capability.commands":            "actor_id:character varying(128):false|fingerprint:character varying(64):false|key:uuid:false|owner_id:character varying(128):false|owner_kind:character varying(16):false|receipt:bytea:false",
		"skill_capability.share_requests":      "consent:boolean:false|created_at:timestamp with time zone:false|id:uuid:false|organization_id:character varying(128):false|result_skill_id:uuid:true|source_digest:character varying(64):false|source_skill_id:uuid:false|source_version:bigint:false|state:character varying(16):false|submitted_by:character varying(128):false|updated_at:timestamp with time zone:false",
		"agent_configuration.skill_admissions": "agent_id:character varying(128):false|allow_personal:boolean:false|organization_id:character varying(128):false|revision:bigint:false|updated_at:timestamp with time zone:false|updated_by:character varying(128):false",
	}
	// Frozen PostgreSQL constraint contracts from this explicit installer DDL.
	// Names, qualified foreign targets, expressions and validation all participate;
	// merely finding a CHECK or primary key cannot admit a weakened schema.
	constraints := map[string]string{
		"agent_configuration.skill_admissions": "b207bf1e2bce9ebd9ae9cc868d5c0b4f4f60c8ef9cb78cf37e300c23e71611a4",
		"skill_capability.commands":            "0dba9a05550b5bf83164e603ebd3edbb82c5fb954d94bfb52e0fdc49eb578b5c",
		"skill_capability.share_requests":      "a136a9aa05483ac138119194bd6c5689a566537d6f5c6c4f40107193ea48afbc",
		"skill_capability.skills":              "bc12f6b39f71ad181445f691ad41a6ab0eb879138b55b6b4398e2e4938768f94",
		"skill_capability.versions":            "019e8629374d2a71f80b1f211c5c7e48a18c708c9a7f77f884d3197445b6b1cc",
	}
	for table, shape := range expected {
		var row struct {
			Shape, Owner string
			Valid        bool
		}
		if e := db.WithContext(ctx).Raw(`SELECT (SELECT string_agg(a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||(NOT a.attnotnull)::text,'|' ORDER BY a.attname) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped) AS shape,pg_get_userbyid(c.relowner) AS owner,NOT EXISTS(SELECT 1 FROM pg_constraint x WHERE x.conrelid=c.oid AND NOT x.convalidated) AND EXISTS(SELECT 1 FROM pg_constraint x WHERE x.conrelid=c.oid AND x.contype='p') AS valid FROM pg_class c WHERE c.oid=to_regclass(?) AND c.relkind='r'`, table).Scan(&row).Error; e != nil {
			return d.ErrUnavailable
		}
		owner := "skill_capability_owner"
		if strings.HasPrefix(table, "agent_configuration.") {
			owner = "agent_configuration_owner"
		}
		if row.Shape != shape || row.Owner != owner || !row.Valid {
			return d.ErrUnavailable
		}
		var contract string
		if e := db.WithContext(ctx).Raw(`SELECT string_agg(x.conname||':'||pg_get_constraintdef(x.oid)||':'||x.convalidated::text,'|' ORDER BY x.conname) FROM pg_constraint x WHERE x.conrelid=to_regclass(?)`, table).Scan(&contract).Error; e != nil {
			return d.ErrUnavailable
		}
		sum := sha256.Sum256([]byte(contract))
		if hex.EncodeToString(sum[:]) != constraints[table] {
			return d.ErrUnavailable
		}
	}
	return nil
}

func GrantRuntime(ctx context.Context, db *gorm.DB, role string) error {
	if !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(role) {
		return d.ErrInvalid
	}
	if e := VerifySchema(ctx, db); e != nil {
		return e
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var safe bool
		if e := tx.Raw(`SELECT NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR pg_has_role(oid,'skill_capability_owner','MEMBER') OR pg_has_role(oid,'agent_configuration_owner','MEMBER')) FROM pg_roles WHERE rolname=?`, role).Scan(&safe).Error; e != nil || !safe {
			return d.ErrForbidden
		}
		quoted := `"` + role + `"`
		for _, sql := range []string{"REVOKE ALL ON SCHEMA skill_capability FROM " + quoted, "REVOKE ALL ON ALL TABLES IN SCHEMA skill_capability FROM " + quoted, "GRANT USAGE ON SCHEMA skill_capability TO " + quoted, "GRANT SELECT,INSERT ON ALL TABLES IN SCHEMA skill_capability TO " + quoted, "GRANT UPDATE(head_version,revision,updated_at) ON skill_capability.skills TO " + quoted, "GRANT UPDATE(state,visibility) ON skill_capability.versions TO " + quoted, "GRANT UPDATE(state,result_skill_id,updated_at) ON skill_capability.share_requests TO " + quoted, "GRANT SELECT,INSERT ON agent_configuration.skill_admissions TO " + quoted, "GRANT UPDATE(allow_personal,revision,updated_by,updated_at) ON agent_configuration.skill_admissions TO " + quoted} {
			if e := tx.Exec(sql).Error; e != nil {
				return e
			}
		}
		var unsafe bool
		if e := tx.Raw(`SELECT has_schema_privilege(?,'skill_capability','CREATE') OR has_table_privilege(?,'skill_capability.versions','DELETE,TRUNCATE,UPDATE') OR has_any_column_privilege(?,'skill_capability.commands','UPDATE') OR has_column_privilege(?,'skill_capability.versions','files','UPDATE') OR has_column_privilege(?,'skill_capability.versions','digest','UPDATE') OR has_column_privilege(?,'skill_capability.skills','owner_id','UPDATE')`, role, role, role, role, role, role).Scan(&unsafe).Error; e != nil || unsafe {
			return d.ErrForbidden
		}
		return nil
	})
}
