package config

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValidationErrors_GroupsByModule(t *testing.T) {
	lines := formatValidationErrors([]error{
		&ValidationError{Field: "database.host", Message: "missing", Hint: "set TASK_PROCESSOR_DATABASE_HOST"},
		&ValidationError{Field: "listingkit.zitadel.projectID", Message: "missing"},
		&ValidationError{Field: "database.user", Message: "missing"},
	})

	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "database.host: missing")
	assert.Contains(t, joined, "hint: set TASK_PROCESSOR_DATABASE_HOST")
	assert.Contains(t, joined, "database.user: missing")
	assert.Contains(t, joined, "[General]")
	assert.Contains(t, joined, "listingkit.zitadel.projectID: missing")
}

func TestFormatValidationErrors_FallbacksToGeneral(t *testing.T) {
	lines := formatValidationErrors([]error{errors.New("plain failure")})
	joined := strings.Join(lines, "\n")
	assert.Contains(t, joined, "[General]")
	assert.Contains(t, joined, "plain failure")
}

func TestValidateConfigWithError_UsesGroupedFormat(t *testing.T) {
	cfg := &Config{ListingKit: ListingKitConfig{Zitadel: ListingKitZitadelConfig{AllowedUsernames: []string{"obsolete-fixture"}}}}

	err := ValidateConfigWithError(cfg)
	if assert.Error(t, err) {
		assert.NotContains(t, err.Error(), "[Management]")
		assert.Contains(t, err.Error(), "[General]")
	}
}
