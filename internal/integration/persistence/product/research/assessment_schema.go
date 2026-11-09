package researchpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/product/research"
)

// InstallAssessmentSchema is called by Product's locked empty-database
// initialization after base Research schema and grants have been installed.
func InstallAssessmentSchema(tx *gorm.DB) error {
	if tx == nil || tx.Dialector.Name() != "postgres" {
		return research.ErrUnavailable
	}
	return tx.Exec(`CREATE TABLE public.research_assessments (
 organization_id varchar(128) NOT NULL,
 actor_id varchar(128) NOT NULL,
 project_id uuid NOT NULL,
 request_key uuid NOT NULL,
 candidate_id uuid NOT NULL,
 candidate_revision bigint NOT NULL CHECK(candidate_revision > 0),
 assessment_revision bigint NOT NULL DEFAULT 1 CHECK(assessment_revision = 1),
 record_json jsonb NOT NULL CHECK(octet_length(record_json::text) <= 32768),
 CONSTRAINT assessment_result CHECK(record_json->'result'->>'status' IS NOT NULL AND record_json->'result'->>'status' IN ('COMPUTABLE','INSUFFICIENT_INPUT','UNSUPPORTED_COST_RULE')),
 created_at timestamptz NOT NULL,
 CONSTRAINT assessment_key PRIMARY KEY(organization_id,actor_id,project_id,request_key),
 CONSTRAINT assessment_revision UNIQUE(organization_id,actor_id,project_id,candidate_id,candidate_revision),
 CONSTRAINT assessment_candidate FOREIGN KEY(organization_id,actor_id,project_id,candidate_id)
   REFERENCES public.research_opportunities(organization_id,actor_id,project_id,id),
 CONSTRAINT assessment_command FOREIGN KEY(organization_id,actor_id,project_id,request_key)
   REFERENCES public.research_commands(organization_id,actor_id,project_id,key)
 );
 REVOKE ALL ON TABLE public.research_assessments FROM PUBLIC, product_research_runtime;
 GRANT SELECT, INSERT ON TABLE public.research_assessments TO product_research_runtime;`).Error
}

func VerifyAssessmentSchema(ctx context.Context, db *gorm.DB) error {
	return verifyAssessmentSchema(ctx, db, true)
}

func verifyAssessmentSchema(ctx context.Context, db *gorm.DB, serving bool) error {
	var verified bool
	err := db.WithContext(ctx).Raw(`WITH expected(name, kind, cols, ref, refcols) AS (VALUES
 ('assessment_key','p',ARRAY['organization_id','actor_id','project_id','request_key'],NULL::text,NULL::text[]),
 ('assessment_revision','u',ARRAY['organization_id','actor_id','project_id','candidate_id','candidate_revision'],NULL,NULL),
 ('assessment_candidate','f',ARRAY['organization_id','actor_id','project_id','candidate_id'],'public.research_opportunities',ARRAY['organization_id','actor_id','project_id','id']),
 ('assessment_command','f',ARRAY['organization_id','actor_id','project_id','request_key'],'public.research_commands',ARRAY['organization_id','actor_id','project_id','key'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(
 SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass(?) AND c.conname=e.name AND c.contype::text=e.kind
 AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(n,i)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.n ORDER BY k.i)=e.cols
 AND (e.ref IS NULL OR (c.confrelid=to_regclass(e.ref) AND c.confdeltype='a' AND c.confupdtype='a'
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(n,i)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.n ORDER BY k.i)=e.refcols))))
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND pg_get_constraintdef(oid)='CHECK ((candidate_revision > 0))')
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND pg_get_constraintdef(oid)='CHECK ((octet_length((record_json)::text) <= 32768))')
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND pg_get_constraintdef(oid)='CHECK ((assessment_revision = 1))')
 AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass(?) AND convalidated AND conname='assessment_result')
 AND EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass(?) AND relkind='r' AND (? OR relowner<>(SELECT oid FROM pg_roles WHERE rolname=current_user)))`,
		assessmentTable, assessmentTable, assessmentTable, assessmentTable, assessmentTable, assessmentTable, !serving).Scan(&verified).Error
	if err != nil || !verified {
		return research.ErrUnavailable
	}
	if db.WithContext(ctx).Exec("SELECT organization_id,actor_id,project_id,request_key,candidate_id,candidate_revision,record_json,created_at FROM "+assessmentTable+" LIMIT 0").Error != nil {
		return research.ErrUnavailable
	}
	return nil
}

// VerifyAssessmentOwnerInstallation is the explicit installer's read-only check.
// It never fixes drift or broadens grants on a previously installed table.
func VerifyAssessmentOwnerInstallation(ctx context.Context, db *gorm.DB) error {
	if verifyAssessmentSchema(ctx, db, false) != nil {
		return research.ErrUnavailable
	}
	var safe bool
	err := db.WithContext(ctx).Raw(`SELECT
 c.relowner=(SELECT oid FROM pg_roles WHERE rolname=current_user)
 AND NOT c.relrowsecurity AND NOT c.relforcerowsecurity
 AND 9=(SELECT count(*) FROM pg_attribute WHERE attrelid=c.oid AND attnum>0 AND NOT attisdropped)
 AND NOT EXISTS(SELECT 1 FROM (VALUES
 ('organization_id','character varying(128)'),('actor_id','character varying(128)'),
 ('project_id','uuid'),('request_key','uuid'),('candidate_id','uuid'),
 ('candidate_revision','bigint'),('assessment_revision','bigint'),('record_json','jsonb'),('created_at','timestamp with time zone')
 ) AS expected(name,kind) WHERE NOT EXISTS(SELECT 1 FROM pg_attribute a
 WHERE a.attrelid=c.oid AND a.attname=expected.name AND a.attnum>0 AND NOT a.attisdropped
 AND a.attnotnull AND pg_catalog.format_type(a.atttypid,a.atttypmod)=expected.kind))
 AND EXISTS(SELECT 1 FROM pg_attribute a JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum
 WHERE a.attrelid=c.oid AND a.attname='assessment_revision' AND pg_get_expr(d.adbin,d.adrelid)='1')
 AND 8=(SELECT count(*) FROM pg_constraint WHERE conrelid=c.oid)
 AND NOT EXISTS(SELECT 1 FROM pg_index WHERE indrelid=c.oid AND (NOT indisvalid OR NOT indisready OR NOT indimmediate))
 AND NOT EXISTS(SELECT 1 FROM pg_trigger WHERE tgrelid=c.oid AND NOT tgisinternal)
 AND has_schema_privilege(?, 'public', 'USAGE')
 AND has_table_privilege(?,c.oid,'SELECT') AND has_table_privilege(?,c.oid,'INSERT')
 AND NOT has_table_privilege(?,c.oid,'UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
 AND NOT has_any_column_privilege(?,c.oid,'UPDATE,REFERENCES,SELECT WITH GRANT OPTION,INSERT WITH GRANT OPTION')
 AND NOT EXISTS(SELECT 1 FROM aclexplode(c.relacl) a
 WHERE a.grantee NOT IN(c.relowner,(SELECT oid FROM pg_roles WHERE rolname=?)))
 AND NOT EXISTS(SELECT 1 FROM pg_attribute t CROSS JOIN LATERAL aclexplode(t.attacl) a
 WHERE t.attrelid=c.oid AND a.grantee<>c.relowner)
 FROM pg_class c WHERE c.oid=to_regclass(?)`,
		RuntimeRole, RuntimeRole, RuntimeRole, RuntimeRole, RuntimeRole, RuntimeRole, assessmentTable).Scan(&safe).Error
	if err != nil || !safe {
		return research.ErrUnavailable
	}
	return nil
}
