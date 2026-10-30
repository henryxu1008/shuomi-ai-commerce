package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	topupconfig "task-processor/internal/integration/wallettopup/config"
	logger "task-processor/internal/platform/logging"

	"github.com/spf13/viper"
)

type envBinding struct {
	Primary string
}

var (
	loadDotEnvMu      sync.Mutex
	loadedDotEnvPaths = map[string]struct{}{}
)

type Config struct {
	StorageCapacity bool            `yaml:"storageCapacity"`
	Referrals       ReferralsConfig `yaml:"-"`
	OpenAI          OpenAIConfig    `yaml:"openai"`
	// BrowserCollector is the opt-in wiring to the standalone anonymous public
	// 1688 browser collector process (design D13).
	BrowserCollector                                      BrowserCollectorConfig        `yaml:"browserCollector"`
	ImageAgent                                            ImageAgentConfig              `yaml:"imageagent"`
	ProductCollectionSourceMedia                          ImageAgentArtifactStoreConfig `yaml:"productCollectionSourceMedia"`
	ProductCollectionSourceMediaIsolatedTrial             bool                          `yaml:"productCollectionSourceMediaIsolatedTrial"`
	ProductCollectionSourceMediaIsolatedTrialDeploymentID string                        `yaml:"productCollectionSourceMediaIsolatedTrialDeploymentID"`
	Database                                              *DatabaseConfig               `yaml:"database"`
	// CommercialOwnerDatabase is the explicit resource-balance owner pool;
	// it must not borrow the Token usage runtime role.
	CommercialOwnerDatabase *DatabaseConfig    `yaml:"commercialOwnerDatabase"`
	ListingKit              ListingKitConfig   `yaml:"listingkit"`
	Workbench               WorkbenchConfig    `yaml:"workbench"`
	WalletTopUp             topupconfig.Config `yaml:"walletTopUp"`
}

// Only the three retired compact image spellings overlap AutomaticEnv. Keep all
// other automatic inputs and explicitly bound environment names unchanged.
type imageClientEnvKeyReplacer struct{}

func (imageClientEnvKeyReplacer) Replace(value string) string {
	switch value {
	case "TASK_PROCESSOR_OPENAI.CLIENTS.IMAGE.APIKEY":
		return "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY"
	case "TASK_PROCESSOR_OPENAI.CLIENTS.IMAGE.BASEURL":
		return "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL"
	case "TASK_PROCESSOR_OPENAI.CLIENTS.IMAGE.APISTYLE":
		return "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE"
	default:
		return strings.ReplaceAll(value, ".", "_")
	}
}

func newViper() *viper.Viper {
	v := viper.NewWithOptions(viper.EnvKeyReplacer(imageClientEnvKeyReplacer{}))
	v.SetConfigType("yaml")
	v.SetEnvPrefix("TASK_PROCESSOR")
	v.AutomaticEnv()
	bindKnownEnvs(v)
	setDefaults(v)
	return v
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
		_ = v.BindEnv(key, binding.Primary)
	}
}

func knownEnvBindings() map[string]envBinding {
	return map[string]envBinding{
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
			Primary: "TASK_PROCESSOR_OPENAI_API_KEY",
		},
		"openai.model": {
			Primary: "TASK_PROCESSOR_OPENAI_MODEL",
		},
		"openai.baseURL": {
			Primary: "TASK_PROCESSOR_OPENAI_BASE_URL",
		},
		"openai.clients.image.apiKey": {
			Primary: "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_KEY",
		},
		"openai.clients.image.baseURL": {
			Primary: "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_BASE_URL",
		},
		"openai.clients.image.apiStyle": {
			Primary: "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_API_STYLE",
		},
		"openai.clients.image.model": {
			Primary: "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_MODEL",
		},
		"openai.clients.image.timeout": {
			Primary: "TASK_PROCESSOR_OPENAI_CLIENTS_IMAGE_TIMEOUT",
		},
		"database.host": {
			Primary: "TASK_PROCESSOR_DATABASE_HOST",
		},
		"database.port": {
			Primary: "TASK_PROCESSOR_DATABASE_PORT",
		},
		"database.user": {
			Primary: "TASK_PROCESSOR_DATABASE_USER",
		},
		"database.password": {
			Primary: "TASK_PROCESSOR_DATABASE_PASSWORD",
		},
		"database.database": {
			Primary: "TASK_PROCESSOR_DATABASE_NAME",
		},
		"database.max_connections": {
			Primary: "TASK_PROCESSOR_DATABASE_MAX_CONNECTIONS",
		},
		"database.max_idle_connections": {
			Primary: "TASK_PROCESSOR_DATABASE_MAX_IDLE_CONNECTIONS",
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
		"listingkit.zitadel.allowedUsernames": {
			Primary: "TASK_PROCESSOR_LISTINGKIT_ZITADEL_ALLOWED_USERNAMES",
		},
		"workbench.enabled": {
			Primary: "TASK_PROCESSOR_WORKBENCH_ENABLED",
		},
	}
}

func loadWithViper(v *viper.Viper) (*Config, error) {
	if err := rejectRetiredProductRuntimeConfig(v); err != nil {
		return nil, err
	}
	cfg := BuildConfig(v)
	if err := ValidateConfigWithError(cfg); err != nil {
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

func LoadConfigFromFile(configFile string) (*Config, error) {
	tryLoadDotEnvForConfig(configFile)

	logger.GetGlobalLogger("core/config").Infof("loading config file: %s", configFile)

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

	v := newViper()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file %s: %w", configFile, err)
	}

	logger.GetGlobalLogger("core/config").Infof("loaded config file: %s", v.ConfigFileUsed())
	return loadWithViperWithoutValidation(v)
}
