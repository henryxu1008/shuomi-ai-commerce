package httpapi

import (
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
)

type PlatformModelChannels struct {
	Service                           *qualified.PlatformChannels
	MaintainerOrganization, ProjectID string
}

func (p *PlatformModelChannels) allows(id authidentity.AuthenticatedIdentity) bool {
	if p == nil || p.MaintainerOrganization == "" || p.ProjectID == "" ||
		id.EffectiveOrganizationID != p.MaintainerOrganization || id.HomeOrganizationID != p.MaintainerOrganization ||
		!slices.Contains(id.Roles, "platform_admin") {
		return false
	}
	return slices.ContainsFunc(id.OrganizationGrants, func(g authidentity.OrganizationGrant) bool {
		return g.OrganizationID == p.MaintainerOrganization && g.ProjectID == p.ProjectID &&
			g.AuthorizationID != "" && g.AuthorizationID == id.EffectiveMemberID && slices.Contains(g.Roles, "platform_admin")
	})
}

func (m modelRoutePricingModule) selectionRoutes() []httproute.Descriptor {
	specs := []struct {
		method, path, action string
		platform             bool
	}{
		{"GET", "/enterprise/text-choice", "list", false},
		{"POST", "/enterprise/text-choice/select", "select", false},
		{"GET", "/enterprise/text-choice/receipts/:key", "receipt", false},
		{"GET", "/platform-channels", "list", true},
		{"POST", "/platform-channels/create", "create", true},
		{"GET", "/platform-channels/create/receipts/:key", "create-receipt", true},
		{"POST", "/platform-channels/:id/publish", "publication", true},
		{"GET", "/platform-channels/:id/receipts/:key", "publication-receipt", true},
		{"POST", "/platform-channels/:id/credentials", "credentials", true},
		{"GET", "/platform-channels/:id/credentials/receipts/:key", "credential-receipt", true},
	}
	routes := make([]httproute.Descriptor, 0, len(specs))
	for _, s := range specs {
		permission := authz.PermissionWorkbenchAgentConfigure
		if s.platform {
			permission = authz.PermissionListingKitPlatformAdm
		}
		routes = append(routes, httproute.Descriptor{
			Method: s.method, Path: modelRouteBase + s.path, Module: m.Name(),
			AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			Permission: permission, RequestTimeout: 30 * time.Second, RejectUnreadRequestBody: s.method == http.MethodGet,
			Handler: func(c *gin.Context) { m.serveSelection(c, s.action, s.platform) },
		})
	}
	return routes
}

func (m modelRoutePricingModule) serveSelection(c *gin.Context, action string, platform bool) {
	ctx := c.Request.Context()
	c.Header("Cache-Control", "no-store")
	actor, org, err := sharedModelIdentity(ctx, m.authorizer, platform, c.Request.Method == http.MethodPost)
	id, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
	if err != nil || org == credential.PlatformOwner || platform && !m.services.PlatformChannels.allows(id) {
		modelRouteFailure(c, errModelRouteForbidden)
		return
	}
	if c.Request.URL.RawQuery != "" || c.Request.URL.ForceQuery {
		modelRouteFailure(c, d.ErrInvalid)
		return
	}
	key := c.GetHeader("Idempotency-Key")
	if c.Request.Method == http.MethodGet {
		key = c.Param("key")
	}
	if action != "list" && !d.ValidFact(key, 128) {
		modelRouteFailure(c, d.ErrInvalid)
		return
	}
	out := gin.H{"organizationId": org}
	if action != "list" {
		out["actorId"], out["requestKey"] = actor, key
	}
	if platform {
		m.servePlatformChannel(c, id, action, key, out)
		return
	}
	shared := m.services.Shared
	if shared == nil {
		modelRouteFailure(c, d.ErrUnavailable)
		return
	}
	if action == "list" {
		view, err := shared.TextChoices(ctx, org)
		if err != nil {
			modelRouteFailure(c, err)
			return
		}
		out["view"] = view
		c.JSON(200, out)
		return
	}
	var r d.Revision
	if action == "select" {
		var input struct {
			d.TextChoiceInput
			ExpectedMemberID string `json:"expectedMemberId"`
		}
		if modelRouteDecode(c, &input) != nil {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		if input.ExpectedMemberID != id.EffectiveMemberID {
			modelRouteFailure(c, d.ErrConflict)
			return
		}
		r, err = shared.ChooseText(ctx, org, actor, key, input.TextChoiceInput)
	} else {
		r, err = shared.Repository.Receipt(ctx, d.Scope{OrganizationID: org, Capability: d.SharedModelCapability}, key)
	}
	if err != nil {
		modelRouteFailure(c, err)
		return
	}
	if r.ActorID != actor || r.Key != key || r.Payload.SharedModels == nil || r.Payload.SharedModels.SchemaVersion != 2 {
		modelRouteFailure(c, d.ErrNotFound)
		return
	}
	out["revision"], out["reference"] = r.ID, r.Payload.SharedModels.Text
	c.JSON(200, out)
}

func (m modelRoutePricingModule) servePlatformChannel(c *gin.Context, id authidentity.AuthenticatedIdentity, action, key string, out gin.H) {
	service := m.services.PlatformChannels.Service
	if service == nil {
		modelRouteFailure(c, d.ErrUnavailable)
		return
	}
	ctx := c.Request.Context()
	actor := id.UserID
	connection := c.Param("id")
	respond := func(value any, err error) {
		if err != nil {
			credentialHTTPFailure(c, err)
			return
		}
		out["result"] = value
		c.JSON(200, out)
	}
	switch action {
	case "list":
		rows, err := service.List(ctx)
		if err != nil {
			credentialHTTPFailure(c, err)
			return
		}
		out["canWrite"] = authz.AllowedOrganization(ctx, m.authorizer, actor, id.EffectiveOrganizationID, id.Roles, authz.PermissionListingKitAdminWrite)
		out["channels"], out["organizations"] = rows, service.CatalogOrganizations()
		c.JSON(200, out)
	case "create":
		var input struct {
			credential.RegistrationInput
			ExpectedMemberID string `json:"expectedMemberId"`
		}
		if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
			return
		}
		value, err := service.Management.CreateRegistration(ctx, credential.PlatformOwner, actor, key, input.RegistrationInput)
		respond(value, err)
	case "create-receipt":
		value, err := service.Management.RegistrationReceipt(ctx, credential.PlatformOwner, actor, key)
		respond(value, err)
	case "publication", "publication-receipt":
		var r d.Revision
		var err error
		if action == "publication" {
			var input struct {
				qualified.PlatformPublishInput
				ExpectedMemberID string `json:"expectedMemberId"`
			}
			if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
				return
			}
			if input.Channel.ConnectionID != connection {
				modelRouteFailure(c, d.ErrInvalid)
				return
			}
			r, err = service.Publish(ctx, actor, key, input.PlatformPublishInput)
		} else {
			r, err = service.Receipt(ctx, connection, actor, key)
		}
		if err != nil {
			modelRouteFailure(c, err)
			return
		}
		out["revision"], out["result"] = r.ID, r.Payload.PlatformChannel
		c.JSON(200, out)
	case "credentials", "credential-receipt":
		var result credential.ManagementResult
		var err error
		if action == "credentials" {
			var input credential.ManagementInput
			if !platformDecode(c, &input, func() string { return input.ExpectedMemberID }, id) {
				return
			}
			result, err = service.Management.Publish(ctx, credential.PlatformOwner, actor, connection, key, input)
		} else {
			result, err = service.Management.Receipt(ctx, credential.PlatformOwner, actor, connection, key)
		}
		respond(result, err)
	}
}

func platformDecode(c *gin.Context, target any, member func() string, id authidentity.AuthenticatedIdentity) bool {
	if modelRouteDecode(c, target) != nil {
		modelRouteFailure(c, d.ErrInvalid)
		return false
	}
	if member() != id.EffectiveMemberID {
		modelRouteFailure(c, d.ErrConflict)
		return false
	}
	return true
}
