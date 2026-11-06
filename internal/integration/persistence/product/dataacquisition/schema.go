// Package dataacquisitionpersistence owns source acquisition jobs and terminal proofs.
package dataacquisitionpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/product/dataacquisition"
)

var Tables = []string{"data_acquisition_jobs", "data_acquisition_items"}

// InstallSchema is only for an explicitly owned empty Product installation.
// Credentials and current Product owners are installed by their own installers.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return dataacquisition.ErrUnavailable
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS data_acquisition_jobs (
 organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,id uuid NOT NULL,
 source_kind varchar(32) NOT NULL DEFAULT 'amazon_data' CHECK(source_kind IN ('amazon_data','shein_us_data')),
 trial_binding char(64), command_key uuid NOT NULL,input_hash varchar(64) NOT NULL CHECK(input_hash ~ '^[0-9a-f]{64}$'),
 credential_id uuid,credential_revision bigint,query_json jsonb NOT NULL CHECK(octet_length(query_json::text)<=65536),
 funding varchar(32) NOT NULL CHECK(funding IN ('enterprise_unallocated','member_allocated')),
 state varchar(24) NOT NULL CHECK(state IN ('ADMITTED','RUNNING','SUCCEEDED','PARTIAL','FAILED','CANCELED')),reason varchar(64) NOT NULL DEFAULT '',
 discovered boolean NOT NULL DEFAULT false,canceled boolean NOT NULL DEFAULT false,quota_reserved integer NOT NULL CHECK(quota_reserved BETWEEN 0 AND 200),
 day_window date NOT NULL,month_window date NOT NULL,created_at timestamptz NOT NULL,deadline timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,id),UNIQUE(organization_id,actor_id,command_key),UNIQUE(organization_id,id),
 UNIQUE(organization_id,actor_id,id,source_kind),
 FOREIGN KEY(organization_id,actor_id,credential_id) REFERENCES data_service_credentials(organization_id,actor_id,id),
 CHECK((credential_id IS NULL AND credential_revision IS NULL) OR (credential_id IS NOT NULL AND credential_revision>0)),
 CHECK((source_kind='amazon_data' AND trial_binding IS NULL) OR (source_kind='shein_us_data' AND trial_binding IS NOT NULL AND trial_binding ~ '^[0-9a-f]{64}$' AND credential_id IS NULL AND quota_reserved=0 AND id=command_key)),
 CHECK(deadline>created_at AND deadline<=created_at+interval '30 minutes'));
CREATE TABLE IF NOT EXISTS data_acquisition_items (
 organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,job_id uuid NOT NULL,id uuid NOT NULL,
 source_kind varchar(32) NOT NULL DEFAULT 'amazon_data',asin varchar(10),native_goods_id varchar(64),source_position integer,evidence_checksum char(71),
 state varchar(24) NOT NULL CHECK(state IN ('PREPARED','FETCHING','PREPARED_EVIDENCE','SAVED','FAILED')),
 intent_json jsonb NOT NULL CHECK(octet_length(intent_json::text)<=8192),
 claim_token uuid,lease_until timestamptz,evidence_json bytea CHECK(octet_length(evidence_json)<=2097152),
 source_json jsonb CHECK(octet_length(source_json::text)<=8192),terminal_evidence uuid,
 reservation_id uuid,charge_state varchar(32) NOT NULL DEFAULT '' CHECK(charge_state IN ('','reserved','committed','released','reconciliation_required')),
 reason varchar(64) NOT NULL DEFAULT '',created_at timestamptz NOT NULL,saved_at timestamptz,
 PRIMARY KEY(organization_id,actor_id,job_id,id),UNIQUE(organization_id,actor_id,job_id,asin),UNIQUE(organization_id,id),
 UNIQUE(organization_id,actor_id,job_id,native_goods_id),UNIQUE(organization_id,actor_id,job_id,source_position),
 FOREIGN KEY(organization_id,actor_id,job_id,source_kind) REFERENCES data_acquisition_jobs(organization_id,actor_id,id,source_kind),
 CHECK((source_kind='amazon_data' AND asin IS NOT NULL AND asin ~ '^[A-Z0-9]{10}$' AND native_goods_id IS NULL AND source_position IS NULL AND evidence_checksum IS NULL) OR
       (source_kind='shein_us_data' AND asin IS NULL AND native_goods_id IS NOT NULL AND native_goods_id ~ '^[1-9][0-9]{0,63}$' AND intent_json='{}'::jsonb AND source_position IS NOT NULL AND source_position BETWEEN 1 AND 20 AND evidence_checksum IS NOT NULL AND evidence_checksum ~ '^sha256:[0-9a-f]{64}$' AND reservation_id IS NULL AND charge_state='')),
 CHECK((state='SAVED' AND source_json IS NOT NULL AND terminal_evidence IS NOT NULL AND (source_kind='shein_us_data' OR reservation_id IS NOT NULL) AND saved_at IS NOT NULL) OR (state<>'SAVED' AND source_json IS NULL AND saved_at IS NULL)),
 CHECK(state<>'FAILED' OR terminal_evidence IS NOT NULL));
CREATE INDEX IF NOT EXISTS data_acquisition_active_org ON data_acquisition_jobs(organization_id,deadline) WHERE state IN ('ADMITTED','RUNNING');
CREATE INDEX IF NOT EXISTS data_acquisition_recent_actor ON data_acquisition_jobs(organization_id,actor_id,created_at DESC,id);
`).Error
}

func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return dataacquisition.ErrUnavailable
	}
	for _, q := range []string{
		"SELECT organization_id,actor_id,member_id,id,command_key,input_hash,credential_id,credential_revision,query_json,funding,state,reason,discovered,canceled,quota_reserved,day_window,month_window,created_at,deadline,source_kind,trial_binding FROM data_acquisition_jobs LIMIT 0",
		"SELECT organization_id,actor_id,job_id,id,asin,state,intent_json,claim_token,lease_until,evidence_json,source_json,terminal_evidence,reservation_id,charge_state,reason,created_at,saved_at,source_kind,native_goods_id,source_position,evidence_checksum FROM data_acquisition_items LIMIT 0",
	} {
		if db.WithContext(ctx).Exec(q).Error != nil {
			return dataacquisition.ErrUnavailable
		}
	}
	var ready bool
	err := db.WithContext(ctx).Raw(`WITH expected(relation,kind,columns,reference,reference_columns) AS (VALUES
 ('data_acquisition_jobs','p',ARRAY['organization_id','actor_id','id'],NULL::text,NULL::text[]),
 ('data_acquisition_jobs','u',ARRAY['organization_id','actor_id','command_key'],NULL,NULL),
 ('data_acquisition_jobs','u',ARRAY['organization_id','id'],NULL,NULL),
 ('data_acquisition_jobs','u',ARRAY['organization_id','actor_id','id','source_kind'],NULL,NULL),
 ('data_acquisition_jobs','f',ARRAY['organization_id','actor_id','credential_id'],'data_service_credentials',ARRAY['organization_id','actor_id','id']),
 ('data_acquisition_items','p',ARRAY['organization_id','actor_id','job_id','id'],NULL,NULL),
 ('data_acquisition_items','u',ARRAY['organization_id','actor_id','job_id','asin'],NULL,NULL),
 ('data_acquisition_items','u',ARRAY['organization_id','id'],NULL,NULL),
 ('data_acquisition_items','u',ARRAY['organization_id','actor_id','job_id','native_goods_id'],NULL,NULL),
 ('data_acquisition_items','u',ARRAY['organization_id','actor_id','job_id','source_position'],NULL,NULL),
 ('data_acquisition_items','f',ARRAY['organization_id','actor_id','job_id','source_kind'],'data_acquisition_jobs',ARRAY['organization_id','actor_id','id','source_kind'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(
 SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass(e.relation) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(id,position) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.position)=e.columns
 AND (e.reference IS NULL OR (c.confrelid=to_regclass(e.reference)
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(id,position) JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.position)=e.reference_columns))))`).Scan(&ready).Error
	if err != nil || !ready {
		return dataacquisition.ErrUnavailable
	}
	return nil
}
