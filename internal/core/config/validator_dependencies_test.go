package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOpenAIConfig_ClientsMustBeComplete(t *testing.T) {
	cfg := OpenAIConfig{
		APIKey:  "",
		Model:   "gemini-2.5-flash",
		BaseURL: "https://example.com/v1",
		Timeout: 30,
		Clients: map[string]OpenAIClientConfig{
			"vision": {
				Model: "gpt-4o-mini",
			},
		},
	}

	errors := ValidateOpenAIConfig(&cfg)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[len(errors)-1].Error(), "openai.clients.vision.apiKey")
}

func TestValidateConfig_CatchesDependencyErrors(t *testing.T) {
	cfg := &Config{
		OpenAI: OpenAIConfig{
			APIKey:  "key",
			Model:   "model",
			BaseURL: "http://example.com/v1",
			Timeout: 10,
			Clients: map[string]OpenAIClientConfig{
				"vision": {},
			},
		},
	}

	errors := ValidateConfig(cfg)
	assert.NotEmpty(t, errors)
}

func TestValidateConfigPreservesValidationGroupOrder(t *testing.T) {
	cfg := &Config{
		OpenAI: OpenAIConfig{Model: "synthetic-model"},
		ListingKit: ListingKitConfig{Zitadel: ListingKitZitadelConfig{
			AllowedUsernames: []string{"synthetic-obsolete-user"},
		}},
		Workbench: WorkbenchConfig{Enabled: true},
	}

	errors := ValidateConfig(cfg)
	previous := -1
	for _, field := range []string{
		"openai.apiKey",
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
