package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFixtureBindsImageAgentArtifactStoreYAML(t *testing.T) {
	cfg, err := loadConfigFixture(t, []byte(`
imageagent:
  artifactStore:
    enabled: true
    provider: s3
    publicBase: https://cdn.example.test/image-agent
    s3:
      bucket: image-agent-artifacts
      region: ap-southeast-1
      endpoint: https://s3.example.test
      accessKeyID: access-key
      secretAccessKey: secret-key
      usePathStyle: true
      artifactMode: cos
      cosImmutableNonVersionedBucketPolicy: true
`))
	require.NoError(t, err)

	store := cfg.ImageAgent.ArtifactStore
	assert.True(t, store.Enabled)
	assert.Equal(t, "s3", store.Provider)
	assert.Equal(t, "https://cdn.example.test/image-agent", store.PublicBase)
	assert.Equal(t, "image-agent-artifacts", store.S3.Bucket)
	assert.Equal(t, "ap-southeast-1", store.S3.Region)
	assert.Equal(t, "https://s3.example.test", store.S3.Endpoint)
	assert.Equal(t, "access-key", store.S3.AccessKeyID)
	assert.Equal(t, "secret-key", store.S3.SecretAccessKey)
	assert.True(t, store.S3.UsePathStyle)
	assert.Equal(t, "cos", store.S3.ArtifactMode)
	assert.True(t, store.S3.COSImmutableNonVersionedBucketPolicy)
}

func TestConfigFixtureBindsImageAgentArtifactStoreEnvironment(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_ENABLED", "true")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PROVIDER", "s3")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_PUBLIC_BASE", "https://env-cdn.example.test")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_BUCKET", "env-bucket")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_REGION", "env-region")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ENDPOINT", "https://env-s3.example.test")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ACCESS_KEY_ID", "env-access")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_SECRET_ACCESS_KEY", "env-secret")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_USE_PATH_STYLE", "true")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_ARTIFACT_MODE", "aws")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ARTIFACT_STORE_S3_COS_IMMUTABLE_NON_VERSIONED_BUCKET_POLICY", "false")

	cfg, err := loadConfigFixture(t, []byte(""))
	require.NoError(t, err)

	store := cfg.ImageAgent.ArtifactStore
	assert.True(t, store.Enabled)
	assert.Equal(t, "s3", store.Provider)
	assert.Equal(t, "https://env-cdn.example.test", store.PublicBase)
	assert.Equal(t, "env-bucket", store.S3.Bucket)
	assert.Equal(t, "env-region", store.S3.Region)
	assert.Equal(t, "https://env-s3.example.test", store.S3.Endpoint)
	assert.Equal(t, "env-access", store.S3.AccessKeyID)
	assert.Equal(t, "env-secret", store.S3.SecretAccessKey)
	assert.True(t, store.S3.UsePathStyle)
	assert.Equal(t, "aws", store.S3.ArtifactMode)
	assert.False(t, store.S3.COSImmutableNonVersionedBucketPolicy)
}

func TestConfigFixtureBindsImageAgentAdmissionEnvironment(t *testing.T) {
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ADMISSION_ENABLED", "true")
	t.Setenv("TASK_PROCESSOR_IMAGEAGENT_ADMISSION_ALLOWED_TENANT_IDS", "373211199677923496,1038")

	cfg, err := loadConfigFixture(t, []byte("imageagent:\n  admission:\n    enabled: false\n    allowedTenantIDs: []\n"))
	require.NoError(t, err)

	assert.True(t, cfg.ImageAgent.Admission.Enabled)
	assert.Equal(t, []string{"373211199677923496", "1038"}, cfg.ImageAgent.Admission.AllowedTenantIDs)
}
