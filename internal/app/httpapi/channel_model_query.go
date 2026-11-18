package httpapi

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/workbenchcontext"
)

type channelModelQuery interface {
	Target(credential.ManagedConnection) *d.ModelBindingReference
	Check(context.Context, string, string, int64, d.ModelBindingReference, func(context.Context) bool) (qualified.ModelQueryResult, error)
}

type modelQueryTarget struct {
	ConnectionID string                  `json:"connectionId"`
	Reference    d.ModelBindingReference `json:"reference"`
}

func (m modelRoutePricingModule) modelQueryRoute() httproute.Descriptor {
	return httproute.Descriptor{
		Method: http.MethodPost, Path: modelRouteBase + "/connections/:id/model-query", Module: m.Name(),
		AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
		Permission: authz.PermissionWorkbenchConnectionTest, RequestTimeout: 30 * time.Second, Handler: m.serveModelQuery,
	}
}

func (m modelRoutePricingModule) modelQueryAllowed(ctx context.Context, id authidentity.AuthenticatedIdentity) bool {
	if ctx.Err() != nil || !credential.ValidID(id.UserID) || !credential.ValidID(id.EffectiveOrganizationID) ||
		!credential.ValidID(id.EffectiveMemberID) || id.TenantID != id.EffectiveOrganizationID || !time.Now().Before(id.TokenExpiresAt) {
		return false
	}
	permissions, err := authz.PermissionsInOrganization(ctx, m.authorizer, id.UserID, id.EffectiveOrganizationID, id.Roles)
	return err == nil && slices.Contains(permissions, authz.PermissionWorkbenchCredentialsManage) &&
		slices.Contains(permissions, authz.PermissionWorkbenchConnectionTest)
}

func (m modelRoutePricingModule) modelQueryTargets(ctx context.Context, id authidentity.AuthenticatedIdentity, connections []credential.ManagedConnection) []modelQueryTarget {
	if m.services.ModelQuery == nil || !m.modelQueryAllowed(ctx, id) {
		return nil
	}
	targets := make([]modelQueryTarget, 0)
	for _, connection := range connections {
		if ref := m.services.ModelQuery.Target(connection); ref != nil {
			targets = append(targets, modelQueryTarget{ConnectionID: connection.ConnectionID, Reference: *ref})
		}
	}
	return targets
}

func (m modelRoutePricingModule) serveModelQuery(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	ctx := c.Request.Context()
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !m.modelQueryAllowed(ctx, id) {
		c.AbortWithStatusJSON(403, gin.H{"code": "FORBIDDEN"})
		return
	}
	var input struct {
		ExpectedMemberID string                  `json:"expectedMemberId"`
		ExpectedRevision int64                   `json:"expectedRevision,string"`
		Reference        d.ModelBindingReference `json:"reference"`
	}
	connection := c.Param("id")
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery || !d.ValidFact(connection, 128) ||
		modelRouteDecode(c, &input) != nil || !d.ValidFact(input.ExpectedMemberID, 128) ||
		input.ExpectedRevision <= 0 || input.ExpectedRevision >= 1<<62 ||
		!d.ValidFact(input.Reference.ID, 64) || !d.ValidFact(input.Reference.Version, 128) {
		credentialHTTPFailure(c, credential.ErrInvalid)
		return
	}
	if input.ExpectedMemberID != id.EffectiveMemberID {
		credentialHTTPFailure(c, credential.ErrConflict)
		return
	}
	token := requestBearerToken(c.GetHeader("Authorization"))
	if m.services.ModelQuery == nil || m.organizationResolver == nil || token == "" {
		credentialHTTPFailure(c, credential.ErrUnavailable)
		return
	}
	authorize := func(ctx context.Context) bool {
		fresh, err := m.organizationResolver.Resolve(ctx, httproute.OrganizationAccessPolicyLiveWrite, workbenchcontext.ResolveInput{
			Identity: id, BearerToken: token, RequestedOrganizationID: id.EffectiveOrganizationID,
		})
		return err == nil && fresh.UserID == id.UserID && fresh.EffectiveOrganizationID == id.EffectiveOrganizationID &&
			fresh.EffectiveMemberID == id.EffectiveMemberID && m.modelQueryAllowed(ctx, fresh)
	}
	result, err := m.services.ModelQuery.Check(ctx, id.EffectiveOrganizationID, connection, input.ExpectedRevision, input.Reference, authorize)
	if err != nil {
		if errors.Is(err, qualified.ErrModelQueryForbidden) {
			c.AbortWithStatusJSON(403, gin.H{"code": "FORBIDDEN"})
		} else {
			credentialHTTPFailure(c, err)
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"organizationId": id.EffectiveOrganizationID, "actorId": id.UserID, "memberId": id.EffectiveMemberID,
		"connectionId": connection, "expectedRevision": strconv.FormatInt(input.ExpectedRevision, 10), "reference": input.Reference, "result": result,
	})
}
