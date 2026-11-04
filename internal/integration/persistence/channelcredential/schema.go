package channelcredentialstore

import (
	"context"
	"gorm.io/gorm"
	"strings"
	credential "task-processor/internal/aicapability/channelcredential"
)

const MetadataView = `CREATE VIEW public.ai_channel_credential_bindings AS SELECT organization_id,connection_id,provider,channel,account,region,base_url,revision,enabled,length(secret_ciphertext::text)>0 AS key_present FROM public.ai_channel_connections`

// Qualify the exact projection rather than trusting a same-named view. Startup
// never recreates it, imports the old table, or repairs a retained schema.
func VerifyMetadataProjection(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return credential.ErrUnavailable
	}
	var row struct {
		Definition string
		Safe       bool
	}
	e := db.WithContext(ctx).Raw(`SELECT pg_get_viewdef(v.oid,true) AS definition,
 v.relkind='v' AND pg_get_userbyid(v.relowner)='product_agent_owner'
 AND t.relkind='r' AND t.relpersistence='p' AND t.relowner=v.relowner
 AND to_regclass('public.ai_client_credentials') IS NULL AS safe
 FROM pg_class v,pg_class t WHERE v.oid=to_regclass('public.ai_channel_credential_bindings') AND t.oid=to_regclass('public.ai_channel_connections')`).Scan(&row).Error
	compact := func(s string) string {
		s = strings.ToLower(s)
		s = strings.ReplaceAll(s, "public.", "")
		s = strings.ReplaceAll(s, "ai_channel_connections.", "")
		return strings.Map(func(r rune) rune {
			switch r {
			case ' ', '\t', '\n', '\r', '(', ')', ';':
				return -1
			}
			return r
		}, s)
	}
	expected := strings.TrimPrefix(MetadataView, "CREATE VIEW public.ai_channel_credential_bindings AS ")
	if e != nil || !row.Safe || compact(row.Definition) != compact(expected) {
		return credential.ErrUnavailable
	}
	return nil
}

// VerifySecretReader is read-only and qualifies only the fixed dedicated role.
// Metadata/config roles are never upgraded to obtain secret-table access.
func VerifySecretReader(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return credential.ErrUnavailable
	}
	if VerifyMetadataProjection(ctx, db) != nil {
		return credential.ErrUnavailable
	}
	var ok bool
	e := db.WithContext(ctx).Raw(`SELECT current_user='ai_channel_secret_reader' AND rolcanlogin AND rolconnlimit BETWEEN 1 AND 2 AND NOT (rolsuper OR rolcreatedb OR rolcreaterole OR rolbypassrls OR rolreplication)
 AND NOT EXISTS(SELECT 1 FROM pg_auth_members WHERE member=r.oid OR roleid=r.oid)
 AND has_table_privilege(current_user,'public.ai_channel_connections','SELECT')
 AND NOT has_any_column_privilege(current_user,'public.ai_channel_connections','INSERT,UPDATE,REFERENCES')
 AND NOT has_table_privilege(current_user,'public.ai_channel_connections','DELETE,TRUNCATE,TRIGGER')
 AND NOT has_table_privilege(current_user,'public.ai_channel_connection_receipts','SELECT,INSERT,UPDATE,DELETE')
 AND NOT has_schema_privilege(current_user,'public','CREATE')
 AND NOT has_database_privilege(current_user,current_database(),'CREATE,TEMP')
 AND NOT EXISTS(SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema' AND c.relkind IN ('r','p','v','m','f')
 AND c.oid<>to_regclass('public.ai_channel_connections') AND has_any_column_privilege(current_user,c.oid,'SELECT,INSERT,UPDATE,REFERENCES'))
 FROM pg_roles r WHERE rolname=current_user`).Scan(&ok).Error
	if e != nil || !ok {
		return credential.ErrUnavailable
	}
	return nil
}

// InstallSchema is only an explicit new-empty-schema operation. It never
// upgrades the model-bound table or repairs retained relations.
func InstallSchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || db == nil || db.Dialector.Name() != "postgres" {
		return credential.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return EnsureSchema(ctx, tx) })
}

func EnsureSchema(ctx context.Context, tx *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || tx == nil || tx.Dialector.Name() != "postgres" {
		return credential.ErrUnavailable
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return credential.ErrUnavailable
	}
	var safe bool
	e := tx.WithContext(ctx).Raw(`SELECT current_user='product_agent_owner' AND current_schema()='public' AND NOT EXISTS(SELECT 1 FROM pg_class WHERE oid IN (to_regclass('public.ai_client_credentials'),to_regclass('public.ai_channel_connections'),to_regclass('public.ai_channel_connection_receipts'),to_regclass('public.ai_channel_credential_bindings')))`).Scan(&safe).Error
	if e != nil || !safe {
		return credential.ErrUnavailable
	}
	if tx.Migrator().CreateTable(&ConnectionRow{}, &ReceiptRow{}) != nil {
		return credential.ErrUnavailable
	}
	if tx.Exec(MetadataView).Error != nil {
		return credential.ErrUnavailable
	}
	return nil
}
