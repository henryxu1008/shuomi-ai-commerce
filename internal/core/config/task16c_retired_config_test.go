package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFixtureRejectsRetiredProductRuntimeYAML(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		key  string
	}{
		{name: "manual upload provider", yaml: "listingkit:\n  imageUpload:\n    provider: s3\n", key: "listingkit.imageUpload"},
		{name: "manual upload local", yaml: "listingkit:\n  imageUpload:\n    local:\n      rootDir: retired-upload-fixture\n", key: "listingkit.imageUpload"},
		{name: "manual upload s3", yaml: "listingkit:\n  imageUpload:\n    s3:\n      bucket: retired-upload-fixture\n", key: "listingkit.imageUpload"},
		{name: "manual upload credential", yaml: "listingkit:\n  imageUpload:\n    s3:\n      secretAccessKey: retired-upload-fixture\n", key: "listingkit.imageUpload"},
		{name: "productimage root", yaml: "productimage:\n  publisher:\n    provider: s3\n", key: "productimage"},
		{name: "product enrich debug", yaml: "debug:\n  productEnrichMockLLM: true\n", key: "debug.productEnrichMockLLM"},
		{name: "product image scene enabled", yaml: "aiCapability:\n  productImageSceneEnabled: true\n", key: "aiCapability.productImageSceneEnabled"},
		{name: "product image scene tenants", yaml: "aiCapability:\n  productImageSceneAllowedTenantIDs: [tenant-a]\n", key: "aiCapability.productImageSceneAllowedTenantIDs"},
		{name: "product enrich text", yaml: "aiCapability:\n  productEnrichTextEnabled: true\n", key: "aiCapability.productEnrichTextEnabled"},
		{name: "product enrich vision", yaml: "aiCapability:\n  productEnrichVisionAllowedTenantIDs: [tenant-a]\n", key: "aiCapability.productEnrichVisionAllowedTenantIDs"},
		{name: "product enrich listing", yaml: "aiCapability:\n  productEnrichListingEnabled: true\n", key: "aiCapability.productEnrichListingEnabled"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := loadConfigFixture(t, []byte("openai:\n  apiKey: test-key\n"+test.yaml))
			require.Error(t, err)
			require.Contains(t, strings.ToLower(err.Error()), strings.ToLower(test.key))
			require.NotContains(t, err.Error(), "retired-upload-fixture")
		})
	}
}

func TestRetiredListingKitUploadEnvDoesNotRecreateApplicationOwner(t *testing.T) {
	for _, suffix := range []string{"PROVIDER", "LOCAL_ROOT_DIR", "S3_BUCKET", "S3_REGION", "S3_ENDPOINT", "S3_ACCESS_KEY_ID", "S3_SECRET_ACCESS_KEY", "S3_PUBLIC_BASE"} {
		t.Setenv("TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_"+suffix, "synthetic-retired-upload")
	}
	t.Setenv("TASK_PROCESSOR_LISTINGKIT_IMAGE_UPLOAD_S3_USE_PATH_STYLE", "true")
	cfg, err := loadConfigFixture(t, []byte("openai:\n  apiKey: test-key\n"))
	require.NoError(t, err, "imgproxy's two valid shared Secret names must not be globally rejected")
	require.NotNil(t, cfg)
	require.Empty(t, cfg.ImageAgent.ArtifactStore.Provider)
	require.Empty(t, cfg.ImageAgent.ArtifactStore.S3.Bucket)
	require.Empty(t, cfg.ImageAgent.ArtifactStore.S3.AccessKeyID)
	require.Empty(t, cfg.ImageAgent.ArtifactStore.S3.SecretAccessKey)
}

func TestConfigFixtureRejectsRetiredProductRuntimeEnvironment(t *testing.T) {
	variables := []string{
		"TASK_PROCESSOR_PRODUCTIMAGE_PUBLISHER_PROVIDER",
		"TASK_PROCESSOR_PRODUCT_IMAGE_PUBLISHER_PROVIDER",
		"PRODUCTIMAGE_PUBLISHER_PROVIDER",
		"PRODUCT_IMAGE_PUBLISHER_PROVIDER",
		"TASK_PROCESSOR_PRODUCTENRICH_MOCK_LLM",
		"TASK_PROCESSOR_PRODUCT_ENRICH_MOCK_LLM",
		"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_IMAGE_SCENE_ENABLED",
		"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_IMAGE_SCENE_ALLOWED_TENANT_IDS",
		"TASK_PROCESSOR_AICAPABILITY_PRODUCTIMAGESCENEENABLED",
		"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_TEXT_ENABLED",
		"TASK_PROCESSOR_AICAPABILITY_PRODUCTENRICHTEXTENABLED",
		"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_VISION_ENABLED",
		"TASK_PROCESSOR_AI_CAPABILITY_PRODUCT_ENRICH_LISTING_ENABLED",
	}

	for _, variable := range variables {
		t.Run(variable, func(t *testing.T) {
			t.Setenv(variable, "retired-value")
			_, err := loadConfigFixture(t, []byte("openai:\n  apiKey: test-key\n"))
			require.Error(t, err)
			require.Contains(t, err.Error(), variable)
		})
	}
}
