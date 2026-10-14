package agentconfigpersistence

import (
	"context"
	"gorm.io/gorm"
	"task-processor/internal/agentconfig"
)

const materialsSchemaVersionCheck = `CHECK (((schema_version)::text = ANY ((ARRAY['title-config-v1'::character varying, 'image-config-v1'::character varying, 'content-config-v1'::character varying, 'listing-materials-config-v1'::character varying])::text[])))`
const materialsParametersCheck = `CHECK (((((schema_version)::text = 'title-config-v1'::text) AND ((agent_id)::text <> ALL ((ARRAY['product.image.agent'::character varying, 'product.content.agent'::character varying, 'listing.materials.agent'::character varying])::text[])) AND ((target_platform)::text <> 'product'::text) AND (image_parameters IS NULL) AND (content_parameters IS NULL)) OR (((schema_version)::text = 'image-config-v1'::text) AND ((agent_id)::text = 'product.image.agent'::text) AND (default_knowledge_base_id IS NULL) AND (image_parameters IS NOT NULL) AND (content_parameters IS NULL) AND ((octet_length(image_parameters) >= 2) AND (octet_length(image_parameters) <= 65536))) OR (((schema_version)::text = 'content-config-v1'::text) AND ((agent_id)::text = 'product.content.agent'::text) AND (image_parameters IS NULL) AND (content_parameters IS NOT NULL) AND ((octet_length(content_parameters) >= 2) AND (octet_length(content_parameters) <= 4096))) OR (((schema_version)::text = 'listing-materials-config-v1'::text) AND ((agent_id)::text = 'listing.materials.agent'::text) AND ((target_platform)::text = 'shein'::text) AND (default_knowledge_base_id IS NULL) AND (image_parameters IS NULL) AND (content_parameters IS NULL))))`
const materialsSchemaVersionDDL = `CHECK(schema_version IN ('title-config-v1','image-config-v1','content-config-v1','listing-materials-config-v1'))`
const materialsParametersDDL = `CHECK ((schema_version='title-config-v1' AND agent_id NOT IN ('product.image.agent','product.content.agent','listing.materials.agent') AND target_platform<>'product' AND image_parameters IS NULL AND content_parameters IS NULL)
 OR (schema_version='image-config-v1' AND agent_id='product.image.agent' AND default_knowledge_base_id IS NULL AND image_parameters IS NOT NULL AND content_parameters IS NULL AND octet_length(image_parameters) BETWEEN 2 AND 65536)
 OR (schema_version='content-config-v1' AND agent_id='product.content.agent' AND image_parameters IS NULL AND content_parameters IS NOT NULL AND octet_length(content_parameters) BETWEEN 2 AND 4096)
 OR (schema_version='listing-materials-config-v1' AND agent_id='listing.materials.agent' AND target_platform='shein' AND default_knowledge_base_id IS NULL AND image_parameters IS NULL AND content_parameters IS NULL))`

func materialsChecksInstalled(tx *gorm.DB) (bool, error) {
	var checks []contentSchemaCheck
	if err := tx.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition, convalidated AS valid, connoinherit AS inherited FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass AND conname IN ('template_revisions_check','template_revisions_schema_version_check')`).Scan(&checks).Error; err != nil {
		return false, err
	}
	return exactSchemaChecks(checks, map[string]string{"template_revisions_check": materialsParametersCheck, "template_revisions_schema_version_check": materialsSchemaVersionCheck}) || exactSchemaChecks(checks, map[string]string{"template_revisions_check": artworkParametersCheck, "template_revisions_schema_version_check": artworkSchemaVersionCheck}), nil
}

// Serving only verifies the exact admitted contract. It never installs or
// changes constraints, activations, templates, modes or historical requests.
func VerifyMaterialsConfigurationSchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil {
		return agentconfig.ErrUnavailable
	}
	updated, err := verifyContentSchemaShape(db.WithContext(ctx))
	if err != nil || !updated {
		return agentconfig.ErrUnavailable
	}
	installed, err := materialsChecksInstalled(db.WithContext(ctx))
	if err != nil || !installed {
		return agentconfig.ErrUnavailable
	}
	return nil
}

func UpdateMaterialsConfigurationSchema(ctx context.Context, db *gorm.DB, runtimeRole string) error {
	return updateConfigurationSchema(ctx, db, runtimeRole, true, false)
}
