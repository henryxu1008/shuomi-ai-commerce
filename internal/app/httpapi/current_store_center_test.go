package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/storecenter"
	storehttp "task-processor/internal/storecenter/httpapi"
)

func TestTikTokLifecycleAdmissionRequiresExplicitSignedBoundary(t *testing.T) {
	var base []httproute.Descriptor
	for _, route := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: route.Method, Path: route.Path})
	}
	base = append(base, currentStoreTestRoutes(t)...)
	callback := httproute.Descriptor{Method: http.MethodPost, Path: storehttp.TikTokLifecyclePath, Module: storehttp.ModuleName, AuthPolicy: httproute.AuthPolicyPublic, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyNone, RequestTimeout: 5 * time.Second, Handler: func(*gin.Context) {}}
	validate := func(routes []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{StoreCenter: true, TikTokShop: enabled})
	}
	all := append(base, callback)
	require.NoError(t, validate(all, true))
	require.Error(t, validate(all, false))
	require.Error(t, validate(base, true))
	for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyCurrentIdentity }, func(d *httproute.Descriptor) {
		d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyLiveWrite
	}, func(d *httproute.Descriptor) { d.RequestTimeout = 0 }, func(d *httproute.Descriptor) { d.Permission = "store.update" }, func(d *httproute.Descriptor) { d.Handler = nil }} {
		changed := append([]httproute.Descriptor(nil), all...)
		mutate(&changed[len(changed)-1])
		require.Error(t, validate(changed, true))
	}
}

func TestCurrentStoreAdmissionUsesRegisteredOfficialRefresh(t *testing.T) {
	h, err := storehttp.NewHandlerWithServiceLifecycle(inertStoreService{}, &storecenter.ServiceLifecycleApplication{})
	require.NoError(t, err)
	require.NoError(t, h.SetOfficialConnections(&storeapp.OfficialConnections{}))
	m := storehttp.NewModule(h)
	m.(interface {
		ConfigureCatalog(*storecenter.Catalog, *storecenter.MetadataRepository)
	}).ConfigureCatalog(&storecenter.Catalog{}, &storecenter.MetadataRepository{})
	require.NoError(t, m.(interface {
		ConfigureBrowser(*storecenter.BrowserOperations) error
	}).ConfigureBrowser(&storecenter.BrowserOperations{}))
	reg := kernelmodule.NewRegistry()
	require.NoError(t, m.Register(reg))
	var base []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	all := append(base, reg.Routes()...)
	validate := func(routes []httproute.Descriptor) error {
		return validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{StoreCenter: true})
	}
	require.NoError(t, validate(all))
	found := false
	for i, r := range all {
		if r.Path != "/api/v1/workbench/stores/:store_id/connection/refresh" {
			continue
		}
		found = true
		for _, mutate := range []func(*httproute.Descriptor){func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyPublic }, func(d *httproute.Descriptor) { d.Permission = authz.PermissionWorkbenchStoreRead }, func(d *httproute.Descriptor) {
			d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
		}, func(d *httproute.Descriptor) { d.Handler = nil }} {
			changed := append([]httproute.Descriptor(nil), all...)
			mutate(&changed[i])
			require.Error(t, validate(changed))
		}
	}
	require.True(t, found, "test must consume the real official refresh route")
}

func currentStoreTestRoutes(t *testing.T) []httproute.Descriptor {
	t.Helper()
	m := currentStoreNativeTestModule(t)
	require.NoError(t, m.(interface {
		ConfigureBrowser(*storecenter.BrowserOperations) error
	}).ConfigureBrowser(&storecenter.BrowserOperations{}))
	registry := kernelmodule.NewRegistry()
	require.NoError(t, m.Register(registry))
	return registry.Routes()
}

// Use the production HTTP module so buildCurrentApplication must perform its
// ConfigureBrowser injection, instead of feeding the admission list back to it.
func currentStoreNativeTestModule(t *testing.T) kernelmodule.Module {
	t.Helper()
	h, err := storehttp.NewHandlerWithServiceLifecycle(inertStoreService{}, &storecenter.ServiceLifecycleApplication{})
	require.NoError(t, err)
	require.NoError(t, h.SetOfficialConnections(&storeapp.OfficialConnections{}))
	m := storehttp.NewModule(h)
	m.(interface {
		ConfigureCatalog(*storecenter.Catalog, *storecenter.MetadataRepository)
	}).ConfigureCatalog(&storecenter.Catalog{}, &storecenter.MetadataRepository{})
	return m
}

func TestCurrentStoreRouteAdmissionRejectsDrift(t *testing.T) {
	var base []httproute.Descriptor
	for _, r := range currentWorkbenchApplicationRoutes {
		base = append(base, httproute.Descriptor{Method: r.Method, Path: r.Path})
	}
	stores := currentStoreTestRoutes(t)
	all := append(base, stores...)
	validate := func(routes []httproute.Descriptor, enabled bool) error {
		return validateCurrentApplicationRoutesInternal(routes, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{StoreCenter: enabled})
	}
	require.NoError(t, validate(base, false))
	require.Error(t, validate(base, true))
	require.Error(t, validate(all, false))
	require.NoError(t, validate(all, true))
	for i := range stores {
		for _, mutate := range []func(*httproute.Descriptor){
			func(d *httproute.Descriptor) { d.AuthPolicy = httproute.AuthPolicyVerifiedIdentity },
			func(d *httproute.Descriptor) {
				d.OrganizationAccessPolicy = httproute.OrganizationAccessPolicyCachedRead
			},
			func(d *httproute.Descriptor) { d.Permission = authz.PermissionWorkbenchStoreRead + "-drift" },
			func(d *httproute.Descriptor) { d.Handler = nil },
		} {
			changed := append([]httproute.Descriptor(nil), all...)
			mutate(&changed[len(base)+i])
			require.Error(t, validate(changed, true))
		}
	}
	for _, action := range []string{"activate", "renew", "reactivate"} {
		require.Error(t, validate(append(append([]httproute.Descriptor(nil), all...), httproute.Descriptor{Method: http.MethodPost, Path: "/api/v1/workbench/stores/:store_id/" + action, Module: storehttp.ModuleName, Handler: func(*gin.Context) {}}), true))
	}
}

func TestCurrentStoreBrowserInventoryRejectsMissingOrDuplicateRoutes(t *testing.T) {
	var routes []httproute.Descriptor
	for _, route := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: route.Method, Path: route.Path})
	}
	routes = append(routes, currentStoreTestRoutes(t)...)
	validate := func(got []httproute.Descriptor) error {
		return validateCurrentApplicationRoutesInternal(got, false, false, false, false, false, false, false, currentApplicationOptionalRoutes{StoreCenter: true})
	}
	require.NoError(t, validate(routes))
	var browserIndexes []int
	for _, expected := range currentStoreBrowserRoutes {
		for index, route := range routes {
			if route.Method == expected.method && route.Path == expected.path {
				browserIndexes = append(browserIndexes, index)
			}
		}
	}
	require.Len(t, browserIndexes, 4)
	for i, index := range browserIndexes {
		t.Run(routes[index].Method+routes[index].Path, func(t *testing.T) {
			missing := append([]httproute.Descriptor(nil), routes[:index]...)
			missing = append(missing, routes[index+1:]...)
			require.Error(t, validate(missing), "all four contracted routes are mandatory")
			duplicate := append([]httproute.Descriptor(nil), routes...)
			duplicate[index] = routes[browserIndexes[(i+1)%len(browserIndexes)]]
			require.Error(t, validate(duplicate), "equal route count cannot hide a duplicate and missing route")
		})
	}
}

func TestCurrentStoreOptionalAssembly(t *testing.T) {
	for _, mode := range []string{"disabled", "enabled", "nil-records", "shared", "missing-owner", "duplicate", "missing-factory"} {
		t.Run(mode, func(t *testing.T) {
			deps := newRouteAuthDependencies()
			calls := 0
			source, owner, records := &gorm.DB{}, emptyStorageCommercialOwner(t), &gorm.DB{}
			factories := currentApplicationFactories{
				buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
					return workbenchContextBuildResult{module: currentApplicationTestModule{name: "base", routes: currentWorkbenchApplicationRoutes}, authDependencies: &deps}, nil
				},
				buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
				buildCommercial:    func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
				buildResourceCharges: func(context.Context, *gorm.DB, *gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer, storecenter.RuntimeCapabilities) (*orgresource.ConsumerChargeService, error) {
					return orgresource.NewConsumerChargeService(currentStoreChargeFixture{}, map[orgresource.ResourceConsumer]orgresource.ConsumerChargeOwner{orgresource.ConsumerStoreService: currentStoreChargeFixture{}})
				},
				buildStoreCenter: func(_ context.Context, r *gorm.DB, _ *authz.ListingKitAuthorizer, charges orgresource.ConsumerChargePort, _ *storeapp.OfficialApplicationRegistry, _ storecenter.RuntimeCapabilities) (kernelmodule.Module, error) {
					calls++
					require.Same(t, records, r)
					require.NotNil(t, charges)
					return currentStoreNativeTestModule(t), nil
				},
			}
			options := []CurrentApplicationOption{WithCommercialOwnerDatabase(owner)}
			switch mode {
			case "nil-records":
				records = nil
			case "shared":
				records = source
			case "missing-owner":
				options = nil
			case "missing-factory":
				factories.buildStoreCenter = nil
			}
			if mode != "disabled" {
				options = append(options, WithStoreCenter(records))
			}
			if mode == "duplicate" {
				options = append(options, WithStoreCenter(records))
			}
			cfg := currentApplicationTestConfig()
			cfg.ListingKit.Zitadel.IssuerURL = "https://issuer.example"
			server, err := buildCurrentApplication(context.Background(), source, cfg, logrus.New(), factories, options...)
			if mode == "enabled" || mode == "disabled" {
				require.NoError(t, err)
				require.NotNil(t, server)
				if mode == "enabled" {
					response := httptest.NewRecorder()
					server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/workbench/stores/11111111-1111-4111-8111-111111111111/browser-operations/22222222-2222-4222-8222-222222222222", nil))
					require.Equal(t, http.StatusUnauthorized, response.Code)
					require.Contains(t, response.Header().Get("Content-Type"), "application/json")
				}
			} else {
				require.Error(t, err)
				require.Nil(t, server)
			}
			if mode == "enabled" {
				require.Equal(t, 1, calls)
			} else {
				require.Zero(t, calls)
			}
		})
	}
}

type currentStoreRouteFixture struct{ routes []httproute.Descriptor }

func (currentStoreRouteFixture) Name() string                { return storehttp.ModuleName }
func (currentStoreRouteFixture) Enabled(*config.Config) bool { return true }
func (m currentStoreRouteFixture) Register(reg *kernelmodule.Registry) error {
	reg.AddRoutes(m.routes...)
	return nil
}

type currentStoreChargeFixture struct{}

func (currentStoreChargeFixture) Reserve(context.Context, orgresource.ConsumerChargeIntent) (orgresource.ConsumerChargeReceipt, error) {
	return orgresource.ConsumerChargeReceipt{}, orgresource.ErrInvalidInput
}
func (currentStoreChargeFixture) Read(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeReceipt, error) {
	return orgresource.ConsumerChargeReceipt{}, orgresource.ErrInvalidInput
}
func (currentStoreChargeFixture) Settle(context.Context, orgresource.ConsumerChargeReceipt, orgresource.ConsumerChargeProof) (orgresource.ConsumerChargeReceipt, error) {
	return orgresource.ConsumerChargeReceipt{}, orgresource.ErrInvalidInput
}
func (currentStoreChargeFixture) ClaimDue(context.Context, []orgresource.ResourceConsumer) ([]orgresource.ConsumerChargeIdentity, error) {
	return nil, nil
}
func (currentStoreChargeFixture) ReadChargeIntent(context.Context, orgresource.ConsumerChargeIdentity) (orgresource.ConsumerChargeIntent, error) {
	return orgresource.ConsumerChargeIntent{}, orgresource.ErrInvalidInput
}
func (currentStoreChargeFixture) ReadChargeProof(context.Context, orgresource.ConsumerChargeReceipt) (orgresource.ConsumerChargeProof, error) {
	return orgresource.ConsumerChargeProof{}, orgresource.ErrInvalidInput
}
