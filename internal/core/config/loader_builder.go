package config

import (
	"os"
	"strconv"
	"strings"
	topupconfig "task-processor/internal/integration/wallettopup/config"
	logger "task-processor/internal/platform/logging"
	"time"

	"github.com/spf13/viper"
)

func BuildConfig(v *viper.Viper) *Config {
	listingKitAllowedTenantIDs := getStringSlice(v, "listingkit.zitadel.allowedTenantIDs")
	listingKitAllowedUserIDs := getStringSlice(v, "listingkit.zitadel.allowedUserIDs")
	listingKitAllowedUsernames := getStringSlice(v, "listingkit.zitadel.allowedUsernames")
	listingKitLegacyUsernameAllowlistConfigured := legacyListingKitUsernameAllowlistConfigured(v)
	listingKitAllowedRoles := getStringSlice(v, "listingkit.zitadel.allowedRoles")
	listingKitZitadelIssuerURL := v.GetString("listingkit.zitadel.issuerURL")
	listingKitZitadelAuthorizationAPIURL := v.GetString("listingkit.zitadel.authorizationAPIURL")
	if strings.TrimSpace(listingKitZitadelAuthorizationAPIURL) == "" {
		listingKitZitadelAuthorizationAPIURL = listingKitZitadelIssuerURL
	}

	cfg := &Config{
		StorageCapacity: v.GetBool("storageCapacity"),
		FeatureFlags:    buildFeatureFlagsConfig(v),
		Observability: ObservabilityConfig{
			Tracing: TracingConfig{
				Enabled:     v.GetBool("observability.tracing.enabled"),
				ServiceName: v.GetString("observability.tracing.serviceName"),
				Endpoint:    v.GetString("observability.tracing.endpoint"),
				Insecure:    v.GetBool("observability.tracing.insecure"),
			},
		},
		OpenAI: OpenAIConfig{
			APIKey:  v.GetString("openai.apiKey"),
			Model:   v.GetString("openai.model"),
			BaseURL: v.GetString("openai.baseURL"),
			Timeout: v.GetInt("openai.timeout"),
			Clients: buildOpenAIClients(v),
		},
		Platforms: PlatformsConfig{
			Temu:  BuildPlatformConfig(v, "platforms.temu"),
			Shein: BuildPlatformConfig(v, "platforms.shein"),
			SDS: SDSPlatformConfig{
				LoginService: BuildLoginServiceConfig(v, "platforms.sds.loginService"),
				AuthRedis: RedisConfig{
					Host:     v.GetString("platforms.sds.authRedis.host"),
					Port:     v.GetInt("platforms.sds.authRedis.port"),
					Password: v.GetString("platforms.sds.authRedis.password"),
					DB:       v.GetInt("platforms.sds.authRedis.db"),
					PoolSize: v.GetInt("platforms.sds.authRedis.pool_size"),
				},
				AuthBootstrap: BuildSDSAuthBootstrapConfig(v, "platforms.sds.authBootstrap"),
			},
			Alibaba1688: Alibaba1688Config{
				Enabled:        v.GetBool("platforms.alibaba1688.enabled"),
				Timeout:        v.GetInt("platforms.alibaba1688.timeout"),
				PoolSize:       v.GetInt("platforms.alibaba1688.poolSize"),
				ProfileRootDir: v.GetString("platforms.alibaba1688.profileRootDir"),
			},
		},
		Browser: BrowserConfig{
			Enabled:        v.GetBool("browser.enabled"),
			Headless:       v.GetBool("browser.headless"),
			BrowserPath:    v.GetString("browser.browserPath"),
			UserDataDir:    v.GetString("browser.userDataDir"),
			PoolSize:       v.GetInt("browser.poolSize"),
			ViewportWidth:  v.GetInt("browser.viewportWidth"),
			ViewportHeight: v.GetInt("browser.viewportHeight"),
			ProxyServer:    v.GetString("browser.proxyServer"),
			RandomConfig: BrowserRandomConfig{
				Enabled:             v.GetBool("browser.randomConfig.enabled"),
				Strategy:            v.GetString("browser.randomConfig.strategy"),
				PresetName:          v.GetString("browser.randomConfig.presetName"),
				FingerprintStrategy: v.GetString("browser.randomConfig.fingerprintStrategy"),
				HealthCheckEnabled:  v.GetBool("browser.randomConfig.healthCheckEnabled"),
				MaxRetries:          v.GetInt("browser.randomConfig.maxRetries"),
				MaxUsesPerInstance:  v.GetInt("browser.randomConfig.maxUsesPerInstance"),
			},
		},
		Amazon: AmazonConfig{
			Enabled:      v.GetBool("amazon.enabled"),
			Zipcodes:     v.GetStringMapString("amazon.zipcodes"),
			CrawlTimeout: v.GetInt("amazon.crawlTimeout"),
			FailureArtifacts: FailureArtifactsConfig{
				Enabled:      v.GetBool("amazon.failureArtifacts.enabled"),
				Directory:    v.GetString("amazon.failureArtifacts.directory"),
				CaptureHTML:  v.GetBool("amazon.failureArtifacts.captureHTML"),
				MaxHTMLBytes: v.GetInt("amazon.failureArtifacts.maxHTMLBytes"),
			},
			RiskControl: AmazonRiskControlConfig{
				CaptchaRecreateThreshold:        v.GetInt("amazon.riskControl.captchaRecreateThreshold"),
				AuthenticationRecreateThreshold: v.GetInt("amazon.riskControl.authenticationRecreateThreshold"),
				BrowserCrashRecreateThreshold:   v.GetInt("amazon.riskControl.browserCrashRecreateThreshold"),
				TimeoutRecreateThreshold:        v.GetInt("amazon.riskControl.timeoutRecreateThreshold"),
				NetworkRecreateThreshold:        v.GetInt("amazon.riskControl.networkRecreateThreshold"),
				ServerErrorRecreateThreshold:    v.GetInt("amazon.riskControl.serverErrorRecreateThreshold"),
			},
			QualityControl: AmazonQualityControlConfig{
				RetryOnValidationFailure:   v.GetBool("amazon.qualityControl.retryOnValidationFailure"),
				ValidationRetryMaxAttempts: v.GetInt("amazon.qualityControl.validationRetryMaxAttempts"),
			},
			ProxyPool: AmazonProxyPoolConfig{
				Enabled:                v.GetBool("amazon.proxyPool.enabled"),
				Strategy:               v.GetString("amazon.proxyPool.strategy"),
				FailureCooldownSeconds: v.GetInt("amazon.proxyPool.failureCooldownSeconds"),
				ProxyFile:              v.GetString("amazon.proxyPool.proxyFile"),
				Proxies: uniqueStringList(
					getStringSlice(v, "amazon.proxyPool.proxies"),
					readLineList(v.GetString("amazon.proxyPool.proxyFile")),
				),
			},
		},
		ImageAgent: ImageAgentConfig{
			Generation: ImageAgentGenerationConfig{
				PriceVersion:   v.GetString("imageagent.generation.priceVersion"),
				PointsPerImage: v.GetInt64("imageagent.generation.pointsPerImage"),
			},
			Admission: ImageAgentAdmissionConfig{
				Enabled:          v.GetBool("imageagent.admission.enabled"),
				AllowedTenantIDs: getStringSlice(v, "imageagent.admission.allowedTenantIDs"),
			},
			ArtifactStore: ImageAgentArtifactStoreConfig{
				IsolatedTrialGeneratedURLs: v.GetBool("imageagent.artifactStore.isolatedTrialGeneratedURLs"),
				Enabled:                    v.GetBool("imageagent.artifactStore.enabled"),
				Provider:                   v.GetString("imageagent.artifactStore.provider"),
				PublicBase:                 v.GetString("imageagent.artifactStore.publicBase"),
				S3: ImageAgentArtifactStoreS3Config{
					CapacityIdentityProvider:             v.GetString("imageagent.artifactStore.s3.capacityIdentityProvider"),
					CapacityMinIODeploymentID:            v.GetString("imageagent.artifactStore.s3.capacityMinIODeploymentID"),
					Bucket:                               v.GetString("imageagent.artifactStore.s3.bucket"),
					Region:                               v.GetString("imageagent.artifactStore.s3.region"),
					Endpoint:                             v.GetString("imageagent.artifactStore.s3.endpoint"),
					AccessKeyID:                          v.GetString("imageagent.artifactStore.s3.accessKeyID"),
					SecretAccessKey:                      v.GetString("imageagent.artifactStore.s3.secretAccessKey"),
					UsePathStyle:                         v.GetBool("imageagent.artifactStore.s3.usePathStyle"),
					ArtifactMode:                         v.GetString("imageagent.artifactStore.s3.artifactMode"),
					COSImmutableNonVersionedBucketPolicy: v.GetBool("imageagent.artifactStore.s3.cosImmutableNonVersionedBucketPolicy"),
				},
			},
		},
		Updater: UpdaterConfig{
			Enabled:            v.GetBool("updater.enabled"),
			UpdateURL:          v.GetString("updater.updateURL"),
			CheckInterval:      v.GetInt("updater.checkInterval"),
			InsecureSkipVerify: v.GetBool("updater.insecureSkipVerify"),
		},
		ListingKit: ListingKitConfig{
			RolePolicyDatabase: rolePolicyDatabase(v),
			PlatformAdminUsers: getStringSlice(v, "listingkit.platformAdminUsers"),
			PlatformAdminRoles: getStringSlice(v, "listingkit.platformAdminRoles"),
			ImageUpload: ListingKitImageUploadConfig{
				Provider: v.GetString("listingkit.imageUpload.provider"),
				Local:    ListingKitImageUploadLocalConfig{RootDir: v.GetString("listingkit.imageUpload.local.rootDir")},
				S3: ListingKitImageUploadS3Config{
					Bucket: v.GetString("listingkit.imageUpload.s3.bucket"), Region: v.GetString("listingkit.imageUpload.s3.region"),
					Endpoint: v.GetString("listingkit.imageUpload.s3.endpoint"), AccessKeyID: v.GetString("listingkit.imageUpload.s3.accessKeyID"),
					SecretAccessKey: v.GetString("listingkit.imageUpload.s3.secretAccessKey"), UsePathStyle: v.GetBool("listingkit.imageUpload.s3.usePathStyle"),
					PublicBase: v.GetString("listingkit.imageUpload.s3.publicBase"),
				},
			},
			Zitadel: ListingKitZitadelConfig{
				IssuerURL:                         listingKitZitadelIssuerURL,
				AuthorizationAPIURL:               listingKitZitadelAuthorizationAPIURL,
				ClientID:                          v.GetString("listingkit.zitadel.clientID"),
				ClientSecret:                      v.GetString("listingkit.zitadel.clientSecret"),
				TenantDirectoryToken:              v.GetString("listingkit.zitadel.tenantDirectoryToken"),
				MemberInvitationToken:             v.GetString("listingkit.zitadel.memberInvitationToken"),
				ProjectID:                         v.GetString("listingkit.zitadel.projectID"),
				AuthorizationRequired:             v.GetBool("listingkit.zitadel.authorizationRequired"),
				AllowedTenantIDs:                  listingKitAllowedTenantIDs,
				AllowedUserIDs:                    listingKitAllowedUserIDs,
				AllowedUsernames:                  listingKitAllowedUsernames,
				LegacyUsernameAllowlistConfigured: listingKitLegacyUsernameAllowlistConfigured,
				AllowedRoles:                      listingKitAllowedRoles,
				SMS: ListingKitZitadelSMSConfig{
					SigningKey:                     v.GetString("listingkit.zitadel.sms.signingKey"),
					TencentSecretID:                v.GetString("listingkit.zitadel.sms.tencentSecretID"),
					TencentSecretKey:               v.GetString("listingkit.zitadel.sms.tencentSecretKey"),
					TencentAppID:                   v.GetString("listingkit.zitadel.sms.tencentAppID"),
					TencentSignName:                v.GetString("listingkit.zitadel.sms.tencentSignName"),
					TencentTemplateID:              v.GetString("listingkit.zitadel.sms.tencentTemplateID"),
					PhoneVerificationExpiryMinutes: phoneVerificationExpiryMinutes(v),
				},
			},
		},
		Workbench: WorkbenchConfig{
			Enabled: v.GetBool("workbench.enabled"),
		},
		WalletTopUp: topupconfig.Load(v),
	}

	cfg.Database = databaseConfigFromViper(v, "database")
	cfg.CommercialDatabase = databaseConfigFromViper(v, "commercialDatabase")
	cfg.CommercialOwnerDatabase = databaseConfigFromViper(v, "commercialOwnerDatabase")

	if v.GetString("redis.host") != "" {
		port := v.GetInt("redis.port")
		if port == 0 {
			port = 6379
		}
		poolSize := v.GetInt("redis.pool_size")
		if poolSize == 0 {
			poolSize = 10
		}
		cfg.Redis = &RedisConfig{
			Host:     v.GetString("redis.host"),
			Port:     port,
			Password: v.GetString("redis.password"),
			DB:       v.GetInt("redis.db"),
			PoolSize: poolSize,
		}
	}

	cfg.Logging = LoggingConfig{
		Level:        v.GetString("logging.level"),
		Format:       v.GetString("logging.format"),
		File:         v.GetString("logging.file"),
		SplitByLevel: buildSplitByLevelConfig(v),
	}

	return cfg
}

func databaseConfigFromViper(v *viper.Viper, prefix string) *DatabaseConfig {
	if v == nil || strings.TrimSpace(v.GetString(prefix+".host")) == "" {
		return nil
	}
	return &DatabaseConfig{
		Host:                  v.GetString(prefix + ".host"),
		Port:                  v.GetInt(prefix + ".port"),
		User:                  v.GetString(prefix + ".user"),
		Password:              v.GetString(prefix + ".password"),
		Database:              v.GetString(prefix + ".database"),
		MaxConnections:        v.GetInt(prefix + ".max_connections"),
		MaxIdleConnections:    v.GetInt(prefix + ".max_idle_connections"),
		ConnectionMaxLifetime: time.Duration(v.GetInt64(prefix + ".connection_max_lifetime")),
	}
}

func buildFeatureFlagsConfig(v *viper.Viper) FeatureFlagsConfig {
	const (
		flagsConfigKey = "featureFlags.flags"
		autoMigrateKey = "product-listing-runtime-auto-migrate"
	)

	rawFlags := v.GetStringMap(flagsConfigKey)
	flags := make(map[string]bool, len(rawFlags)+1)
	for key := range rawFlags {
		flags[key] = v.GetBool(flagsConfigKey + "." + key)
	}
	flags[autoMigrateKey] = v.GetBool(flagsConfigKey + "." + autoMigrateKey)

	return FeatureFlagsConfig{Flags: flags}
}

func legacyListingKitUsernameAllowlistConfigured(v *viper.Viper) bool {
	const key = "listingkit.zitadel.allowedUsernames"
	if v == nil {
		return false
	}
	if v.InConfig(key) || len(getStringSlice(v, key)) > 0 {
		return true
	}

	binding := knownEnvBindings()[key]
	for _, name := range append([]string{binding.Primary}, binding.Deprecated...) {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

func BuildHTTPClientConfig(v *viper.Viper, prefix string) HTTPClientConfig {
	return HTTPClientConfig{
		BaseURL:            v.GetString(prefix + ".baseURL"),
		Timeout:            time.Duration(v.GetInt(prefix+".timeout")) * time.Second,
		MaxRetries:         v.GetInt(prefix + ".maxRetries"),
		RetryDelay:         time.Duration(v.GetInt(prefix+".retryDelay")) * time.Second,
		MaxIdleConns:       v.GetInt(prefix + ".maxIdleConns"),
		MaxConnsPerHost:    v.GetInt(prefix + ".maxConnsPerHost"),
		InsecureSkipVerify: v.GetBool(prefix + ".insecureSkipVerify"),
		Headers:            v.GetStringMapString(prefix + ".headers"),
	}
}

func buildOpenAIClients(v *viper.Viper) map[string]OpenAIClientConfig {
	raw := v.GetStringMap("openai.clients")
	if len(raw) == 0 {
		return nil
	}

	defaultKey := v.GetString("openai.apiKey")
	defaultBase := v.GetString("openai.baseURL")
	defaultTimeout := v.GetInt("openai.timeout")

	clients := make(map[string]OpenAIClientConfig, len(raw))
	for name := range raw {
		prefix := "openai.clients." + name

		apiKey := v.GetString(prefix + ".apiKey")
		if apiKey == "" {
			apiKey = defaultKey
		}
		baseURL := v.GetString(prefix + ".baseURL")
		if baseURL == "" {
			baseURL = defaultBase
		}
		timeout := v.GetInt(prefix + ".timeout")
		if timeout == 0 {
			timeout = defaultTimeout
		}

		clients[name] = OpenAIClientConfig{
			APIKey:   apiKey,
			Model:    v.GetString(prefix + ".model"),
			BaseURL:  baseURL,
			Timeout:  timeout,
			APIStyle: v.GetString(prefix + ".apiStyle"),
		}
	}
	return clients
}

// Preserve malformed configuration as an invalid value for NewService to reject.
// GetInt would turn invalid strings into zero and truncate YAML floats, making
// invalid input indistinguishable from an explicitly disabled or valid setting.
func phoneVerificationExpiryMinutes(v *viper.Viper) int {
	switch raw := v.Get("listingkit.zitadel.sms.phoneVerificationExpiryMinutes").(type) {
	case nil:
		return 0
	case int:
		return raw
	case string:
		if value, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			return value
		}
	}
	return -1
}

func buildSplitByLevelConfig(v *viper.Viper) []logger.LevelFileConfig {
	raw := v.Get("logging.split_by_level")
	if raw == nil {
		return nil
	}

	list, ok := raw.([]any)
	if !ok {
		return nil
	}

	configs := make([]logger.LevelFileConfig, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}

		cfg := logger.LevelFileConfig{
			File: getStringFromMap(m, "file"),
		}

		if levelsRaw, ok := m["levels"]; ok {
			switch values := levelsRaw.(type) {
			case []any:
				for _, level := range values {
					if s, ok := level.(string); ok {
						cfg.Levels = append(cfg.Levels, s)
					}
				}
			case []string:
				cfg.Levels = values
			}
		}

		if cfg.File != "" && len(cfg.Levels) > 0 {
			configs = append(configs, cfg)
		}
	}

	return configs
}

func rolePolicyDatabase(v *viper.Viper) *DatabaseConfig {
	if !v.IsSet("listingkit.rolePolicyDatabase") {
		return nil
	}
	var cfg DatabaseConfig
	if v.UnmarshalKey("listingkit.rolePolicyDatabase", &cfg) != nil {
		return &DatabaseConfig{}
	}
	return &cfg
}
