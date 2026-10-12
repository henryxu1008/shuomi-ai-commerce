package imageagentworker

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/objectstore"
	"task-processor/internal/imageagent/temporal"
	"task-processor/internal/integration/httpimage"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/integration/s3"
	"time"
)

func configureStorageCapacity(cfg *config.Config, commercial *gorm.DB, repository imageagent.Repository, artifacts temporal.DurableArtifactStore, generation *imageagent.GenerationExecution, recovery imageagent.GenerationRecovery) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if cfg == nil {
		return errors.New("storage capacity configuration unavailable")
	}
	if !cfg.StorageCapacity {
		return commercialstore.VerifyStorageCapacityDisabled(ctx, commercial)
	}
	if commercial == nil {
		return errors.New("storage capacity requires its commercial owner")
	}
	if e := store.VerifyRuntime(ctx, commercial); e != nil {
		return e
	}
	capacity, e := store.New(commercial)
	if e != nil {
		return e
	}
	facts, ok := repository.(imageagent.StorageGenerationFactReader)
	if !ok {
		return errors.New("storage capacity requires original generation facts")
	}
	artifactStore, ok := artifacts.(*objectstore.DurableArtifactStore)
	if !ok {
		return errors.New("storage capacity requires immutable durable artifacts")
	}
	s := cfg.ImageAgent.ArtifactStore.S3
	ns, e := s3.ResolveCapacityNamespace(ctx, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle}, s.Bucket, s.ArtifactMode)
	if e != nil {
		return errors.New("image storage account admission failed")
	}
	prefix := imageagent.PublishedArtifactPrefix + "/"
	if e = capacity.EnsureNamespace(ctx, ns, prefix, func(check context.Context) error {
		return s3.VerifyEmptyCapacityPrefix(check, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle}, s.Bucket, prefix, s.ArtifactMode)
	}); e != nil {
		return errors.New("image storage capacity namespace admission failed")
	}
	if generation != nil {
		generation.ConfigureStorageCapacity(capacity, httpimage.DefaultMaxBodyBytes)
	}
	imageagent.ConfigureGenerationRecoveryStorage(recovery, capacity)
	artifactStore.ConfigureStorageCapacity(capacity, ns, facts)
	return nil
}
