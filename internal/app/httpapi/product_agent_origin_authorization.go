package httpapi

import (
	"context"
	"net/http"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/product/review"
)

// Cross-administrator Review consumes the same exact service IAM read used by
// current durable consumers. It neither impersonates a request identity nor
// substitutes the local configuration snapshot for current membership.
func productAgentOriginAuthorization(cfg *config.Config, permissions *authz.ListingKitAuthorizer) func(context.Context, agent.ExecutionIdentity) error {
	if cfg == nil || permissions == nil || cfg.ListingKit.Zitadel.AuthorizationAPIURL == "" || cfg.ListingKit.Zitadel.TenantDirectoryToken == "" || cfg.ListingKit.Zitadel.ProjectID == "" {
		return func(context.Context, agent.ExecutionIdentity) error { return review.ErrUnavailable }
	}
	settings := cfg.ListingKit.Zitadel
	client := zitadel.NewAuthorizationClient(settings.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second})
	return func(ctx context.Context, origin agent.ExecutionIdentity) error {
		if ctx == nil || !origin.Valid() {
			return review.ErrForbidden
		}
		grant, err := client.ReadExactServiceProjectAuthorization(ctx, settings.TenantDirectoryToken, origin.ActorID, settings.ProjectID, origin.OrganizationID)
		if err != nil {
			return review.ErrUnavailable
		}
		if !grant.Found || grant.State != "STATE_ACTIVE" || grant.AuthorizationID != origin.MemberID {
			return review.ErrForbidden
		}
		for _, permission := range []string{authz.PermissionLocalAgentWrite, authz.PermissionWorkbenchAgentUse} {
			allowed, err := authz.AuthorizeOrganization(ctx, permissions, origin.ActorID, origin.OrganizationID, grant.Roles, permission)
			if err != nil {
				return review.ErrUnavailable
			}
			if !allowed {
				return review.ErrForbidden
			}
		}
		return nil
	}
}
