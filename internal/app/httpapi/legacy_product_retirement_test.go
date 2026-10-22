package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"task-processor/internal/authz"
	"task-processor/internal/httproute"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"task-processor/internal/core/config"
	kernelmodule "task-processor/internal/kernel/module"
)

func currentRetirementTestApplication(t *testing.T) (*http.Server, []httproute.Descriptor) {
	t.Helper()
	dependencies := newRouteAuthDependencies()
	factories := currentApplicationFactories{
		buildWorkbench: func(*config.Config, *logrus.Logger) (workbenchContextBuildResult, error) {
			return workbenchContextBuildResult{module: currentApplicationTestModule{name: "workbench", routes: currentWorkbenchApplicationRoutes}, authDependencies: &dependencies}, nil
		},
		buildSourceAccount: func(*gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
		buildCommercial:    func(*gorm.DB, *gorm.DB, *authz.ListingKitAuthorizer) (kernelmodule.Module, error) { return nil, nil },
	}
	server, err := buildCurrentApplication(context.Background(), &gorm.DB{}, currentApplicationTestConfig(), logrus.New(), factories)
	if err != nil {
		t.Fatal(err)
	}
	routes := make([]httproute.Descriptor, 0, len(currentWorkbenchApplicationRoutes))
	for _, route := range currentWorkbenchApplicationRoutes {
		routes = append(routes, httproute.Descriptor{Method: route.Method, Path: route.Path})
	}
	return server, routes
}

func TestLegacyProductRoutesAreNotRegisteredByProductionComposition(t *testing.T) {
	server, routes := currentRetirementTestApplication(t)
	if len(routes) == 0 {
		t.Fatal("production composition registered no routes")
	}

	for _, route := range []struct {
		method string
		path   string
	}{
		{method: http.MethodPost, path: "/api/v1/products/generate"},
		{method: http.MethodGet, path: "/api/v1/products/tasks/task-1"},
		{method: http.MethodPost, path: "/api/v1/images/process"},
		{method: http.MethodGet, path: "/api/v1/images/tasks/task-1"},
		{method: http.MethodPost, path: "/api/v1/images/tasks/task-1/review"},
	} {
		request := httptest.NewRequest(route.method, route.path, nil)
		response := httptest.NewRecorder()
		server.Handler.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want 404", route.method, route.path, response.Code)
		}
	}
}

func TestProductionRuntimeRegistryHasNoLegacyProductModules(t *testing.T) {
	cfg := &config.Config{}
	modules := []kernelmodule.Module{newCoreHTTPModule()}
	_, err := buildRuntimeBundleFromModules(cfg, modules)
	if err != nil {
		t.Fatalf("build production runtime bundle: %v", err)
	}
	if err := validateNoLegacyProductRuntime(modules); err != nil {
		t.Fatal(err)
	}
	assertModuleRegistered(t, modules, "system")

	for _, name := range []string{"product", "product-image"} {
		mutatedModules := append(append([]kernelmodule.Module(nil), modules...), legacyProductMutationModule{name: name})
		if err := validateNoLegacyProductRuntime(mutatedModules); err == nil {
			t.Fatalf("legacy module mutation %q was not detected", name)
		}
	}
}

func validateNoLegacyProductRuntime(modules []kernelmodule.Module) error {
	for _, module := range modules {
		if module == nil {
			continue
		}
		if module.Name() == "product" || module.Name() == "product-image" {
			return fmt.Errorf("legacy runtime module %q is composed", module.Name())
		}
	}
	return nil
}

func assertModuleRegistered(t *testing.T, modules []kernelmodule.Module, name string) {
	t.Helper()
	for _, module := range modules {
		if module != nil && module.Name() == name {
			return
		}
	}
	t.Fatalf("production module %q is not registered", name)
}

type legacyProductMutationModule struct{ name string }

func (m legacyProductMutationModule) Name() string                        { return m.name }
func (legacyProductMutationModule) Enabled(*config.Config) bool           { return true }
func (legacyProductMutationModule) Register(*kernelmodule.Registry) error { return nil }
