package config

// ImageAgentConfig owns the durable runtime settings consumed by ImageAgent.
type ImageAgentConfig struct {
	Generation    ImageAgentGenerationConfig    `mapstructure:"generation" yaml:"generation"`
	Admission     ImageAgentAdmissionConfig     `mapstructure:"admission" yaml:"admission"`
	ArtifactStore ImageAgentArtifactStoreConfig `mapstructure:"artifactStore" yaml:"artifactStore"`
}

// Generation contains only an explicit admission ceiling, never a customer price.
// Every charge comes from the shared official tariff and the original ledger.
type ImageAgentGenerationConfig struct {
	Enabled            bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	UnitVersion        string `mapstructure:"unitVersion" yaml:"unitVersion" json:"unitVersion"`
	MaximumMicropoints int64  `mapstructure:"maximumMicropoints" yaml:"maximumMicropoints" json:"maximumMicropoints,string"`
}

func (c ImageAgentGenerationConfig) Configured() bool {
	return c.Enabled && c.UnitVersion == "ai-micropoint-v1" && c.MaximumMicropoints > 0
}

// ImageAgentAdmissionConfig is the explicit tenant boundary for provider work.
// Route permissions identify the caller; this gate identifies eligible tenants.
type ImageAgentAdmissionConfig struct {
	Enabled          bool     `mapstructure:"enabled" yaml:"enabled"`
	AllowedTenantIDs []string `mapstructure:"allowedTenantIDs" yaml:"allowedTenantIDs"`
}

type ImageAgentArtifactStoreConfig struct {
	Enabled                    bool                            `mapstructure:"enabled" yaml:"enabled"`
	Provider                   string                          `mapstructure:"provider" yaml:"provider"`
	PublicBase                 string                          `mapstructure:"publicBase" yaml:"publicBase"`
	IsolatedTrialGeneratedURLs bool                            `mapstructure:"isolatedTrialGeneratedURLs" yaml:"isolatedTrialGeneratedURLs"`
	S3                         ImageAgentArtifactStoreS3Config `mapstructure:"s3" yaml:"s3"`
}

type ImageAgentArtifactStoreS3Config struct {
	CapacityIdentityProvider             string `mapstructure:"capacityIdentityProvider" yaml:"capacityIdentityProvider"`
	CapacityMinIODeploymentID            string `mapstructure:"capacityMinIODeploymentID" yaml:"capacityMinIODeploymentID"`
	Bucket                               string `mapstructure:"bucket" yaml:"bucket"`
	Region                               string `mapstructure:"region" yaml:"region"`
	Endpoint                             string `mapstructure:"endpoint" yaml:"endpoint"`
	AccessKeyID                          string `mapstructure:"accessKeyID" yaml:"accessKeyID"`
	SecretAccessKey                      string `mapstructure:"secretAccessKey" yaml:"secretAccessKey"`
	UsePathStyle                         bool   `mapstructure:"usePathStyle" yaml:"usePathStyle"`
	ArtifactMode                         string `mapstructure:"artifactMode" yaml:"artifactMode"`
	COSImmutableNonVersionedBucketPolicy bool   `mapstructure:"cosImmutableNonVersionedBucketPolicy" yaml:"cosImmutableNonVersionedBucketPolicy"`
}
