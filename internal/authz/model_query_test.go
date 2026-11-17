package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelQueryPermissionRequiresExplicitEnterpriseGrant(t *testing.T) {
	const permission = "workbench.ai.connection.test"
	require.Contains(t, WorkbenchPermissions(), permission)
	a, err := NewListingKitAuthorizer([]string{"maintainer"}, nil)
	require.NoError(t, err)
	for _, role := range []string{"listingkit_admin", "platform_admin", "listingkit_operator", "listingkit_viewer"} {
		require.False(t, a.Authorize("ordinary", []string{role}, permission))
	}
	require.False(t, a.Authorize("maintainer", nil, permission))
	ctx := context.Background()
	role := EnterpriseRoleKey("org-a", 1)
	a.SetRolePolicyReader(rolePolicyFixture{"org-a": {role: {"ai-connection-test"}}})
	require.True(t, AllowedOrganization(ctx, a, "operator", "org-a", []string{role}, permission))
	require.False(t, AllowedOrganization(ctx, a, "operator", "org-b", []string{role}, permission))
	require.False(t, AllowedOrganization(ctx, a, "operator", "org-a", []string{role}, PermissionWorkbenchCredentialsManage))
}
