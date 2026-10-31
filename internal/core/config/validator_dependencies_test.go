package config

import (
	"testing"
)

func TestValidateConfigPreservesValidationGroupOrder(t *testing.T) {
	cfg := &Config{
		ListingKit: ListingKitConfig{Zitadel: ListingKitZitadelConfig{
			AllowedUsernames: []string{"synthetic-obsolete-user"},
		}},
		Workbench: WorkbenchConfig{Enabled: true},
	}

	errors := ValidateConfig(cfg)
	previous := -1
	for _, field := range []string{
		"listingkit.zitadel.allowedUsernames",
		"listingkit.zitadel.projectID",
	} {
		found := -1
		for index, err := range errors {
			if validationError, ok := err.(*ValidationError); ok && validationError.Field == field {
				found = index
				break
			}
		}
		if found == -1 {
			t.Fatalf("validation group field %s is missing from %v", field, errors)
		}
		if found <= previous {
			t.Fatalf("validation group field %s at %d must follow index %d", field, found, previous)
		}
		previous = found
	}
}
