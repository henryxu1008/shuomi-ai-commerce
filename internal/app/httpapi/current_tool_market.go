package httpapi

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/httproute"
	tmstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/toolmarket"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/toolmarketauth"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	tm "github.com/henryxu1008/shuomi-ai-commerce/internal/toolmarket"
	tmhttp "github.com/henryxu1008/shuomi-ai-commerce/internal/toolmarket/httpapi"
	"gorm.io/gorm"
)

type ToolMarketDependencies struct {
	DB      *gorm.DB
	Package *tm.PackageConfig
}

func WithToolMarket(d ToolMarketDependencies) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.toolMarketConfigs++; o.toolMarket = &d }
}

type toolMarketModule struct{ routes []httproute.Descriptor }

func (toolMarketModule) Name() string                    { return tmhttp.ModuleName }
func (m toolMarketModule) Enabled(c *config.Config) bool { return c != nil && c.Workbench.Enabled }
func (m toolMarketModule) Register(r *kernelmodule.Registry) error {
	r.AddRoutes(m.routes...)
	return nil
}

func buildToolMarket(ctx context.Context, d ToolMarketDependencies, deps routeAuthDependencies, a *authz.ListingKitAuthorizer, readiness tm.Readiness) (kernelmodule.Module, error) {
	if d.DB == nil {
		return nil, tm.ErrUnavailable
	}
	var safe bool
	if err := d.DB.WithContext(ctx).Raw(`SELECT current_user='tool_market_runtime' AND NOT rolsuper AND NOT rolcreatedb AND NOT rolcreaterole AND NOT rolbypassrls AND NOT pg_has_role(current_user,'tool_market_owner','MEMBER') AND NOT has_schema_privilege(current_user,'tool_market','CREATE') FROM pg_roles WHERE rolname=current_user`).Scan(&safe).Error; err != nil || !safe {
		return nil, errors.New("tool market requires its restricted serving role")
	}
	if err := tmstore.VerifySchema(ctx, d.DB); err != nil {
		return nil, err
	}
	repo, err := tmstore.New(d.DB)
	if err != nil {
		return nil, err
	}
	admission, err := toolmarketauth.New(deps.organizationResolver, a)
	if err != nil {
		return nil, err
	}
	h := &tmhttp.Handler{Repository: repo, Bind: admission.Bind, ReadAuthorize: admission.AuthorizeRead, Authorize: admission.Authorize, Readiness: readiness}
	if d.Package != nil {
		_ = h.ConfigurePackage(*d.Package)
	}
	routes, err := tmhttp.BuildRoutes(h)
	if err != nil {
		return nil, err
	}
	return toolMarketModule{routes: routes}, nil
}
