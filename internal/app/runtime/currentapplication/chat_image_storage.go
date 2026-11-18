package currentapplication

import (
	"context"
	"errors"

	"gorm.io/gorm"
	capstore "task-processor/internal/integration/persistence/storagecapacity"
	s3integration "task-processor/internal/integration/s3"
	"task-processor/internal/ledger/storagecapacity"
)

// ChatImageStorageRuntime carries the private object and capacity ports to the application composer.
type ChatImageStorageRuntime struct {
	Objects  *s3integration.Uploader
	Capacity *storagecapacity.Meter
}

func prepareChatImageStorage(ctx context.Context, cfg *Config, db *gorm.DB) (*ChatImageStorageRuntime, error) {
	if cfg.AIWorkbench == nil || cfg.AIWorkbench.ChatImageStorage == nil {
		return nil, nil
	}
	settings := *cfg.AIWorkbench.ChatImageStorage
	if !cfg.StorageCapacity || db == nil || settings.Bucket == "" || settings.AccessKeyID == "" || settings.SecretAccessKey == "" ||
		settings.Mode != "aws" && settings.Mode != "cos" || settings.Mode == "cos" && !settings.COSImmutableNonVersionedBucketPolicy {
		return nil, errors.New("Chat images require metered private immutable storage")
	}
	if cfg.SourceMedia != nil && cfg.SourceMedia.S3.Bucket == settings.Bucket && cfg.SourceMedia.S3.Endpoint == settings.Endpoint {
		return nil, errors.New("Chat images require a bucket separate from public source media")
	}
	clientConfig := settings.capacityClientConfig()
	namespace, err := s3integration.ResolveCapacityNamespace(ctx, clientConfig, settings.Bucket, settings.Mode)
	if err != nil {
		return nil, errors.New("Chat images storage identity is not admitted")
	}
	repo, err := capstore.New(db)
	if err != nil {
		return nil, err
	}
	if err = repo.EnsureNamespace(ctx, namespace, "chat-image/", func(ctx context.Context) error {
		return s3integration.VerifyEmptyCapacityPrefix(ctx, clientConfig, settings.Bucket, "chat-image/", settings.Mode)
	}); err != nil {
		return nil, errors.New("Chat images capacity namespace is not admitted")
	}
	client, err := s3integration.NewKnowledgeClient(clientConfig)
	if err != nil {
		return nil, errors.New("Chat images private storage client is unavailable")
	}
	objects, err := s3integration.NewUploaderWithOptions(client, s3integration.UploaderOptions{
		Bucket: settings.Bucket, ArtifactCapabilities: s3integration.ArtifactStorageCapabilities{
			Mode: s3integration.ArtifactStorageMode(settings.Mode), COSImmutableNonVersionedBucketPolicy: settings.COSImmutableNonVersionedBucketPolicy,
		},
	})
	if err != nil {
		return nil, err
	}
	return &ChatImageStorageRuntime{Objects: objects, Capacity: &storagecapacity.Meter{Writer: repo, Namespace: namespace}}, nil
}
