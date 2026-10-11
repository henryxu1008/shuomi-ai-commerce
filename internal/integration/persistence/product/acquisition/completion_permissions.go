package acquisition

import (
	"context"
	"gorm.io/gorm"
	completionstore "task-processor/internal/integration/persistence/listing/completiondraft"
	preparationstore "task-processor/internal/integration/persistence/listing/preparation"
	"task-processor/internal/product/sourcing"
)

const completionPrivileges = `,('listing_completion_draft_records','SELECT'),('listing_completion_draft_records','INSERT'),('listing_completion_draft_heads','SELECT'),('listing_completion_draft_heads','INSERT'),('listing_completion_draft_heads','UPDATE'),('listing_completion_draft_commands','SELECT'),('listing_completion_draft_commands','INSERT')`
const completionRuntimeGrant = "GRANT SELECT,INSERT ON public.listing_completion_draft_records,public.listing_completion_draft_commands TO source_acquisition_runtime; GRANT SELECT,INSERT,UPDATE ON public.listing_completion_draft_heads TO source_acquisition_runtime"

// Owner-only additive installation: no REVOKE, data rewrite, execution grants,
// runtime credential or role creation. Unrelated existing grants are preserved.
func GrantCompletionDraftPermissions(ctx context.Context, db *gorm.DB) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" {
		return sourcing.ErrAcquisitionUnavailable
	}
	var safe bool
	if db.WithContext(ctx).Raw(`SELECT current_database() NOT IN ('postgres','template0','template1') AND current_user<>?
 AND EXISTS(SELECT 1 FROM pg_roles r WHERE r.rolname=? AND r.rolcanlogin AND NOT r.rolsuper AND NOT r.rolcreatedb AND NOT r.rolcreaterole AND NOT r.rolreplication AND NOT r.rolbypassrls AND NOT EXISTS(SELECT 1 FROM pg_auth_members m WHERE m.member=r.oid))`, RuntimeRole, RuntimeRole).Scan(&safe).Error != nil || !safe {
		return sourcing.ErrAcquisitionUnavailable
	}
	if err := completionstore.VerifySchema(ctx, db); err != nil {
		return err
	}
	if err := preparationstore.VerifySchema(ctx, db); err != nil {
		return err
	}
	for _, statement := range []string{preparationRuntimeGrant, completionRuntimeGrant} {
		if db.WithContext(ctx).Exec(statement).Error != nil {
			return sourcing.ErrAcquisitionUnavailable
		}
	}
	return nil
}
