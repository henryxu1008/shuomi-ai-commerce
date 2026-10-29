package authz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdministratorModulesRequireCompletePermissionCoverage(t *testing.T) {
	a, err := NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	removed, err := a.enforcer.RemovePolicy("listingkit_admin", PermissionWorkbenchStoreCreate)
	require.NoError(t, err)
	require.True(t, removed)
	catalog := append(MenuModules(),
		MenuModule{ID: "empty", Available: true},
		MenuModule{ID: "blank", Available: true, Permissions: []string{"", " "}},
		MenuModule{ID: "unavailable", Permissions: []string{PermissionWorkbenchStoreRead}},
		MenuModule{ID: "narrow", Available: true, Permissions: []string{PermissionWorkbenchStoreRead, "workbench.store.browser.use"}},
	)
	modules := a.fixedRoleModules("listingkit_admin", catalog)
	require.NotContains(t, modules, "stores", "one allowed permission is not the full module")
	require.NotContains(t, modules, "narrow", "new narrow capabilities require explicit policy")
	require.NotContains(t, modules, "empty")
	require.NotContains(t, modules, "blank")
	require.NotContains(t, modules, "unavailable")
	require.Contains(t, modules, "members")
}
