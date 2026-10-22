package httpapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"task-processor/internal/core/config"
	"task-processor/internal/httproute"
	kernelmodule "task-processor/internal/kernel/module"
)

func TestCoreHTTPModuleRegistersHealthRoute(t *testing.T) {
	t.Parallel()

	reg := kernelmodule.NewRegistry()

	err := newCoreHTTPModule().Register(reg)
	require.NoError(t, err)

	routes := reg.Routes()
	require.Len(t, routes, 1)
	require.Equal(t, http.MethodGet, routes[0].Method)
	require.Equal(t, "/health", routes[0].Path)
	require.Equal(t, "system", routes[0].Module)
}

func TestHTTPModuleRegisterRejectsNilRegistrar(t *testing.T) {
	t.Parallel()

	err := (httpModule{name: "broken"}).Register(kernelmodule.NewRegistry())
	require.EqualError(t, err, "http module broken has no registrar")
}

func TestBuildHTTPServerBundleFromModulesSkipsDisabledModules(t *testing.T) {
	enabledRoute := httproute.Descriptor{
		Method: http.MethodGet,
		Path:   "/enabled",
		Module: "enabled",
		Handler: func(c *gin.Context) {
			c.Status(http.StatusOK)
		},
	}
	disabledRoute := httproute.Descriptor{
		Method: http.MethodGet,
		Path:   "/disabled",
		Module: "disabled",
		Handler: func(c *gin.Context) {
			c.Status(http.StatusTeapot)
		},
	}

	server, routes, err := buildHTTPServerBundleFromModules(18080, &config.Config{}, []kernelmodule.Module{
		httpModule{
			name:     "enabled",
			register: func(reg *kernelmodule.Registry) error { reg.AddRoutes(enabledRoute); return nil },
		},
		httpModule{
			name: "disabled",
			enabled: func(*config.Config) bool {
				return false
			},
			register: func(reg *kernelmodule.Registry) error { reg.AddRoutes(disabledRoute); return nil },
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"GET /enabled"}, routeKeys(routes))

	router, ok := server.Handler.(*gin.Engine)
	require.True(t, ok)

	req := httptest.NewRequest(http.MethodGet, "/enabled", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	req = httptest.NewRequest(http.MethodGet, "/disabled", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusNotFound, resp.Code)
}

func TestBuildHTTPServerBundleFromModulesSkipsNilModules(t *testing.T) {
	t.Parallel()

	server, routes, err := buildHTTPServerBundleFromModules(18080, &config.Config{}, []kernelmodule.Module{
		nil,
		httpModule{
			name: "enabled",
			register: func(reg *kernelmodule.Registry) error {
				reg.AddRoutes(httproute.Descriptor{
					Method: http.MethodGet,
					Path:   "/enabled",
					Module: "enabled",
					Handler: func(c *gin.Context) {
						c.Status(http.StatusOK)
					},
				})
				return nil
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"GET /enabled"}, routeKeys(routes))

	router, ok := server.Handler.(*gin.Engine)
	require.True(t, ok)

	req := httptest.NewRequest(http.MethodGet, "/enabled", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestBuildRuntimeBundleFromModulesCollectsRoutes(t *testing.T) {
	t.Parallel()

	bundle, err := buildRuntimeBundleFromModules(&config.Config{}, []kernelmodule.Module{
		httpModule{
			name: "custom",
			register: func(reg *kernelmodule.Registry) error {
				reg.AddRoutes(httproute.Descriptor{
					Method: http.MethodGet,
					Path:   "/health",
					Module: "custom",
					Handler: func(c *gin.Context) {
						c.Status(http.StatusOK)
					},
				})
				return nil
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"GET /health"}, routeKeys(bundle.routes))
}

func routeKeys(routes []httproute.Descriptor) []string {
	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, fmt.Sprintf("%s %s", route.Method, route.Path))
	}
	return keys
}
