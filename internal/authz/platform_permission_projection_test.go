package authz

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// Keep the narrow interface path under test as well as the scoped batch reader.
type staticPermissionReader struct{ StaticAuthorizer }

func TestPlatformPermissionReadbackMatchesExistingAuthority(t *testing.T) {
	authorizer, err := NewListingKitAuthorizer([]string{"configured-user"}, []string{"configured-role"})
	require.NoError(t, err)
	for _, tc := range []struct {
		name, user, role string
		want             bool
	}{
		{"platform role", "member", "platform_admin", true},
		{"configured role", "member", "configured-role", true},
		{"configured user", "configured-user", "listingkit_admin", true},
		{"enterprise admin", "member", "listingkit_admin", false},
		{"operator", "member", "listingkit_operator", false},
		{"viewer", "member", "listingkit_viewer", false},
		{"unconfigured user", "other-user", "", false},
		{"unconfigured role", "member", "other-role", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roles := []string{tc.role}
			require.Equal(t, tc.want, authorizer.Authorize(tc.user, roles, PermissionListingKitPlatformAdm))
			for name, reader := range map[string]StaticAuthorizer{
				"scoped batch":     authorizer,
				"static interface": staticPermissionReader{authorizer},
			} {
				t.Run(name, func(t *testing.T) {
					permissions, err := PermissionsInOrganization(context.Background(), reader, tc.user, "org-a", roles)
					require.NoError(t, err)
					require.Equal(t, tc.want, slices.Contains(permissions, PermissionListingKitPlatformAdm))
				})
			}
		})
	}
}

func TestPlatformPermissionReadbackDoesNotGrantEnterpriseModulesOrFailedPolicy(t *testing.T) {
	authorizer, err := NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	modules := make([]string, 0, len(enterpriseModules))
	for _, module := range enterpriseModules {
		modules = append(modules, module.ID)
	}
	key := EnterpriseRoleKey("org-a", 1)
	authorizer.SetRolePolicyReader(rolePolicyFixture{"org-a": {key: modules}})
	for _, organization := range []string{"org-a", "org-b"} {
		permissions, err := PermissionsInOrganization(context.Background(), authorizer, "member", organization, []string{key})
		require.NoError(t, err)
		require.NotContains(t, permissions, PermissionListingKitPlatformAdm, "all enterprise modules still cannot grant platform maintenance")
	}
	authorizer.SetRolePolicyReader(failedRolePolicy{})
	permissions, err := PermissionsInOrganization(context.Background(), authorizer, "member", "org-a", []string{key})
	require.ErrorIs(t, err, ErrRolePolicyUnavailable)
	require.Empty(t, permissions)
}
