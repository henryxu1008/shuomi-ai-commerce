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

	setBrowserDefaults(v, defaults)
	setAmazonDefaults(v, defaults)
	setImageAgentDefaults(v, defaults)
	setPlatformDefaults(v, defaults)
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

func setBrowserDefaults(v *viper.Viper, defaults *Config) {
	b := defaults.Browser

	v.SetDefault("browser.enabled", b.Enabled)
	v.SetDefault("browser.headless", b.Headless)
	v.SetDefault("browser.browserPath", b.BrowserPath)
	v.SetDefault("browser.userDataDir", b.UserDataDir)
	v.SetDefault("browser.poolSize", b.PoolSize)
	v.SetDefault("browser.viewportWidth", b.ViewportWidth)
	v.SetDefault("browser.viewportHeight", b.ViewportHeight)
	v.SetDefault("browser.proxyServer", b.ProxyServer)

	rc := b.RandomConfig
	v.SetDefault("browser.randomConfig.enabled", rc.Enabled)
	v.SetDefault("browser.randomConfig.strategy", rc.Strategy)
	v.SetDefault("browser.randomConfig.presetName", rc.PresetName)
	v.SetDefault("browser.randomConfig.fingerprintStrategy", rc.FingerprintStrategy)
	v.SetDefault("browser.randomConfig.healthCheckEnabled", rc.HealthCheckEnabled)
	v.SetDefault("browser.randomConfig.maxRetries", rc.MaxRetries)
	v.SetDefault("browser.randomConfig.maxUsesPerInstance", rc.MaxUsesPerInstance)
}

func setAmazonDefaults(v *viper.Viper, defaults *Config) {
	a := defaults.Amazon

	v.SetDefault("amazon.enabled", a.Enabled)
	v.SetDefault("amazon.failureArtifacts.enabled", a.FailureArtifacts.Enabled)
	v.SetDefault("amazon.failureArtifacts.directory", a.FailureArtifacts.Directory)
	v.SetDefault("amazon.failureArtifacts.captureHTML", a.FailureArtifacts.CaptureHTML)
	v.SetDefault("amazon.failureArtifacts.maxHTMLBytes", a.FailureArtifacts.MaxHTMLBytes)
	v.SetDefault("amazon.riskControl.captchaRecreateThreshold", a.RiskControl.CaptchaRecreateThreshold)
	v.SetDefault("amazon.riskControl.authenticationRecreateThreshold", a.RiskControl.AuthenticationRecreateThreshold)
	v.SetDefault("amazon.riskControl.browserCrashRecreateThreshold", a.RiskControl.BrowserCrashRecreateThreshold)
	v.SetDefault("amazon.riskControl.timeoutRecreateThreshold", a.RiskControl.TimeoutRecreateThreshold)
	v.SetDefault("amazon.riskControl.networkRecreateThreshold", a.RiskControl.NetworkRecreateThreshold)
	v.SetDefault("amazon.riskControl.serverErrorRecreateThreshold", a.RiskControl.ServerErrorRecreateThreshold)
	v.SetDefault("amazon.qualityControl.retryOnValidationFailure", a.QualityControl.RetryOnValidationFailure)
	v.SetDefault("amazon.qualityControl.validationRetryMaxAttempts", a.QualityControl.ValidationRetryMaxAttempts)
	v.SetDefault("amazon.proxyPool.enabled", a.ProxyPool.Enabled)
	v.SetDefault("amazon.proxyPool.strategy", a.ProxyPool.Strategy)
	v.SetDefault("amazon.proxyPool.failureCooldownSeconds", a.ProxyPool.FailureCooldownSeconds)
	v.SetDefault("amazon.proxyPool.proxyFile", a.ProxyPool.ProxyFile)
	v.SetDefault("amazon.proxyPool.proxies", a.ProxyPool.Proxies)

}

func setPlatformDefaults(v *viper.Viper, defaults *Config) {
	setAlibaba1688Defaults(v, &defaults.Platforms.Alibaba1688)
}

func setAlibaba1688Defaults(v *viper.Viper, p *Alibaba1688Config) {
	v.SetDefault("platforms.alibaba1688.enabled", p.Enabled)
	v.SetDefault("platforms.alibaba1688.timeout", p.Timeout)
	v.SetDefault("platforms.alibaba1688.poolSize", p.PoolSize)
	v.SetDefault("platforms.alibaba1688.profileRootDir", p.ProfileRootDir)
}
