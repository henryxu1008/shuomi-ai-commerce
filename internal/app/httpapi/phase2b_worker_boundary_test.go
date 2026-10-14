package httpapi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLegacyBuildHandlersFacadeStaysRetired(t *testing.T) {
	t.Parallel()

	_, err := os.Stat("handlers_legacy.go")
	require.Truef(t, os.IsNotExist(err), "handlers_legacy.go still exists; use module runtime bootstrap instead of the legacy handler facade")
}

func TestHTTPAPIServerDoesNotOwnListingKitAuthMiddlewareSelection(t *testing.T) {
	t.Parallel()

	serverSrc, err := os.ReadFile("server.go")
	require.NoError(t, err)
	serverContent := string(serverSrc)

	for _, marker := range []string{
		`"task-processor/internal/authz/httpapi"`,
		"NewZitadelAuthMiddlewareFromEnv(",
		"RouteRequiresZitadelAuth(",
		"NewRouteRoleMiddleware(",
	} {
		require.NotContains(t, serverContent, marker)
	}

	authSrc, err := os.ReadFile("server_auth.go")
	require.NoError(t, err)
	authContent := string(authSrc)
	for _, marker := range []string{
		`"task-processor/internal/authz/httpapi"`,
		"zitadelruntime.NewMiddleware(",
		"RouteRequiresZitadelAuth(",
		"NewRouteRoleMiddlewareWithAuthorizer(",
	} {
		require.Contains(t, authContent, marker)
	}
	for _, forbidden := range []string{
		"ConfigureListingKitZitadelAuth(",
		"ConfigureListingKitAuthorization(",
		"NewZitadelAuthMiddlewareFromEnv(",
		"NewRouteRoleMiddleware(route)",
	} {
		require.NotContains(t, authContent, forbidden)
	}
}

func TestHTTPModulesDoNotOwnFeatureHTTPAPIModuleSelectionSignatures(t *testing.T) {
	t.Parallel()

	modulesSrc, err := os.ReadFile("http_modules.go")
	require.NoError(t, err)
	modulesContent := string(modulesSrc)

	for _, marker := range []string{
		"built *productenrichhttpapi.Module",
		"imageBuilt *productimagehttpapi.Module",
		"built *amazonlistinghttpapi.Module",
		"built *listingkithttpapi.Module",
		"built *productModuleResult",
		"imageBuilt *imageModuleResult",
		"built *amazonListingModuleResult",
		"built *listingKitModuleResult",
	} {
		require.NotContains(t, modulesContent, marker)
	}
}
