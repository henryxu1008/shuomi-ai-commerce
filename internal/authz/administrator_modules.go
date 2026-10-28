package authz

import "strings"

// AdministratorModules is display metadata from the fixed built-in policy.
// It must not authorize business requests or inherit actor/enterprise privileges.
func AdministratorModules() []string {
	return DefaultListingKitAuthorizer().fixedRoleModules("listingkit_admin", MenuModules())
}

func (a *ListingKitAuthorizer) fixedRoleModules(role string, catalog []MenuModule) []string {
	modules := []string{}
	for _, module := range catalog {
		if !module.Available {
			continue
		}
		complete, hasPermission := true, false
		for _, permission := range module.Permissions {
			if strings.TrimSpace(permission) == "" {
				continue
			}
			hasPermission = true
			if !a.Authorize("", []string{role}, permission) {
				complete = false
				break
			}
		}
		if complete && hasPermission {
			modules = append(modules, module.ID)
		}
	}
	return modules
}
