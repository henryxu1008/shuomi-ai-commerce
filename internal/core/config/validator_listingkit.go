package config

// ValidateListingKitConfig rejects obsolete identity configuration while the
// deprecated field remains loadable solely so legacy YAML and env inputs cannot
// be silently ignored.
func ValidateListingKitConfig(listingKit *ListingKitConfig) []error {
	if listingKit == nil || (!listingKit.Zitadel.LegacyUsernameAllowlistConfigured && len(listingKit.Zitadel.AllowedUsernames) == 0) {
		return nil
	}

	return []error{&ValidationError{
		Field:   "listingkit.zitadel.allowedUsernames",
		Message: "obsolete ZITADEL username allowlist configuration is not supported",
		Hint:    "remove it and use current identity and organization permissions; configure platform administrators with platformAdminUsers or platformAdminRoles",
	}}
}
