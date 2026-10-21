package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
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

func LoadConfigWithFallback(configPath string, logger *logrus.Logger) (*Config, error) {
	if logger != nil {
		logger.Infof("received config path: %q", configPath)
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %s", configPath)
		}
		return nil, fmt.Errorf("stat config file %s: %w", configPath, err)
	}

	cfg, err := LoadConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	if logger != nil {
		logger.Infof("config loaded successfully")
	}

	return cfg, nil
}

func LoadAmazonCrawlerAPIConfigWithFallback(configPath string, logger *logrus.Logger) (*Config, error) {
	if logger != nil {
		logger.Infof("received amazon crawler api config path: %q", configPath)
	}

	if _, err := os.Stat(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file does not exist: %s", configPath)
		}
		return nil, fmt.Errorf("stat config file %s: %w", configPath, err)
	}

	cfg, err := LoadConfigFromFileWithoutValidation(configPath)
	if err != nil {
		return nil, err
	}

	if err := validateAmazonCrawlerAPIConfig(cfg); err != nil {
		return nil, err
	}

	if logger != nil {
		logger.Infof("amazon crawler api config loaded successfully")
		logger.Debugf("   - Browser.Enabled: %v", cfg.Browser.Enabled)
		logger.Debugf("   - Amazon.Enabled: %v", cfg.Amazon.Enabled)
		logger.Debugf("   - Redis.Enabled: %v", cfg.Redis != nil)
	}

	return cfg, nil
}

func validateAmazonCrawlerAPIConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config validation failed:\n[General]\n  - config: configuration cannot be nil")
	}

	var errors []error
	errors = append(errors, ValidateBrowserConfig(&cfg.Browser)...)
	errors = append(errors, ValidateAmazonConfig(&cfg.Amazon)...)

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf("config validation failed:\n%s", strings.Join(formatValidationErrors(errors), "\n"))
}

func NewDefaultConfig() *Config {
	return &Config{
		FeatureFlags: FeatureFlagsConfig{
			Flags: map[string]bool{"product-listing-runtime-auto-migrate": true},
		},
		Observability: ObservabilityConfig{
			Tracing: TracingConfig{
				Enabled:     false,
				ServiceName: "task-processor",
				Endpoint:    "",
				Insecure:    false,
			},
		},
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
			SPAPI: SPAPIConfig{
				Enabled:                false,
				Region:                 "us-east-1",
				DefaultMarketplace:     "ATVPDKIKX0DER",
				DefaultFulfillmentType: "FBM",
				DefaultCondition:       "New",
			},
		},
		ImageAgent: ImageAgentConfig{},
		ListingKit: ListingKitConfig{
			ImageUpload: ListingKitImageUploadConfig{},
			Zitadel:     ListingKitZitadelConfig{},
		},
		Platforms: PlatformsConfig{
			Temu: PlatformConfig{
				HTTPClient: *DefaultHTTPClientConfig(),
			},
			Shein: PlatformConfig{
				HTTPClient: *DefaultHTTPClientConfig(),
			},
			Alibaba1688: Alibaba1688Config{
				Enabled:        false,
				Timeout:        120,
				PoolSize:       2,
				ProfileRootDir: "./.local/tmp/browser-profiles/1688-accounts",
			},
		},
	}
}
