package httpapi

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/operationscockpitobservations"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/operationscockpitauth"
	persistence "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/operationscockpit"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	c "github.com/henryxu1008/shuomi-ai-commerce/internal/operationscockpit"
	cockpithttp "github.com/henryxu1008/shuomi-ai-commerce/internal/operationscockpit/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/shopdiagnosis"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
	"gorm.io/gorm"
	"time"
)

func WithOperationsCockpit() CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.operationsCockpits++ }
}

type operationsCockpitModule struct {
	routes         []httproute.Descriptor
	diagnosisReads *shopdiagnosis.NativeReader
	bind           func(context.Context, string) (context.Context, error)
}

func (operationsCockpitModule) Name() string { return cockpithttp.ModuleName }
func (operationsCockpitModule) Enabled(cfg *config.Config) bool {
	return cfg != nil && cfg.Workbench.Enabled
}
func (m operationsCockpitModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}
func buildOperationsCockpit(ctx context.Context, db *gorm.DB, deps routeAuthDependencies, permissions *authz.ListingKitAuthorizer, capabilities storecenter.RuntimeCapabilities, observations *o.Service) (kernelmodule.Module, error) {
	if !capabilities.OperationsCockpit {
		return nil, c.ErrUnavailable
	}
	if err := storecenter.VerifyCurrentSchemaForCapabilities(ctx, db, capabilities); err != nil {
		return nil, err
	}
	if err := storecenter.VerifyRuntimePermissionsForCapabilities(ctx, db, capabilities); err != nil {
		return nil, err
	}
	boundary, err := operationscockpitauth.New(deps.organizationResolver, permissions, db)
	if err != nil {
		return nil, err
	}
	repo, err := persistence.New(ctx, db, boundary, time.Now)
	if err != nil {
		return nil, err
	}
	h := &cockpithttp.Handler{Repository: repo, Bind: boundary.Bind, Current: boundary.Current, Directory: boundary.ListStores, Revalidate: boundary.ReadStores, Observations: (operationscockpitobservations.Orders{Service: observations}).Read, Now: time.Now, Scope: func(ctx context.Context) (c.Scope, error) {
		id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		s := c.Scope{OrganizationID: id.EffectiveOrganizationID, ActorID: id.UserID}
		if !ok || !s.Valid() || id.TenantID != s.OrganizationID || !id.TokenExpiresAt.After(time.Now()) {
			return c.Scope{}, c.ErrForbidden
		}
		return s, nil
	}}
	routes, err := cockpithttp.BuildRoutes(h)
	if err != nil {
		return nil, err
	}
	reads := &shopdiagnosis.NativeReader{Repository: repo, Boundary: boundary}
	if observations != nil {
		reads.Observations = observations
	}
	return operationsCockpitModule{routes: routes, diagnosisReads: reads, bind: boundary.Bind}, nil
}
