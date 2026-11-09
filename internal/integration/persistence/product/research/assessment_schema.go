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
 AND EXISTS(SELECT 1 FROM pg_class WHERE oid=to_regclass(?) AND relkind='r' AND relowner<>(SELECT oid FROM pg_roles WHERE rolname=current_user))`,
		assessmentTable, assessmentTable, assessmentTable, assessmentTable, assessmentTable, assessmentTable).Scan(&verified).Error
	if err != nil || !verified {
		return research.ErrUnavailable
	}
	if db.WithContext(ctx).Exec("SELECT organization_id,actor_id,project_id,request_key,candidate_id,candidate_revision,record_json,created_at FROM "+assessmentTable+" LIMIT 0").Error != nil {
		return research.ErrUnavailable
	}
	return nil
}
