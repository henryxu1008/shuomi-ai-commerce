package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	topupconfig "task-processor/internal/integration/wallettopup/config"
	"task-processor/internal/pkg/watermark"
	logger "task-processor/internal/platform/logging"

	"github.com/spf13/viper"
)

type envBinding struct {
	Primary    string
	Deprecated []string
}

var (
	loadDotEnvMu      sync.Mutex
	loadedDotEnvPaths = map[string]struct{}{}
)

type Config struct {
	StorageCapacity bool                `yaml:"storageCapacity"`
	Referrals       ReferralsConfig     `yaml:"-"`
	FeatureFlags    FeatureFlagsConfig  `yaml:"featureFlags"`
	Observability   ObservabilityConfig `yaml:"observability"`
	Logging         LoggingConfig       `yaml:"logging"`
	OpenAI          OpenAIConfig        `yaml:"openai"`
	Browser         BrowserConfig       `yaml:"browser"`
	// BrowserCollector is the opt-in wiring to the standalone anonymous public
	// 1688 browser collector process (design D13).
	BrowserCollector                                      BrowserCollectorConfig        `yaml:"browserCollector"`
	Amazon                                                AmazonConfig                  `yaml:"amazon"`
	Updater                                               UpdaterConfig                 `yaml:"updater"`
	Platforms                                             PlatformsConfig               `yaml:"platforms"`
	Watermark                                             *watermark.Config             `yaml:"watermark"`
	ImageAgent                                            ImageAgentConfig              `yaml:"imageagent"`
	ProductCollectionSourceMedia                          ImageAgentArtifactStoreConfig `yaml:"productCollectionSourceMedia"`
	ProductCollectionSourceMediaIsolatedTrial             bool                          `yaml:"productCollectionSourceMediaIsolatedTrial"`
	ProductCollectionSourceMediaIsolatedTrialDeploymentID string                        `yaml:"productCollectionSourceMediaIsolatedTrialDeploymentID"`
	Database                                              *DatabaseConfig               `yaml:"database"`
	// CommercialDatabase is optional for workers that settle observed AI
	// usage into the commercial owner.
	CommercialDatabase *DatabaseConfig `yaml:"commercialDatabase"`
	// CommercialOwnerDatabase is the explicit resource-balance owner pool;
	// it must not borrow the Token usage runtime role.
	CommercialOwnerDatabase *DatabaseConfig    `yaml:"commercialOwnerDatabase"`
	Redis                   *RedisConfig       `yaml:"redis"`
	Prompts                 PromptsConfig      `yaml:"prompts"`
	ListingKit              ListingKitConfig   `yaml:"listingkit"`
	Workbench               WorkbenchConfig    `yaml:"workbench"`
	WalletTopUp             topupconfig.Config `yaml:"walletTopUp"`
}

type FeatureFlagsConfig struct {
	Flags map[string]bool `yaml:"flags"`
}

type ObservabilityConfig struct {
	Tracing TracingConfig `yaml:"tracing"`
}

type TracingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	ServiceName string `yaml:"serviceName"`
	Endpoint    string `yaml:"endpoint"`
	Insecure    bool   `yaml:"insecure"`
}

type PromptsConfig struct {
	Dir       string `yaml:"dir"`
	HotReload bool   `yaml:"hotReload"`
}

type LoggingConfig struct {
	Level        string                   `yaml:"level"`
	Format       string                   `yaml:"format"`
	File         string                   `yaml:"file"`
	SplitByLevel []logger.LevelFileConfig `yaml:"split_by_level"`
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("TASK_PROCESSOR")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindKnownEnvs(v)
	setDefaults(v)
	return v
}

func tryLoadDotEnv() {
	loadDotEnvCandidates(dotEnvCandidates())
}

func tryLoadDotEnvForConfig(configFile string) {
	loadDotEnvCandidates(dotEnvCandidatesForConfig(configFile))
}

func dotEnvCandidates() []string {
	candidates := []string{
		".env",
		filepath.Join(".", ".env"),
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates,
			filepath.Join(exeDir, ".env"),
			filepath.Join(exeDir, "..", ".env"),
		)
	}

	return uniqueCleanPaths(candidates)
}

func dotEnvCandidatesForConfig(configFile string) []string {
	baseCandidates := dotEnvCandidates()
	if strings.TrimSpace(configFile) == "" {
		return baseCandidates
	}

	configDir := filepath.Dir(configFile)
	configBase := strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile))
	candidates := []string{
		filepath.Join(configDir, ".env."+configBase),
		filepath.Join(configDir, configBase+".env"),
	}
	candidates = append(candidates, baseCandidates...)

	return uniqueCleanPaths(candidates)
}

func uniqueCleanPaths(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		cleaned := filepath.Clean(candidate)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		result = append(result, cleaned)
	}
	return result
}

func loadDotEnvCandidates(candidates []string) {
	for _, candidate := range candidates {
		if loadDotEnvFileIfNeeded(candidate) {
			logger.GetGlobalLogger("core/config").Infof("loaded .env file: %s", candidate)
		}
	}
}

func loadDotEnvFileIfNeeded(path string) bool {
	cleaned := filepath.Clean(path)

	loadDotEnvMu.Lock()
	if _, ok := loadedDotEnvPaths[cleaned]; ok {
		loadDotEnvMu.Unlock()
		return false
	}
	loadDotEnvMu.Unlock()

	if err := loadDotEnvFile(cleaned); err != nil {
		return false
	}

	loadDotEnvMu.Lock()
	loadedDotEnvPaths[cleaned] = struct{}{}
	loadDotEnvMu.Unlock()
	return true
}

func resetDotEnvLoadStateForTests() {
	loadDotEnvMu.Lock()
	defer loadDotEnvMu.Unlock()
	loadedDotEnvPaths = map[string]struct{}{}
}

func loadDotEnvFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set env %s from %s: %w", key, path, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan .env file %s: %w", path, err)
	}

	return nil
}

func bindKnownEnvs(v *viper.Viper) {
	for key, binding := range knownEnvBindings() {
		args := []string{key, binding.Primary}
		args = append(args, binding.Deprecated...)
		_ = v.BindEnv(args...)
	}
}

func knownEnvBindings() map[string]envBinding {
	return map[string]envBinding{
		"observability.tracing.enabled": {
			Primary: "TASK_PROCESSOR_OBSERVABILITY_TRACING_ENABLED",
		},
		"observability.tracing.serviceName": {
			Primary: "TASK_PROCESSOR_OBSERVABILITY_TRACING_SERVICE_NAME",
		},
		"observability.tracing.endpoint": {
			Primary: "TASK_PROCESSOR_OBSERVABILITY_TRACING_ENDPOINT",
		},
		"observability.tracing.insecure": {
			Primary: "TASK_PROCESSOR_OBSERVABILITY_TRACING_INSECURE",
		},
		"featureFlags.flags.product-listing-runtime-auto-migrate": {
			Primary: "TASK_PROCESSOR_API_RUNTIME_AUTOMIGRATE",
		},
		"imageagent.admission.enabled": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ADMISSION_ENABLED",
		},
		"imageagent.admission.allowedTenantIDs": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ADMISSION_ALLOWED_TENANT_IDS",
		},
		"imageagent.artifactStore.enabled": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_ENABLED",
		},
		"imageagent.artifactStore.provider": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PROVIDER",
		},
		"imageagent.artifactStore.publicBase": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PUBLIC_BASE",
		},
		"imageagent.artifactStore.s3.bucket": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_BUCKET",
		},
		"imageagent.artifactStore.s3.region": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_REGION",
		},
		"imageagent.artifactStore.s3.endpoint": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ENDPOINT",
		},
		"imageagent.artifactStore.s3.accessKeyID": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ACCESS_KEY_ID",
		},
		"imageagent.artifactStore.s3.secretAccessKey": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_SECRET_ACCESS_KEY",
		},
		"imageagent.artifactStore.s3.usePathStyle": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_USE_PATH_STYLE",
		},
		"imageagent.artifactStore.s3.artifactMode": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ARTIFACT_MODE",
		},
		"imageagent.artifactStore.s3.cosImmutableNonVersionedBucketPolicy": {
			Primary: "TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_COS_IMMUTABLE_NON_VERSIONED_BUCKET_POLICY",
		},
		"openai.apiKey": {
			Primary:    "TASK_PROCESSOR_OPENAI_API_KEY",
			Deprecated: []string{"OPENAI_API_KEY"},
		},
		"openai.model": {
			Primary:    "TASK_PROCESSOR_OPENAI_MODEL",
			Deprecated: []string{"OPENAI_MODEL"},
		},
		"openai.baseURL": {
			Primary:    "TASK_PROCESSOR_OPENAI_BASE_URL",
			Deprecated: []string{"OPENAI_BASE_URL"},
		},
		"openai.clients.image.apiKey": {
			Primary:    "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY",
			Deprecated: []string{"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APIKEY"},
		},
		"openai.clients.image.baseURL": {
			Primary:    "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL",
			Deprecated: []string{"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASEURL"},
		},
		"openai.clients.image.apiStyle": {
			Primary:    "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE",
			Deprecated: []string{"TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_APISTYLE"},
		},
		"openai.clients.image.model": {
			Primary:    "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL",
			Deprecated: nil,
		},
		"openai.clients.image.timeout": {
			Primary:    "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_TIMEOUT",
			Deprecated: nil,
		},
		"amazon.spapi.clientID": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_CLIENT_ID",
			Deprecated: []string{"AMAZON_SPAPI_CLIENT_ID"},
		},
		"amazon.spapi.clientSecret": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_CLIENT_SECRET",
			Deprecated: []string{"AMAZON_SPAPI_CLIENT_SECRET"},
		},
		"amazon.spapi.refreshToken": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_REFRESH_TOKEN",
			Deprecated: []string{"AMAZON_SPAPI_REFRESH_TOKEN"},
		},
		"amazon.spapi.region": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_REGION",
			Deprecated: []string{"AMAZON_SPAPI_REGION"},
		},
		"amazon.spapi.defaultMarketplace": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_DEFAULT_MARKETPLACE",
			Deprecated: []string{"TASK_PROCESSOR_AMAZON_SPAPI_MARKETPLACE_ID", "AMAZON_SPAPI_DEFAULT_MARKETPLACE", "AMAZON_SPAPI_MARKETPLACE_ID"},
		},
		"amazon.spapi.defaultFulfillmentType": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_DEFAULT_FULFILLMENT_TYPE",
			Deprecated: []string{"AMAZON_SPAPI_DEFAULT_FULFILLMENT_TYPE"},
		},
		"amazon.spapi.defaultCondition": {
			Primary:    "TASK_PROCESSOR_AMAZON_SPAPI_DEFAULT_CONDITION",
			Deprecated: []string{"AMAZON_SPAPI_DEFAULT_CONDITION"},
		},
		"amazon.remoteAPI.enabled": {
			Primary:    "TASK_PROCESSOR_AMAZON_REMOTE_API_ENABLED",
			Deprecated: []string{"AMAZON_REMOTE_API_ENABLED"},
		},
		"amazon.remoteAPI.baseURL": {
			Primary:    "TASK_PROCESSOR_AMAZON_REMOTE_API_BASE_URL",
			Deprecated: []string{"AMAZON_REMOTE_API_BASE_URL"},
		},
		"amazon.remoteAPI.timeout": {
			Primary:    "TASK_PROCESSOR_AMAZON_REMOTE_API_TIMEOUT",
			Deprecated: []string{"AMAZON_REMOTE_API_TIMEOUT"},
		},
		"browser.browserPath": {
			Primary:    "TASK_PROCESSOR_BROWSER_PATH",
			Deprecated: []string{"BROWSER_PATH"},
		},
		"browser.userDataDir": {
			Primary:    "TASK_PROCESSOR_BROWSER_USER_DATA_DIR",
			Deprecated: []string{"BROWSER_USER_DATA_DIR"},
		},
		"browser.headless": {
			Primary:    "TASK_PROCESSOR_BROWSER_HEADLESS",
			Deprecated: []string{"BROWSER_HEADLESS"},
		},
		"browser.poolSize": {
			Primary:    "TASK_PROCESSOR_BROWSER_POOL_SIZE",
			Deprecated: []string{"BROWSER_POOL_SIZE"},
		},
		"platforms.alibaba1688.enabled": {
			Primary:    "TASK_PROCESSOR_PLATFORM_1688_ENABLED",
			Deprecated: []string{"PLATFORM_1688_ENABLED"},
		},
		"database.host": {
			Primary:    "TASK_PROCESSOR_DATABASE_HOST",
			Deprecated: []string{"DB_HOST"},
		},
		"database.port": {
			Primary:    "TASK_PROCESSOR_DATABASE_PORT",
			Deprecated: []string{"DB_PORT"},
		},
		"database.user": {
			Primary:    "TASK_PROCESSOR_DATABASE_USER",
			Deprecated: []string{"DB_USER"},
		},
		"database.password": {
			Primary:    "TASK_PROCESSOR_DATABASE_PASSWORD",
			Deprecated: []string{"DB_PASSWORD"},
		},
		"database.database": {
			Primary:    "TASK_PROCESSOR_DATABASE_NAME",
			Deprecated: []string{"DB_NAME"},
		},
		"database.max_connections": {
			Primary:    "TASK_PROCESSOR_DATABASE_MAX_CONNECTIONS",
			Deprecated: []string{"DB_MAX_CONNECTIONS"},
		},
		"database.max_idle_connections": {
			Primary:    "TASK_PROCESSOR_DATABASE_MAX_IDLE_CONNECTIONS",
			Deprecated: []string{"DB_MAX_IDLE_CONNECTIONS"},
		},
		"commercialDatabase.host": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_HOST",
		},
		"commercialDatabase.port": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_PORT",
		},
		"commercialDatabase.user": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_USER",
		},
		"commercialDatabase.password": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_PASSWORD",
		},
		"commercialDatabase.database": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_NAME",
		},
		"commercialDatabase.max_connections": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_MAX_CONNECTIONS",
		},
		"commercialDatabase.max_idle_connections": {
			Primary: "TASK_PROCESSOR_COMMERCIAL_DATABASE_MAX_IDLE_CONNECTIONS",
		},
		"redis.host": {
			Primary:    "TASK_PROCESSOR_REDIS_HOST",
			Deprecated: []string{"REDIS_HOST"},
		},
		"redis.port": {
			Primary:    "TASK_PROCESSOR_REDIS_PORT",
			Deprecated: []string{"REDIS_PORT"},
		},
		"redis.password": {
			Primary:    "TASK_PROCESSOR_REDIS_PASSWORD",
			Deprecated: []string{"REDIS_PASSWORD"},
		},
		"redis.db": {
			Primary:    "TASK_PROCESSOR_REDIS_DB",
			Deprecated: []string{"REDIS_DB"},
		},
		"redis.pool_size": {
			Primary:    "TASK_PROCESSOR_REDIS_POOL_SIZE",
			Deprecated: []string{"REDIS_POOL_SIZE"},
		},
		"platforms.shein.cookieRedis.host": {
			Primary: "TASK_PROCESSOR_SHEIN_COOKIE_REDIS_HOST",
		},
		"platforms.shein.cookieRedis.port": {
			Primary: "TASK_PROCESSOR_SHEIN_COOKIE_REDIS_PORT",
		},
		"platforms.shein.cookieRedis.password": {
			Primary: "TASK_PROCESSOR_SHEIN_COOKIE_REDIS_PASSWORD",
		},
		"platforms.shein.cookieRedis.db": {
			Primary: "TASK_PROCESSOR_SHEIN_COOKIE_REDIS_DB",
		},
		"platforms.shein.cookieRedis.pool_size": {
			Primary: "TASK_PROCESSOR_SHEIN_COOKIE_REDIS_POOL_SIZE",
		},
		"platforms.shein.loginService.tenantID": {
			Primary: "TASK_PROCESSOR_SHEIN_LOGIN_SERVICE_TENANT_ID",
		},
		"platforms.shein.loginService.identifier": {
			Primary: "TASK_PROCESSOR_SHEIN_LOGIN_SERVICE_IDENTIFIER",
		},
		"platforms.shein.loginService.defaultHeadless": {
			Primary: "TASK_PROCESSOR_SHEIN_LOGIN_SERVICE_DEFAULT_HEADLESS",
		},
		"platforms.shein.loginService.forceHeadless": {
			Primary: "TASK_PROCESSOR_SHEIN_LOGIN_SERVICE_FORCE_HEADLESS",
		},
		"platforms.shein.loginService.executionMode": {
			Primary: "TASK_PROCESSOR_SHEIN_LOGIN_SERVICE_EXECUTION_MODE",
		},
		"platforms.sds.loginService.baseURL": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_BASE_URL",
		},
		"platforms.sds.loginService.sharedKey": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_SHARED_KEY",
		},
		"platforms.sds.loginService.tenantID": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_TENANT_ID",
		},
		"platforms.sds.loginService.identifier": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_IDENTIFIER",
		},
		"platforms.sds.loginService.merchantName": {
			Primary: "TASK_PROCESSOR_SDS_MERCHANT_NAME",
		},
		"platforms.sds.loginService.username": {
			Primary: "TASK_PROCESSOR_SDS_USERNAME",
		},
		"platforms.sds.loginService.password": {
			Primary: "TASK_PROCESSOR_SDS_PASSWORD",
		},
		"platforms.sds.loginService.defaultHeadless": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_DEFAULT_HEADLESS",
		},
		"platforms.sds.loginService.cloakBrowserEnabled": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_CLOAKBROWSER_ENABLED",
		},
		"platforms.sds.loginService.cloakBrowserPath": {
			Primary: "TASK_PROCESSOR_SDS_LOGIN_SERVICE_CLOAKBROWSER_PATH",
		},
		"platforms.sds.authRedis.host": {
			Primary: "TASK_PROCESSOR_SDS_AUTH_REDIS_HOST",
		},
		"platforms.sds.authRedis.port": {
			Primary: "TASK_PROCESSOR_SDS_AUTH_REDIS_PORT",
		},
		"platforms.sds.authRedis.password": {
			Primary: "TASK_PROCESSOR_SDS_AUTH_REDIS_PASSWORD",
		},
		"platforms.sds.authRedis.db": {
			Primary: "TASK_PROCESSOR_SDS_AUTH_REDIS_DB",
		},
		"platforms.sds.authRedis.pool_size": {
			Primary: "TASK_PROCESSOR_SDS_AUTH_REDIS_POOL_SIZE",
		},
		"platforms.sds.authBootstrap.staticAccessToken": {
			Primary: "TASK_PROCESSOR_SDS_ACCESS_TOKEN",
		},
		"platforms.sds.authBootstrap.staticOutToken": {
			Primary: "TASK_PROCESSOR_SDS_OUT_ACCESS_TOKEN",
		},
		"platforms.sds.authBootstrap.staticMerchantID": {
			Primary: "TASK_PROCESSOR_SDS_MERCHANT_ID",
		},
		"platforms.sds.authBootstrap.staticCookie": {
			Primary: "TASK_PROCESSOR_SDS_COOKIE",
		},
		"platforms.sds.authBootstrap.loginDomainName": {
			Primary: "TASK_PROCESSOR_SDS_DOMAIN_NAME",
		},
		"platforms.sds.authBootstrap.loginVerifyCaptchaParam": {
			Primary: "TASK_PROCESSOR_SDS_VERIFY_CAPTCHA_PARAM",
		},
		"platforms.sds.authBootstrap.loginExtraInfo": {
			Primary: "TASK_PROCESSOR_SDS_EXTRA_INFO",
		},
		"listingkit.imageUpload.provider": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_PROVIDER",
		},
		"listingkit.imageUpload.local.rootDir": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_LOCAL_ROOT_DIR",
		},
		"listingkit.imageUpload.s3.bucket": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_BUCKET",
		},
		"listingkit.imageUpload.s3.region": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_REGION",
		},
		"listingkit.imageUpload.s3.endpoint": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_ENDPOINT",
		},
		"listingkit.imageUpload.s3.accessKeyID": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_ACCESS_KEY_ID",
		},
		"listingkit.imageUpload.s3.secretAccessKey": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_SECRET_ACCESS_KEY",
		},
		"listingkit.imageUpload.s3.usePathStyle": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_USE_PATH_STYLE",
		},
		"listingkit.imageUpload.s3.publicBase": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_PUBLIC_BASE",
		},
		"listingkit.platformAdminUsers": {
			Primary: "LISTINGKIT_PLATFORM_ADMIN_USERS",
		},
		"listingkit.platformAdminRoles": {
			Primary: "LISTINGKIT_PLATFORM_ADMIN_ROLES",
		},
		"listingkit.zitadel.issuerURL": {
			Primary: "ZITADEL_ISSUER_URL",
		},
		"listingkit.zitadel.authorizationAPIURL": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHORIZATION_API_URL",
		},
		"listingkit.zitadel.clientID": {
			Primary: "ZITADEL_CLIENT_ID",
		},
		"listingkit.zitadel.clientSecret": {
			Primary: "ZITADEL_CLIENT_SECRET",
		},
		"listingkit.zitadel.tenantDirectoryToken": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_TENANT_DIRECTORY_TOKEN",
		},
		"listingkit.zitadel.memberInvitationToken": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_MEMBER_INVITATION_TOKEN",
		},
		"listingkit.zitadel.projectID": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_PROJECT_ID",
		},
		"listingkit.zitadel.sms.signingKey": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_SIGNING_KEY",
		},
		"listingkit.zitadel.sms.tencentSecretID": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_ID",
		},
		"listingkit.zitadel.sms.tencentSecretKey": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SECRET_KEY",
		},
		"listingkit.zitadel.sms.tencentAppID": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_APP_ID",
		},
		"listingkit.zitadel.sms.tencentSignName": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_SIGN_NAME",
		},
		"listingkit.zitadel.sms.tencentTemplateID": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_TENCENT_SMS_TEMPLATE_ID",
		},
		"listingkit.zitadel.sms.phoneVerificationExpiryMinutes": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_SMS_PHONE_VERIFICATION_EXPIRY_MINUTES",
		},
		"listingkit.zitadel.authorizationRequired": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_AUTHZ_REQUIRED",
		},
		"listingkit.zitadel.allowedTenantIDs": {
			Primary:    "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS",
			Deprecated: []string{"LISTINGKIT_ZITADEL_ALLOWED_TENANT_IDS"},
		},
		"listingkit.zitadel.allowedUserIDs": {
			Primary:    "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USER_IDS",
			Deprecated: []string{"LISTINGKIT_ZITADEL_ALLOWED_USER_IDS"},
		},
		"listingkit.zitadel.allowedUsernames": {
			Primary:    "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USERNAMES",
			Deprecated: []string{"LISTINGKIT_ZITADEL_ALLOWED_USERNAMES"},
		},
		"listingkit.zitadel.allowedRoles": {
			Primary:    "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_ROLES",
			Deprecated: []string{"LISTINGKIT_ZITADEL_ALLOWED_ROLES"},
		},
		"workbench.enabled": {
			Primary: "TASK_PROCESSOR_WORKBENCH_ENABLED",
		},
	}
}

func deprecatedEnvWarnings() []string {
	var warnings []string

	for key, binding := range knownEnvBindings() {
		_, primarySet := os.LookupEnv(binding.Primary)
		for _, deprecated := range binding.Deprecated {
			_, deprecatedSet := os.LookupEnv(deprecated)
			if !deprecatedSet {
				continue
			}
			// 已设置新名称时不再提示旧别名（避免系统环境仍带 REDIS_* 等与项目 TASK_PROCESSOR_* 并存时刷屏）
			if primarySet {
				continue
			}

			warning := fmt.Sprintf("environment variable %s is deprecated; use %s for %s instead", deprecated, binding.Primary, key)
			warnings = append(warnings, warning)
		}
	}

	return warnings
}

func logDeprecatedEnvUsage() {
	logger := logger.GetGlobalLogger("core/config")
	for _, warning := range deprecatedEnvWarnings() {
		logger.Warn(warning)
	}
}

func loadWithViper(v *viper.Viper) (*Config, error) {
	if err := rejectRetiredProductRuntimeConfig(v); err != nil {
		return nil, err
	}
	cfg := BuildConfig(v)
	if err := cfg.ValidateWithError(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadWithViperWithoutValidation(v *viper.Viper) (*Config, error) {
	if err := rejectRetiredProductRuntimeConfig(v); err != nil {
		return nil, err
	}
	return BuildConfig(v), nil
}

func LoadConfig() (*Config, error) {
	tryLoadDotEnv()

	env := os.Getenv("TASK_PROCESSOR_ENV")
	if env == "" {
		env = "dev"
	}

	configName := fmt.Sprintf("config-%s", env)
	logger.GetGlobalLogger("core/config").Infof("loading config file: %s.yaml", configName)
	logDeprecatedEnvUsage()

	v := newViper()
	v.SetConfigName(configName)

	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		v.AddConfigPath(filepath.Join(exeDir, "config"))
		v.AddConfigPath(exeDir)
	}

	v.AddConfigPath("./config")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/task-processor/")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config %s.yaml: %w", configName, err)
	}

	logger.GetGlobalLogger("core/config").Infof("loaded config file: %s", v.ConfigFileUsed())
	return loadWithViper(v)
}

func LoadConfigFromFile(configFile string) (*Config, error) {
	tryLoadDotEnvForConfig(configFile)

	logger.GetGlobalLogger("core/config").Infof("loading config file: %s", configFile)
	logDeprecatedEnvUsage()

	v := newViper()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", configFile, err)
	}

	logger.GetGlobalLogger("core/config").Infof("loaded config file: %s", v.ConfigFileUsed())
	return loadWithViper(v)
}

func LoadConfigFromFileWithoutValidation(configFile string) (*Config, error) {
	tryLoadDotEnvForConfig(configFile)

	logger.GetGlobalLogger("core/config").Infof("loading config file: %s", configFile)
	logDeprecatedEnvUsage()

	v := newViper()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", configFile, err)
	}

	logger.GetGlobalLogger("core/config").Infof("loaded config file: %s", v.ConfigFileUsed())
	return loadWithViperWithoutValidation(v)
}
