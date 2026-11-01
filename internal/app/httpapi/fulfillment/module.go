package fulfillmentmodule

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"io"
	"mime"
	"net/http"
	sigjson "sigs.k8s.io/json"
	app "task-processor/internal/app/fulfillment"
	runtime "task-processor/internal/app/runtime/fulfillment"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/core/config"
	f "task-processor/internal/fulfillment"
	"task-processor/internal/httproute"
	kernel "task-processor/internal/kernel/module"
	"task-processor/internal/storecenter"
	"time"
)

const BasePath = "/api/v1/workbench/fulfillment"

type Dependencies struct {
	Configuration *app.Configuration
	Starter       app.Starter
	NewWorker     func(*app.Service) (runtime.Worker, error)
	Lifecycle     *runtime.Lifecycle
	Stores        storecenter.FulfillmentReader
	Authorization app.Authorization
	Applications  *storeapp.OfficialApplicationRegistry
	Bind          func(context.Context, string) (context.Context, error)
}
type Module struct {
	Service      *app.Service
	dependencies Dependencies
	address      map[string]bool
}

func (Module) Name() string                         { return "fulfillment" }
func (Module) Enabled(c *config.Config) bool        { return c != nil && c.Workbench.Enabled }
func (m *Module) Register(r *kernel.Registry) error { r.AddRoutes(m.Routes()...); return nil }
func New(ctx context.Context, d Dependencies) (*Module, error) {
	if d.Configuration == nil || len(d.Configuration.ERPs) == 0 || d.Starter == nil || d.NewWorker == nil || d.Lifecycle == nil || d.Stores == nil || d.Applications == nil || d.Bind == nil {
		return nil, f.ErrUnavailable
	}
	access := &storeapp.FulfillmentAccess{Reader: d.Stores, Authorization: d.Authorization, Applications: d.Applications}
	service := &app.Service{ERPs: d.Configuration.ERPs, Aliases: d.Configuration.Aliases, SupplyMovement: d.Configuration.SupplyMovement, Starter: d.Starter, Open: func(ctx context.Context, subject storecenter.FulfillmentSubject, store, kind string) (app.Merchant, error) {
		return access.Open(ctx, subject, store, kind)
	}}
	module := &Module{Service: service, dependencies: d, address: map[string]bool{}}
	for org, erp := range d.Configuration.ERPs {
		raw, err := erp.Execute(ctx, f.Command{OperationID: uuid.NewString(), Organization: org, Command: "configuration", Payload: json.RawMessage(`{}`)})
		if err != nil {
			return nil, err
		}
		var native struct {
			Organization string `json:"organization"`
			Version      string `json:"version"`
			Movement     string `json:"supply_movement"`
			Address      bool   `json:"address_enabled"`
		}
		if json.Unmarshal(raw, &native) != nil || native.Organization != org || native.Version != "0.1.0" || native.Movement != d.Configuration.SupplyMovement[org] {
			return nil, f.ErrUnavailable
		}
		module.address[org] = native.Address
	}
	worker, err := d.NewWorker(service)
	if err != nil {
		return nil, err
	}
	d.Lifecycle.Worker = worker
	return module, nil
}
func (m *Module) Routes() []httproute.Descriptor {
	routes := []httproute.Descriptor{}
	for _, spec := range []struct{ Method, Path string }{{"GET", "/capabilities"}, {"POST", "/commands"}} {
		routes = append(routes, httproute.Descriptor{Method: spec.Method, Path: BasePath + spec.Path, Module: "fulfillment", Permission: "workbench.fulfillment.read", AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite, RequestTimeout: 60 * time.Second, RejectUnreadRequestBody: spec.Method == "GET", Handler: func(c *gin.Context) { m.handle(c, spec.Method) }})
	}
	return routes
}
func (m *Module) handle(c *gin.Context, method string) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	if m == nil || m.Service == nil || m.dependencies.Bind == nil || c.Request.URL.RawQuery != "" {
		failure(c, f.ErrUnavailable)
		return
	}
	ctx, err := m.dependencies.Bind(c.Request.Context(), c.GetHeader("Authorization"))
	if err != nil {
		failure(c, f.ErrForbidden)
		return
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.EffectiveOrganizationID == "" || identity.TenantID != identity.EffectiveOrganizationID || !time.Now().Before(identity.TokenExpiresAt) {
		failure(c, f.ErrForbidden)
		return
	}
	scope := f.Scope{OrganizationID: identity.EffectiveOrganizationID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID}
	if method == "GET" {
		if c.Request.ContentLength != 0 {
			failure(c, f.ErrInvalid)
			return
		}
		_, err = m.dependencies.Authorization.AuthorizeFulfillment(ctx, storecenter.FulfillmentSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID, Purpose: "fulfillment.read"})
		if err != nil {
			failure(c, err)
			return
		}
		c.JSON(200, gin.H{"available": m.dependencies.Lifecycle.Ready() && m.Service.ERPs[scope.OrganizationID] != nil, "addressConfigured": m.address[scope.OrganizationID], "realPlatformValidation": "NOT_RUN"})
		return
	}
	if !m.dependencies.Lifecycle.Ready() {
		failure(c, f.ErrUnavailable)
		return
	}
	media, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || media != "application/json" {
		failure(c, f.ErrInvalid)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 256*1024))
	if err != nil {
		failure(c, f.ErrInvalid)
		return
	}
	var input app.Input
	issues, err := sigjson.UnmarshalStrict(body, &input)
	if err != nil || len(issues) != 0 {
		failure(c, f.ErrInvalid)
		return
	}
	result, err := m.Service.Execute(ctx, scope, input)
	if err != nil {
		failure(c, err)
		return
	}
	if len(result) > 2*1024*1024 || !json.Valid(result) {
		failure(c, f.ErrUnknown)
		return
	}
	c.Data(200, "application/json", result)
}
func failure(c *gin.Context, err error) {
	code, status := "FULFILLMENT_UNAVAILABLE", 503
	switch {
	case errors.Is(err, f.ErrForbidden):
		code, status = "FULFILLMENT_FORBIDDEN", 403
	case errors.Is(err, f.ErrInvalid):
		code, status = "FULFILLMENT_INVALID", 400
	case errors.Is(err, f.ErrConflict):
		code, status = "FULFILLMENT_CONFLICT", 409
	case errors.Is(err, f.ErrUnknown):
		code, status = "FULFILLMENT_UNKNOWN", 409
	}
	c.JSON(status, gin.H{"code": code})
}
