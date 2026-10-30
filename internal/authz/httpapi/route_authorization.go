package httpapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

type RoleAuthorizationFailure string

const (
	RoleAuthorizationIdentityRequired      RoleAuthorizationFailure = "identity_required"
	RoleAuthorizationPermissionDenied      RoleAuthorizationFailure = "permission_denied"
	RoleAuthorizationDependencyUnavailable RoleAuthorizationFailure = "dependency_unavailable"
)

type RoleAuthorizationResponder func(*gin.Context, RoleAuthorizationFailure, string)

func RouteRequiresZitadelAuth(route httproute.Descriptor) bool {
	return route.AuthPolicy == httproute.AuthPolicyVerifiedIdentity
}

func routeRequiredPermission(route httproute.Descriptor) string {
	return strings.TrimSpace(route.Permission)
}

func NewRouteRoleMiddlewareWithAuthorizer(route httproute.Descriptor, authorizer *authz.ListingKitAuthorizer) gin.HandlerFunc {
	return NewRouteRoleMiddlewareWithAuthorizerAndResponder(route, authorizer, defaultRoleAuthorizationResponder)
}

func NewRouteRoleMiddlewareWithAuthorizerAndResponder(route httproute.Descriptor, authorizer *authz.ListingKitAuthorizer, respond RoleAuthorizationResponder) gin.HandlerFunc {
	if respond == nil {
		respond = defaultRoleAuthorizationResponder
	}
	requiredPermission := routeRequiredPermission(route)
	if requiredPermission == "" {
		return nil
	}
	if authorizer == nil {
		var err error
		authorizer, err = authz.NewListingKitAuthorizer(nil, nil)
		if err != nil {
			return func(c *gin.Context) {
				respond(c, RoleAuthorizationDependencyUnavailable, requiredPermission)
			}
		}
	}
	return func(c *gin.Context) {
		identity, ok := authidentity.AuthenticatedIdentityFromContext(c.Request.Context())
		if !ok {
			respond(c, RoleAuthorizationIdentityRequired, requiredPermission)
			return
		}
		allowed := authorizer.Authorize(identity.UserID, identity.Roles, requiredPermission)
		if identity.EffectiveOrganizationID != "" {
			var err error
			allowed, err = authorizer.AuthorizeScoped(c.Request.Context(), identity.UserID, identity.EffectiveOrganizationID, identity.Roles, requiredPermission)
			if err != nil {
				respond(c, RoleAuthorizationDependencyUnavailable, requiredPermission)
				return
			}
		}
		if allowed {
			c.Next()
			return
		}
		respond(c, RoleAuthorizationPermissionDenied, requiredPermission)
	}
}

func defaultRoleAuthorizationResponder(c *gin.Context, failure RoleAuthorizationFailure, requiredPermission string) {
	switch failure {
	case RoleAuthorizationDependencyUnavailable:
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"error":   "listingkit_authorization_unavailable",
			"message": "ListingKit authorization is not available",
		})
	case RoleAuthorizationIdentityRequired:
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":   "listingkit_permission_denied",
			"message": "ZITADEL identity is required to access this ListingKit route",
		})
	default:
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":               "listingkit_permission_denied",
			"message":             "ZITADEL identity is not allowed to access this ListingKit route",
			"required_permission": requiredPermission,
		})
	}
}
