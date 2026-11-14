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
	resolve := productAgentOriginRoles(cfg, permissions)
	return func(ctx context.Context, origin agent.ExecutionIdentity) error {
		_, err := resolve(ctx, origin)
		return err
	}
}

// The role result comes from the same exact, live membership check used for
// durable Review origins. It is never derived from a saved browser token.
func productAgentOriginRoles(cfg *config.Config, permissions *authz.ListingKitAuthorizer) func(context.Context, agent.ExecutionIdentity) ([]string, error) {
	if cfg == nil || permissions == nil || cfg.ListingKit.Zitadel.AuthorizationAPIURL == "" || cfg.ListingKit.Zitadel.TenantDirectoryToken == "" || cfg.ListingKit.Zitadel.ProjectID == "" {
		return func(context.Context, agent.ExecutionIdentity) ([]string, error) { return nil, review.ErrUnavailable }
	}
	settings := cfg.ListingKit.Zitadel
	client := zitadel.NewAuthorizationClient(settings.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second})
	users, err := dataserviceauth.NewActiveUserClient(settings.AuthorizationAPIURL, nil)
	if err != nil {
		return func(context.Context, agent.ExecutionIdentity) ([]string, error) { return nil, review.ErrUnavailable }
	}
	return func(ctx context.Context, origin agent.ExecutionIdentity) ([]string, error) {
		if ctx == nil || !origin.Valid() {
			return nil, review.ErrForbidden
		}
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		grant, err := client.ReadExactServiceProjectAuthorization(ctx, settings.TenantDirectoryToken, origin.ActorID, settings.ProjectID, origin.OrganizationID)
		if err != nil {
			return nil, review.ErrUnavailable
		}
		if !grant.Found || grant.State != "STATE_ACTIVE" || grant.AuthorizationID != origin.MemberID {
			return nil, review.ErrForbidden
		}
		active, err := users.IsUserActive(ctx, settings.TenantDirectoryToken, origin.ActorID)
		if err != nil {
			return nil, review.ErrUnavailable
		}
		if !active {
			return nil, review.ErrForbidden
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
				return nil, review.ErrUnavailable
			}
			if !allowed {
				return nil, review.ErrForbidden
			}
		}
		return append([]string(nil), grant.Roles...), nil
	}
}
