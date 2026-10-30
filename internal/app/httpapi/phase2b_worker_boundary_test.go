package httpapi

import (
	"os"
	"path/filepath"
	"strings"
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
		"workbenchAuthenticationMiddleware(dependencies.workbenchVerifier)",
		"RouteRequiresZitadelAuth(",
		"NewRouteRoleMiddlewareWithAuthorizer(",
	} {
		require.Contains(t, authContent, marker)
	}
	for _, forbidden := range []string{
		"zitadelruntime.NewMiddleware(",
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

	_, err := os.Lstat("http_modules.go")
	require.True(t, os.IsNotExist(err), "the retired assembly file must be absent")
	sources, err := filepath.Glob("*.go")
	require.NoError(t, err)
	require.NotEmpty(t, sources, "the retained HTTP owner must remain readable")
	var modulesContent strings.Builder
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		body, err := os.ReadFile(source)
		require.NoError(t, err)
		modulesContent.Write(body)
		modulesContent.WriteByte('\n')
	}

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
		require.NotContains(t, modulesContent.String(), marker)
	}
}
