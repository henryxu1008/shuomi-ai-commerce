package module

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"task-processor/internal/httproute"
)

func TestRegistryAddRoutesPreservesOrder(t *testing.T) {
	reg := NewRegistry()
	handler := func(c *gin.Context) {}

	reg.AddRoutes(
		httproute.Descriptor{Method: http.MethodGet, Path: "/health", Module: "system", Handler: handler},
		httproute.Descriptor{Method: http.MethodPost, Path: "/jobs", Module: "jobs", Handler: handler},
	)

	routes := reg.Routes()
	require.Len(t, routes, 2)
	require.Equal(t, "/health", routes[0].Path)
	require.Equal(t, "/jobs", routes[1].Path)
}

func TestRegistryRoutesReturnsDefensiveCopy(t *testing.T) {
	reg := NewRegistry()
	handler := func(c *gin.Context) {}

	reg.AddRoutes(
		httproute.Descriptor{Method: http.MethodGet, Path: "/health", Module: "system", Handler: handler},
	)

	routes := reg.Routes()
	routes[0].Path = "/mutated"

	again := reg.Routes()
	require.Equal(t, "/health", again[0].Path)
}
