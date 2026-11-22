package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	capacitystore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/storagecapacity"
	"gorm.io/gorm"

	zitadelruntime "github.com/henryxu1008/shuomi-ai-commerce/internal/authruntime/zitadel"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authz"
	billing "github.com/henryxu1008/shuomi-ai-commerce/internal/commercial/billing"
	billinghttp "github.com/henryxu1008/shuomi-ai-commerce/internal/commercial/billing/httpapi"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	orgresourceadapter "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/orgresource"
	commercialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/commercialbilling"
	moneystore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/money"
	kernelmodule "github.com/henryxu1008/shuomi-ai-commerce/internal/kernel/module"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/orgresource"
)

func buildCommercialBillingModule(ctx context.Context, commercialDB, moneyDB *gorm.DB, authorizer *authz.ListingKitAuthorizer, cfg *config.Config) (kernelmodule.Module, error) {
	if ctx == nil || commercialDB == nil || moneyDB == nil || authorizer == nil || cfg == nil {
		return nil, errors.New("commercial billing or canonical money database unavailable")
	}
	if cfg.WalletTopUp.Alipay.Enabled || cfg.WalletTopUp.WeChat.Enabled {
		if err := moneystore.VerifyProviderTopUpRuntime(ctx, moneyDB); err != nil {
			return nil, errors.New("canonical money top-up runtime privileges unavailable")
		}
	}
	wallet, err := moneystore.New(moneyDB)
	if err != nil {
		return nil, err
	}
	commercial, err := commercialstore.New(commercialDB)
	if err != nil {
		return nil, err
	}
	resourceRepository, err := orgresourceadapter.NewGormRepository(commercialDB, orgresourceadapter.TransactionConfig{})
	if err != nil {
		return nil, err
	}
	grantService, err := orgresource.NewPurchasedResourceGrantService(resourceRepository, orgresourceadapter.TrustedCommercialGrantAuthorizer{})
	if err != nil {
		return nil, err
	}
	service, err := billing.NewService(commercial, commercial, commercial, commercial, wallet, grantService)
	if err != nil {
		return nil, err
	}
	module := commercialBillingModule{handler: billinghttp.NewHandler(service), pricing: dataRowPricingModule{store: commercial, authorizer: authorizer}}
	if cfg.StorageCapacity {
		capacity, e := capacitystore.New(commercialDB)
		if e != nil {
			return nil, e
		}
		if e = capacitystore.VerifyRuntime(ctx, commercialDB); e != nil {
			return nil, errors.New("storage capacity runtime tables or privileges unavailable")
		}
		service.ConfigureStorage(commercial)
		module.handler.ConfigureStorage(capacity)
	}
	zitadel := cfg.ListingKit.Zitadel
	if (cfg.WalletTopUp.Alipay.Enabled || cfg.WalletTopUp.WeChat.Enabled) && (strings.TrimSpace(zitadel.TenantDirectoryToken) == "" || strings.TrimSpace(zitadel.AuthorizationAPIURL) == "" || strings.TrimSpace(zitadel.ProjectID) == "") {
		return nil, errors.New("financial recovery directory authorization configuration unavailable")
	}
	recoveryAuthorizer := financialRecoveryAuthorizer{reader: zitadelruntime.NewAuthorizationClient(zitadel.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second}), serviceToken: zitadel.TenantDirectoryToken, projectID: zitadel.ProjectID, authorizer: authorizer}
	module.reconcileResources = func(run context.Context) error { return service.ReconcileRecoverableResourceOrders(run, 50) }
	if err := configureWalletTopUps(ctx, service, module.handler, commercial, wallet, cfg.WalletTopUp, topUpRuntimeAuthorizer{directory: recoveryAuthorizer}); err != nil {
		return nil, err
	}
	module.reconcileTopUps = func(run context.Context) error { return service.ReconcileRecoverableTopUps(run) }
	return module, nil
}

type commercialBillingModule struct {
	pricing            dataRowPricingModule
	handler            *billinghttp.Handler
	reconcileResources func(context.Context) error
	reconcileTopUps    func(context.Context) error
}

func (commercialBillingModule) Name() string { return "commercial-billing" }

func (m commercialBillingModule) Enabled(cfg *config.Config) bool {
	return m.handler != nil && cfg != nil && cfg.Workbench.Enabled
}

func (m commercialBillingModule) Register(registry *kernelmodule.Registry) error {
	routes, err := billinghttp.Routes(m.handler)
	if err != nil {
		return err
	}
	registry.AddRoutes(routes...)
	registry.AddRoutes(m.pricing.routes()...)
	return nil
}
