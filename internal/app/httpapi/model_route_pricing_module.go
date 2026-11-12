package httpapi

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/url"
	strict "sigs.k8s.io/json"
	"strconv"
	"strings"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	fx "task-processor/internal/app/runtime/automaticfx"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"time"
)

const modelRouteBase = "/api/v1/model-route-pricing"

var errModelRouteForbidden = errors.New("FORBIDDEN")

type ModelRoutePricingDependencies struct {
	Shared      *d.SharedModelService
	Prices      *d.PlatformPricesService
	Credentials *credential.Management
	AutomaticFX AutomaticFXControl
}

type AutomaticFXControl interface {
	Status(context.Context) (fx.StatusSnapshot, error)
	Start(context.Context) error
	Stop(context.Context) error
}

func WithModelRoutePricing(deps ModelRoutePricingDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.modelRoutePricing = &deps }
}

type modelRoutePricingModule struct {
	services   ModelRoutePricingDependencies
	authorizer *authz.ListingKitAuthorizer
}

func (modelRoutePricingModule) Name() string                { return "model-route-pricing" }
func (modelRoutePricingModule) Enabled(*config.Config) bool { return true }
func (m modelRoutePricingModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes()...)
	return nil
}
func (m modelRoutePricingModule) routes() []httproute.Descriptor {
	specs := []struct{ method, path, op string }{{"GET", "", "capabilities"}}
	specs = append(specs, struct{ method, path, op string }{"GET", "/platform-prices/automatic-fx", "fx-status"}, struct{ method, path, op string }{"POST", "/platform-prices/automatic-fx/start", "fx-start"}, struct{ method, path, op string }{"POST", "/platform-prices/automatic-fx/stop", "fx-stop"})
	for _, family := range []string{"enterprise", "platform-prices"} {
		specs = append(specs, struct{ method, path, op string }{"GET", "/" + family, "current"})
		for _, op := range []string{"history", "receipts/:key"} {
			specs = append(specs, struct{ method, path, op string }{"GET", "/" + family + "/" + op, strings.Split(op, "/")[0]})
		}
		for _, op := range []string{"validate", "publish"} {
			specs = append(specs, struct{ method, path, op string }{"POST", "/" + family + "/" + op, op})
		}
	}
	specs = append(specs, struct{ method, path, op string }{"GET", "/enterprise/options", "options"})
	out := []httproute.Descriptor{}
	for _, spec := range specs {
		permission := authz.PermissionWorkbenchAgentConfigure
		platform := strings.HasPrefix(spec.path, "/platform-prices")
		if platform {
			permission = authz.PermissionListingKitPlatformAdm
		}
		out = append(out, httproute.Descriptor{Method: spec.method, Path: modelRouteBase + spec.path, Module: m.Name(), AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Permission: permission, RequestTimeout: 30 * time.Second, RejectUnreadRequestBody: spec.method == "GET" || strings.HasPrefix(spec.op, "fx-"), Handler: func(c *gin.Context) { m.serve(c, spec.op, platform) }})
	}
	return append(out, m.credentialRoutes()...)
}
func validateModelRouteDescriptor(descriptor httproute.Descriptor) error {
	for _, r := range (modelRoutePricingModule{}).routes() {
		if descriptor.Method == r.Method && descriptor.Path == r.Path && descriptor.Module == r.Module && descriptor.AuthPolicy == r.AuthPolicy && descriptor.OrganizationAccessPolicy == r.OrganizationAccessPolicy && descriptor.Permission == r.Permission && descriptor.RequestTimeout == r.RequestTimeout && descriptor.RejectUnreadRequestBody == r.RejectUnreadRequestBody && descriptor.OrganizationTargetResolver == nil && descriptor.Handler != nil {
			return nil
		}
	}
	return errors.New("model configuration loses organization/permission boundary")
}
func sharedModelIdentity(ctx context.Context, a *authz.ListingKitAuthorizer, platform, write bool) (string, string, error) {
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	permission := authz.PermissionWorkbenchAgentConfigure
	if platform {
		permission = authz.PermissionListingKitPlatformAdm
	}
	if !ok || ctx.Err() != nil || !d.ValidFact(i.UserID, 128) || !d.ValidFact(i.EffectiveOrganizationID, 128) || !d.ValidFact(i.EffectiveMemberID, 128) || i.TenantID != i.EffectiveOrganizationID || !time.Now().Before(i.TokenExpiresAt) || !authz.AllowedOrganization(ctx, a, i.UserID, i.EffectiveOrganizationID, i.Roles, permission) {
		return "", "", errModelRouteForbidden
	}
	if platform && write && !authz.AllowedOrganization(ctx, a, i.UserID, i.EffectiveOrganizationID, i.Roles, authz.PermissionListingKitAdminWrite) {
		return "", "", errModelRouteForbidden
	}
	return i.UserID, i.EffectiveOrganizationID, nil
}
func modelRouteIdentity(ctx context.Context, a *authz.ListingKitAuthorizer, write bool) (string, string, error) {
	return sharedModelIdentity(ctx, a, true, write)
}
func modelRouteFailure(c *gin.Context, e error) {
	status, code := 503, "UNAVAILABLE"
	switch {
	case errors.Is(e, errModelRouteForbidden):
		status, code = 403, "FORBIDDEN"
	case errors.Is(e, d.ErrInvalid):
		status, code = 400, "INVALID"
	case errors.Is(e, d.ErrNotFound):
		status, code = 404, "NOT_FOUND"
	case errors.Is(e, d.ErrConflict):
		status, code = 409, "CONFLICT"
	case errors.Is(e, d.ErrNeedsConfiguration):
		status, code = 409, "NEEDS_CONFIGURATION"
	case errors.Is(e, d.ErrOutcomeUnknown):
		code = "OUTCOME_UNKNOWN"
	}
	c.Header("Cache-Control", "no-store")
	c.AbortWithStatusJSON(status, gin.H{"code": code})
}

func modelRouteDecode(c *gin.Context, target any) error {
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" || c.GetHeader("Content-Encoding") != "" {
		return d.ErrInvalid
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, (16<<10)+1))
	if err != nil || len(raw) > 16<<10 || !strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
		return d.ErrInvalid
	}
	violations, err := strict.UnmarshalStrict(raw, target)
	if err != nil || len(violations) > 0 {
		return d.ErrInvalid
	}
	return nil
}
func (m modelRoutePricingModule) serve(c *gin.Context, op string, platform bool) {
	c.Header("Cache-Control", "no-store")
	ctx := c.Request.Context()
	actor, org, err := sharedModelIdentity(ctx, m.authorizer, platform, op == "publish" || op == "fx-start" || op == "fx-stop")
	if err != nil {
		modelRouteFailure(c, err)
		return
	}
	query, err := url.ParseQuery(c.Request.URL.RawQuery)
	if err != nil || c.Request.URL.ForceQuery || len(c.Request.URL.RawQuery) > 512 {
		modelRouteFailure(c, d.ErrInvalid)
		return
	}
	for key, values := range query {
		if op != "history" || (key != "limit" && key != "cursor") || len(values) != 1 {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
	}
	if strings.HasPrefix(op, "fx-") {
		if m.services.AutomaticFX == nil {
			modelRouteFailure(c, d.ErrUnavailable)
			return
		}
		switch op {
		case "fx-status":
			state, e := m.services.AutomaticFX.Status(ctx)
			if e != nil {
				modelRouteFailure(c, d.ErrOutcomeUnknown)
				return
			}
			c.JSON(200, gin.H{"state": state.State, "checkedAt": state.CheckedAt, "stopRequested": state.StopRequested})
			return
		case "fx-start":
			if e := m.services.AutomaticFX.Start(ctx); e != nil {
				modelRouteFailure(c, d.ErrOutcomeUnknown)
				return
			}
			c.JSON(200, gin.H{"state": "STARTED"})
			return
		case "fx-stop":
			if e := m.services.AutomaticFX.Stop(ctx); e != nil {
				modelRouteFailure(c, d.ErrOutcomeUnknown)
				return
			}
			c.JSON(200, gin.H{"state": "QUIESCED"})
			return
		}
	}
	if op == "capabilities" {
		c.JSON(200, gin.H{"organizationId": org, "unitVersion": aicapability.AIPointMicrounitV1, "available": m.services.Shared != nil, "canMaintainPrices": authz.AllowedOrganization(ctx, m.authorizer, actor, org, authidentityRoles(ctx), authz.PermissionListingKitPlatformAdm)})
		return
	}
	scope := d.Scope{OrganizationID: org, Capability: d.SharedModelCapability}
	var repo d.Repository
	if platform {
		if m.services.Prices != nil {
			repo = m.services.Prices.Repository
		}
		scope = d.PlatformPricesScope()
	} else if m.services.Shared != nil {
		repo = m.services.Shared.Repository
	}
	if repo == nil {
		modelRouteFailure(c, d.ErrUnavailable)
		return
	}
	switch op {
	case "current":
		var revision d.Revision
		var preview *d.SharedModelPreview
		if platform {
			revision, err = m.services.Prices.Current(ctx)
		} else {
			var value d.SharedModelPreview
			revision, value, err = m.services.Shared.Current(ctx, org)
			if err == nil {
				preview = &value
			}
		}
		status := "READY"
		if errors.Is(err, d.ErrNotFound) {
			status = "UNCONFIGURED"
		} else if err != nil {
			modelRouteFailure(c, err)
			return
		}
		if preview != nil && !preview.Ready {
			status = "NEEDS_CONFIGURATION"
		}
		var current *d.Revision
		if revision.ID != "" {
			current = &revision
		}
		c.JSON(200, gin.H{"status": status, "revision": current, "preview": publicSharedPreview(preview)})
	case "options":
		options, ok := m.services.Shared.Catalog.(d.SharedModelOptions)
		if !ok {
			modelRouteFailure(c, d.ErrUnavailable)
			return
		}
		rows, e := options.Options(ctx, org)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		channels := []string{"openai-direct"}
		for _, row := range rows {
			found := false
			for _, channel := range channels {
				found = found || row.Channel == channel
			}
			if !found {
				channels = append(channels, row.Channel)
			}
		}
		c.JSON(200, gin.H{"organizationId": org, "channels": channels, "models": rows})
	case "history":
		limit := 20
		if query.Has("limit") {
			limit, err = strconv.Atoi(query.Get("limit"))
			if err != nil || strconv.Itoa(limit) != query.Get("limit") || limit < 1 || limit > 50 {
				modelRouteFailure(c, d.ErrInvalid)
				return
			}
		}
		cursor := query.Get("cursor")
		if cursor != "" && !d.ValidFact(cursor, 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		rows, e := repo.History(ctx, scope, cursor, limit)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		next := ""
		if len(rows) == limit {
			next = rows[len(rows)-1].ID
		}
		c.JSON(200, gin.H{"revisions": rows, "nextCursor": next})
	case "receipts":
		key := c.Param("key")
		if !d.ValidFact(key, 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		r, e := repo.Receipt(ctx, scope, key)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"receipt": r})
	case "validate", "publish":
		var input d.SharedModelInput
		var priceInput d.PlatformPricesInput
		var target any = &input
		if platform {
			target = &priceInput
		}
		if err = modelRouteDecode(c, target); err != nil {
			modelRouteFailure(c, err)
			return
		}
		expected := input.ExpectedRevision
		if platform {
			expected = priceInput.ExpectedRevision
		}
		if !d.ValidFact(expected, 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		if op == "validate" {
			current, e := repo.Current(ctx, scope)
			if e != nil && !errors.Is(e, d.ErrNotFound) {
				modelRouteFailure(c, e)
				return
			}
			id := current.ID
			if id == "" {
				id = d.NoRevision
			}
			if expected != id {
				modelRouteFailure(c, d.ErrConflict)
				return
			}
			if platform {
				snapshot, e := d.FreezePlatformPrices(priceInput, "preview", nil, time.Now().UTC())
				if e != nil {
					modelRouteFailure(c, e)
					return
				}
				c.JSON(200, gin.H{"expectedRevision": expected, "snapshot": snapshot})
				return
			}
			preview, e := m.services.Shared.Preview(ctx, org, input.Selection)
			if e != nil {
				modelRouteFailure(c, e)
				return
			}
			c.JSON(200, gin.H{"expectedRevision": expected, "selection": input.Selection, "preview": publicSharedPreview(&preview)})
			return
		}
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || !d.ValidFact(keys[0], 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		var revision d.Revision
		if platform {
			revision, err = m.services.Prices.Publish(ctx, actor, keys[0], priceInput)
		} else {
			revision, err = m.services.Shared.Publish(ctx, org, actor, keys[0], input)
		}
		if err != nil {
			modelRouteFailure(c, err)
			return
		}
		c.JSON(200, gin.H{"receipt": revision})
	}
}
func authidentityRoles(ctx context.Context) []string {
	i, _ := authidentity.AuthenticatedIdentityFromContext(ctx)
	return i.Roles
}

func publicSharedPreview(preview *d.SharedModelPreview) any {
	if preview == nil {
		return nil
	}
	models := make([]d.ModelOption, 0, len(preview.Models))
	for _, q := range preview.Models {
		option := d.ModelOption{Category: q.Category, Reference: q.Reference, Channel: q.Channel, Model: q.Model, Ready: q.Ready, Reason: q.Reason}
		if q.Price.Validate() == nil {
			price := q.Price
			option.Price = &price
		}
		models = append(models, option)
	}
	return gin.H{"ready": preview.Ready, "models": models}
}
