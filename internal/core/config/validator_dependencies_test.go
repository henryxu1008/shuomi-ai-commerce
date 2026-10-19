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
		Enabled:           true,
		DataFreshnessDays: 7,
		CrawlTimeout:      30,
		RiskControl: AmazonRiskControlConfig{
			CaptchaRecreateThreshold:        1,
			AuthenticationRecreateThreshold: 1,
			BrowserCrashRecreateThreshold:   1,
			TimeoutRecreateThreshold:        3,
			NetworkRecreateThreshold:        2,
			ServerErrorRecreateThreshold:    3,
		},
		RegionGuard: AmazonRegionGuardConfig{
			Enabled:                 true,
			FailureThreshold:        3,
			EvaluationWindowSeconds: 300,
			CooldownSeconds:         180,
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
			Enabled:           true,
			DataFreshnessDays: 1,
			CrawlTimeout:      1,
			RiskControl: AmazonRiskControlConfig{
				CaptchaRecreateThreshold:        1,
				AuthenticationRecreateThreshold: 1,
				BrowserCrashRecreateThreshold:   1,
				TimeoutRecreateThreshold:        3,
				NetworkRecreateThreshold:        2,
				ServerErrorRecreateThreshold:    3,
			},
			RegionGuard: AmazonRegionGuardConfig{
				Enabled:                 true,
				FailureThreshold:        3,
				EvaluationWindowSeconds: 300,
				CooldownSeconds:         180,
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

	errors := cfg.Validate()
	assert.NotEmpty(t, errors)
}
