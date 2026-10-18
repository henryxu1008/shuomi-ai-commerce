package config

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestImageWorkerLoaderKeepsMeteringAndExplicitStorageIdentity(t *testing.T) {
	v := newViper()
	require.NoError(t, v.ReadConfig(strings.NewReader(`
storageCapacity: true
commercialOwnerDatabase:
  host: 127.0.0.1
  port: 5433
  user: commercial_owner_runtime
  password: fixture
  database: commercial
  max_connections: 2
imageagent:
  artifactStore:
    isolatedTrialGeneratedURLs: true
    s3:
      capacityIdentityProvider: minio-deployment
      capacityMinIODeploymentID: 0b86217b-c946-4d62-861e-11f331957e3b
`)))
	cfg, err := loadWithViperWithoutValidation(v)
	require.NoError(t, err)
	require.True(t, cfg.StorageCapacity)
	require.NotNil(t, cfg.CommercialOwnerDatabase)
	require.Equal(t, "commercial_owner_runtime", cfg.CommercialOwnerDatabase.User)
	require.True(t, cfg.ImageAgent.ArtifactStore.IsolatedTrialGeneratedURLs)
	require.Equal(t, "minio-deployment", cfg.ImageAgent.ArtifactStore.S3.CapacityIdentityProvider)
	require.Equal(t, "0b86217b-c946-4d62-861e-11f331957e3b", cfg.ImageAgent.ArtifactStore.S3.CapacityMinIODeploymentID)
}
