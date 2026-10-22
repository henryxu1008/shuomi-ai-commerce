package config

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurrentConfigDoesNotRequireRetiredTaskInputs(t *testing.T) {
	for _, name := range []string{"Processor", "Worker", "RabbitMQ", "Debug", "ListingControlPlane", "Updater", "Watermark", "FeatureFlags", "Observability"} {
		_, exists := reflect.TypeOf(Config{}).FieldByName(name)
		assert.False(t, exists, "retired root %s", name)
	}
	for _, name := range []string{"SheinSubmitDebugDumpDir", "GenerationUsageLedgerEnabled", "GenerationUsageLedgerTenantIDs"} {
		_, exists := reflect.TypeOf(ListingKitConfig{}).FieldByName(name)
		assert.False(t, exists, "retired ListingKit field %s", name)
	}
	platformType := reflect.TypeOf(PlatformConfig{})
	var names []string
	for i := 0; i < platformType.NumField(); i++ {
		names = append(names, platformType.Field(i).Name)
	}
	assert.ElementsMatch(t, []string{"HTTPClient"}, names)
	v := viper.New()
	setDefaults(v)
	keys := v.AllKeys()
	for key := range knownEnvBindings() {
		keys = append(keys, key)
	}
	for _, key := range keys {
		normalized := strings.ToLower(key)
		assert.False(t, normalized == "watermark" || strings.HasPrefix(normalized, "watermark."), "retired watermark default/env %s", key)
		for _, prefix := range []string{"featureflags", "observability"} {
			assert.False(t, normalized == prefix || strings.HasPrefix(normalized, prefix+"."), "retired host config default/env %s", key)
		}
		for _, prefix := range []string{"processor.", "worker.", "rabbitmq.", "listingControlPlane.", "debug.save_publish_json", "listingkit.sheinSubmitDebugDumpDir", "listingkit.generationUsageLedger", "updater."} {
			assert.False(t, strings.HasPrefix(normalized, strings.ToLower(prefix)), "retired default/env %s", key)
		}
		for _, field := range []string{"dataFreshnessDays", "productDedupe", "regionGuard", "concurrencyControl", "remoteAPI"} {
			prefix := strings.ToLower("amazon." + field)
			assert.False(t, normalized == prefix || strings.HasPrefix(normalized, prefix+"."), "retired Amazon default/env %s", key)
		}
		for _, platform := range []string{"temu", "shein"} {
			for _, field := range []string{"enabled", "schedulerEnabled", "autoPricing", "listingPricing", "productSync", "inventorySync", "activityRegistration", "sync", "monitor", "configPaths", "cookieRedis", "loginService", "fetchMode"} {
				prefix := strings.ToLower("platforms." + platform + "." + field)
				assert.False(t, normalized == prefix || strings.HasPrefix(normalized, prefix+"."), "retired platform default/env %s", key)
			}
		}
	}
	for _, cfg := range []*Config{NewDefaultConfig(), BuildConfig(v)} {
		cfg.OpenAI.APIKey = "fixture-key"
		require.Empty(t, cfg.Validate(), "current defaults must not require retired worker/task inputs")
	}
}

func TestLoadFromBytesRetainsPlatformTransportLoginAndRetry(t *testing.T) {
	cfg, err := LoadFromBytes([]byte(`
openai:
  apiKey: fixture-key
platforms:
  shein:
    httpClient:
      timeout: 42
      insecureSkipVerify: true
`))
	require.NoError(t, err)
	assert.Equal(t, 42*time.Second, cfg.Platforms.Shein.HTTPClient.Timeout)
	assert.True(t, cfg.Platforms.Shein.HTTPClient.InsecureSkipVerify)
	assert.Equal(t, 3, DefaultRetryConfig().MaxRetries)
}

func TestBrowserConfigDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	assert.True(t, v.GetBool("browser.enabled"))
	assert.True(t, v.GetBool("browser.headless"))
	assert.Equal(t, "./.local/chrome/chrome.exe", v.GetString("browser.browserPath"))
	assert.Equal(t, "", v.GetString("browser.userDataDir"))
	assert.Equal(t, 3, v.GetInt("browser.poolSize"))
	assert.Equal(t, 1920, v.GetInt("browser.viewportWidth"))
	assert.Equal(t, 1080, v.GetInt("browser.viewportHeight"))
	assert.Equal(t, "", v.GetString("browser.proxyServer"))
	assert.True(t, v.GetBool("browser.randomConfig.enabled"))
	assert.Equal(t, "random", v.GetString("browser.randomConfig.strategy"))
	assert.Equal(t, "windows_high_end", v.GetString("browser.randomConfig.presetName"))
	assert.Equal(t, "random", v.GetString("browser.randomConfig.fingerprintStrategy"))
	assert.True(t, v.GetBool("browser.randomConfig.healthCheckEnabled"))
	assert.Equal(t, 3, v.GetInt("browser.randomConfig.maxRetries"))
	assert.Equal(t, 25, v.GetInt("browser.randomConfig.maxUsesPerInstance"))
}

func TestAmazonConfigDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	assert.False(t, v.GetBool("amazon.enabled"))
	assert.Equal(t, 1, v.GetInt("amazon.riskControl.captchaRecreateThreshold"))
	assert.Equal(t, 1, v.GetInt("amazon.riskControl.authenticationRecreateThreshold"))
	assert.Equal(t, 1, v.GetInt("amazon.riskControl.browserCrashRecreateThreshold"))
	assert.Equal(t, 3, v.GetInt("amazon.riskControl.timeoutRecreateThreshold"))
	assert.True(t, v.GetBool("amazon.qualityControl.retryOnValidationFailure"))
	assert.Equal(t, 2, v.GetInt("amazon.qualityControl.validationRetryMaxAttempts"))
	assert.False(t, v.GetBool("amazon.spapi.enabled"))
	assert.Equal(t, "us-east-1", v.GetString("amazon.spapi.region"))
	assert.Equal(t, "ATVPDKIKX0DER", v.GetString("amazon.spapi.defaultMarketplace"))
	assert.Equal(t, "FBM", v.GetString("amazon.spapi.defaultFulfillmentType"))
	assert.Equal(t, "New", v.GetString("amazon.spapi.defaultCondition"))
}

func TestAlibaba1688ConfigDefaults(t *testing.T) {
	v := viper.New()
	setDefaults(v)

	assert.False(t, v.GetBool("platforms.alibaba1688.enabled"))
	assert.Equal(t, 120, v.GetInt("platforms.alibaba1688.timeout"))
	assert.Equal(t, 2, v.GetInt("platforms.alibaba1688.poolSize"))
	assert.Equal(t, "./.local/tmp/browser-profiles/1688-accounts", v.GetString("platforms.alibaba1688.profileRootDir"))
}

func TestConfigBuild(t *testing.T) {
	v := viper.New()
	v.Set("browser.enabled", true)
	v.Set("browser.poolSize", 5)
	v.Set("browser.userDataDir", "./.local/tmp/browser-profiles/1688")
	v.Set("browser.randomConfig.enabled", true)
	v.Set("browser.randomConfig.strategy", "stable")
	v.Set("browser.randomConfig.presetName", "mac_high_end")
	v.Set("amazon.enabled", true)
	v.Set("platforms.alibaba1688.profileRootDir", "C:/task-processor-test/1688-profiles")
	v.Set("platforms.temu.httpClient.insecureSkipVerify", true)
	v.Set("openai.clients.image.model", "nanobanana")
	v.Set("openai.clients.image.timeout", 180)
	v.Set("openai.clients.image.apiStyle", "nanobanana")

	cfg := BuildConfig(v)

	assert.True(t, cfg.Browser.Enabled)
	assert.Equal(t, 5, cfg.Browser.PoolSize)
	assert.Equal(t, "./.local/tmp/browser-profiles/1688", cfg.Browser.UserDataDir)
	assert.True(t, cfg.Browser.RandomConfig.Enabled)
	assert.Equal(t, "stable", cfg.Browser.RandomConfig.Strategy)
	assert.Equal(t, "mac_high_end", cfg.Browser.RandomConfig.PresetName)
	assert.True(t, cfg.Amazon.Enabled)
	assert.True(t, cfg.Platforms.Temu.HTTPClient.InsecureSkipVerify)
	assert.Equal(t, "C:/task-processor-test/1688-profiles", cfg.Platforms.Alibaba1688.ProfileRootDir)
	assert.Equal(t, "nanobanana", cfg.OpenAI.Clients["image"].Model)
	assert.Equal(t, 180, cfg.OpenAI.Clients["image"].Timeout)
	assert.Equal(t, "nanobanana", cfg.OpenAI.Clients["image"].APIStyle)
}

func TestConfigBuildIncludesListingKitConfig(t *testing.T) {
	v := viper.New()
	v.Set("listingkit.platformAdminUsers", []string{"user-a", "user-b"})
	v.Set("listingkit.platformAdminRoles", []string{"role-a", "role-b"})
	v.Set("listingkit.zitadel.issuerURL", "https://issuer.example")
	v.Set("listingkit.zitadel.clientID", "listingkit-client")
	v.Set("listingkit.zitadel.clientSecret", "listingkit-secret")
	v.Set("listingkit.zitadel.authorizationRequired", true)
	v.Set("listingkit.zitadel.allowedTenantIDs", []string{"tenant-a", "tenant-b"})
	v.Set("listingkit.zitadel.allowedUserIDs", []string{"user-a", "user-b"})
	v.Set("listingkit.zitadel.allowedUsernames", []string{"alice", "bob"})
	v.Set("listingkit.zitadel.allowedRoles", []string{"listingkit_admin", "platform_admin"})
	v.Set("imageagent.admission.enabled", true)
	v.Set("imageagent.admission.allowedTenantIDs", []string{"tenant-image-a", "tenant-image-b"})

	cfg := BuildConfig(v)

	assert.Equal(t, []string{"user-a", "user-b"}, cfg.ListingKit.PlatformAdminUsers)
	assert.Equal(t, []string{"role-a", "role-b"}, cfg.ListingKit.PlatformAdminRoles)
	assert.Equal(t, "https://issuer.example", cfg.ListingKit.Zitadel.IssuerURL)
	assert.Equal(t, "listingkit-client", cfg.ListingKit.Zitadel.ClientID)
	assert.Equal(t, "listingkit-secret", cfg.ListingKit.Zitadel.ClientSecret)
	assert.True(t, cfg.ListingKit.Zitadel.AuthorizationRequired)
	assert.Equal(t, []string{"tenant-a", "tenant-b"}, cfg.ListingKit.Zitadel.AllowedTenantIDs)
	assert.Equal(t, []string{"user-a", "user-b"}, cfg.ListingKit.Zitadel.AllowedUserIDs)
	assert.Equal(t, []string{"alice", "bob"}, cfg.ListingKit.Zitadel.AllowedUsernames)
	assert.Equal(t, []string{"listingkit_admin", "platform_admin"}, cfg.ListingKit.Zitadel.AllowedRoles)
	assert.True(t, cfg.ImageAgent.Admission.Enabled)
	assert.Equal(t, []string{"tenant-image-a", "tenant-image-b"}, cfg.ImageAgent.Admission.AllowedTenantIDs)
}

func TestConfigValidation(t *testing.T) {
	validConfig := &Config{
		OpenAI: OpenAIConfig{
			APIKey:  "test-key",
			Model:   "test-model",
			BaseURL: "http://example.com/v1",
			Timeout: 30,
		},
		Browser: BrowserConfig{
			Enabled:        true,
			PoolSize:       3,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			RandomConfig: BrowserRandomConfig{
				Enabled:             true,
				Strategy:            "random",
				PresetName:          "windows_high_end",
				FingerprintStrategy: "random",
				MaxRetries:          3,
				MaxUsesPerInstance:  25,
			},
		},
		Amazon: AmazonConfig{
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
		},
	}

	errors := validConfig.Validate()
	assert.Empty(t, errors)

	invalidConfig := &Config{
		Browser: BrowserConfig{
			Enabled:        true,
			PoolSize:       0,
			ViewportWidth:  0,
			ViewportHeight: 0,
			RandomConfig: BrowserRandomConfig{
				Enabled:             true,
				Strategy:            "invalid_strategy",
				FingerprintStrategy: "invalid_fingerprint",
				MaxRetries:          -1,
				MaxUsesPerInstance:  -1,
			},
		},
		Amazon: AmazonConfig{
			Enabled: true,
		},
	}

	errors = invalidConfig.Validate()
	assert.NotEmpty(t, errors)
	assert.True(t, len(errors) >= 4)
}

func TestBrowserConfigValidation(t *testing.T) {
	validConfig := &Config{
		Browser: BrowserConfig{
			Enabled:        true,
			PoolSize:       3,
			ViewportWidth:  1920,
			ViewportHeight: 1080,
			RandomConfig: BrowserRandomConfig{
				Enabled:             true,
				Strategy:            "random",
				PresetName:          "windows_high_end",
				FingerprintStrategy: "random",
				MaxRetries:          3,
				MaxUsesPerInstance:  25,
			},
		},
	}

	errors := ValidateBrowserConfig(&validConfig.Browser)
	assert.Empty(t, errors)

	invalidConfig := &Config{
		Browser: BrowserConfig{
			Enabled:        true,
			PoolSize:       0,
			ViewportWidth:  0,
			ViewportHeight: 0,
			RandomConfig: BrowserRandomConfig{
				Enabled:             true,
				Strategy:            "invalid_strategy",
				FingerprintStrategy: "invalid_fingerprint",
				MaxRetries:          -1,
				MaxUsesPerInstance:  -1,
			},
		},
	}

	errors = ValidateBrowserConfig(&invalidConfig.Browser)
	assert.NotEmpty(t, errors)
	assert.True(t, len(errors) >= 3)
}
