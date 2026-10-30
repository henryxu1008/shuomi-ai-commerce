package config

import (
	"github.com/spf13/viper"
)

func setDefaults(v *viper.Viper) {
	defaults := NewDefaultConfig()

	v.SetDefault("openai.apiKey", defaults.OpenAI.APIKey)
	v.SetDefault("openai.model", defaults.OpenAI.Model)
	v.SetDefault("openai.baseURL", defaults.OpenAI.BaseURL)
	v.SetDefault("openai.timeout", defaults.OpenAI.Timeout)

	v.SetDefault("listingkit.platformAdminUsers", defaults.ListingKit.PlatformAdminUsers)
	v.SetDefault("listingkit.platformAdminRoles", defaults.ListingKit.PlatformAdminRoles)
	v.SetDefault("listingkit.imageUpload.provider", defaults.ListingKit.ImageUpload.Provider)
	v.SetDefault("listingkit.imageUpload.local.rootDir", defaults.ListingKit.ImageUpload.Local.RootDir)
	v.SetDefault("listingkit.imageUpload.s3.bucket", defaults.ListingKit.ImageUpload.S3.Bucket)
	v.SetDefault("listingkit.imageUpload.s3.region", defaults.ListingKit.ImageUpload.S3.Region)
	v.SetDefault("listingkit.imageUpload.s3.endpoint", defaults.ListingKit.ImageUpload.S3.Endpoint)
	v.SetDefault("listingkit.imageUpload.s3.accessKeyID", defaults.ListingKit.ImageUpload.S3.AccessKeyID)
	v.SetDefault("listingkit.imageUpload.s3.secretAccessKey", defaults.ListingKit.ImageUpload.S3.SecretAccessKey)
	v.SetDefault("listingkit.imageUpload.s3.usePathStyle", defaults.ListingKit.ImageUpload.S3.UsePathStyle)
	v.SetDefault("listingkit.imageUpload.s3.publicBase", defaults.ListingKit.ImageUpload.S3.PublicBase)
	v.SetDefault("listingkit.zitadel.issuerURL", defaults.ListingKit.Zitadel.IssuerURL)
	v.SetDefault("listingkit.zitadel.authorizationAPIURL", defaults.ListingKit.Zitadel.AuthorizationAPIURL)
	v.SetDefault("listingkit.zitadel.clientID", defaults.ListingKit.Zitadel.ClientID)
	v.SetDefault("listingkit.zitadel.clientSecret", defaults.ListingKit.Zitadel.ClientSecret)
	v.SetDefault("listingkit.zitadel.authorizationRequired", defaults.ListingKit.Zitadel.AuthorizationRequired)
	v.SetDefault("listingkit.zitadel.allowedTenantIDs", defaults.ListingKit.Zitadel.AllowedTenantIDs)
	v.SetDefault("listingkit.zitadel.allowedUserIDs", defaults.ListingKit.Zitadel.AllowedUserIDs)
	v.SetDefault("listingkit.zitadel.allowedUsernames", defaults.ListingKit.Zitadel.AllowedUsernames)
	v.SetDefault("listingkit.zitadel.allowedRoles", defaults.ListingKit.Zitadel.AllowedRoles)
	v.SetDefault("listingkit.zitadel.sms.signingKey", defaults.ListingKit.Zitadel.SMS.SigningKey)
	v.SetDefault("listingkit.zitadel.sms.tencentSecretID", defaults.ListingKit.Zitadel.SMS.TencentSecretID)
	v.SetDefault("listingkit.zitadel.sms.tencentSecretKey", defaults.ListingKit.Zitadel.SMS.TencentSecretKey)
	v.SetDefault("listingkit.zitadel.sms.tencentAppID", defaults.ListingKit.Zitadel.SMS.TencentAppID)
	v.SetDefault("listingkit.zitadel.sms.tencentSignName", defaults.ListingKit.Zitadel.SMS.TencentSignName)
	v.SetDefault("listingkit.zitadel.sms.tencentTemplateID", defaults.ListingKit.Zitadel.SMS.TencentTemplateID)
	v.SetDefault("listingkit.zitadel.sms.phoneVerificationExpiryMinutes", defaults.ListingKit.Zitadel.SMS.PhoneVerificationExpiryMinutes)
	v.SetDefault("workbench.enabled", defaults.Workbench.Enabled)

	setImageAgentDefaults(v, defaults)
}

func setImageAgentDefaults(v *viper.Viper, defaults *Config) {
	v.SetDefault("imageagent.admission.enabled", defaults.ImageAgent.Admission.Enabled)
	v.SetDefault("imageagent.admission.allowedTenantIDs", defaults.ImageAgent.Admission.AllowedTenantIDs)
	store := defaults.ImageAgent.ArtifactStore
	v.SetDefault("imageagent.artifactStore.enabled", store.Enabled)
	v.SetDefault("imageagent.artifactStore.provider", store.Provider)
	v.SetDefault("imageagent.artifactStore.publicBase", store.PublicBase)
	v.SetDefault("imageagent.artifactStore.s3.bucket", store.S3.Bucket)
	v.SetDefault("imageagent.artifactStore.s3.region", store.S3.Region)
	v.SetDefault("imageagent.artifactStore.s3.endpoint", store.S3.Endpoint)
	v.SetDefault("imageagent.artifactStore.s3.accessKeyID", store.S3.AccessKeyID)
	v.SetDefault("imageagent.artifactStore.s3.secretAccessKey", store.S3.SecretAccessKey)
	v.SetDefault("imageagent.artifactStore.s3.usePathStyle", store.S3.UsePathStyle)
	v.SetDefault("imageagent.artifactStore.s3.artifactMode", store.S3.ArtifactMode)
	v.SetDefault("imageagent.artifactStore.s3.cosImmutableNonVersionedBucketPolicy", store.S3.COSImmutableNonVersionedBucketPolicy)
}
