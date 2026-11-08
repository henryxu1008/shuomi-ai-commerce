package httpapi

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	agentstore "task-processor/internal/integration/persistence/agent"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
)

func TestImageAnalysisAssemblyWithHistoryOnly(t *testing.T) {
	db := &gorm.DB{}
	snapshots, err := catalogstore.NewBoundedSnapshotReader(db, 1<<20)
	require.NoError(t, err)
	authorizer, err := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, err)
	parent := &productAgentApplication{
		config: ProductAgentDependencies{AssetDB: db}, snapshots: snapshots,
		authorizer: authorizer, store: &agentstore.Store{},
	}
	t.Run("history only", func(t *testing.T) {
		analysis, err := buildImageAnalysisApplication(parent, nil, &FullImageSetDependencies{AssetDB: db}, nil)
		require.NoError(t, err)
		require.NotNil(t, analysis)
		require.Nil(t, analysis.materials.ReadGenerated)
	})
	t.Run("configured artifact reader", func(t *testing.T) {
		worker := &config.Config{}
		worker.ImageAgent.ArtifactStore = config.ImageAgentArtifactStoreConfig{
			Enabled: true, Provider: "s3", PublicBase: "https://images.example.test",
			S3: config.ImageAgentArtifactStoreS3Config{
				Bucket: "image-assets", Region: "us-east-1", Endpoint: "https://s3.example.test",
				AccessKeyID: "test-access", SecretAccessKey: "test-secret", ArtifactMode: "aws",
			},
		}
		analysis, err := buildImageAnalysisApplication(parent, nil, &FullImageSetDependencies{AssetDB: db, WorkerConfig: worker}, nil)
		require.NoError(t, err)
		require.NotNil(t, analysis)
		require.NotNil(t, analysis.materials.ReadGenerated)
	})
}
