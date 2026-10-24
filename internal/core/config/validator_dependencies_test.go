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

func TestValidateAmazonConfig_SPAPIRequiresCredentials(t *testing.T) {
	cfg := AmazonConfig{
		Enabled:      true,
		CrawlTimeout: 30,
		RiskControl: AmazonRiskControlConfig{
			CaptchaRecreateThreshold:        1,
			AuthenticationRecreateThreshold: 1,
			BrowserCrashRecreateThreshold:   1,
			TimeoutRecreateThreshold:        3,
			NetworkRecreateThreshold:        2,
			ServerErrorRecreateThreshold:    3,
		},
		QualityControl: AmazonQualityControlConfig{
			RetryOnValidationFailure:   true,
			ValidationRetryMaxAttempts: 2,
		},
		SPAPI: SPAPIConfig{
			Enabled: true,
			Region:  "us-east-1",
		},
	}

	errors := ValidateAmazonConfig(&cfg)
	assert.NotEmpty(t, errors)
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
		Browser: BrowserConfig{
			Enabled:        true,
			PoolSize:       1,
			ViewportWidth:  1,
			ViewportHeight: 1,
		},
		Amazon: AmazonConfig{
			Enabled:      true,
			CrawlTimeout: 1,
			RiskControl: AmazonRiskControlConfig{
				CaptchaRecreateThreshold:        1,
				AuthenticationRecreateThreshold: 1,
				BrowserCrashRecreateThreshold:   1,
				TimeoutRecreateThreshold:        3,
				NetworkRecreateThreshold:        2,
				ServerErrorRecreateThreshold:    3,
			},
			QualityControl: AmazonQualityControlConfig{
				RetryOnValidationFailure:   true,
				ValidationRetryMaxAttempts: 2,
			},
			SPAPI: SPAPIConfig{
				Enabled: true,
				Region:  "us-east-1",
			},
		},
	}

	errors := ValidateConfig(cfg)
	assert.NotEmpty(t, errors)
}

func TestValidateConfigPreservesValidationGroupOrder(t *testing.T) {
	cfg := &Config{
		OpenAI:    OpenAIConfig{Model: "synthetic-model"},
		Browser:   BrowserConfig{Enabled: true},
		Amazon:    AmazonConfig{Enabled: true},
		Platforms: PlatformsConfig{Alibaba1688: Alibaba1688Config{Enabled: true}},
		ListingKit: ListingKitConfig{Zitadel: ListingKitZitadelConfig{
			AllowedUsernames: []string{"synthetic-obsolete-user"},
		}},
		Workbench: WorkbenchConfig{Enabled: true},
	}

	errors := ValidateConfig(cfg)
	previous := -1
	for _, field := range []string{
		"openai.apiKey",
		"browser.poolSize",
		"amazon.crawlTimeout",
		"platforms.alibaba1688.timeout",
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
