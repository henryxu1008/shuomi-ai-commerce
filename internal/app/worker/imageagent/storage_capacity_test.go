package imageagentworker

import (
	"context"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	commercialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/commercialbilling"
	capacitystore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/storagecapacity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/s3"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/ledger/storagecapacity"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestImageStorageCapacityUsesExplicitExistingMinIOIdentityContract(t *testing.T) {
	pin := "0b86217b-c946-4d62-861e-11f331957e3b"
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		require.Equal(t, "/minio/admin/v3/info", r.URL.Path)
		require.NotEmpty(t, r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{"deploymentID":"` + pin + `"}`))
	}))
	defer server.Close()
	cfg := config.ImageAgentArtifactStoreS3Config{Region: "us-east-1", Endpoint: server.URL, AccessKeyID: "controlled", SecretAccessKey: "fixture", Bucket: "images", UsePathStyle: true, ArtifactMode: "aws", CapacityIdentityProvider: s3.MinIOCapacityIdentityProvider, CapacityMinIODeploymentID: pin}
	client := imageStorageCapacityClientConfig(cfg)
	namespace, err := s3.ResolveCapacityNamespace(context.Background(), client, cfg.Bucket, cfg.ArtifactMode)
	require.NoError(t, err)
	require.NotEmpty(t, namespace)
	require.Equal(t, 1, requests)
	client.CapacityMinIODeploymentID = "1b86217b-c946-4d62-861e-11f331957e3b"
	_, err = s3.ResolveCapacityNamespace(context.Background(), client, cfg.Bucket, cfg.ArtifactMode)
	require.Error(t, err, "a different physical deployment cannot inherit the namespace")
}

func TestStorageRollbackRejectsOrganizationWorkerBeforeDependencies(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, commercialstore.AutoMigrate(db))
	cfg := &config.Config{}
	require.NoError(t, configureStorageCapacity(cfg, db, nil, nil, nil, nil), "empty owner allows the original disabled mode")
	repo, err := capacitystore.New(db)
	require.NoError(t, err)
	require.NoError(t, repo.EnsureNamespace(context.Background(), storagecapacity.Digest("original-worker-bucket"), "published/", func(context.Context) error { return nil }))
	require.Error(t, configureStorageCapacity(cfg, db, nil, nil, nil, nil), "registered owner must refuse an unmetered worker")
	require.Error(t, configureStorageCapacity(cfg, nil, nil, nil, nil, nil), "missing owner is not a provably empty owner")
}
