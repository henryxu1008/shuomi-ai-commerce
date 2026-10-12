package completiondraftpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/listing/preparation"
)

var Tables = []string{"listing_completion_draft_records", "listing_completion_draft_heads", "listing_completion_draft_commands"}

// InstallSchema is explicit owner maintenance, never called by serving.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return preparation.ErrUnavailable
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS listing_completion_draft_records (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 id uuid NOT NULL, source_id uuid NOT NULL, draft_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0),
 source_json jsonb NOT NULL, content_json jsonb NOT NULL, content_hash varchar(64) NOT NULL CHECK(length(content_hash)=64), created_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,id), UNIQUE(organization_id,actor_id,draft_id,revision),
 UNIQUE(organization_id,actor_id,member_id,id,source_id,draft_id,revision),
 FOREIGN KEY(organization_id,actor_id,source_id) REFERENCES listing_preparation_sources(organization_id,actor_id,id));
CREATE TABLE IF NOT EXISTS listing_completion_draft_heads (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 source_id uuid NOT NULL, record_id uuid NOT NULL, draft_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(organization_id,actor_id,source_id),
 FOREIGN KEY(organization_id,actor_id,source_id) REFERENCES listing_preparation_sources(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,member_id,record_id,source_id,draft_id,revision) REFERENCES listing_completion_draft_records(organization_id,actor_id,member_id,id,source_id,draft_id,revision));
CREATE TABLE IF NOT EXISTS listing_completion_draft_commands (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 command_key uuid NOT NULL, input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64), input_json jsonb NOT NULL,
 record_id uuid NOT NULL, source_id uuid NOT NULL, draft_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0),
 PRIMARY KEY(organization_id,actor_id,command_key),
 FOREIGN KEY(organization_id,actor_id,member_id,record_id,source_id,draft_id,revision) REFERENCES listing_completion_draft_records(organization_id,actor_id,member_id,id,source_id,draft_id,revision));`).Error
}

func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return preparation.ErrUnavailable
	}
	for _, q := range []string{
		"SELECT organization_id,actor_id,member_id,id,source_id,draft_id,revision,source_json,content_json,content_hash,created_at FROM listing_completion_draft_records LIMIT 0",
		"SELECT organization_id,actor_id,member_id,source_id,record_id,draft_id,revision FROM listing_completion_draft_heads LIMIT 0",
		"SELECT organization_id,actor_id,member_id,command_key,input_hash,input_json,record_id,source_id,draft_id,revision FROM listing_completion_draft_commands LIMIT 0",
	} {
		if db.WithContext(ctx).Exec(q).Error != nil {
			return preparation.ErrUnavailable
		}
	}
	var ready bool
	err := db.WithContext(ctx).Raw(`WITH expected(relation,kind,columns,reference,reference_columns) AS (VALUES
 ('listing_completion_draft_records','p',ARRAY['organization_id','actor_id','id'],NULL::text,NULL::text[]),
 ('listing_completion_draft_records','u',ARRAY['organization_id','actor_id','draft_id','revision'],NULL,NULL),
 ('listing_completion_draft_records','u',ARRAY['organization_id','actor_id','member_id','id','source_id','draft_id','revision'],NULL,NULL),
 ('listing_completion_draft_records','f',ARRAY['organization_id','actor_id','source_id'],'listing_preparation_sources',ARRAY['organization_id','actor_id','id']),
 ('listing_completion_draft_heads','p',ARRAY['organization_id','actor_id','source_id'],NULL,NULL),
 ('listing_completion_draft_heads','f',ARRAY['organization_id','actor_id','source_id'],'listing_preparation_sources',ARRAY['organization_id','actor_id','id']),
 ('listing_completion_draft_heads','f',ARRAY['organization_id','actor_id','member_id','record_id','source_id','draft_id','revision'],'listing_completion_draft_records',ARRAY['organization_id','actor_id','member_id','id','source_id','draft_id','revision']),
 ('listing_completion_draft_commands','p',ARRAY['organization_id','actor_id','command_key'],NULL,NULL),
 ('listing_completion_draft_commands','f',ARRAY['organization_id','actor_id','member_id','record_id','source_id','draft_id','revision'],'listing_completion_draft_records',ARRAY['organization_id','actor_id','member_id','id','source_id','draft_id','revision'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass('public.'||e.relation) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(id,position) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.position)=e.columns
 AND (e.reference IS NULL OR (c.confrelid=to_regclass('public.'||e.reference)
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY k(id,position) JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.position)=e.reference_columns))))`).Scan(&ready).Error
	if err != nil || !ready {
		return preparation.ErrUnavailable
	}
	return nil
}
