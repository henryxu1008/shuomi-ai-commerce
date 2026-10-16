package acquisition

import (
	"context"
	"strings"

	"gorm.io/gorm"
	"task-processor/internal/product/sourcing"
)

const supplyRecordRuntimeGrant = "GRANT SELECT,INSERT ON public.listing_target_records,public.listing_target_record_commands,public.listing_submission_official_intents,public.listing_submission_official_receipts TO source_acquisition_runtime"
const supplyExecutionRuntimeGrant = "GRANT SELECT,INSERT,UPDATE ON public.listing_preparation_targets,public.listing_preparation_operations,public.listing_preparation_operation_items,public.listing_submission_execution_attempts,public.listing_submission_target_fences TO source_acquisition_runtime"
const supplyReviewRuntimeGrant = "GRANT SELECT ON public.product_title_proposals TO source_acquisition_runtime"

// VerifyOwnerRuntimePermissions is catalog-only maintenance for this fixed
// runtime role. It shares serving's allow/deny query without changing session
// authorization, reading its credential or admitting the owner as a runtime.
func VerifyOwnerRuntimePermissions(ctx context.Context, db *gorm.DB, capabilities RuntimeCapabilities) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return sourcing.ErrAcquisitionUnavailable
	}
	query := strings.Replace(runtimePermissionsFor(capabilities), currentRuntimeSubject, "(SELECT '"+RuntimeRole+"'::text AS runtime_role)", 1)
	var facts struct {
		RoleName            string
		RequiredPrivileges  bool
		ForbiddenPrivileges bool
	}
	err := db.WithContext(ctx).Raw(query).Scan(&facts).Error
	if err != nil || facts.RoleName == "" || facts.RoleName == RuntimeRole || !facts.RequiredPrivileges || facts.ForbiddenPrivileges {
		return sourcing.ErrAcquisitionUnavailable
	}
	var grantable bool
	// Role memberships and ownership are already forbidden by the same query.
	// PUBLIC and direct role ACLs therefore cover effective grant options,
	// including column-only grants which table ACL checks would miss.
	if db.WithContext(ctx).Raw(`WITH grantees(oid) AS (SELECT oid FROM pg_roles WHERE rolname=? UNION ALL SELECT 0::oid),
 effective_acl AS (
 SELECT p.* FROM pg_database d CROSS JOIN LATERAL aclexplode(d.datacl) p WHERE d.datname=current_database()
 UNION ALL SELECT p.* FROM pg_namespace n CROSS JOIN LATERAL aclexplode(n.nspacl) p WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema'
 UNION ALL SELECT p.* FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(c.relacl) p WHERE n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema'
 UNION ALL SELECT p.* FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace CROSS JOIN LATERAL aclexplode(a.attacl) p WHERE a.attnum>0 AND NOT a.attisdropped AND n.nspname NOT LIKE 'pg_%' AND n.nspname<>'information_schema')
 SELECT EXISTS(SELECT 1 FROM effective_acl WHERE is_grantable AND grantee IN (SELECT oid FROM grantees))`, RuntimeRole).Scan(&grantable).Error != nil || grantable {
		return sourcing.ErrAcquisitionUnavailable
	}
	return verifyRuntimeSchema(ctx, db, capabilities)
}

// GrantSupplyChainPermissions adds only the admitted new table grants. The
// caller holds the existing Product initialization transaction and lock.
// No original grants, schema/database ACLs, roles or credentials are changed.
func GrantSupplyChainPermissions(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return sourcing.ErrAcquisitionUnavailable
	}
	for _, statement := range []string{supplyRecordRuntimeGrant, supplyExecutionRuntimeGrant, supplyReviewRuntimeGrant} {
		if db.WithContext(ctx).Exec(statement).Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
	}
	return nil
}
