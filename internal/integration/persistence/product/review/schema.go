package reviewpersistence

import (
	"context"

	"gorm.io/gorm"
	"task-processor/internal/product/review"
)

// VerifySchema checks this owner's installed read/operation schema without
// installing, repairing or interpreting any retained business records.
func VerifySchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return review.ErrUnavailable
	}
	var ready bool
	err := db.WithContext(ctx).Raw(`WITH expected(table_name,name,kind) AS (VALUES
 ('product_title_proposals','org','text'),('product_title_proposals','id','text'),('product_title_proposals','owner','text'),('product_title_proposals','state','text'),('product_title_proposals','payload','bytea'),
 ('product_title_operations','org','text'),('product_title_operations','actor','text'),('product_title_operations','operation_key','text'),('product_title_operations','fingerprint','text'),('product_title_operations','response','bytea')),
 actual AS (SELECT c.relname::text AS table_name,a.attname::text AS name,format_type(a.atttypid,a.atttypmod) AS kind,a.attnotnull AS required
 FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='public' AND c.relname IN ('product_title_proposals','product_title_operations') AND a.attnum>0 AND NOT a.attisdropped),
 keys(table_name,columns) AS (VALUES ('product_title_proposals',ARRAY['org','id']),('product_title_operations',ARRAY['org','actor','operation_key'])),
 checks(table_name,name,definition) AS (VALUES
 ('product_title_proposals','ck_product_title_proposals_state','CHECK ((state = ANY (ARRAY[''pending''::text, ''accepted''::text, ''rejected''::text, ''applied''::text])))'),
 ('product_title_proposals','ck_product_title_proposals_payload_size','CHECK ((octet_length(payload) <= 65536))'),
 ('product_title_proposals','ck_product_title_proposals_state_payload','CHECK ((state = ((convert_from(payload, ''UTF8''::name))::jsonb ->> ''State''::text)))'),
 ('product_title_operations','product_title_operations_response_check','CHECK ((octet_length(response) <= 65536))')),
 indexes(name,columns) AS (VALUES ('ix_product_title_proposals_actionable_admin',ARRAY['org','id']),('ix_product_title_proposals_actionable_owner',ARRAY['org','owner','id']))
 SELECT NOT EXISTS(SELECT 1 FROM expected e FULL JOIN actual a USING(table_name,name) WHERE e.kind IS DISTINCT FROM a.kind OR a.required IS DISTINCT FROM true)
 AND NOT EXISTS(SELECT 1 FROM keys e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass('public.'||e.table_name) AND c.contype='p' AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY k(number,position) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.number ORDER BY k.position)=e.columns))
 AND NOT EXISTS(SELECT 1 FROM checks e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.'||e.table_name)
 AND c.conname=e.name AND c.contype='c' AND c.convalidated AND NOT c.condeferrable AND pg_get_constraintdef(c.oid)=e.definition))
 AND NOT EXISTS(SELECT 1 FROM indexes e WHERE NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace JOIN pg_index i ON i.indexrelid=c.oid
 WHERE n.nspname='public' AND c.relname=e.name AND i.indrelid=to_regclass('public.product_title_proposals') AND i.indisvalid AND i.indisready AND NOT i.indisunique AND i.indexprs IS NULL
 AND i.indnkeyatts=array_length(e.columns,1)
 AND ARRAY(SELECT a.attname::text FROM unnest(i.indkey::smallint[]) WITH ORDINALITY k(number,position) JOIN pg_attribute a ON a.attrelid=i.indrelid AND a.attnum=k.number WHERE k.position<=i.indnkeyatts ORDER BY k.position)=e.columns
 AND pg_get_expr(i.indpred,i.indrelid)='(state = ANY (ARRAY[''pending''::text, ''accepted''::text]))'))`).Scan(&ready).Error
	if err != nil || !ready {
		return review.ErrUnavailable
	}
	return nil
}
