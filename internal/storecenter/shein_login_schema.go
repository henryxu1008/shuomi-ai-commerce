package storecenter

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

const sheinLoginTable = "workbench_store_shein_login_bindings"
const sheinLoginDDL = `CREATE TABLE public.workbench_store_shein_login_bindings (
 organization_id VARCHAR(200) NOT NULL,store_id CHAR(36) NOT NULL,
 installation_id CHAR(36) NOT NULL,binding_revision CHAR(36) NOT NULL,config_fingerprint VARCHAR(64) NOT NULL,
 generation BIGINT NOT NULL,state VARCHAR(32) NOT NULL,active_attempt_id VARCHAR(36) NOT NULL,owner_nonce VARCHAR(36) NOT NULL,closed_confirmed BOOLEAN NOT NULL,
 PRIMARY KEY(organization_id,store_id),
 CONSTRAINT shein_login_store FOREIGN KEY(organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT shein_login_generation CHECK(generation>0 AND octet_length(config_fingerprint)=64),
 CONSTRAINT shein_login_state CHECK(
  (state='active' AND ((active_attempt_id='' AND owner_nonce='' AND closed_confirmed) OR (active_attempt_id<>'' AND owner_nonce<>'' AND NOT closed_confirmed))) OR
  (state='quiescing' AND active_attempt_id<>'' AND owner_nonce<>'' AND NOT closed_confirmed) OR
  (state='quiesced' AND active_attempt_id='' AND owner_nonce='' AND closed_confirmed))
)`

// InstallSheinLoginCapability is an explicit schema-owner operation for an
// authorized fresh Store installation. Serving never invokes it.
func InstallSheinLoginCapability(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return ErrDependencyUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if e := tx.Table("public.workbench_stores").Count(&count).Error; e != nil || count != 0 {
			return errors.New("SHEIN worker capability requires an authorized fresh empty Store")
		}
		for _, q := range []string{sheinLoginDDL,
			`DO $$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO shein_login_worker',current_database()); END $$`,
			`GRANT USAGE ON SCHEMA public TO shein_login_worker`,
			`GRANT SELECT ON public.workbench_stores,public.workbench_store_member_grants,public.workbench_store_audit_logs TO shein_login_worker`,
			`GRANT UPDATE(external_store_id,identity_key,version,updated_by,updated_at) ON public.workbench_stores TO shein_login_worker`,
			`GRANT INSERT ON public.workbench_store_audit_logs TO shein_login_worker`,
			`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_shein_login_bindings TO shein_login_worker`,
		} {
			if e := tx.Exec(q).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
func verifySheinLoginSchema(ctx context.Context, q coreSchemaQueryer) error {
	var valid bool
	e := q.QueryRowContext(ctx, `SELECT count(*)=4 AND bool_and(k.convalidated AND NOT k.condeferrable AND NOT k.condeferred AND
  (k.conname,k.contype::text) IN (('workbench_store_shein_login_bindings_pkey','p'),('shein_login_store','f'),('shein_login_generation','c'),('shein_login_state','c')))
 FROM pg_constraint k WHERE k.conrelid='public.workbench_store_shein_login_bindings'::regclass`).Scan(&valid)
	if e != nil || !valid {
		return errors.New("SHEIN login schema boundary unavailable")
	}
	// Exact PostgreSQL 16 canonical definitions, matching the core schema owner.
	definitions := map[string]string{
		"workbench_store_shein_login_bindings_pkey": `PRIMARY KEY (organization_id, store_id)`,
		"shein_login_store":                         `FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)`,
		"shein_login_generation":                    `CHECK (((generation > 0) AND (octet_length((config_fingerprint)::text) = 64)))`,
		"shein_login_state":                         `CHECK (((((state)::text = 'active'::text) AND ((((active_attempt_id)::text = ''::text) AND ((owner_nonce)::text = ''::text) AND closed_confirmed) OR (((active_attempt_id)::text <> ''::text) AND ((owner_nonce)::text <> ''::text) AND (NOT closed_confirmed)))) OR (((state)::text = 'quiescing'::text) AND ((active_attempt_id)::text <> ''::text) AND ((owner_nonce)::text <> ''::text) AND (NOT closed_confirmed)) OR (((state)::text = 'quiesced'::text) AND ((active_attempt_id)::text = ''::text) AND ((owner_nonce)::text = ''::text) AND closed_confirmed)))`,
	}
	for name, want := range definitions {
		var definition string
		if err := q.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='public.workbench_store_shein_login_bindings'::regclass AND conname=$1`, name).Scan(&definition); err != nil || definition != want {
			return errors.New("SHEIN login constraint contract mismatch")
		}
	}

	return nil
}

// VerifySheinLoginRuntime checks the dedicated role, exact column UPDATE set,
// every accessible user relation and dangerous role/schema/database privileges.
func VerifySheinLoginRuntime(ctx context.Context, db *gorm.DB) error {
	if e := VerifyCurrentSchemaForCapabilities(ctx, db, RuntimeCapabilities{SheinLogin: true}); e != nil {
		return e
	}
	sqlDB, e := db.DB()
	if e != nil {
		return e
	}
	var role string
	var allowed bool
	e = sqlDB.QueryRowContext(ctx, `SELECT current_user,
 current_schema()='public' AND current_schemas(false)=ARRAY['public']::name[]
 AND has_database_privilege(current_user,current_database(),'CONNECT') AND has_schema_privilege(current_user,'public','USAGE')
 AND NOT has_database_privilege(current_user,current_database(),'CREATE') AND NOT has_database_privilege(current_user,current_database(),'TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=current_user AND (rolsuper OR rolcreaterole OR rolcreatedb OR rolreplication OR rolbypassrls))
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=(SELECT oid FROM pg_roles WHERE rolname=current_user))
 AND NOT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname !~ '^pg_' AND nspname<>'information_schema' AND has_schema_privilege(current_user,oid,'CREATE'))
 AND NOT EXISTS(SELECT 1 FROM (VALUES ('workbench_stores','SELECT'),('workbench_store_member_grants','SELECT'),('workbench_store_audit_logs','SELECT'),('workbench_store_audit_logs','INSERT'),('workbench_store_shein_login_bindings','SELECT'),('workbench_store_shein_login_bindings','INSERT'),('workbench_store_shein_login_bindings','UPDATE')) t(rel,priv) WHERE NOT has_table_privilege(current_user,('public.'||rel)::regclass,priv))
 AND NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid='public.workbench_stores'::regclass AND a.attnum>0 AND NOT a.attisdropped AND
 has_column_privilege(current_user,a.attrelid,a.attnum,'UPDATE')<>(a.attname IN ('external_store_id','identity_key','version','updated_by','updated_at')))
 AND NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(acldefault('r',c.relowner)) p
 WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema' AND c.relkind IN ('r','p','v','m','f')
 AND CASE WHEN p.privilege_type IN ('SELECT','INSERT','UPDATE','REFERENCES') THEN has_any_column_privilege(current_user,c.oid,p.privilege_type) ELSE has_table_privilege(current_user,c.oid,p.privilege_type) END
 AND NOT(n.nspname='public' AND ((c.relname,p.privilege_type) IN (('workbench_stores','SELECT'),('workbench_stores','UPDATE'),('workbench_store_member_grants','SELECT'),('workbench_store_audit_logs','SELECT'),('workbench_store_audit_logs','INSERT'),('workbench_store_shein_login_bindings','SELECT'),('workbench_store_shein_login_bindings','INSERT'),('workbench_store_shein_login_bindings','UPDATE')))))`).Scan(&role, &allowed)
	if e != nil || role != "shein_login_worker" || !allowed {
		return errors.New("SHEIN login runtime privilege boundary unavailable")
	}
	return nil
}
