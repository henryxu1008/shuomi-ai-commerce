package config

import (
	"os"
	"strings"
	topupconfig "task-processor/internal/integration/wallettopup/config"
	"time"

	"github.com/spf13/viper"
)

func BuildConfig(v *viper.Viper) *Config {
	listingKitAllowedUsernames := getStringSlice(v, "listingkit.zitadel.allowedUsernames")
	listingKitLegacyUsernameAllowlistConfigured := legacyListingKitUsernameAllowlistConfigured(v)
	listingKitZitadelIssuerURL := v.GetString("listingkit.zitadel.issuerURL")
	listingKitZitadelAuthorizationAPIURL := v.GetString("listingkit.zitadel.authorizationAPIURL")
	if strings.TrimSpace(listingKitZitadelAuthorizationAPIURL) == "" {
		listingKitZitadelAuthorizationAPIURL = listingKitZitadelIssuerURL
	}

	cfg := &Config{
		StorageCapacity: v.GetBool("storageCapacity"),
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
		ListingKit: ListingKitConfig{
			RolePolicyDatabase: rolePolicyDatabase(v),
			PlatformAdminUsers: getStringSlice(v, "listingkit.platformAdminUsers"),
			PlatformAdminRoles: getStringSlice(v, "listingkit.platformAdminRoles"),
			Zitadel: ListingKitZitadelConfig{
				IssuerURL:                         listingKitZitadelIssuerURL,
				AuthorizationAPIURL:               listingKitZitadelAuthorizationAPIURL,
				ClientID:                          v.GetString("listingkit.zitadel.clientID"),
				ClientSecret:                      v.GetString("listingkit.zitadel.clientSecret"),
				TenantDirectoryToken:              v.GetString("listingkit.zitadel.tenantDirectoryToken"),
				MemberInvitationToken:             v.GetString("listingkit.zitadel.memberInvitationToken"),
				ProjectID:                         v.GetString("listingkit.zitadel.projectID"),
				AllowedUsernames:                  listingKitAllowedUsernames,
				LegacyUsernameAllowlistConfigured: listingKitLegacyUsernameAllowlistConfigured,
			},
		},
		Workbench: WorkbenchConfig{
			Enabled: v.GetBool("workbench.enabled"),
		},
		WalletTopUp: topupconfig.Load(v),
	}

	cfg.Database = databaseConfigFromViper(v, "database")
	cfg.CommercialOwnerDatabase = databaseConfigFromViper(v, "commercialOwnerDatabase")

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

func legacyListingKitUsernameAllowlistConfigured(v *viper.Viper) bool {
	const key = "listingkit.zitadel.allowedUsernames"
	if v == nil {
		return false
	}
	if v.InConfig(key) || len(getStringSlice(v, key)) > 0 {
		return true
	}

	for _, name := range []string{knownEnvBindings()[key].Primary, "LISTINGKIT_ZITADEL_ALLOWED_USERNAMES"} {
		if value, ok := os.LookupEnv(name); ok && strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
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
