package currentapplication

import (
	"github.com/stretchr/testify/require"
	coreconfig "task-processor/internal/core/config"
	"testing"
)

func TestSourceMediaForCurrentCollectionsDoesNotRequireOfficialPlatform(t *testing.T) {
	c := acquisitionRuntimeConfig()
	c.ProductCollections = true
	c.SourceMedia = &coreconfig.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://files.example.org/products", S3: coreconfig.ImageAgentArtifactStoreS3Config{Bucket: "products", Region: "us-east-1", AccessKeyID: "synthetic-access", SecretAccessKey: "synthetic-secret", ArtifactMode: "aws"}}
	require.Nil(t, c.StoreCenter)
	require.Nil(t, c.SupplyChain)
	require.NoError(t, c.validate(), "current Product file uploads do not consume platform rules")
}

func TestSourceMediaMapsCompleteStorageWithoutEnablingAgent(t *testing.T) {
	c := supplyRuntimeConfig(t)
	wanted := coreconfig.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://files.example.org/products", S3: coreconfig.ImageAgentArtifactStoreS3Config{Bucket: "source-products", Region: "us-east-1", Endpoint: "https://storage.example.org", AccessKeyID: "synthetic-access", SecretAccessKey: "synthetic-secret", UsePathStyle: true, ArtifactMode: "aws"}}
	c.SourceMedia = &wanted
	require.NoError(t, c.validate())
	core := c.CoreConfig()
	require.Equal(t, wanted, core.ProductCollectionSourceMedia)
	require.False(t, core.ImageAgent.Admission.Enabled)
	require.False(t, core.ImageAgent.ArtifactStore.Enabled)
	require.Nil(t, c.ImageAgent)
}

func TestSourceMediaRejectsIncompleteOrIncorrectStorage(t *testing.T) {
	c := supplyRuntimeConfig(t)
	base := coreconfig.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://files.example.org/products", S3: coreconfig.ImageAgentArtifactStoreS3Config{Bucket: "products", Region: "us-east-1", AccessKeyID: "synthetic-access", SecretAccessKey: "synthetic-secret", ArtifactMode: "aws"}}
	for name, change := range map[string]func(*coreconfig.ImageAgentArtifactStoreConfig){"no credentials": func(v *coreconfig.ImageAgentArtifactStoreConfig) { v.S3.SecretAccessKey = "" }, "no bucket": func(v *coreconfig.ImageAgentArtifactStoreConfig) { v.S3.Bucket = "" }, "private public URL": func(v *coreconfig.ImageAgentArtifactStoreConfig) { v.PublicBase = "https://127.0.0.1/images" }, "cos missing immutable policy": func(v *coreconfig.ImageAgentArtifactStoreConfig) {
		v.S3.ArtifactMode = "cos"
		v.S3.Endpoint = "https://storage.example.org"
	}, "unknown mode": func(v *coreconfig.ImageAgentArtifactStoreConfig) { v.S3.ArtifactMode = "other" }} {
		t.Run(name, func(t *testing.T) {
			value := base
			change(&value)
			c.SourceMedia = &value
			require.Error(t, c.validate())
		})
	}
	c.SourceMedia = &base
	c.SupplyChain = nil
	require.NoError(t, c.validate())
	c.ProductCollections = false
	require.Error(t, c.validate())
	c.SourceMedia = nil
	require.NoError(t, c.validate())
}

func TestSourceMediaTrialConfigRequiresExplicitMeteredCurrentCollections(t *testing.T) {
	c := acquisitionRuntimeConfig()
	c.ProductCollections = true
	c.MoneyOwnerDatabase = &DatabaseConfig{Host: "127.0.0.1", Port: 5432, User: "money_owner_runtime", Password: "fixture-password", Database: "money", MaxConnections: 2}
	c.SourceMedia = &coreconfig.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://localhost:39744/source-media-assets/source-media-trial", S3: coreconfig.ImageAgentArtifactStoreS3Config{Bucket: "source-media-trial", Region: "us-east-1", Endpoint: "http://127.0.0.1:9103", AccessKeyID: "fixture", SecretAccessKey: "fixture", UsePathStyle: true, ArtifactMode: "aws"}}
	require.Error(t, c.validate(), "private origins are not implicitly admitted")
	c.SourceMediaIsolatedTrial = true
	require.Error(t, c.validate(), "trial does not bypass capacity")
	c.StorageCapacity = true
	require.Error(t, c.validate(), "MinIO requires its provider deployment pin")
	c.SourceMediaIsolatedTrialDeploymentID = "d7b806bb-6f4f-47be-999c-99f027252244"
	require.NoError(t, c.validate())
	require.True(t, c.CoreConfig().ProductCollectionSourceMediaIsolatedTrial)
	require.Equal(t, c.SourceMediaIsolatedTrialDeploymentID, c.CoreConfig().ProductCollectionSourceMediaIsolatedTrialDeploymentID)
	c.SourceMedia.S3.UsePathStyle = false
	require.Error(t, c.validate())
	c.SourceMedia.S3.UsePathStyle = true
	c.SourceMedia.S3.Endpoint = "https://real-storage.example.org"
	require.Error(t, c.validate())
	c.SourceMedia = nil
	require.Error(t, c.validate(), "stale flag cannot enable another capability")
}
