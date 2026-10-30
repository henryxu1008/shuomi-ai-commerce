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

	bundle, err := buildRuntimeBundleFromModules(&config.Config{}, []kernelmodule.Module{
		runtimeBundleTestModule{
			name:     "enabled",
			register: func(reg *kernelmodule.Registry) error { reg.AddRoutes(enabledRoute); return nil },
		},
		runtimeBundleTestModule{
			name:     "disabled",
			disabled: true,
			register: func(reg *kernelmodule.Registry) error { reg.AddRoutes(disabledRoute); return nil },
		},
	})
	require.NoError(t, err)
	routes := bundle.routes
	server := buildCurrentApplicationHTTPServer(routes, routeAuthDependencies{})
	require.Equal(t, []string{"GET /enabled"}, routeKeys(routes))

	router := server.Handler

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

	bundle, err := buildRuntimeBundleFromModules(&config.Config{}, []kernelmodule.Module{
		nil,
		runtimeBundleTestModule{
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
	routes := bundle.routes
	server := buildCurrentApplicationHTTPServer(routes, routeAuthDependencies{})
	require.Equal(t, []string{"GET /enabled"}, routeKeys(routes))

	router := server.Handler

	req := httptest.NewRequest(http.MethodGet, "/enabled", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
}

func TestBuildRuntimeBundleFromModulesCollectsRoutes(t *testing.T) {
	t.Parallel()

	bundle, err := buildRuntimeBundleFromModules(&config.Config{}, []kernelmodule.Module{
		runtimeBundleTestModule{
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
	failure := fmt.Errorf("fixture registration failure")
	_, err = buildRuntimeBundleFromModules(&config.Config{}, []kernelmodule.Module{runtimeBundleTestModule{name: "broken", register: func(*kernelmodule.Registry) error { return failure }}})
	require.ErrorIs(t, err, failure)
}

func routeKeys(routes []httproute.Descriptor) []string {
	keys := make([]string, 0, len(routes))
	for _, route := range routes {
		keys = append(keys, fmt.Sprintf("%s %s", route.Method, route.Path))
	}
	return keys
}

// A test-only Module fixture; production composition uses each current domain module.
type runtimeBundleTestModule struct {
	name     string
	disabled bool
	register func(*kernelmodule.Registry) error
}

func (m runtimeBundleTestModule) Name() string                { return m.name }
func (m runtimeBundleTestModule) Enabled(*config.Config) bool { return !m.disabled }
func (m runtimeBundleTestModule) Register(reg *kernelmodule.Registry) error {
	if m.register != nil {
		return m.register(reg)
	}
	return nil
}
