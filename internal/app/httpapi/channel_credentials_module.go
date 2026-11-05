package httpapi

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	credential "task-processor/internal/aicapability/channelcredential"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
)

func (m modelRoutePricingModule) credentialRoutes() []httproute.Descriptor {
	var out []httproute.Descriptor
	for _, spec := range []struct{ method, path, operation string }{
		{"GET", "/connections", "list"},
		{"POST", "/connections/:id/commands", "publish"},
		{"GET", "/connections/:id/receipts/:key", "receipt"},
	} {
		out = append(out, httproute.Descriptor{Method: spec.method, Path: modelRouteBase + spec.path, Module: m.Name(), AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Permission: authz.PermissionWorkbenchCredentialsManage, RequestTimeout: 30 * time.Second, RejectUnreadRequestBody: spec.method == "GET", Handler: func(c *gin.Context) { m.serveCredentials(c, spec.operation) }})
	}
	return out
}

func credentialHTTPFailure(c *gin.Context, e error) {
	status, code := 503, "UNAVAILABLE"
	switch {
	case errors.Is(e, credential.ErrInvalid):
		status, code = 400, "INVALID"
	case errors.Is(e, credential.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, credential.ErrConflict):
		status, code = 409, "CONFLICT"
	case errors.Is(e, credential.ErrOutcomeUnknown):
		code = "OUTCOME_UNKNOWN"
	}
	c.AbortWithStatusJSON(status, gin.H{"code": code})
}

func (m modelRoutePricingModule) serveCredentials(c *gin.Context, operation string) {
	c.Header("Cache-Control", "no-store")
	ctx := c.Request.Context()
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || ctx.Err() != nil || !credential.ValidID(id.UserID) || !credential.ValidID(id.EffectiveOrganizationID) || !credential.ValidID(id.EffectiveMemberID) || id.TenantID != id.EffectiveOrganizationID || !time.Now().Before(id.TokenExpiresAt) || !authz.AllowedOrganization(ctx, m.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles, authz.PermissionWorkbenchCredentialsManage) {
		c.AbortWithStatusJSON(403, gin.H{"code": "FORBIDDEN"})
		return
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		credentialHTTPFailure(c, credential.ErrInvalid)
		return
	}
	manager := m.services.Credentials
	if manager == nil {
		credentialHTTPFailure(c, credential.ErrUnavailable)
		return
	}
	if operation == "list" {
		connections, e := manager.List(ctx, id.EffectiveOrganizationID)
		if e != nil {
			credentialHTTPFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"organizationId": id.EffectiveOrganizationID, "canManage": manager.CanManage(), "connections": connections})
		return
	}
	connection := c.Param("id")
	key := c.Param("key")
	var result credential.ManagementResult
	var e error
	if operation == "publish" {
		var input credential.ManagementInput
		if modelRouteDecode(c, &input) != nil {
			credentialHTTPFailure(c, credential.ErrInvalid)
			return
		}
		key = c.GetHeader("Idempotency-Key")
		result, e = manager.Publish(ctx, id.EffectiveOrganizationID, id.UserID, connection, key, input)
	} else {
		result, e = manager.Receipt(ctx, id.EffectiveOrganizationID, id.UserID, connection, key)
	}
	if e != nil {
		credentialHTTPFailure(c, e)
		return
	}
	c.JSON(200, gin.H{"receipt": result})
}
