package membership

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/authz"
)

func TestAdministratorProjectionIsConsistentAcrossViewersAndMemberQueries(t *testing.T) {
	key := authz.EnterpriseRoleKey("effective-b", 3)
	roles := roleStoreFixture{[]RoleDefinition{{ID: key, Name: "查看成员", Modules: []string{"members"}, Version: 1}}}
	a, err := authz.NewListingKitAuthorizer([]string{"actor", "protected-member"}, nil)
	require.NoError(t, err)
	a.SetRolePolicyReader(roles)
	d := &directoryStub{page: Page{Total: 3, Items: []Member{
		{ID: "admin", UserID: "admin-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin"}, State: "active"},
		{ID: "custom", UserID: "custom-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{key}, State: "active"},
		{ID: "protected", UserID: "protected-member", OrganizationID: "effective-b", ProjectID: "project", Roles: []string{"listingkit_admin", key}, State: "active"},
	}}}
	s := NewService(d, a, "project")
	s.SetRoleStore(roles)
	var fixed RoleDefinition
	for _, viewer := range []string{"listingkit_admin", key} {
		t.Run(viewer, func(t *testing.T) {
			ctx := scopedContext(viewer)
			catalog, err := s.Roles(ctx)
			require.NoError(t, err)
			members, err := s.List(ctx, PageRequest{Limit: 20})
			require.NoError(t, err)
			detail, err := s.Read(ctx, "admin")
			require.NoError(t, err)
			require.Equal(t, catalog.Items[0], members.RoleDefinitions[0])
			require.Equal(t, catalog.Items[0], detail.RoleDefinitions[0])
			require.Contains(t, catalog.Items[0].Modules, "stores")
			if fixed.ID == "" {
				fixed = catalog.Items[0]
			}
			require.Equal(t, fixed, catalog.Items[0])
			require.Equal(t, roles.roles[0], catalog.Items[1])
			require.NotContains(t, members.Items[1].Permissions, authz.PermissionWorkbenchStoreCreate)
			require.Contains(t, members.Items[1].Permissions, authz.PermissionWorkbenchOrganizationMemberRead)
			require.False(t, members.Items[2].CanChangeRole)
			require.False(t, members.Items[2].CanRemove)
		})
	}
}
