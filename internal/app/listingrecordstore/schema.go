package recordpersistence

import (
	"context"
	_ "embed"
	"gorm.io/gorm"
	"task-processor/internal/listing/record"
)

//go:embed schema.sql
var schemaDDL string

// InstallSchema is only for an explicitly confirmed empty owner database.
func InstallSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return record.ErrUnavailable
	}
	return db.Exec(schemaDDL).Error
}

// VerifySchema is read-only and uses public-qualified catalog identities. It is
// deliberately separate from the isolated trial constructor's admission.
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return record.ErrUnavailable
	}
	var ready bool
	if err := db.WithContext(ctx).Raw(offlineSchemaQuery).Scan(&ready).Error; err != nil || !ready {
		return record.ErrUnavailable
	}
	return nil
}

const offlineSchemaQuery = `WITH expected(t,n,k) AS (VALUES
 ('listing_shein_records','id','uuid'),('listing_shein_records','organization_id','character varying(200)'),
 ('listing_shein_records','owner_user_id','character varying(200)'),('listing_shein_records','operation_id','character varying(128)'),
 ('listing_shein_records','product_key','character varying(128)'),('listing_shein_records','snapshot_version','bigint'),
 ('listing_shein_records','store_id','uuid'),('listing_shein_records','country','character varying(2)'),('listing_shein_records','language','character varying(2)'),
 ('listing_shein_records','action','character varying(16)'),('listing_shein_records','input_hash','character varying(71)'),
 ('listing_shein_records','product_hash','character varying(71)'),('listing_shein_records','asset_inventory_version','bigint'),
 ('listing_shein_records','asset_inventory_hash','character varying(71)'),('listing_shein_records','rule_revision','character varying(128)'),
 ('listing_shein_records','policy_revision','character varying(128)'),('listing_shein_records','package_hash','character varying(71)'),
 ('listing_shein_records','diagnostic_hash','character varying(71)'),('listing_shein_records','diagnostic_status','character varying(32)'),
 ('listing_shein_records','payload','bytea'),('listing_shein_records','diagnostic','bytea'),('listing_shein_records','created_at','timestamp with time zone'),
 ('listing_shein_record_operations','organization_id','character varying(200)'),('listing_shein_record_operations','operation_id','character varying(128)'),
 ('listing_shein_record_operations','owner_user_id','character varying(200)'),('listing_shein_record_operations','input_hash','character varying(71)'),
 ('listing_shein_record_operations','record_id','uuid'),('listing_shein_record_operations','created_at','timestamp with time zone')),
 actual AS (SELECT c.relname::text t,a.attname::text n,format_type(a.atttypid,a.atttypmod) k,a.attnotnull required
 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace ns ON ns.oid=c.relnamespace
 WHERE ns.nspname='public' AND c.relname IN ('listing_shein_records','listing_shein_record_operations') AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped),
 keys(t,cols,primary_key) AS (VALUES ('listing_shein_records',ARRAY['id'],true),
 ('listing_shein_record_operations',ARRAY['organization_id','operation_id'],true),('listing_shein_record_operations',ARRAY['record_id'],false))
 SELECT NOT EXISTS(SELECT 1 FROM expected e FULL JOIN actual a USING(t,n) WHERE e.k IS DISTINCT FROM a.k OR NOT coalesce(a.required,false))
 AND NOT EXISTS(SELECT 1 FROM keys e WHERE NOT EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=to_regclass('public.'||e.t)
 AND i.indisunique AND i.indisvalid AND i.indisready AND i.indimmediate AND (NOT e.primary_key OR i.indisprimary) AND i.indpred IS NULL AND i.indexprs IS NULL
 AND ARRAY(SELECT a.attname::text FROM unnest(i.indkey) WITH ORDINALITY k(num,pos) JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=k.num ORDER BY pos)=e.cols))
 AND EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.listing_shein_record_operations') AND c.confrelid=to_regclass('public.listing_shein_records')
 AND c.contype='f' AND c.convalidated AND c.condeferrable AND c.condeferred AND c.confdeltype='a' AND c.confupdtype='a'
 AND pg_get_constraintdef(c.oid)='FOREIGN KEY (record_id) REFERENCES listing_shein_records(id) DEFERRABLE INITIALLY DEFERRED')
 AND (SELECT count(*)=2 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.listing_shein_records') AND c.convalidated AND c.contype='c'
 AND pg_get_constraintdef(c.oid) IN ('CHECK (((octet_length(payload) >= 1) AND (octet_length(payload) <= 2097152)))','CHECK (((octet_length(diagnostic) >= 1) AND (octet_length(diagnostic) <= 2097152)))'))
 AND EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=to_regclass('public.listing_shein_records') AND i.indisvalid AND i.indisready AND i.indpred IS NULL AND i.indexprs IS NULL
 AND ARRAY(SELECT a.attname::text FROM unnest(i.indkey) WITH ORDINALITY k(num,pos) JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=k.num ORDER BY pos)=ARRAY['organization_id','owner_user_id','created_at','id']
 AND ARRAY(SELECT x FROM unnest(i.indoption) x)=ARRAY[0,0,3,3]::smallint[])
 AND (SELECT count(*)=2 FROM pg_attrdef d JOIN pg_attribute a ON a.attrelid=d.adrelid AND a.attnum=d.adnum
 WHERE a.attname='created_at' AND d.adrelid IN (to_regclass('public.listing_shein_records'),to_regclass('public.listing_shein_record_operations')) AND pg_get_expr(d.adbin,d.adrelid)='CURRENT_TIMESTAMP')`
