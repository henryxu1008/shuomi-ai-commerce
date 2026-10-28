package membership

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
)

func TestBrowserAdministratorProjectionSeparatesFixedRoleFromExplicitGrants(t *testing.T) {
	key := authz.EnterpriseRoleKey("effective-b", 3)
	roles := roleStoreFixture{[]RoleDefinition{{ID: key, Name: "浏览器使用", Modules: []string{"store-browser"}, Version: 1}}}
	a, err := authz.NewListingKitAuthorizer([]string{"actor"}, nil)
	require.NoError(t, err)
	a.SetRolePolicyReader(roles)
	d := &directoryStub{page: Page{Total: 3, Items: []Member{
		{ID: "admin", UserID: "admin-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin"}, State: "active"},
		{ID: "browser", UserID: "browser-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{key}, State: "active"},
		{ID: "protected", UserID: "protected-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin", "platform_admin", key}, State: "active"},
	}}}
	s := NewService(d, a, "project")
	s.SetRoleStore(roles)
	ctx := scopedContext("listingkit_admin")
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	require.True(t, ok)
	identity.Roles = append(identity.Roles, key)
	identity.OrganizationGrants[0].Roles = append([]string{}, identity.Roles...)
	ctx = authidentity.WithAuthenticatedIdentity(ctx, identity)
	catalog, err := s.Roles(ctx)
	require.NoError(t, err)
	members, err := s.List(ctx, PageRequest{Limit: 20})
	require.NoError(t, err)
	detail, err := s.Read(ctx, "admin")
	require.NoError(t, err)
	require.Equal(t, catalog.Items[0], members.RoleDefinitions[0])
	require.Equal(t, catalog.Items[0], detail.RoleDefinitions[0])
	require.Contains(t, catalog.Items[0].Modules, "stores")
	require.NotContains(t, catalog.Items[0].Modules, "store-browser")
	require.NotContains(t, members.Items[0].Permissions, authz.PermissionWorkbenchStoreBrowserUse)
	require.Contains(t, members.Items[1].Permissions, authz.PermissionWorkbenchStoreBrowserUse)
	require.NotContains(t, members.Items[1].Permissions, authz.PermissionWorkbenchStoreUpdate)
	require.Contains(t, members.Items[2].Permissions, authz.PermissionWorkbenchStoreBrowserUse)
	require.False(t, members.Items[2].CanChangeRole)
	require.False(t, members.Items[2].CanRemove)
	available := false
	for _, module := range catalog.Catalog {
		if module.ID == "store-browser" {
			available = module.Available
		}
	}
	require.True(t, available, "module must remain available for explicit enterprise roles")
}
