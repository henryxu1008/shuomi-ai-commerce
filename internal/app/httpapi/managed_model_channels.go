package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	qualified "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/routeconfig"
)

func (m modelRoutePricingModule) managedChannelRoutes() []httproute.Descriptor {
	var routes []httproute.Descriptor
	for _, entry := range []struct{ method, path, action string }{
		{http.MethodGet, "", "list"},
		{http.MethodPost, "/create", "create"},
		{http.MethodGet, "/create/receipts/:key", "create-receipt"},
		{http.MethodPost, "/:id/name", "rename"},
		{http.MethodGet, "/:id/name/receipts/:key", "rename-receipt"},
		{http.MethodPost, "/:id/models", "binding"},
		{http.MethodGet, "/:id/models/receipts/:key", "binding-receipt"},
	} {
		routes = append(routes, httproute.Descriptor{Method: entry.method, Path: modelRouteBase + "/managed-channels" + entry.path,
			Module: m.Name(), AuthPolicy: httproute.AuthPolicyCurrentIdentity,
			OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Permission: authz.PermissionWorkbenchChannelsManage,
			RequestTimeout: 30 * time.Second, RejectUnreadRequestBody: entry.method == http.MethodGet,
			Handler: func(c *gin.Context) { m.serveManagedChannel(c, entry.action) }})
	}
	return routes
}

func (m modelRoutePricingModule) serveManagedChannel(c *gin.Context, action string) {
	c.Header("Cache-Control", "no-store")
	ctx := c.Request.Context()
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	org, actor, member := id.EffectiveOrganizationID, id.UserID, id.EffectiveMemberID
	forbidden := func() { c.AbortWithStatusJSON(403, gin.H{"code": "FORBIDDEN"}) }
	if !ok || org == credential.PlatformOwner || ctx.Err() != nil || !credential.ValidID(org) || !credential.ValidID(actor) || !credential.ValidID(member) ||
		id.TenantID != org || !time.Now().Before(id.TokenExpiresAt) {
		forbidden()
		return
	}
	permissions, err := authz.PermissionsInOrganization(ctx, m.authorizer, actor, org, id.Roles)
	if err != nil || !slices.Contains(permissions, authz.PermissionWorkbenchChannelsManage) {
		forbidden()
		return
	}
	canSave := slices.Contains(permissions, authz.PermissionWorkbenchCredentialsManage)
	ownedImageRegistration := slices.Contains(m.ownedImageTenants, org)
	if (action == "create" || action == "create-receipt") && !canSave {
		forbidden()
		return
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		credentialHTTPFailure(c, credential.ErrInvalid)
		return
	}
	manager := m.services.Credentials
	if manager == nil || m.services.Shared == nil {
		credentialHTTPFailure(c, credential.ErrUnavailable)
		return
	}
	catalog, ok := m.services.Shared.Catalog.(*qualified.SharedCatalog)
	if !ok {
		credentialHTTPFailure(c, credential.ErrUnavailable)
		return
	}
	managed := catalog.Managed
	if action == "list" {
		channels, err := manager.RegisteredConnections(ctx, org)
		if err != nil {
			credentialHTTPFailure(c, err)
			return
		}
		bindings := []d.ManagedBinding{}
		bindingTemplates := []string{}
		if managed != nil {
			bindings, err = managed.Bindings.ManagedBindings(ctx, org)
			if err != nil {
				modelRouteFailure(c, err)
				return
			}
			for _, template := range []string{credential.OfficialMiniTemplate, credential.CompatibleMiniTemplate} {
				if managed.Supports(org, template) {
					bindingTemplates = append(bindingTemplates, template)
				}
			}
		}
		readiness := []gin.H{}
		for i := range channels {
			for _, binding := range bindings {
				if binding.ConnectionID == channels[i].ConnectionID && binding.Enabled {
					ref := binding.Reference()
					q, err := catalog.QualifyModel(ctx, org, d.ModelCategoryText, ref)
					if err != nil {
						modelRouteFailure(c, err)
						return
					}
					readiness = append(readiness, gin.H{"bindingId": binding.BindingID, "ready": q.Ready, "reason": q.Reason})
					channels[i].Models = append(channels[i].Models, credential.ManagedModel{ID: ref.ID, Version: ref.Version, Category: "text", Model: qualified.OpenAIQueryModel})
				}
			}
		}
		response := gin.H{
			"organizationId": org, "channels": channels, "bindings": bindings, "bindingTemplates": bindingTemplates,
			"canSave": canSave && manager.CanRegister(), "canRename": manager.CanRegister(), "canAttach": managed != nil, "bindingReadiness": readiness,
			"modelQueryTargets": m.modelQueryTargets(ctx, id, channels),
		}
		if ownedImageRegistration {
			response["grsaiImageRegistration"] = true
		}
		c.JSON(200, response)
		return
	}
	key := c.GetHeader("Idempotency-Key")
	if c.Request.Method == http.MethodGet {
		key = c.Param("key")
	}
	if !credential.ValidID(key) {
		credentialHTTPFailure(c, credential.ErrInvalid)
		return
	}
	envelope := gin.H{"organizationId": org, "actorId": actor, "memberId": member, "requestKey": key}
	switch action {
	case "create", "create-receipt":
		var result credential.RegistrationResult
		var err error
		if action == "create" {
			var body struct {
				ExpectedMemberID string                       `json:"expectedMemberId"`
				Input            credential.RegistrationInput `json:"input"`
			}
			if modelRouteDecode(c, &body) != nil || body.ExpectedMemberID != member {
				credentialHTTPFailure(c, credential.ErrInvalid)
				return
			}
			if body.Input.Template == credential.GRSaiImageTemplate && !ownedImageRegistration {
				forbidden()
				return
			}
			result, err = manager.CreateRegistration(ctx, org, actor, key, body.Input)
		} else {
			result, err = manager.RegistrationReceipt(ctx, org, actor, key)
		}
		if err != nil {
			credentialHTTPFailure(c, err)
			return
		}
		envelope["result"] = result
	case "rename", "rename-receipt":
		connection := c.Param("id")
		var result credential.Registration
		var err error
		if action == "rename" {
			var body struct {
				ExpectedMemberID string                             `json:"expectedMemberId"`
				Input            credential.RenameRegistrationInput `json:"input"`
			}
			if modelRouteDecode(c, &body) != nil || body.ExpectedMemberID != member {
				credentialHTTPFailure(c, credential.ErrInvalid)
				return
			}
			result, err = manager.RenameRegistration(ctx, credential.RenameRegistrationCommand{
				OrganizationID: org, ActorID: actor, RequestKey: key, ConnectionID: connection, Input: body.Input,
			})
		} else {
			result, err = manager.RenameReceipt(ctx, org, actor, key)
		}
		if err != nil {
			credentialHTTPFailure(c, err)
			return
		}
		if result.ConnectionID != connection {
			credentialHTTPFailure(c, credential.ErrNotFound)
			return
		}
		envelope["result"] = result
	default:
		writer, ok := m.services.Shared.Repository.(d.ManagedBindingWriter)
		if managed == nil || !ok {
			modelRouteFailure(c, d.ErrUnavailable)
			return
		}
		connection := c.Param("id")
		if !credential.IsRegisteredConnection(connection) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		var result d.ManagedBinding
		var err error
		if action == "binding" {
			var body struct {
				ExpectedMemberID string                `json:"expectedMemberId"`
				Input            d.ManagedBindingInput `json:"input"`
			}
			if modelRouteDecode(c, &body) != nil || body.ExpectedMemberID != member {
				modelRouteFailure(c, d.ErrInvalid)
				return
			}
			command := d.ManagedBindingCommand{OrganizationID: org, ConnectionID: connection, ActorID: actor, RequestKey: key, Input: body.Input}
			result, err = writer.WriteManagedBinding(ctx, command, func(template string) error {
				if !managed.Supports(org, template) {
					return d.ErrNeedsConfiguration
				}
				return nil
			})
		} else {
			result, err = writer.ManagedBindingReceipt(ctx, org, actor, key)
		}
		if err != nil {
			modelRouteFailure(c, err)
			return
		}
		if result.ConnectionID != connection {
			modelRouteFailure(c, d.ErrNotFound)
			return
		}
		envelope["result"] = result
	}
	c.JSON(200, envelope)
}
