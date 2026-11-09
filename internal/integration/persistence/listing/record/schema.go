package recordpersistence

import (
	"context"
	"gorm.io/gorm"
	record "task-processor/internal/listing/record/target"
)

var Tables = []string{"listing_target_records", "listing_preparation_targets", "listing_target_record_commands", "listing_target_split_commands"}

// InstallSchema belongs to explicit initialization of the Product owner pool.
// Serving construction only verifies these facts and never runs DDL.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return record.ErrUnavailable
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS listing_target_records (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 id uuid NOT NULL, target_id uuid NOT NULL, source_id uuid NOT NULL, store_id uuid NOT NULL,
 site varchar(32) NOT NULL CONSTRAINT listing_target_records_sites_v2 CHECK(site IN ('shein-us','temu-supplier','temu-local','tiktokshop-us','walmart-us')), revision bigint NOT NULL CHECK(revision>0),
 purpose varchar(32) NOT NULL DEFAULT 'standard', partition_id varchar(36) NOT NULL DEFAULT '', family_id varchar(36) NOT NULL DEFAULT '',
 CONSTRAINT listing_target_records_purpose CHECK ((purpose='standard' AND partition_id='' AND family_id='') OR (purpose='shein_color_local' AND site='shein-us' AND length(partition_id)=36 AND length(family_id)=36)),
 input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64), body_hash varchar(64) NOT NULL CHECK(length(body_hash)=64),
 body_json jsonb NOT NULL, created_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,id), UNIQUE(organization_id,actor_id,target_id,id,revision),
 UNIQUE(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,target_id,id,revision),
 FOREIGN KEY(organization_id,actor_id,source_id) REFERENCES listing_preparation_sources(organization_id,actor_id,id));
CREATE TABLE IF NOT EXISTS listing_preparation_targets (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 id uuid NOT NULL, source_id uuid NOT NULL, store_id uuid NOT NULL, site varchar(32) NOT NULL CONSTRAINT listing_preparation_targets_sites_v2 CHECK(site IN ('shein-us','temu-supplier','temu-local','tiktokshop-us','walmart-us')),
 current_record_id uuid NOT NULL, revision bigint NOT NULL CHECK(revision>0),
 purpose varchar(32) NOT NULL DEFAULT 'standard', partition_id varchar(36) NOT NULL DEFAULT '', family_id varchar(36) NOT NULL DEFAULT '',
 CONSTRAINT listing_preparation_targets_purpose CHECK ((purpose='standard' AND partition_id='' AND family_id='') OR (purpose='shein_color_local' AND site='shein-us' AND length(partition_id)=36 AND length(family_id)=36)),
 PRIMARY KEY(organization_id,actor_id,id), UNIQUE(organization_id,actor_id,source_id,store_id,site,purpose,partition_id),
 FOREIGN KEY(organization_id,actor_id,source_id) REFERENCES listing_preparation_sources(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,id,current_record_id,revision) REFERENCES listing_target_records(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,target_id,id,revision));
CREATE TABLE IF NOT EXISTS listing_target_record_commands (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, member_id varchar(128) NOT NULL,
 command_key uuid NOT NULL, record_id uuid NOT NULL, input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64),
 PRIMARY KEY(organization_id,actor_id,command_key),
 FOREIGN KEY(organization_id,actor_id,record_id) REFERENCES listing_target_records(organization_id,actor_id,id));
CREATE TABLE IF NOT EXISTS listing_target_split_commands (
 organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,
 command_key uuid NOT NULL,input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64),family_id varchar(36) NOT NULL,
 source_id uuid NOT NULL,store_id uuid NOT NULL,site varchar(32) NOT NULL CHECK(site='shein-us'),purpose varchar(32) NOT NULL CHECK(purpose='shein_color_local'),
 base_record_id uuid NOT NULL,completion_record_id uuid NOT NULL,
 first_target_id uuid NOT NULL,first_record_id uuid NOT NULL,first_revision bigint NOT NULL CHECK(first_revision=1),
 second_target_id uuid NOT NULL,second_record_id uuid NOT NULL,second_revision bigint NOT NULL CHECK(second_revision=1),
 created_at timestamptz NOT NULL,PRIMARY KEY(organization_id,actor_id,command_key),UNIQUE(organization_id,actor_id,family_id),
 CHECK(first_target_id<>second_target_id AND first_record_id<>second_record_id),
 FOREIGN KEY(organization_id,actor_id,base_record_id) REFERENCES listing_target_records(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,first_target_id,first_record_id,first_revision) REFERENCES listing_target_records(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,target_id,id,revision),
 FOREIGN KEY(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,second_target_id,second_record_id,second_revision) REFERENCES listing_target_records(organization_id,actor_id,member_id,source_id,store_id,site,purpose,family_id,target_id,id,revision));`).Error
}

func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return record.ErrUnavailable
	}
	for _, query := range []string{
		"SELECT organization_id,actor_id,member_id,id,target_id,source_id,store_id,site,revision,input_hash,body_hash,body_json,created_at FROM listing_target_records LIMIT 0",
		"SELECT organization_id,actor_id,member_id,id,source_id,store_id,site,current_record_id,revision FROM listing_preparation_targets LIMIT 0",
		"SELECT organization_id,actor_id,member_id,command_key,record_id,input_hash FROM listing_target_record_commands LIMIT 0",
		"SELECT purpose,partition_id,family_id FROM listing_target_records LIMIT 0",
		"SELECT purpose,partition_id,family_id FROM listing_preparation_targets LIMIT 0",
		"SELECT organization_id,actor_id,member_id,command_key,input_hash,family_id,source_id,store_id,site,purpose,base_record_id,completion_record_id,first_target_id,first_record_id,first_revision,second_target_id,second_record_id,second_revision,created_at FROM listing_target_split_commands LIMIT 0",
	} {
		if db.WithContext(ctx).Exec(query).Error != nil {
			return record.ErrUnavailable
		}
	}
	var ready bool
	err := db.WithContext(ctx).Raw(`WITH expected(relation,kind,columns,reference,reference_columns) AS (VALUES
 ('listing_target_records','p',ARRAY['organization_id','actor_id','id'],NULL::text,NULL::text[]),
 ('listing_target_records','u',ARRAY['organization_id','actor_id','target_id','id','revision'],NULL,NULL),
 ('listing_target_records','f',ARRAY['organization_id','actor_id','source_id'],'listing_preparation_sources',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_targets','p',ARRAY['organization_id','actor_id','id'],NULL,NULL),
 ('listing_preparation_targets','u',ARRAY['organization_id','actor_id','source_id','store_id','site','purpose','partition_id'],NULL,NULL),
 ('listing_preparation_targets','f',ARRAY['organization_id','actor_id','source_id'],'listing_preparation_sources',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_targets','f',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','id','current_record_id','revision'],'listing_target_records',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','target_id','id','revision']),
 ('listing_target_split_commands','p',ARRAY['organization_id','actor_id','command_key'],NULL,NULL),
 ('listing_target_split_commands','u',ARRAY['organization_id','actor_id','family_id'],NULL,NULL),
 ('listing_target_split_commands','f',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','first_target_id','first_record_id','first_revision'],'listing_target_records',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','target_id','id','revision']),
 ('listing_target_split_commands','f',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','second_target_id','second_record_id','second_revision'],'listing_target_records',ARRAY['organization_id','actor_id','member_id','source_id','store_id','site','purpose','family_id','target_id','id','revision']),
 ('listing_target_record_commands','p',ARRAY['organization_id','actor_id','command_key'],NULL,NULL),
 ('listing_target_record_commands','f',ARRAY['organization_id','actor_id','record_id'],'listing_target_records',ARRAY['organization_id','actor_id','id'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass(e.relation) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.position)=e.columns
 AND (e.reference IS NULL OR (c.confrelid=to_regclass(e.reference)
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.position)=e.reference_columns))))`).Scan(&ready).Error
	if err != nil || !ready {
		return record.ErrUnavailable
	}
	var sitesReady bool
	err = db.WithContext(ctx).Raw(`SELECT count(*)=2 FROM pg_constraint c
 WHERE ((c.conrelid=to_regclass('listing_target_records') AND c.conname='listing_target_records_sites_v2')
 OR (c.conrelid=to_regclass('listing_preparation_targets') AND c.conname='listing_preparation_targets_sites_v2'))
 AND c.contype='c' AND c.convalidated AND NOT c.condeferrable
 AND position('shein-us' in pg_get_constraintdef(c.oid))>0
 AND position('temu-supplier' in pg_get_constraintdef(c.oid))>0
 AND position('temu-local' in pg_get_constraintdef(c.oid))>0 AND position('tiktokshop-us' in pg_get_constraintdef(c.oid))>0 AND position('walmart-us' in pg_get_constraintdef(c.oid))>0`).Scan(&sitesReady).Error
	if err != nil || !sitesReady {
		return record.ErrUnavailable
	}
	var purposeReady bool
	err = db.WithContext(ctx).Raw(`SELECT count(*)=2 FROM pg_constraint c WHERE c.conrelid IN (to_regclass('listing_target_records'),to_regclass('listing_preparation_targets')) AND c.conname IN ('listing_target_records_purpose','listing_preparation_targets_purpose') AND c.contype='c' AND c.convalidated AND NOT c.condeferrable AND position('shein_color_local' in pg_get_constraintdef(c.oid))>0 AND position('standard' in pg_get_constraintdef(c.oid))>0 AND position('partition_id' in pg_get_constraintdef(c.oid))>0 AND position('family_id' in pg_get_constraintdef(c.oid))>0`).Scan(&purposeReady).Error
	if err != nil || !purposeReady {
		return record.ErrUnavailable
	}

	return nil
}
