package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/url"
	"strconv"
	"strings"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"time"

	"github.com/gin-gonic/gin"
	strict "sigs.k8s.io/json"
)

const modelRouteBase = "/api/v1/model-route-pricing"

var errModelRouteForbidden = errors.New("FORBIDDEN")

type ModelRoutePricingDependencies struct{ Text, Image *d.Service }

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
	specs := []struct{ method, path, op string }{{"GET", "", "capabilities"}, {"GET", "/:family/:capability", "current"}, {"GET", "/:family/:capability/bindings", "bindings"}, {"GET", "/:family/:capability/options", "options"}, {"GET", "/:family/:capability/history", "history"}, {"GET", "/:family/:capability/receipts/:key", "receipt"}, {"POST", "/:family/:capability/validate", "validate"}, {"POST", "/:family/:capability/publish", "publish"}}
	out := []httproute.Descriptor{}
	for _, s := range specs {
		out = append(out, httproute.Descriptor{Method: s.method, Path: modelRouteBase + s.path, Module: m.Name(), AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, Permission: authz.PermissionListingKitPlatformAdm, RequestTimeout: 30 * time.Second, RejectUnreadRequestBody: s.method == "GET", Handler: func(c *gin.Context) { m.serve(c, s.op) }})
	}
	return out
}
func validateModelRouteDescriptor(d httproute.Descriptor) error {
	for _, r := range (modelRoutePricingModule{}).routes() {
		if d.Method == r.Method && d.Path == r.Path && d.Module == r.Module && d.AuthPolicy == r.AuthPolicy && d.OrganizationAccessPolicy == r.OrganizationAccessPolicy && d.Permission == r.Permission && d.RequestTimeout == r.RequestTimeout && d.RejectUnreadRequestBody == r.RejectUnreadRequestBody && d.OrganizationTargetResolver == nil && d.Handler != nil {
			return nil
		}
	}
	return errors.New("model route pricing loses current organization/platform permission boundary")
}
func modelRouteIdentity(ctx context.Context, a *authz.ListingKitAuthorizer, write bool) (string, string, error) {
	i, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || ctx.Err() != nil || !d.ValidFact(i.UserID, 128) || !d.ValidFact(i.EffectiveOrganizationID, 128) || !d.ValidFact(i.EffectiveMemberID, 128) || i.TenantID != i.EffectiveOrganizationID || !time.Now().Before(i.TokenExpiresAt) || !authz.AllowedOrganization(ctx, a, i.UserID, i.EffectiveOrganizationID, i.Roles, authz.PermissionListingKitPlatformAdm) {
		return "", "", errModelRouteForbidden
	}
	if write && !authz.AllowedOrganization(ctx, a, i.UserID, i.EffectiveOrganizationID, i.Roles, authz.PermissionListingKitAdminWrite) {
		return "", "", errModelRouteForbidden
	}
	return i.UserID, i.EffectiveOrganizationID, nil
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
func modelRouteBody(c *gin.Context) (d.Input, error) {
	var in d.Input
	media, _, e := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if e != nil || media != "application/json" || c.GetHeader("Content-Encoding") != "" {
		return in, d.ErrInvalid
	}
	raw, e := io.ReadAll(io.LimitReader(c.Request.Body, (16<<10)+1))
	if e != nil || len(raw) > 16<<10 || !strings.HasPrefix(strings.TrimSpace(string(raw)), "{") {
		return in, d.ErrInvalid
	}
	violations, e := strict.UnmarshalStrict(raw, &in)
	if e != nil || len(violations) > 0 {
		return in, d.ErrInvalid
	}
	return in, nil
}
func (m modelRoutePricingModule) serve(c *gin.Context, op string) {
	c.Header("Cache-Control", "no-store")
	actor, org, e := modelRouteIdentity(c.Request.Context(), m.authorizer, op == "publish")
	if e != nil {
		modelRouteFailure(c, e)
		return
	}
	q, e := url.ParseQuery(c.Request.URL.RawQuery)
	if e != nil || c.Request.URL.ForceQuery || len(c.Request.URL.RawQuery) > 512 {
		modelRouteFailure(c, d.ErrInvalid)
		return
	}
	for k, v := range q {
		if op != "history" || (k != "cursor" && k != "limit") || len(v) != 1 {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
	}
	if op == "capabilities" {
		rows := []gin.H{}
		for family, s := range map[string]*d.Service{"text": m.services.Text, "image": m.services.Image} {
			if s != nil {
				for _, sc := range s.Catalog.Scopes(org) {
					rows = append(rows, gin.H{"family": family, "capability": sc.Capability})
				}
			}
		}
		c.JSON(200, gin.H{"organizationId": org, "capabilities": rows})
		return
	}
	s := m.services.Text
	if c.Param("family") == "image" {
		s = m.services.Image
	} else if c.Param("family") != "text" {
		modelRouteFailure(c, d.ErrInvalid)
		return
	}
	scope := d.Scope{OrganizationID: org, Capability: c.Param("capability")}
	supported := false
	if s != nil && s.Catalog != nil {
		for _, v := range s.Catalog.Scopes(org) {
			supported = supported || v == scope
		}
	}
	if !supported || (scope.Capability == d.ImageCapability) != (c.Param("family") == "image") {
		modelRouteFailure(c, d.ErrUnavailable)
		return
	}
	ctx := c.Request.Context()
	switch op {
	case "current":
		r, e := s.Current(ctx, scope)
		status := "READY"
		if errors.Is(e, d.ErrNotFound) {
			status = "UNCONFIGURED"
		} else if errors.Is(e, d.ErrNeedsConfiguration) {
			status = "NEEDS_CONFIGURATION"
		} else if e != nil {
			modelRouteFailure(c, e)
			return
		}
		var current *d.Revision
		if r.ID != "" {
			current = &r
		}
		c.JSON(200, gin.H{"status": status, "revision": current})
	case "bindings":
		v, e := s.Bindings(ctx, scope)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"bindings": v})
	case "options":
		v, e := s.Options(ctx, scope)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"revisions": v})
	case "history":
		limit := 20
		if q.Has("limit") {
			limit, e = strconv.Atoi(q.Get("limit"))
			if e != nil || strconv.Itoa(limit) != q.Get("limit") {
				modelRouteFailure(c, d.ErrInvalid)
				return
			}
		}
		rows, e := s.Repository.History(ctx, scope, q.Get("cursor"), limit)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		next := ""
		if len(rows) == limit {
			next = rows[len(rows)-1].ID
		}
		c.JSON(200, gin.H{"revisions": rows, "nextCursor": next})
	case "receipt":
		key := c.Param("key")
		if !d.ValidFact(key, 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		r, e := s.Repository.Receipt(ctx, scope, key)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"receipt": r})
	case "validate", "publish":
		in, e := modelRouteBody(c)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		if op == "validate" {
			p, e := s.Validate(ctx, scope, in)
			if e != nil {
				modelRouteFailure(c, e)
				return
			}
			raw, _ := json.Marshal(p)
			digest := sha256.Sum256(raw)
			c.JSON(200, gin.H{"payload": p, "expectedRevision": in.ExpectedRevision, "digest": hex.EncodeToString(digest[:])})
			return
		}
		keys := c.Request.Header.Values("Idempotency-Key")
		if len(keys) != 1 || !d.ValidFact(keys[0], 128) {
			modelRouteFailure(c, d.ErrInvalid)
			return
		}
		r, e := s.Publish(ctx, scope, actor, keys[0], in)
		if e != nil {
			modelRouteFailure(c, e)
			return
		}
		c.JSON(200, gin.H{"receipt": r})
	}
}
