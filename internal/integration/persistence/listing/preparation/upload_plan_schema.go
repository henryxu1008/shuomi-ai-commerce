package preparationpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/listing/preparation"
)

var UploadPlanTables = []string{"listing_preparation_upload_plans", "listing_preparation_upload_plan_items"}

func InstallUploadPlanSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "postgres" {
		return preparation.ErrUnavailable
	}
	return db.Exec(`CREATE TABLE IF NOT EXISTS listing_preparation_upload_plans (
 organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,
 id uuid NOT NULL,command_key uuid NOT NULL,input_hash varchar(64) NOT NULL CHECK(length(input_hash)=64),input_json jsonb NOT NULL,
 preparation_id uuid NOT NULL,store_id uuid NOT NULL,store_binding_hash varchar(64) NOT NULL CHECK(length(store_binding_hash)=64),
 cancelled boolean NOT NULL DEFAULT false,
 item_count bigint NOT NULL CHECK(item_count>0),revision bigint NOT NULL CHECK(revision>0),created_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,id),UNIQUE(organization_id,actor_id,command_key),
 FOREIGN KEY(organization_id,actor_id,preparation_id) REFERENCES listing_preparations(organization_id,actor_id,id));
CREATE TABLE IF NOT EXISTS listing_preparation_upload_plan_items (
 organization_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,member_id varchar(128) NOT NULL,
 plan_id uuid NOT NULL,source_id uuid NOT NULL,source_version bigint NOT NULL CHECK(source_version>0),revision bigint NOT NULL CHECK(revision>0),
 adapt_operation_id uuid,optimize_operation_id uuid,upload_operation_id uuid,
 PRIMARY KEY(organization_id,actor_id,plan_id,source_id),
 FOREIGN KEY(organization_id,actor_id,plan_id) REFERENCES listing_preparation_upload_plans(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,source_id) REFERENCES listing_preparation_sources(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,adapt_operation_id) REFERENCES listing_preparation_operations(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,optimize_operation_id) REFERENCES listing_preparation_operations(organization_id,actor_id,id),
 FOREIGN KEY(organization_id,actor_id,upload_operation_id) REFERENCES listing_preparation_operations(organization_id,actor_id,id));`).Error
}

func VerifyUploadPlanSchema(ctx context.Context, db *gorm.DB) error {
	if VerifyOperationSchema(ctx, db) != nil {
		return preparation.ErrUnavailable
	}
	for _, q := range []string{"SELECT organization_id,actor_id,member_id,id,command_key,input_hash,input_json,preparation_id,store_id,store_binding_hash,item_count,revision,cancelled,created_at FROM listing_preparation_upload_plans LIMIT 0", "SELECT organization_id,actor_id,member_id,plan_id,source_id,source_version,revision,adapt_operation_id,optimize_operation_id,upload_operation_id FROM listing_preparation_upload_plan_items LIMIT 0"} {
		if db.WithContext(ctx).Exec(q).Error != nil {
			return preparation.ErrUnavailable
		}
	}
	var ready bool
	err := db.WithContext(ctx).Raw(`WITH expected(relation,kind,columns,reference,reference_columns) AS (VALUES
 ('listing_preparation_upload_plans','p',ARRAY['organization_id','actor_id','id'],NULL::text,NULL::text[]),
 ('listing_preparation_upload_plans','u',ARRAY['organization_id','actor_id','command_key'],NULL,NULL),
 ('listing_preparation_upload_plans','f',ARRAY['organization_id','actor_id','preparation_id'],'listing_preparations',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_upload_plan_items','p',ARRAY['organization_id','actor_id','plan_id','source_id'],NULL,NULL),
 ('listing_preparation_upload_plan_items','f',ARRAY['organization_id','actor_id','plan_id'],'listing_preparation_upload_plans',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_upload_plan_items','f',ARRAY['organization_id','actor_id','source_id'],'listing_preparation_sources',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_upload_plan_items','f',ARRAY['organization_id','actor_id','adapt_operation_id'],'listing_preparation_operations',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_upload_plan_items','f',ARRAY['organization_id','actor_id','optimize_operation_id'],'listing_preparation_operations',ARRAY['organization_id','actor_id','id']),
 ('listing_preparation_upload_plan_items','f',ARRAY['organization_id','actor_id','upload_operation_id'],'listing_preparation_operations',ARRAY['organization_id','actor_id','id'])
 ) SELECT NOT EXISTS(SELECT 1 FROM expected e WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c
 WHERE c.conrelid=to_regclass(e.relation) AND c.contype::text=e.kind AND c.convalidated AND NOT c.condeferrable
 AND ARRAY(SELECT a.attname::text FROM unnest(c.conkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.id ORDER BY k.position)=e.columns
 AND (e.reference IS NULL OR (c.confrelid=to_regclass(e.reference)
 AND ARRAY(SELECT a.attname::text FROM unnest(c.confkey) WITH ORDINALITY AS k(id,position)
 JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.id ORDER BY k.position)=e.reference_columns))))`).Scan(&ready).Error
	if err != nil || !ready {
		return preparation.ErrUnavailable
	}
	return nil
}
