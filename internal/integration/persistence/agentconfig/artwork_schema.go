package agentconfigpersistence

import (
	"context"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/agentconfig"
)

const priorArtworkTargetCheck = `CHECK (((target_platform)::text = ANY ((ARRAY['product'::character varying, 'shein'::character varying, 'temu'::character varying, 'amazon'::character varying])::text[])))`
const artworkTargetCheck = `CHECK (((target_platform)::text = ANY ((ARRAY['product'::character varying, 'shein'::character varying, 'temu'::character varying, 'amazon'::character varying, 'sds'::character varying])::text[])))`
const artworkTargetDDL = `CHECK(target_platform IN ('product','shein','temu','amazon','sds'))`

var artworkSchemaVersionCheck = strings.Replace(materialsSchemaVersionCheck, `'listing-materials-config-v1'::character varying`, `'listing-materials-config-v1'::character varying, 'pod-artwork-config-v1'::character varying`, 1)
var artworkSchemaVersionDDL = strings.Replace(materialsSchemaVersionDDL, `'listing-materials-config-v1'`, `'listing-materials-config-v1','pod-artwork-config-v1'`, 1)
var artworkParametersDDL = strings.TrimSuffix(strings.Replace(materialsParametersDDL, `'listing.materials.agent')`, `'listing.materials.agent','pod.design.agent')`, 1), ")") + `
 OR (schema_version='pod-artwork-config-v1' AND agent_id='pod.design.agent' AND target_platform='sds' AND default_knowledge_base_id IS NULL AND image_parameters IS NOT NULL AND content_parameters IS NULL AND octet_length(image_parameters) BETWEEN 2 AND 8192))`
var artworkParametersCheck = strings.TrimSuffix(strings.Replace(materialsParametersCheck, `'listing.materials.agent'::character varying])`, `'listing.materials.agent'::character varying, 'pod.design.agent'::character varying])`, 1), "))") + ` OR (((schema_version)::text = 'pod-artwork-config-v1'::text) AND ((agent_id)::text = 'pod.design.agent'::text) AND ((target_platform)::text = 'sds'::text) AND (default_knowledge_base_id IS NULL) AND (image_parameters IS NOT NULL) AND (content_parameters IS NULL) AND ((octet_length(image_parameters) >= 2) AND (octet_length(image_parameters) <= 8192)))))`

func artworkChecksInstalled(tx *gorm.DB) (bool, error) {
	var checks []contentSchemaCheck
	if err := tx.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition, convalidated AS valid, connoinherit AS inherited FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass AND conname IN ('template_revisions_check','template_revisions_schema_version_check')`).Scan(&checks).Error; err != nil {
		return false, err
	}
	return exactSchemaChecks(checks, map[string]string{"template_revisions_check": artworkParametersCheck, "template_revisions_schema_version_check": artworkSchemaVersionCheck}), nil
}

func artworkTargetInstalled(tx *gorm.DB) (bool, error) {
	var checks []contentSchemaCheck
	if err := tx.Raw(`SELECT conname AS name, pg_get_constraintdef(oid) AS definition, convalidated AS valid, connoinherit AS inherited FROM pg_constraint WHERE conrelid='agent_configuration.template_revisions'::regclass AND conname='template_revisions_target_platform_check'`).Scan(&checks).Error; err != nil {
		return false, err
	}
	if exactSchemaChecks(checks, map[string]string{"template_revisions_target_platform_check": artworkTargetCheck}) {
		return true, nil
	}
	if exactSchemaChecks(checks, map[string]string{"template_revisions_target_platform_check": priorArtworkTargetCheck}) {
		return false, nil
	}
	return false, agentconfig.ErrInvalid
}

// Serving verifies; only the explicit initializer extends the admitted checks.
func VerifyArtworkConfigurationSchema(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil {
		return agentconfig.ErrUnavailable
	}
	updated, err := verifyContentSchemaShape(db.WithContext(ctx))
	if err != nil || !updated {
		return agentconfig.ErrUnavailable
	}
	installed, err := artworkChecksInstalled(db.WithContext(ctx))
	if err != nil || !installed {
		return agentconfig.ErrUnavailable
	}
	installed, err = artworkTargetInstalled(db.WithContext(ctx))
	if err != nil || !installed {
		return agentconfig.ErrUnavailable
	}
	return nil
}

func UpdateArtworkConfigurationSchema(ctx context.Context, db *gorm.DB, runtimeRole string) error {
	return updateConfigurationSchema(ctx, db, runtimeRole, true, true)
}
