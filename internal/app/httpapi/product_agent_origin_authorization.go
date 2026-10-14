package httpapi

import (
	"context"
	"net/http"
	"time"

	"task-processor/internal/agent"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/integration/dataserviceauth"
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
	users, err := dataserviceauth.NewActiveUserClient(settings.AuthorizationAPIURL, nil)
	if err != nil {
		return func(context.Context, agent.ExecutionIdentity) error { return review.ErrUnavailable }
	}
	return func(ctx context.Context, origin agent.ExecutionIdentity) error {
		if ctx == nil || !origin.Valid() {
			return review.ErrForbidden
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		grant, err := client.ReadExactServiceProjectAuthorization(ctx, settings.TenantDirectoryToken, origin.ActorID, settings.ProjectID, origin.OrganizationID)
		if err != nil {
			return review.ErrUnavailable
		}
		if !grant.Found || grant.State != "STATE_ACTIVE" || grant.AuthorizationID != origin.MemberID {
			return review.ErrForbidden
		}
		active, err := users.IsUserActive(ctx, settings.TenantDirectoryToken, origin.ActorID)
		if err != nil {
			return review.ErrUnavailable
		}
		if !active {
			return review.ErrForbidden
		}
		for _, permission := range []string{authz.PermissionLocalAgentWrite, authz.PermissionWorkbenchAgentUse} {
			user := origin.ActorID
			if permission == authz.PermissionLocalAgentWrite {
				// Match freshIdentity/Review: configured global user grants cannot
				// replace the current enterprise role for local product writes.
				user = ""
			}
			allowed, err := authz.AuthorizeOrganization(ctx, permissions, user, origin.OrganizationID, grant.Roles, permission)
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
