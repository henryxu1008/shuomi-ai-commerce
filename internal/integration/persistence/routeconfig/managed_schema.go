package routeconfigstore

import (
	"context"

	"gorm.io/gorm"
	d "task-processor/internal/aicapability/routeconfig"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
)

// InstallManagedChannels runs only from the explicit Run DB owner command.
// The exact original owner and roles remain responsible for their own objects.
func InstallManagedChannels(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || ctx.Err() != nil || db == nil || db.Dialector.Name() != "postgres" {
		return d.ErrUnavailable
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := channelstore.InstallRegistrations(ctx, tx); err != nil {
			return err
		}
		return tx.Exec(`CREATE TABLE public.ai_enterprise_model_bindings (
 organization_id varchar(128) NOT NULL,
 binding_id varchar(64) NOT NULL,
 connection_id varchar(128) NOT NULL,
 template varchar(128) NOT NULL CHECK (template IN ('openai-mini-v1','openai-compatible-mini-v1')),
 enabled boolean NOT NULL,
 PRIMARY KEY (organization_id,binding_id),
 FOREIGN KEY (organization_id,connection_id) REFERENCES public.ai_channel_registrations(organization_id,connection_id)
);
CREATE UNIQUE INDEX ai_enterprise_model_binding_active ON public.ai_enterprise_model_bindings(organization_id,connection_id) WHERE enabled;
CREATE TABLE public.ai_enterprise_model_binding_receipts (
 organization_id varchar(128) NOT NULL,
 actor_id varchar(128) NOT NULL,
 request_key varchar(128) NOT NULL,
 fingerprint varchar(128) NOT NULL,
 payload bytea NOT NULL CHECK (octet_length(payload)<=4096),
 PRIMARY KEY (organization_id,actor_id,request_key)
);
GRANT SELECT ON public.ai_channel_registrations TO product_agent_runtime, model_route_config_writer;
GRANT SELECT,INSERT ON public.ai_channel_registrations TO title_credential_writer;
GRANT UPDATE(name,revision) ON public.ai_channel_registrations TO title_credential_writer;
GRANT SELECT ON public.ai_enterprise_model_bindings TO product_agent_runtime;
GRANT SELECT,INSERT ON public.ai_enterprise_model_bindings, public.ai_enterprise_model_binding_receipts TO model_route_config_writer;
GRANT UPDATE(enabled) ON public.ai_enterprise_model_bindings TO model_route_config_writer;`).Error
	})
}

func ManagedChannelSchemaInstalled(ctx context.Context, db *gorm.DB) bool {
	if ctx == nil || db == nil || ctx.Err() != nil {
		return false
	}
	var allowed bool
	q := db.WithContext(ctx).Raw(`SELECT count(*)=3 AND bool_and(relkind='r' AND relpersistence='p'
 AND pg_get_userbyid(relowner)='product_agent_owner')
 FROM pg_class WHERE oid IN (to_regclass('public.ai_channel_registrations'),
 to_regclass('public.ai_enterprise_model_bindings'),to_regclass('public.ai_enterprise_model_binding_receipts'))`).Scan(&allowed)
	return q.Error == nil && allowed
}

// VerifyManagedChannelAccess checks installed grants without changing them.
// The regular metadata reader and model writer must never read encrypted Keys.
func VerifyManagedChannelAccess(ctx context.Context, db *gorm.DB, writer bool) bool {
	if !ManagedChannelSchemaInstalled(ctx, db) {
		return false
	}
	var ok bool
	access := db.WithContext(ctx).Raw(`SELECT
 has_table_privilege(current_user,'public.ai_channel_registrations','SELECT')
 AND NOT has_any_column_privilege(current_user,'public.ai_channel_registrations','INSERT,UPDATE,REFERENCES')
 AND NOT has_table_privilege(current_user,'public.ai_channel_registrations','DELETE,TRUNCATE,TRIGGER,SELECT WITH GRANT OPTION')
 AND has_table_privilege(current_user,'public.ai_enterprise_model_bindings','SELECT')
 AND NOT has_table_privilege(current_user,'public.ai_enterprise_model_bindings','UPDATE,DELETE,TRUNCATE,TRIGGER,REFERENCES,SELECT WITH GRANT OPTION,INSERT WITH GRANT OPTION')
 AND has_table_privilege(current_user,'public.ai_enterprise_model_bindings','INSERT')=?
 AND NOT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid='public.ai_enterprise_model_bindings'::regclass AND a.attnum>0 AND NOT a.attisdropped
 AND (has_column_privilege(current_user,a.attrelid,a.attname,'UPDATE')<>(? AND a.attname='enabled')
 OR has_column_privilege(current_user,a.attrelid,a.attname,'UPDATE WITH GRANT OPTION')))
 AND has_table_privilege(current_user,'public.ai_enterprise_model_binding_receipts','SELECT')=?
 AND has_table_privilege(current_user,'public.ai_enterprise_model_binding_receipts','INSERT')=?
 AND NOT has_table_privilege(current_user,'public.ai_enterprise_model_binding_receipts','UPDATE,DELETE,TRUNCATE,TRIGGER,REFERENCES,SELECT WITH GRANT OPTION,INSERT WITH GRANT OPTION')
 AND NOT has_any_column_privilege(current_user,'public.ai_enterprise_model_binding_receipts','UPDATE,REFERENCES')
 AND NOT has_column_privilege(current_user,'public.ai_channel_connections','secret_ciphertext','SELECT')`, writer, writer, writer, writer).Scan(&ok)
	return access.Error == nil && ok
}
