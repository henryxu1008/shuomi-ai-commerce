package config

import (
	"bytes"
	"fmt"
)

func LoadFromBytes(data []byte) (*Config, error) {
	tryLoadDotEnv()
	logDeprecatedEnvUsage()

	v := newViper()
	if len(data) > 0 {
		if err := v.ReadConfig(bytes.NewReader(data)); err != nil {
			return nil, fmt.Errorf("parse yaml config: %w", err)
		}
	}

	return loadWithViper(v)
}

func NewDefaultConfig() *Config {
	return &Config{
		OpenAI: OpenAIConfig{
			APIKey:  "",
			Model:   "gemini-2.0-flash",
			BaseURL: "https://ai.linkcloudai.com/v1",
			Timeout: 120,
		},
		Browser: BrowserConfig{
			Enabled:        true,
			Headless:       true,
			BrowserPath:    "./.local/chrome/chrome.exe",
			UserDataDir:    "",
			PoolSize:       3,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			RandomConfig: BrowserRandomConfig{
				Enabled:             true,
				Strategy:            "random",
				PresetName:          "windows_high_end",
				FingerprintStrategy: "random",
				HealthCheckEnabled:  true,
				MaxRetries:          3,
				MaxUsesPerInstance:  25,
			},
		},
		Amazon: AmazonConfig{
			Enabled: false,
			FailureArtifacts: FailureArtifactsConfig{
				Enabled:      false,
				Directory:    "./.local/tmp/amazon-failure-artifacts",
				CaptureHTML:  true,
				MaxHTMLBytes: 262144,
			},
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
			ProxyPool: AmazonProxyPoolConfig{
				Enabled:                false,
				Strategy:               "round_robin",
				FailureCooldownSeconds: 300,
			},
		},
		ImageAgent: ImageAgentConfig{},
		ListingKit: ListingKitConfig{
			ImageUpload: ListingKitImageUploadConfig{},
			Zitadel:     ListingKitZitadelConfig{},
		},
		Platforms: PlatformsConfig{
			Alibaba1688: Alibaba1688Config{
				Enabled:        false,
				Timeout:        120,
				PoolSize:       2,
				ProfileRootDir: "./.local/tmp/browser-profiles/1688-accounts",
			},
		},
	}
}
