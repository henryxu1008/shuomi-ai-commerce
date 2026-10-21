package config

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type stubConfigSource struct {
	name string
	data []byte
}

func (s stubConfigSource) Read() ([]byte, error) { return s.data, nil }

func (s stubConfigSource) Watch(_ context.Context, _ func([]byte)) error { return nil }

func (s stubConfigSource) Name() string { return s.name }

func TestUniqueCleanPathsPreservesFirstOccurrenceOrder(t *testing.T) {
	got := uniqueCleanPaths([]string{"./.env", ".env", "config/../.env", "config/.env", "./config/../.env"})
	require.Equal(t, []string{".env", filepath.Join("config", ".env")}, got)
}

func TestNewViper_BindsPrimaryEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_AMAZON_SPAPI_CLIENT_ID", "amzn-client")
	t.Setenv("TASK_PROCESSOR_AMAZON_SPAPI_DEFAULT_MARKETPLACE", "ATVPDKIKX0DER")
	t.Setenv("TASK_PROCESSOR_AMAZON_REMOTE_API_BASE_URL", "http://crawler.internal:8080")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY", "image-key")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL", "https://image.example.test/v1")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE", "nanobanana")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL", "nano-banana-fast")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_TIMEOUT", "300")
	t.Setenv("TASK_PROCESSOR_COMMERCIAL_DATABASE_HOST", "commercial-db")

	v := newViper()

	assert.Equal(t, "amzn-client", v.GetString("amazon.spapi.clientID"))
	assert.Equal(t, "ATVPDKIKX0DER", v.GetString("amazon.spapi.defaultMarketplace"))
	assert.Equal(t, "http://crawler.internal:8080", v.GetString("amazon.remoteAPI.baseURL"))
	assert.Equal(t, "image-key", v.GetString("openai.clients.image.apiKey"))
	assert.Equal(t, "https://image.example.test/v1", v.GetString("openai.clients.image.baseURL"))
	assert.Equal(t, "nanobanana", v.GetString("openai.clients.image.apiStyle"))
	assert.Equal(t, "nano-banana-fast", v.GetString("openai.clients.image.model"))
	assert.Equal(t, 300, v.GetInt("openai.clients.image.timeout"))
	assert.Equal(t, "commercial-db", v.GetString("commercialDatabase.host"))
}

func TestBuildConfigLoadsCommercialDatabaseForWorkerUsageSettlement(t *testing.T) {
	v := newViper()
	v.Set("commercialDatabase.host", "commercial-db")
	v.Set("commercialDatabase.port", 5434)
	v.Set("commercialDatabase.user", "commercial_runtime")
	v.Set("commercialDatabase.password", "secret-not-logged")
	v.Set("commercialDatabase.database", "commercial")
	v.Set("commercialDatabase.max_connections", 4)

	cfg := BuildConfig(v)
	require.NotNil(t, cfg.CommercialDatabase)
	assert.Equal(t, "commercial-db", cfg.CommercialDatabase.Host)
	assert.Equal(t, 5434, cfg.CommercialDatabase.Port)
	assert.Equal(t, "commercial_runtime", cfg.CommercialDatabase.User)
	assert.Equal(t, "commercial", cfg.CommercialDatabase.Database)
	assert.Equal(t, 4, cfg.CommercialDatabase.MaxConnections)
}

func TestBuildConfigLoadsExplicitCommercialResourceOwner(t *testing.T) {
	v := newViper()
	v.Set("commercialOwnerDatabase.host", "commercial-db")
	v.Set("commercialOwnerDatabase.port", 5434)
	v.Set("commercialOwnerDatabase.user", "commercial_owner_runtime")
	v.Set("commercialOwnerDatabase.database", "commercial")
	v.Set("commercialOwnerDatabase.max_connections", 4)
	cfg := BuildConfig(v)
	require.NotNil(t, cfg.CommercialOwnerDatabase)
	assert.Equal(t, "commercial_owner_runtime", cfg.CommercialOwnerDatabase.User)
	assert.Equal(t, "commercial-db", cfg.CommercialOwnerDatabase.Host)
	assert.Equal(t, 5434, cfg.CommercialOwnerDatabase.Port)
	assert.Equal(t, "commercial", cfg.CommercialOwnerDatabase.Database)
	assert.Nil(t, cfg.CommercialDatabase, "no implicit Token-pool alias")
}

func TestTracingConfigDefaultsToDisabledTaskProcessorService(t *testing.T) {
	tracingType := reflect.TypeOf(TracingConfig{})
	assert.Equal(t, 4, tracingType.NumField(), "tracing config must remain the four approved transport values")
	cfg := BuildConfig(newViper())

	assert.False(t, cfg.Observability.Tracing.Enabled)
	assert.Equal(t, "task-processor", cfg.Observability.Tracing.ServiceName)
	assert.Empty(t, cfg.Observability.Tracing.Endpoint)
	assert.False(t, cfg.Observability.Tracing.Insecure)
}

func TestTracingConfigBindsEveryEnvironmentVariable(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OBSERVABILITY_TRACING_ENABLED", "true")
	t.Setenv("TASK_PROCESSOR_OBSERVABILITY_TRACING_SERVICE_NAME", "listing-api")
	t.Setenv("TASK_PROCESSOR_OBSERVABILITY_TRACING_ENDPOINT", "collector.internal:4317")
	t.Setenv("TASK_PROCESSOR_OBSERVABILITY_TRACING_INSECURE", "true")

	cfg := BuildConfig(newViper())

	assert.True(t, cfg.Observability.Tracing.Enabled)
	assert.Equal(t, "listing-api", cfg.Observability.Tracing.ServiceName)
	assert.Equal(t, "collector.internal:4317", cfg.Observability.Tracing.Endpoint)
	assert.True(t, cfg.Observability.Tracing.Insecure)
}

func TestCommittedConfigArtifactsDeclareEveryTracingField(t *testing.T) {
	for _, path := range []string{
		filepath.Join("..", "..", "..", "config", "config-dev.yaml"),
		filepath.Join("..", "..", "..", "config", "config-test.yaml"),
		filepath.Join("..", "..", "..", "config", "config-prod.yaml"),
	} {
		t.Run(filepath.Base(path), func(t *testing.T) {
			cfg, err := LoadConfigFromFileWithoutValidation(path)
			require.NoError(t, err)
			assert.False(t, cfg.Observability.Tracing.Enabled)
			assert.Equal(t, "task-processor", cfg.Observability.Tracing.ServiceName)
			assert.Empty(t, cfg.Observability.Tracing.Endpoint)
			assert.False(t, cfg.Observability.Tracing.Insecure)
		})
	}
}

func TestGetStringSlice_SplitsCommaSeparatedSingleEntry(t *testing.T) {
	t.Setenv("LISTINGKIT_PLATFORM_ADMIN_USERS", "user-a,user-b,user-c,user-d")

	v := newViper()

	assert.Equal(t,
		[]string{"user-a", "user-b", "user-c", "user-d"},
		getStringSlice(v, "listingkit.platformAdminUsers"),
	)
}

func TestNewViper_BindsLegacyEnvironmentAliases(t *testing.T) {
	t.Setenv("AMAZON_SPAPI_CLIENT_SECRET", "legacy-secret")
	t.Setenv("AMAZON_SPAPI_MARKETPLACE_ID", "LEGACY-MARKET")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APIKEY", "legacy-image-key")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASEURL", "https://legacy-image.example.test/v1")
	t.Setenv("TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APISTYLE", "legacy-style")

	v := newViper()

	assert.Equal(t, "legacy-secret", v.GetString("amazon.spapi.clientSecret"))
	assert.Equal(t, "LEGACY-MARKET", v.GetString("amazon.spapi.defaultMarketplace"))
	assert.Equal(t, "legacy-image-key", v.GetString("openai.clients.image.apiKey"))
	assert.Equal(t, "https://legacy-image.example.test/v1", v.GetString("openai.clients.image.baseURL"))
	assert.Equal(t, "legacy-style", v.GetString("openai.clients.image.apiStyle"))
}

func TestNewViper_BindsDatabaseEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_DATABASE_HOST", "db.example")
	t.Setenv("TASK_PROCESSOR_DATABASE_PORT", "30432")
	t.Setenv("DB_NAME", "legacy-db")

	v := newViper()

	assert.Equal(t, "db.example", v.GetString("database.host"))
	assert.Equal(t, 30432, v.GetInt("database.port"))
	assert.Equal(t, "legacy-db", v.GetString("database.database"))
}

func TestFeatureFlagsDefaultProductListingRuntimeAutoMigrate(t *testing.T) {
	cfg := NewDefaultConfig()

	assert.Equal(t, map[string]bool{
		"product-listing-runtime-auto-migrate": true,
	}, cfg.FeatureFlags.Flags)
}

func TestLoadFromBytesLoadsFeatureFlagFromYAML(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "sk-test")

	cfg, err := LoadFromBytes([]byte(strings.Join([]string{
		"featureFlags:",
		"  flags:",
		"    product-listing-runtime-auto-migrate: false",
	}, "\n")))
	require.NoError(t, err)

	assert.Equal(t, false, cfg.FeatureFlags.Flags["product-listing-runtime-auto-migrate"])
}

func TestLoadFromBytesPreservesEveryBooleanFeatureFlagFromYAML(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "sk-test")
	t.Setenv("TASK_PROCESSOR_API_RUNTIME_AUTOMIGRATE", "false")

	cfg, err := LoadFromBytes([]byte(strings.Join([]string{
		"featureFlags:",
		"  flags:",
		"    product-listing-runtime-auto-migrate: true",
		"    secondary-runtime-capability: true",
	}, "\n")))
	require.NoError(t, err)

	assert.Equal(t, map[string]bool{
		"product-listing-runtime-auto-migrate": false,
		"secondary-runtime-capability":         true,
	}, cfg.FeatureFlags.Flags)
}

func TestBuildConfigReturnsIndependentFeatureFlagMaps(t *testing.T) {
	v := newViper()
	v.Set("featureFlags.flags.secondary-runtime-capability", true)

	first := BuildConfig(v)
	second := BuildConfig(v)
	first.FeatureFlags.Flags["secondary-runtime-capability"] = false
	first.FeatureFlags.Flags["test-only-mutation"] = true

	assert.Equal(t, true, second.FeatureFlags.Flags["secondary-runtime-capability"])
	assert.NotContains(t, second.FeatureFlags.Flags, "test-only-mutation")
	assert.Equal(t, true, v.GetBool("featureFlags.flags.secondary-runtime-capability"))
}

func TestLoadFromBytesFeatureFlagEnvironmentOverride(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "sk-test")
	t.Setenv("TASK_PROCESSOR_API_RUNTIME_AUTOMIGRATE", "false")

	cfg, err := LoadFromBytes([]byte(strings.Join([]string{
		"featureFlags:",
		"  flags:",
		"    product-listing-runtime-auto-migrate: true",
	}, "\n")))
	require.NoError(t, err)

	assert.Equal(t, false, cfg.FeatureFlags.Flags["product-listing-runtime-auto-migrate"])
}

func TestCommittedFeatureFlagConfigArtifactsEnableProductListingRuntimeAutoMigrate(t *testing.T) {
	for _, configName := range []string{"config-dev.yaml", "config-prod.yaml", "config-test.yaml"} {
		t.Run(configName, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join("..", "..", "..", "config", configName))
			require.NoError(t, err)

			var document map[string]any
			require.NoError(t, yaml.Unmarshal(contents, &document))
			featureFlags, ok := document["featureFlags"].(map[string]any)
			require.True(t, ok, "featureFlags must be a YAML mapping")
			flags, ok := featureFlags["flags"].(map[string]any)
			require.True(t, ok, "featureFlags.flags must be a YAML mapping")
			assert.Equal(t, true, flags["product-listing-runtime-auto-migrate"])
		})
	}
}

func TestNewViper_BindsRedisEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_REDIS_HOST", "redis.example")
	t.Setenv("TASK_PROCESSOR_REDIS_PORT", "6380")
	t.Setenv("REDIS_PASSWORD", "legacy-pass")

	v := newViper()

	assert.Equal(t, "redis.example", v.GetString("redis.host"))
	assert.Equal(t, 6380, v.GetInt("redis.port"))
	assert.Equal(t, "legacy-pass", v.GetString("redis.password"))
}

func TestNewViper_BindsSDSLoginServiceEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_BASE_URL", "http://login.svc:8000")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_SHARED_KEY", "sds-key")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_TENANT_ID", "manual")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_IDENTIFIER", "default")
	t.Setenv("TASK_PROCESSOR_SDS_MERCHANT_NAME", "merchant")
	t.Setenv("TASK_PROCESSOR_SDS_USERNAME", "tester")
	t.Setenv("TASK_PROCESSOR_SDS_PASSWORD", "secret")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_DEFAULT_HEADLESS", "true")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_CLOAKBROWSER_ENABLED", "true")
	t.Setenv("TASK_PROCESSOR_SDS_LOGIN_SERVICE_CLOAKBROWSER_PATH", "C:/cloak/chrome.exe")

	v := newViper()

	assert.Equal(t, "http://login.svc:8000", v.GetString("platforms.sds.loginService.baseURL"))
	assert.Equal(t, "sds-key", v.GetString("platforms.sds.loginService.sharedKey"))
	assert.Equal(t, "manual", v.GetString("platforms.sds.loginService.tenantID"))
	assert.Equal(t, "default", v.GetString("platforms.sds.loginService.identifier"))
	assert.Equal(t, "merchant", v.GetString("platforms.sds.loginService.merchantName"))
	assert.Equal(t, "tester", v.GetString("platforms.sds.loginService.username"))
	assert.Equal(t, "secret", v.GetString("platforms.sds.loginService.password"))
	assert.True(t, v.GetBool("platforms.sds.loginService.defaultHeadless"))
	assert.True(t, v.GetBool("platforms.sds.loginService.cloakBrowserEnabled"))
	assert.Equal(t, "C:/cloak/chrome.exe", v.GetString("platforms.sds.loginService.cloakBrowserPath"))
}

func TestNewViper_BindsSDSAuthRedisEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_SDS_AUTH_REDIS_HOST", "sds-redis.example")
	t.Setenv("TASK_PROCESSOR_SDS_AUTH_REDIS_PORT", "6381")
	t.Setenv("TASK_PROCESSOR_SDS_AUTH_REDIS_PASSWORD", "sds-pass")
	t.Setenv("TASK_PROCESSOR_SDS_AUTH_REDIS_DB", "9")
	t.Setenv("TASK_PROCESSOR_SDS_AUTH_REDIS_POOL_SIZE", "16")

	v := newViper()

	assert.Equal(t, "sds-redis.example", v.GetString("platforms.sds.authRedis.host"))
	assert.Equal(t, 6381, v.GetInt("platforms.sds.authRedis.port"))
	assert.Equal(t, "sds-pass", v.GetString("platforms.sds.authRedis.password"))
	assert.Equal(t, 9, v.GetInt("platforms.sds.authRedis.db"))
	assert.Equal(t, 16, v.GetInt("platforms.sds.authRedis.pool_size"))
}

func TestNewViper_BindsSDSAuthBootstrapEnvironmentVariables(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_SDS_ACCESS_TOKEN", "access-token")
	t.Setenv("TASK_PROCESSOR_SDS_OUT_ACCESS_TOKEN", "out-token")
	t.Setenv("TASK_PROCESSOR_SDS_MERCHANT_ID", "12345")
	t.Setenv("TASK_PROCESSOR_SDS_COOKIE", "cookie=value")
	t.Setenv("TASK_PROCESSOR_SDS_DOMAIN_NAME", "www.sdsdiy.com")
	t.Setenv("TASK_PROCESSOR_SDS_VERIFY_CAPTCHA_PARAM", "captcha-param")
	t.Setenv("TASK_PROCESSOR_SDS_EXTRA_INFO", "{\"risk\":1}")

	v := newViper()

	assert.Equal(t, "access-token", v.GetString("platforms.sds.authBootstrap.staticAccessToken"))
	assert.Equal(t, "out-token", v.GetString("platforms.sds.authBootstrap.staticOutToken"))
	assert.Equal(t, int64(12345), v.GetInt64("platforms.sds.authBootstrap.staticMerchantID"))
	assert.Equal(t, "cookie=value", v.GetString("platforms.sds.authBootstrap.staticCookie"))
	assert.Equal(t, "www.sdsdiy.com", v.GetString("platforms.sds.authBootstrap.loginDomainName"))
	assert.Equal(t, "captcha-param", v.GetString("platforms.sds.authBootstrap.loginVerifyCaptchaParam"))
	assert.Equal(t, "{\"risk\":1}", v.GetString("platforms.sds.authBootstrap.loginExtraInfo"))
}

func TestNewViper_BindsListingKitEnvironmentVariables(t *testing.T) {
	t.Setenv("LISTINGKIT_DEBUG_SUBMIT_DUMP_DIR", "D:/tmp/shein-submit-dumps")
	t.Setenv("LISTINGKIT_PLATFORM_ADMIN_USERS", "user-a,user-b")
	t.Setenv("LISTINGKIT_PLATFORM_ADMIN_ROLES", "role-a,role-b")
	t.Setenv("ZITADEL_ISSUER_URL", "https://issuer.example")
	t.Setenv("ZITADEL_CLIENT_ID", "listingkit-client")
	t.Setenv("ZITADEL_CLIENT_SECRET", "listingkit-secret")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_TENANT_DIRECTORY_TOKEN", "directory-token")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHZ_REQUIRED", "true")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS", "tenant-a,tenant-b")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS", "user-a,user-b")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USERNAMES", "alice,bob")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES", "listingkit_admin,platform_admin")

	v := newViper()

	assert.Equal(t, []string{"user-a", "user-b"}, getStringSlice(v, "listingkit.platformAdminUsers"))
	assert.Equal(t, []string{"role-a", "role-b"}, getStringSlice(v, "listingkit.platformAdminRoles"))
	assert.Equal(t, "https://issuer.example", v.GetString("listingkit.zitadel.issuerURL"))
	assert.Equal(t, "listingkit-client", v.GetString("listingkit.zitadel.clientID"))
	assert.Equal(t, "listingkit-secret", v.GetString("listingkit.zitadel.clientSecret"))
	assert.Equal(t, "directory-token", v.GetString("listingkit.zitadel.tenantDirectoryToken"))
	assert.True(t, v.GetBool("listingkit.zitadel.authorizationRequired"))
	assert.Equal(t, []string{"tenant-a", "tenant-b"}, getStringSlice(v, "listingkit.zitadel.allowedTenantIDs"))
	assert.Equal(t, []string{"user-a", "user-b"}, getStringSlice(v, "listingkit.zitadel.allowedUserIDs"))
	assert.Equal(t, []string{"alice", "bob"}, getStringSlice(v, "listingkit.zitadel.allowedUsernames"))
	assert.Equal(t, []string{"listingkit_admin", "platform_admin"}, getStringSlice(v, "listingkit.zitadel.allowedRoles"))
}

func TestKnownEnvBindingsDoesNotExposeListingKitAuthenticationToggle(t *testing.T) {
	_, exists := knownEnvBindings()["listingkit.zitadel.authRequired"]
	assert.False(t, exists, "ListingKit authentication must not be configurable")
}

func TestKnownEnvBindingsDoNotExposeListingKitOwnerScopeToggle(t *testing.T) {
	bindings := knownEnvBindings()
	_, ok := bindings["listingkit.ownerScopeRequired"]
	assert.False(t, ok)

	for _, binding := range bindings {
		assert.NotEqual(t, "TASK_PROCESSOR_LISTINGKIT_OWNER_SCOPE_REQUIRED", binding.Primary)
		assert.NotContains(t, binding.Deprecated, "TASK_PROCESSOR_LISTINGKIT_ZITADEL_OWNER_SCOPE_REQUIRED")
	}
}

func TestCommittedListingKitConfigArtifactsDoNotExposeOwnerScopeToggle(t *testing.T) {
	for _, configName := range []string{"config-dev.yaml", "config-prod.yaml", "config-test.yaml"} {
		t.Run(configName, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join("..", "..", "..", "config", configName))
			require.NoError(t, err)

			var document map[string]any
			require.NoError(t, yaml.Unmarshal(contents, &document))
			listingKit, ok := document["listingkit"].(map[string]any)
			require.True(t, ok, "listingkit must be a YAML mapping")
			_, exposed := listingKit["ownerScopeRequired"]
			assert.False(t, exposed, "committed config must not offer an owner scope toggle")
		})
	}
}

func TestCommittedBrowserConfigArtifactsDefaultToHeadless(t *testing.T) {
	for _, configName := range []string{"config-dev.yaml", "config-prod.yaml", "config-test.yaml"} {
		t.Run(configName, func(t *testing.T) {
			contents, err := os.ReadFile(filepath.Join("..", "..", "..", "config", configName))
			require.NoError(t, err)

			var document map[string]any
			require.NoError(t, yaml.Unmarshal(contents, &document))
			browser, ok := document["browser"].(map[string]any)
			require.True(t, ok, "browser must be a YAML mapping")
			assert.Equal(t, true, browser["headless"], "committed browser config must default to headless")
		})
	}
}

func TestLoadConfigFromFileDoesNotExposeListingKitOwnerScopeToggle(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_OWNER_SCOPE_REQUIRED", "false")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_OWNER_SCOPE_REQUIRED", "false")

	configPath := filepath.Join(t.TempDir(), "config-test.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(strings.Join([]string{
		"openai:",
		"  apiKey: test-openai-key",
		"  model: gemini-2.5-flash",
		"  baseURL: https://api.example.test/v1",
		"  timeout: 30",
		"listingkit:",
		"  zitadel:",
		"    issuerURL: https://issuer.file.example",
	}, "\n")), 0o600))

	cfg, err := LoadConfigFromFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "https://issuer.file.example", cfg.ListingKit.Zitadel.IssuerURL)
	_, exposed := reflect.TypeOf(cfg.ListingKit).FieldByName("OwnerScopeRequired")
	assert.False(t, exposed, "real config loading must not expose either former owner scope environment variable")
}

func TestBuildConfigReadsMemberInvitationCredentials(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_MEMBER_INVITATION_TOKEN", "write-token")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID", "project-1")

	cfg := BuildConfig(newViper())

	assert.Equal(t, "write-token", cfg.ListingKit.Zitadel.MemberInvitationToken)
	assert.Equal(t, "project-1", cfg.ListingKit.Zitadel.ProjectID)
}

func TestBuildReadsZitadelAuthorizationAndWorkbenchConfig(t *testing.T) {
	t.Setenv("ZITADEL_ISSUER_URL", "https://issuer.example")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHORIZATION_API_URL", "https://authorization-proxy.example")
	t.Setenv("TASK_PROCESSOR_WORKBENCH_ENABLED", "true")

	cfg := BuildConfig(newViper())

	assert.Equal(t, "https://authorization-proxy.example", cfg.ListingKit.Zitadel.AuthorizationAPIURL)
	assert.True(t, cfg.Workbench.Enabled)
}

func TestBuildDefaultsZitadelAuthorizationAPIURLConfig(t *testing.T) {
	v := newViper()
	v.Set("listingkit.zitadel.issuerURL", "https://issuer.example")
	v.Set("listingkit.zitadel.authorizationAPIURL", "")

	cfg := BuildConfig(v)

	assert.Equal(t, "https://issuer.example", cfg.ListingKit.Zitadel.AuthorizationAPIURL)
}

func TestWorkbenchZitadelConfigDefaultsDisabled(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_WORKBENCH_ENABLED", "false")
	assert.False(t, BuildConfig(newViper()).Workbench.Enabled)
}

func TestLoadFromBytesRejectsEnabledWorkbenchWithoutCompleteZitadelConfig(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_WORKBENCH_ENABLED", "true")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID", "")
	t.Setenv("ZITADEL_ISSUER_URL", "https://issuer.example")

	_, err := LoadFromBytes([]byte(strings.Join([]string{
		"openai:",
		"  apiKey: test-openai-key",
		"  model: gemini-2.5-flash",
		"  baseURL: https://api.example.test/v1",
		"  timeout: 30",
	}, "\n")))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "listingkit.zitadel.projectID")
}

func TestBuildConfigReadsTencentSMSWebhookCredentials(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_SIGNING_KEY", "signing-key")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_ID", "secret-id")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_KEY", "secret-key")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_APP_ID", "1400000000")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SIGN_NAME", "ListingKit")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_TEMPLATE_ID", "100001")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONE_VERIFICATION_EXPIRY_MINUTES", "60")

	cfg := BuildConfig(newViper())

	assert.Equal(t, "signing-key", cfg.ListingKit.Zitadel.SMS.SigningKey)
	assert.Equal(t, "secret-id", cfg.ListingKit.Zitadel.SMS.TencentSecretID)
	assert.Equal(t, "secret-key", cfg.ListingKit.Zitadel.SMS.TencentSecretKey)
	assert.Equal(t, "1400000000", cfg.ListingKit.Zitadel.SMS.TencentAppID)
	assert.Equal(t, "ListingKit", cfg.ListingKit.Zitadel.SMS.TencentSignName)
	assert.Equal(t, "100001", cfg.ListingKit.Zitadel.SMS.TencentTemplateID)
	assert.Equal(t, 60, cfg.ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
}

func TestBuildConfigRejectsMalformedPhoneVerificationExpiry(t *testing.T) {
	const envKey = "TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONE_VERIFICATION_EXPIRY_MINUTES"
	for _, value := range []string{"abc", "1.5", "true", "999999999999999999999999999999999"} {
		t.Run("env/"+value, func(t *testing.T) {
			t.Setenv(envKey, value)
			require.Equal(t, -1, BuildConfig(newViper()).ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
		})
	}
	for _, value := range []string{"abc", "1.5", "1.0", "true", "[]", "{}", `""`} {
		t.Run("yaml/"+value, func(t *testing.T) {
			t.Setenv(envKey, "")
			v := newViper()
			v.SetConfigType("yaml")
			require.NoError(t, v.ReadConfig(strings.NewReader("listingkit:\n  zitadel:\n    sms:\n      phoneVerificationExpiryMinutes: "+value+"\n")))
			require.Equal(t, -1, BuildConfig(v).ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
		})
	}
}

func TestBuildConfigPreservesValidPhoneVerificationExpiry(t *testing.T) {
	const envKey = "TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONE_VERIFICATION_EXPIRY_MINUTES"
	for _, tc := range []struct {
		raw  string
		want int
	}{{"0", 0}, {"1", 1}, {"60", 60}, {`"60"`, 60}, {"-1", -1}, {"61", 61}} {
		t.Run(tc.raw, func(t *testing.T) {
			t.Setenv(envKey, "")
			v := newViper()
			v.SetConfigType("yaml")
			require.NoError(t, v.ReadConfig(strings.NewReader("listingkit:\n  zitadel:\n    sms:\n      phoneVerificationExpiryMinutes: "+tc.raw+"\n")))
			require.Equal(t, tc.want, BuildConfig(v).ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
		})
	}
	t.Setenv(envKey, "")
	require.Zero(t, BuildConfig(newViper()).ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
}

func TestDeprecatedEnvWarnings_ReportsLegacyAliases(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "legacy")

	warnings := deprecatedEnvWarnings()

	assert.NotEmpty(t, warnings)
	assert.Contains(t, strings.Join(warnings, "\n"), "OPENAI_API_KEY is deprecated")
}

func TestDeprecatedEnvWarnings_SkipsWhenPrimaryEnvSet(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "primary")
	t.Setenv("OPENAI_API_KEY", "legacy")

	warnings := deprecatedEnvWarnings()

	for _, w := range warnings {
		assert.NotContains(t, w, "OPENAI_API_KEY")
	}
}

func TestLoadDotEnvFile_PopulatesUnsetEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	err := os.WriteFile(envPath, []byte(strings.Join([]string{
		"# local development overrides",
		"TASK_PROCESSOR_OPENAI_API_KEY=dotenv-openai-key",
		"export TASK_PROCESSOR_OPENAI_BASE_URL=https://example.test/v1",
		"",
	}, "\n")), 0o600)
	require.NoError(t, err)

	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "")
	require.NoError(t, os.Unsetenv("TASK_PROCESSOR_OPENAI_API_KEY"))
	t.Setenv("TASK_PROCESSOR_OPENAI_BASE_URL", "")
	require.NoError(t, os.Unsetenv("TASK_PROCESSOR_OPENAI_BASE_URL"))

	require.NoError(t, loadDotEnvFile(envPath))

	assert.Equal(t, "dotenv-openai-key", os.Getenv("TASK_PROCESSOR_OPENAI_API_KEY"))
	assert.Equal(t, "https://example.test/v1", os.Getenv("TASK_PROCESSOR_OPENAI_BASE_URL"))
}

func TestLoadDotEnvFile_DoesNotOverrideExistingEnvironmentVariables(t *testing.T) {
	tempDir := t.TempDir()
	envPath := filepath.Join(tempDir, ".env")

	err := os.WriteFile(envPath, []byte("TASK_PROCESSOR_OPENAI_API_KEY=from-dotenv\n"), 0o600)
	require.NoError(t, err)

	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "from-process")

	require.NoError(t, loadDotEnvFile(envPath))

	assert.Equal(t, "from-process", os.Getenv("TASK_PROCESSOR_OPENAI_API_KEY"))
}

func TestLoadFromBytes_AppliesEnvironmentOverrides(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_OPENAI_API_KEY", "env-openai-key")
	t.Setenv("TASK_PROCESSOR_REDIS_HOST", "redis.env.local")
	cfg, err := LoadFromBytes([]byte(`
openai:
  apiKey: file-key
  model: gemini-2.5-flash
  baseURL: https://api.example.test/v1
  timeout: 30
  clients:
    vision:
      model: gemini-2.5-flash
      timeout: 30
redis:
  host: redis.file.local
`))
	require.NoError(t, err)
	assert.Equal(t, "env-openai-key", cfg.OpenAI.APIKey)
	assert.Equal(t, "env-openai-key", cfg.OpenAI.ToClientConfigs()["vision"].APIKey)
	require.NotNil(t, cfg.Redis)
	assert.Equal(t, "redis.env.local", cfg.Redis.Host)
}

func TestLoadFromBytes_AppliesViperManagedEnvironmentOverrides(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_BROWSER_HEADLESS", "false")
	t.Setenv("TASK_PROCESSOR_BROWSER_POOL_SIZE", "7")
	t.Setenv("TASK_PROCESSOR_REDIS_HOST", "redis.env.local")
	t.Setenv("TASK_PROCESSOR_REDIS_PORT", "6381")
	cfg, err := LoadFromBytes([]byte(`
openai:
  apiKey: test-openai-key
  model: gemini-2.5-flash
  baseURL: https://api.example.test/v1
  timeout: 30
browser:
  headless: true
  poolSize: 1
redis:
  host: redis.file.local
  port: 6379
`))
	require.NoError(t, err)
	assert.False(t, cfg.Browser.Headless)
	assert.Equal(t, 7, cfg.Browser.PoolSize)
	require.NotNil(t, cfg.Redis)
	assert.Equal(t, "redis.env.local", cfg.Redis.Host)
	assert.Equal(t, 6381, cfg.Redis.Port)
}

func TestConfigManagerLoad_UsesSameEnvironmentOverrideBehavior(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_BROWSER_HEADLESS", "false")
	t.Setenv("TASK_PROCESSOR_BROWSER_POOL_SIZE", "19")
	mgr := NewConfigManager(logrus.New())
	source := stubConfigSource{name: "memory", data: []byte(`
openai:
  apiKey: test-openai-key
  model: gemini-2.5-flash
  baseURL: https://api.example.test/v1
  timeout: 30
browser:
  headless: true
  poolSize: 1
`)}
	cfg, err := mgr.Load(source)
	require.NoError(t, err)
	assert.False(t, cfg.Browser.Headless)
	assert.Equal(t, 19, cfg.Browser.PoolSize)
}

func TestConfigFixturesDoNotCarryRetiredManagementConfig(t *testing.T) {
	content, err := os.ReadFile("config_env_test.go")
	require.NoError(t, err)

	retiredManagementBlock := "manage" + "ment:"
	require.NotContains(t, string(content), `"`+retiredManagementBlock+`"`)
}

func TestDotEnvCandidatesForConfig_PrioritizesScopedEnvFile(t *testing.T) {
	candidates := dotEnvCandidatesForConfig(filepath.Join("config", "config-amazon-crawler-api.yaml"))

	require.NotEmpty(t, candidates)
	assert.Equal(t, filepath.Clean(filepath.Join("config", ".env.config-amazon-crawler-api")), candidates[0])
	assert.Equal(t, filepath.Clean(filepath.Join("config", "config-amazon-crawler-api.env")), candidates[1])
}

func TestLoadDotEnvCandidates_LoadsScopedEnvBeforeSharedDotEnv(t *testing.T) {
	resetDotEnvLoadStateForTests()
	t.Cleanup(func() {
		resetDotEnvLoadStateForTests()
		_ = os.Unsetenv("TASK_PROCESSOR_PLATFORM_1688_ENABLED")
		_ = os.Unsetenv("TASK_PROCESSOR_OPENAI_API_KEY")
	})

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	sharedEnvPath := filepath.Join(tempDir, ".env")
	scopedEnvPath := filepath.Join(configDir, ".env.config-amazon-crawler-api")

	require.NoError(t, os.WriteFile(sharedEnvPath, []byte(strings.Join([]string{
		"TASK_PROCESSOR_PLATFORM_1688_ENABLED=true",
		"TASK_PROCESSOR_OPENAI_API_KEY=shared-key",
	}, "\n")), 0o600))
	require.NoError(t, os.WriteFile(scopedEnvPath, []byte(strings.Join([]string{
		"TASK_PROCESSOR_PLATFORM_1688_ENABLED=false",
	}, "\n")), 0o600))

	_ = os.Unsetenv("TASK_PROCESSOR_PLATFORM_1688_ENABLED")
	_ = os.Unsetenv("TASK_PROCESSOR_OPENAI_API_KEY")

	loadDotEnvCandidates([]string{scopedEnvPath, sharedEnvPath})

	assert.Equal(t, "false", os.Getenv("TASK_PROCESSOR_PLATFORM_1688_ENABLED"))
	assert.Equal(t, "shared-key", os.Getenv("TASK_PROCESSOR_OPENAI_API_KEY"))
}

func TestLoadConfigFromFile_AssemblesListingKitAndZitadelConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config-test.yaml")
	configBody := strings.Join([]string{
		"openai:",
		"  apiKey: \"test-openai-key\"",
		"  model: \"gemini-2.5-flash\"",
		"  baseURL: \"https://api.example.test/v1\"",
		"  timeout: 30",
		"listingkit:",
		"  platformAdminUsers: [\"user-a\"]",
		"  platformAdminRoles: [\"platform_admin\"]",
		"  zitadel:",
		"    issuerURL: \"https://issuer.file.example\"",
		"    clientID: \"file-client\"",
		"    clientSecret: \"file-secret\"",
		"    authorizationRequired: false",
	}, "\n")
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o600))

	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES", "listingkit_admin,platform_admin")
	t.Setenv("LISTINGKIT_PLATFORM_ADMIN_USERS", "user-b,user-c")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHZ_REQUIRED", "1")

	cfg, err := LoadConfigFromFile(configPath)
	require.NoError(t, err)

	assert.Equal(t, []string{"user-b", "user-c"}, cfg.ListingKit.PlatformAdminUsers)
	assert.Equal(t, []string{"platform_admin"}, cfg.ListingKit.PlatformAdminRoles)
	assert.True(t, cfg.ListingKit.Zitadel.AuthorizationRequired)
	assert.Equal(t, "https://issuer.file.example", cfg.ListingKit.Zitadel.IssuerURL)
	assert.Equal(t, "file-client", cfg.ListingKit.Zitadel.ClientID)
	assert.Equal(t, []string{"listingkit_admin", "platform_admin"}, cfg.ListingKit.Zitadel.AllowedRoles)
}

func TestLoadConfigFromFile_CannotDisableListingKitAuthentication(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config-test.yaml")
	configBody := strings.Join([]string{
		"openai:",
		"  apiKey: \"test-openai-key\"",
		"  model: \"gemini-2.5-flash\"",
		"  baseURL: \"https://api.example.test/v1\"",
		"  timeout: 30",
		"listingkit:",
		"  zitadel:",
		"    authorizationRequired: false",
	}, "\n")
	require.NoError(t, os.WriteFile(configPath, []byte(configBody), 0o600))

	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHZ_REQUIRED", "0")
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES", "listingkit_admin,platform_admin")

	cfg, err := LoadConfigFromFile(configPath)
	require.NoError(t, err)

	assert.False(t, cfg.ListingKit.Zitadel.AuthorizationRequired)
	assert.Equal(t, []string{"listingkit_admin", "platform_admin"}, cfg.ListingKit.Zitadel.AllowedRoles)
}
