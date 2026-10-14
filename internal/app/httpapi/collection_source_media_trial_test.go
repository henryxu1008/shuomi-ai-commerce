package httpapi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	imageapp "task-processor/internal/app/imageagent"
	"task-processor/internal/app/productsourcing"
	"task-processor/internal/core/config"
	s3 "task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
	"task-processor/internal/product/collection"
)

type trialMediaPort struct{}

func (trialMediaPort) PutImmutable(context.Context, s3.ImmutableObjectPut) error { return nil }
func (trialMediaPort) ReadObject(context.Context, string, int64) ([]byte, s3.ObjectInspection, error) {
	return nil, s3.ObjectInspection{}, nil
}
func (trialMediaPort) PublicURL(key string) string {
	return "https://localhost:39744/source-media-assets/source-media-trial/" + key
}

func TestCollectionSourceMediaTrialWiresOriginalReaderAndRequiresCapacity(t *testing.T) {
	cfg := &config.Config{StorageCapacity: true, ProductCollectionSourceMediaIsolatedTrial: true, ProductCollectionSourceMedia: config.ImageAgentArtifactStoreConfig{Enabled: true, Provider: "s3", PublicBase: "https://localhost:39744/source-media-assets/source-media-trial", S3: config.ImageAgentArtifactStoreS3Config{Bucket: "source-media-trial", Endpoint: "http://127.0.0.1:9103", ArtifactMode: "aws", UsePathStyle: true}}}
	var options currentApplicationOptions
	WithCollectionSourceMedia(trialMediaPort{})(&options)
	media := options.collectionSourceMedia
	require.ErrorIs(t, configureCollectionSourceMediaTrial(media, cfg), collection.ErrUnavailable)
	media.Capacity = &storagecapacity.Meter{}
	require.ErrorIs(t, configureCollectionSourceMediaTrial(media, cfg), collection.ErrUnavailable)
	cfg.ProductCollectionSourceMediaIsolatedTrialDeploymentID = "d7b806bb-6f4f-47be-999c-99f027252244"
	require.NoError(t, configureCollectionSourceMediaTrial(media, cfg))
	require.NotNil(t, media.TrialURLs)
	reader := newImageSetManualMedia(media, imageMediaScopeAuthority{}).(imageapp.ImageSetManualMedia)
	port := reader.Media.(productsourcing.SourceMedia)
	require.Same(t, media.TrialURLs, port.TrialURLs)
	require.Same(t, media.Capacity, port.Capacity)
	cfg.StorageCapacity = false
	require.ErrorIs(t, configureCollectionSourceMediaTrial(media, cfg), collection.ErrUnavailable)
	cfg.StorageCapacity = true
	cfg.ProductCollectionSourceMedia.S3.Endpoint = "https://storage.example.org"
	require.ErrorIs(t, configureCollectionSourceMediaTrial(media, cfg), collection.ErrUnavailable)
	cfg.ProductCollectionSourceMediaIsolatedTrial = false
	cfg.ProductCollectionSourceMediaIsolatedTrialDeploymentID = ""
	require.NoError(t, configureCollectionSourceMediaTrial(media, cfg))
	require.Nil(t, media.TrialURLs, "normal mode cannot retain a trial policy")
}
