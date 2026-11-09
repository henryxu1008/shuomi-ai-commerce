package researchpersistence

import (
	"context"
	"gorm.io/gorm"
	platformdatabase "task-processor/internal/platform/database"
	"task-processor/internal/product/research"
)

var Tables = []string{"research_opportunities", "research_evidence_links", "research_commands"}

func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return research.ErrUnavailable
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS research_opportunities (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, project_id uuid NOT NULL, id uuid NOT NULL,
 platform varchar(16) NOT NULL, market varchar(4) NOT NULL, native_product_id varchar(64) NOT NULL, selected_link_id uuid,
 state varchar(32) NOT NULL CHECK(state IN ('TO_RESEARCH','PENDING_CONFIRMATION','APPROVED_FOR_TEST','REJECTED')),
 reason text NOT NULL DEFAULT '' CHECK(octet_length(reason)<=4096), revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(organization_id,actor_id,project_id,id), CONSTRAINT research_opportunity_identity UNIQUE(organization_id,actor_id,project_id,platform,market,native_product_id));
 CREATE TABLE IF NOT EXISTS research_evidence_links (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, project_id uuid NOT NULL, id uuid NOT NULL,
 candidate_id uuid NOT NULL, reference_json jsonb NOT NULL CHECK(octet_length(reference_json::text)<=8192),
 removed boolean NOT NULL DEFAULT false, created_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,project_id,id),
 FOREIGN KEY(organization_id,actor_id,project_id,candidate_id) REFERENCES research_opportunities(organization_id,actor_id,project_id,id));
 CREATE TABLE IF NOT EXISTS research_commands (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, project_id uuid NOT NULL, key uuid NOT NULL,
 input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64), candidate_id uuid,
 intent_json jsonb NOT NULL CHECK(octet_length(intent_json::text)<=65536),
 receipt_json jsonb NOT NULL CHECK(octet_length(receipt_json::text)<=65536),
 decision_json jsonb CHECK(octet_length(decision_json::text)<=16384), created_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,project_id,key));
 CREATE INDEX IF NOT EXISTS research_candidate_evidence ON research_evidence_links(organization_id,actor_id,project_id,candidate_id,created_at DESC,id);
 CREATE INDEX IF NOT EXISTS research_candidate_history ON research_commands(organization_id,actor_id,project_id,candidate_id,created_at DESC,key);
 `).Error
}
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return research.ErrUnavailable
	}
	for _, q := range []string{
		"SELECT organization_id,actor_id,project_id,id,platform,market,native_product_id,selected_link_id,state,reason,revision FROM research_opportunities LIMIT 0",
		"SELECT organization_id,actor_id,project_id,id,candidate_id,reference_json,removed,created_at FROM research_evidence_links LIMIT 0",
		"SELECT organization_id,actor_id,project_id,key,input_hash,candidate_id,intent_json,receipt_json,decision_json,created_at FROM research_commands LIMIT 0",
	} {
		if db.WithContext(ctx).Exec(q).Error != nil {
			return research.ErrUnavailable
		}
	}
	var ready bool
	if db.WithContext(ctx).Raw(`WITH expected(relation,kind,columns,reference,reference_columns) AS (VALUES
 ('research_opportunities','p',ARRAY['organization_id','actor_id','project_id','id'],NULL::text,NULL::text[]),
 ('research_opportunities','u',ARRAY['organization_id','actor_id','project_id','platform','market','native_product_id'],NULL,NULL),
 ('research_evidence_links','p',ARRAY['organization_id','actor_id','project_id','id'],NULL,NULL),
 ('research_commands','p',ARRAY['organization_id','actor_id','project_id','key'],NULL,NULL),
 ('research_evidence_links','f',ARRAY['organization_id','actor_id','project_id','candidate_id'],'research_opportunities',ARRAY['organization_id','actor_id','project_id','id'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass(e.relation) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.position)=e.columns
 AND (e.reference IS NULL OR (c.confrelid=to_regclass(e.reference) AND c.confdeltype='a'
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.position)=e.reference_columns))))`).Scan(&ready).Error != nil || !ready {
		return research.ErrUnavailable
	}
	for _, expected := range []struct{ table, column, expression string }{
		{"research_opportunities", "state", "CHECK (((state)::text = ANY ((ARRAY['TO_RESEARCH'::character varying, 'PENDING_CONFIRMATION'::character varying, 'APPROVED_FOR_TEST'::character varying, 'REJECTED'::character varying])::text[])))"},
		{"research_opportunities", "revision", "CHECK ((revision > 0))"},
		{"research_opportunities", "reason", "CHECK ((octet_length(reason) <= 4096))"},
		{"research_evidence_links", "reference_json", "CHECK ((octet_length((reference_json)::text) <= 8192))"},
		{"research_commands", "input_hash", "CHECK ((length((input_hash)::text) = 64))"},
		{"research_commands", "intent_json", "CHECK ((octet_length((intent_json)::text) <= 65536))"},
		{"research_commands", "receipt_json", "CHECK ((octet_length((receipt_json)::text) <= 65536))"},
		{"research_commands", "decision_json", "CHECK ((octet_length((decision_json)::text) <= 16384))"},
	} {
		var definitions []string
		if db.WithContext(ctx).Raw(`SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=ANY(c.conkey) WHERE c.conrelid=to_regclass(?) AND c.contype='c' AND c.convalidated AND a.attname=?`, expected.table, expected.column).Scan(&definitions).Error != nil {
			return research.ErrUnavailable
		}
		matched := false
		for _, definition := range definitions {
			matched = matched || platformdatabase.SchemaConstraintEqual(definition, expected.expression)
		}
		if !matched {
			return research.ErrUnavailable
		}
	}
	return nil
}
