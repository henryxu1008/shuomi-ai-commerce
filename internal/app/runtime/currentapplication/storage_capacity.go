package currentapplication

import (
	"context"
	"errors"
	"gorm.io/gorm"
	coreconfig "task-processor/internal/core/config"
	commercialstore "task-processor/internal/integration/persistence/commercialbilling"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/integration/s3"
	"task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
)

func (s KnowledgeStorageConfig) capacityClientConfig() s3.ClientConfig {
	return s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: true, CapacityIdentityProvider: s.CapacityIdentityProvider, CapacityMinIODeploymentID: s.CapacityMinIODeploymentID}
}

func (s KnowledgeStorageConfig) validateCapacityIdentity() error {
	return s.capacityClientConfig().ValidateCapacityIdentity(s.Bucket, s.Mode)
}

func prepareStorageCapacity(ctx context.Context, c *Config, db *gorm.DB, service *knowledge.Service, processor *knowledge.Processor, imageWorker *coreconfig.Config, ecoservices *EcoservicesRuntime) error {
	if !c.StorageCapacity {
		if db == nil && c.CommercialOwnerDatabase == nil && c.SourceMedia == nil && (c.Knowledge == nil || !c.Knowledge.Enabled) && (c.Ecoservices == nil || !c.Ecoservices.Enabled) && c.ImageAgent == nil {
			return nil
		}
		return commercialstore.VerifyStorageCapacityDisabled(ctx, db)
	}
	if db == nil || c.MoneyOwnerDatabase == nil || c.CommercialOwnerDatabase == nil {
		return errors.New("enterprise storage requires commercial and money owners")
	}
	if c.ImageAgent != nil && (imageWorker == nil || !imageWorker.StorageCapacity) {
		return errors.New("storage capacity requires the metered organization image worker")
	}
	if c.ImageAgent != nil {
		owner := imageWorker.CommercialOwnerDatabase
		if owner == nil || !sameDatabaseTarget(*c.CommercialOwnerDatabase, DatabaseConfig{Host: owner.Host, Port: owner.Port, Database: owner.Database}) {
			return errors.New("storage capacity requires the same commercial owner for application and image worker")
		}
	}
	repo, e := store.New(db)
	if e != nil {
		return e
	}
	if e = store.VerifyRuntime(ctx, db); e != nil {
		return e
	}
	if c.Ecoservices != nil && c.Ecoservices.Enabled {
		s := c.Ecoservices.Storage
		client := s.capacityClientConfig()
		ns, err := s3.ResolveCapacityNamespace(ctx, client, s.Bucket, s.Mode)
		if err != nil {
			return errors.New("service-request attachment account admission failed")
		}
		if e = repo.EnsureNamespace(ctx, ns, "ecoservices/files/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, client, s.Bucket, "ecoservices/files/", s.Mode)
		}); e != nil {
			return errors.New("service-request attachment namespace admission failed")
		}
		if ecoservices == nil {
			return storagecapacity.ErrUnavailable
		}
		ecoservices.Capacity = &storagecapacity.Meter{Writer: repo, Namespace: ns}
	}
	if c.SourceMedia != nil {
		s := c.SourceMedia.S3
		ns, err := s3.ResolveSourceMediaCapacityNamespace(ctx, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle}, s.Bucket, s.ArtifactMode, c.SourceMediaIsolatedTrial, c.SourceMediaIsolatedTrialDeploymentID)
		if err != nil {
			return errors.New("source-media storage account admission failed")
		}
		e = repo.EnsureNamespace(ctx, ns, "product-source/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle}, s.Bucket, "product-source/", s.ArtifactMode)
		})
		if e != nil {
			return errors.New("source-media storage capacity namespace admission failed")
		}
	}
	if c.Knowledge != nil && c.Knowledge.Enabled {
		s := c.Knowledge.Storage
		client := s.capacityClientConfig()
		ns, err := s3.ResolveCapacityNamespace(ctx, client, s.Bucket, s.Mode)
		if err != nil {
			return errors.New("knowledge storage account admission failed")
		}
		e = repo.EnsureNamespace(ctx, ns, "knowledge/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, client, s.Bucket, "knowledge/", s.Mode)
		})
		if e != nil {
			return errors.New("knowledge storage capacity namespace admission failed")
		}
		if service == nil || processor == nil {
			return storagecapacity.ErrUnavailable
		}
		meter := &storagecapacity.Meter{Writer: repo, Namespace: ns}
		service.ConfigureStorageCapacity(meter)
		processor.ConfigureStorageCapacity(meter)
	}
	return nil
}
