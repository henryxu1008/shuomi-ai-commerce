package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadConfigFixture arranges synthetic YAML for the current configuration owner.
func loadConfigFixture(t *testing.T, data []byte) (*Config, error) {
	t.Helper()
	v := newViper()
	if len(data) > 0 {
		if err := v.ReadConfig(strings.NewReader(string(data))); err != nil {
			return nil, err
		}
	}
	return loadWithViper(v)
}

func TestCurrentConfigDoesNotRequireRetiredTaskInputs(t *testing.T) {
	for _, name := range []string{"Processor", "Worker", "RabbitMQ", "Debug", "ListingControlPlane", "Updater", "Watermark", "FeatureFlags", "Observability", "Browser", "Amazon", "Platforms"} {
		_, exists := reflect.TypeOf(Config{}).FieldByName(name)
		assert.False(t, exists, "retired root %s", name)
	}
	for _, name := range []string{"SheinSubmitDebugDumpDir", "GenerationUsageLedgerEnabled", "GenerationUsageLedgerTenantIDs"} {
		_, exists := reflect.TypeOf(ListingKitConfig{}).FieldByName(name)
		assert.False(t, exists, "retired ListingKit field %s", name)
	}
	v := viper.New()
	setDefaults(v)
	keys := v.AllKeys()
	for key := range knownEnvBindings() {
		keys = append(keys, key)
	}
	for _, key := range keys {
		normalized := strings.ToLower(key)
		for _, prefix := range []string{"browser", "amazon", "platforms"} {
			assert.False(t, normalized == prefix || strings.HasPrefix(normalized, prefix+"."), "retired platform API default/env %s", key)
		}
		assert.False(t, normalized == "watermark" || strings.HasPrefix(normalized, "watermark."), "retired watermark default/env %s", key)
		for _, prefix := range []string{"featureflags", "observability"} {
			assert.False(t, normalized == prefix || strings.HasPrefix(normalized, prefix+"."), "retired host config default/env %s", key)
		}
		for _, prefix := range []string{"processor.", "worker.", "rabbitmq.", "listingControlPlane.", "debug.save_publish_json", "listingkit.sheinSubmitDebugDumpDir", "listingkit.generationUsageLedger", "updater.", "logging.", "redis.", "prompts."} {
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
		require.Empty(t, ValidateConfig(cfg), "current defaults must not require retired worker/task inputs")
	}
}

func TestConfigBuild(t *testing.T) {
	v := viper.New()
	v.Set("openai.clients.image.model", "nanobanana")
	v.Set("openai.clients.image.timeout", 180)
	v.Set("openai.clients.image.apiStyle", "nanobanana")

	cfg := BuildConfig(v)

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
	v.Set("listingkit.zitadel.allowedUsernames", []string{"alice", "bob"})
	v.Set("imageagent.admission.enabled", true)
	v.Set("imageagent.admission.allowedTenantIDs", []string{"tenant-image-a", "tenant-image-b"})

	cfg := BuildConfig(v)

	assert.Equal(t, []string{"user-a", "user-b"}, cfg.ListingKit.PlatformAdminUsers)
	assert.Equal(t, []string{"role-a", "role-b"}, cfg.ListingKit.PlatformAdminRoles)
	assert.Equal(t, "https://issuer.example", cfg.ListingKit.Zitadel.IssuerURL)
	assert.Equal(t, "listingkit-client", cfg.ListingKit.Zitadel.ClientID)
	assert.Equal(t, "listingkit-secret", cfg.ListingKit.Zitadel.ClientSecret)
	assert.Equal(t, []string{"alice", "bob"}, cfg.ListingKit.Zitadel.AllowedUsernames)
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
	}

	errors := ValidateConfig(validConfig)
	assert.Empty(t, errors)

	invalidConfig := &Config{OpenAI: OpenAIConfig{Clients: map[string]OpenAIClientConfig{"fixture": {}}}}

	errors = ValidateConfig(invalidConfig)
	assert.NotEmpty(t, errors)
	assert.True(t, len(errors) >= 4)
}
