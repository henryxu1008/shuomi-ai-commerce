package imageagentworker

import (
	"context"
	"errors"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/objectstore"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent/temporal"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
	commercialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/commercialbilling"
	store "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/storagecapacity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/s3"
	"gorm.io/gorm"
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
	client := imageStorageCapacityClientConfig(s)
	ns, e := s3.ResolveCapacityNamespace(ctx, client, s.Bucket, s.ArtifactMode)
	if e != nil {
		return errors.New("image storage account admission failed")
	}
	prefix := imageagent.PublishedArtifactPrefix + "/"
	if e = capacity.EnsureNamespace(ctx, ns, prefix, func(check context.Context) error {
		return s3.VerifyEmptyCapacityPrefix(check, client, s.Bucket, prefix, s.ArtifactMode)
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

func imageStorageCapacityClientConfig(s config.ImageAgentArtifactStoreS3Config) s3.ClientConfig {
	return s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle, CapacityIdentityProvider: s.CapacityIdentityProvider, CapacityMinIODeploymentID: s.CapacityMinIODeploymentID}
}
