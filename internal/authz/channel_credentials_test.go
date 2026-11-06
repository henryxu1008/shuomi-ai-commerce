package authz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelCredentialManagementRequiresAnExplicitGrant(t *testing.T) {
	const permission = "workbench.ai.credentials.manage"
	require.Contains(t, WorkbenchPermissions(), permission, "the independent permission must be available to the existing permission projection")
	authorizer, err := NewListingKitAuthorizer([]string{"platform-maintainer"}, nil)
	require.NoError(t, err)
	for _, role := range []string{"admin", "listingkit_admin", "platform_admin", "listingkit_operator"} {
		require.False(t, authorizer.Authorize("ordinary", []string{role}, permission), "existing roles must not silently acquire Key management: %s", role)
	}
	require.False(t, authorizer.Authorize("platform-maintainer", nil, permission), "price maintenance does not confer enterprise Key management")
}
