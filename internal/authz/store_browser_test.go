package authz

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStoreBrowserIsExplicitModuleAndDoesNotRequireStoreUpdate(t *testing.T) {
	a, err := NewListingKitAuthorizer([]string{"protected-user"}, []string{"protected-role"})
	require.NoError(t, err)
	for _, role := range []string{"listingkit_admin", "listingkit_operator", "platform_admin", "protected-role"} {
		require.False(t, a.Authorize("member", []string{role}, PermissionWorkbenchStoreBrowserUse), role)
	}
	require.False(t, a.Authorize("protected-user", nil, PermissionWorkbenchStoreBrowserUse))
	require.NotContains(t, ModulePermissions([]string{"stores"}), PermissionWorkbenchStoreBrowserUse)
	require.Contains(t, ModulePermissions([]string{"store-browser"}), PermissionWorkbenchStoreRead)
	require.NotContains(t, ModulePermissions([]string{"store-browser"}), PermissionWorkbenchStoreUpdate)
	key := EnterpriseRoleKey("org-a", 1)
	policy := rolePolicyFixture{"org-a": {key: {"store-browser"}}}
	a.SetRolePolicyReader(policy)
	allowed, err := a.AuthorizeScoped(context.Background(), "member", "org-a", []string{key}, PermissionWorkbenchStoreBrowserUse)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = a.AuthorizeScoped(context.Background(), "member", "org-a", []string{key}, PermissionWorkbenchStoreUpdate)
	require.NoError(t, err)
	require.False(t, allowed)
	policy["org-a"][key] = []string{}
	allowed, err = a.AuthorizeScoped(context.Background(), "member", "org-a", []string{key}, PermissionWorkbenchStoreBrowserUse)
	require.NoError(t, err)
	require.False(t, allowed)
	a.SetRolePolicyReader(failedRolePolicy{})
	allowed, err = a.AuthorizeScoped(context.Background(), "member", "org-a", []string{key}, PermissionWorkbenchStoreBrowserUse)
	require.Error(t, err)
	require.False(t, allowed)
}
