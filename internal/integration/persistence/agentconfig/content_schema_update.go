package agentconfigpersistence

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"gorm.io/gorm"
	"task-processor/internal/agentconfig"
)

const oldContentSchemaVersionCheck = `CHECK (((schema_version)::text = ANY ((ARRAY['title-config-v1'::character varying, 'image-config-v1'::character varying])::text[])))`
const contentSchemaVersionCheck = `CHECK (((schema_version)::text = ANY ((ARRAY['title-config-v1'::character varying, 'image-config-v1'::character varying, 'content-config-v1'::character varying])::text[])))`
const oldContentParametersCheck = `CHECK (((((schema_version)::text = 'title-config-v1'::text) AND ((agent_id)::text <> 'product.image.agent'::text) AND ((target_platform)::text <> 'product'::text) AND (image_parameters IS NULL)) OR (((schema_version)::text = 'image-config-v1'::text) AND ((agent_id)::text = 'product.image.agent'::text) AND (default_knowledge_base_id IS NULL) AND (image_parameters IS NOT NULL) AND ((octet_length(image_parameters) >= 2) AND (octet_length(image_parameters) <= 65536)))))`
const contentParametersCheck = `CHECK (((((schema_version)::text = 'title-config-v1'::text) AND ((agent_id)::text <> ALL ((ARRAY['product.image.agent'::character varying, 'product.content.agent'::character varying])::text[])) AND ((target_platform)::text <> 'product'::text) AND (image_parameters IS NULL) AND (content_parameters IS NULL)) OR (((schema_version)::text = 'image-config-v1'::text) AND ((agent_id)::text = 'product.image.agent'::text) AND (default_knowledge_base_id IS NULL) AND (image_parameters IS NOT NULL) AND (content_parameters IS NULL) AND ((octet_length(image_parameters) >= 2) AND (octet_length(image_parameters) <= 65536))) OR (((schema_version)::text = 'content-config-v1'::text) AND ((agent_id)::text = 'product.content.agent'::text) AND (image_parameters IS NULL) AND (content_parameters IS NOT NULL) AND ((octet_length(content_parameters) >= 2) AND (octet_length(content_parameters) <= 4096)))))`
const contentSchemaVersionDDL = `CHECK(schema_version IN ('title-config-v1','image-config-v1','content-config-v1'))`
const contentParametersDDL = `CHECK ((schema_version='title-config-v1' AND agent_id NOT IN ('product.image.agent','product.content.agent') AND target_platform<>'product' AND image_parameters IS NULL AND content_parameters IS NULL)
 OR (schema_version='image-config-v1' AND agent_id='product.image.agent' AND default_knowledge_base_id IS NULL AND image_parameters IS NOT NULL AND content_parameters IS NULL AND octet_length(image_parameters) BETWEEN 2 AND 65536)
 OR (schema_version='content-config-v1' AND agent_id='product.content.agent' AND image_parameters IS NULL AND content_parameters IS NOT NULL AND octet_length(content_parameters) BETWEEN 2 AND 4096))`

type contentSchemaCheck struct {
	Name, Definition string
	Valid, Inherited bool
}

// The only accepted inputs are the exact current pre-content contract and the
// exact content contract. Foreign or partial shapes are not silently repaired.
func verifyContentSchemaShape(tx *gorm.DB) (bool, error) {
	var columns []struct {
		Name  string
		Valid bool
	}
	if err := tx.Raw(`SELECT attname AS name, atttypid='bytea'::regtype AND NOT attnotnull AND NOT atthasdef AND attidentity='' AND attgenerated='' AS valid
 FROM pg_attribute WHERE attrelid='agent_configuration.template_revisions'::regclass AND attnum>0 AND NOT attisdropped AND attname='content_parameters'`).Scan(&columns).Error; err != nil {
		return false, err
	}
	updated := len(columns) == 1
	if updated && !columns[0].Valid {
		return false, agentconfig.ErrInvalid
	}
	var checks []contentSchemaCheck
	if err := tx.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition, convalidated AS valid, connoinherit AS inherited
 FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass AND contype='c'
 AND conkey && ARRAY(SELECT attnum FROM pg_attribute WHERE attrelid='agent_configuration.template_revisions'::regclass AND attname IN ('schema_version','image_parameters','content_parameters','agent_id'))`).Scan(&checks).Error; err != nil {
		return false, err
	}
	expected := map[string]string{"template_revisions_check": oldContentParametersCheck, "template_revisions_schema_version_check": oldContentSchemaVersionCheck}
	if updated {
		expected["template_revisions_check"], expected["template_revisions_schema_version_check"] = contentParametersCheck, contentSchemaVersionCheck
	}
	if len(checks) != len(expected) {
		return false, agentconfig.ErrInvalid
	}
	for _, check := range checks {
		if check.Definition != expected[check.Name] || !check.Valid || check.Inherited {
			return false, fmt.Errorf("%w: content constraint %s differs from current contract", agentconfig.ErrInvalid, check.Name)
		}
	}
	var unsafe bool
	if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid='agent_configuration.template_revisions'::regclass AND a.attname='content_parameters' AND NOT a.attisdropped
 AND (EXISTS(SELECT 1 FROM pg_index i WHERE i.indrelid=a.attrelid AND a.attnum=ANY(i.indkey)) OR EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=a.attrelid AND c.contype<>'c' AND a.attnum=ANY(c.conkey))))`).Scan(&unsafe).Error; err != nil {
		return false, err
	}
	if unsafe {
		return false, agentconfig.ErrInvalid
	}
	return updated, nil
}

// UpdateContentConfigurationSchema is initializer-only: one bounded DDL
// transaction under the existing non-login owner, without any business DML.
func UpdateContentConfigurationSchema(ctx context.Context, db *gorm.DB, runtimeRole string) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(runtimeRole) {
		return agentconfig.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var allowed bool
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_roles r JOIN pg_namespace n ON n.nspowner=r.oid JOIN pg_class c ON c.relnamespace=n.oid AND c.relowner=r.oid
 WHERE r.rolname='agent_configuration_owner' AND n.nspname='agent_configuration' AND c.relname='template_revisions' AND c.relkind='r' AND NOT c.relispartition AND NOT c.relrowsecurity
 AND NOT(r.rolcanlogin OR r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolbypassrls))`).Scan(&allowed).Error; err != nil || !allowed {
			return fmt.Errorf("%w: schema owner validation failed (%v)", agentconfig.ErrInvalid, err)
		}
		if err := tx.Raw(`SELECT EXISTS(SELECT 1 FROM pg_roles WHERE rolname=? AND NOT(rolsuper OR rolcreatedb OR rolcreaterole OR rolreplication OR rolbypassrls) AND NOT pg_has_role(oid,'agent_configuration_owner','MEMBER'))`, runtimeRole).Scan(&allowed).Error; err != nil || !allowed {
			return fmt.Errorf("%w: runtime role validation failed (%v)", agentconfig.ErrInvalid, err)
		}
		if err := tx.Raw(`SELECT has_schema_privilege(?,'agent_configuration','USAGE') AND NOT has_schema_privilege(?,'agent_configuration','CREATE') AND has_table_privilege(?,'agent_configuration.template_revisions','SELECT') AND has_table_privilege(?,'agent_configuration.template_revisions','INSERT') AND NOT has_any_column_privilege(?,'agent_configuration.template_revisions','UPDATE') AND NOT has_table_privilege(?,'agent_configuration.template_revisions','DELETE,TRUNCATE,REFERENCES,TRIGGER')`, runtimeRole, runtimeRole, runtimeRole, runtimeRole, runtimeRole, runtimeRole).Scan(&allowed).Error; err != nil || !allowed {
			return fmt.Errorf("%w: runtime authority validation failed (%v)", agentconfig.ErrInvalid, err)
		}
		if err := tx.Exec("SET LOCAL lock_timeout='3s'").Error; err != nil {
			return err
		}
		if err := tx.Exec("SET LOCAL statement_timeout='8s'").Error; err != nil {
			return err
		}
		if err := tx.Exec("SET LOCAL ROLE agent_configuration_owner").Error; err != nil {
			return err
		}
		if err := tx.Exec("LOCK TABLE agent_configuration.template_revisions IN ACCESS EXCLUSIVE MODE").Error; err != nil {
			return err
		}
		updated, err := verifyContentSchemaShape(tx)
		if err != nil {
			return err
		}
		if !updated {
			if err := tx.Exec(`ALTER TABLE agent_configuration.template_revisions
 ADD COLUMN content_parameters bytea,
 DROP CONSTRAINT template_revisions_schema_version_check,
 DROP CONSTRAINT template_revisions_check,
 ADD CONSTRAINT template_revisions_schema_version_check ` + contentSchemaVersionDDL + `,
 ADD CONSTRAINT template_revisions_check ` + contentParametersDDL).Error; err != nil {
				return err
			}
		}
		if _, err := verifyContentSchemaShape(tx); err != nil {
			return err
		}
		// Preserve existing grants, roles, and all original facts. The new carrier
		// inherits no mutation authority over immutable revisions.
		if err := tx.Exec(`GRANT SELECT (content_parameters), INSERT (content_parameters) ON agent_configuration.template_revisions TO "` + runtimeRole + `"`).Error; err != nil {
			return err
		}
		return tx.Exec("RESET ROLE").Error
	})
}
