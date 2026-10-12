package currentapplication

import (
	"context"
	"errors"
	"gorm.io/gorm"
	coreconfig "task-processor/internal/core/config"
	store "task-processor/internal/integration/persistence/storagecapacity"
	"task-processor/internal/integration/s3"
	"task-processor/internal/knowledge"
	"task-processor/internal/ledger/storagecapacity"
)

func prepareStorageCapacity(ctx context.Context, c *Config, db *gorm.DB, service *knowledge.Service, processor *knowledge.Processor, imageWorker *coreconfig.Config, ecoservices *EcoservicesRuntime) error {
	if !c.StorageCapacity {
		return nil
	}
	if db == nil || c.MoneyOwnerDatabase == nil {
		return errors.New("enterprise storage requires commercial and money owners")
	}
	if c.ImageAgent != nil && (imageWorker == nil || !imageWorker.StorageCapacity) {
		return errors.New("storage capacity requires the metered organization image worker")
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
		ns := s3.CapacityNamespace(s.Mode, s.Endpoint, s.Region, s.Bucket)
		if e = repo.EnsureNamespace(ctx, ns, "ecoservices/files/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: true}, s.Bucket, "ecoservices/files/", s.Mode)
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
		ns := s3.CapacityNamespace(s.ArtifactMode, s.Endpoint, s.Region, s.Bucket)
		e = repo.EnsureNamespace(ctx, ns, "product-source/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: s.UsePathStyle}, s.Bucket, "product-source/", s.ArtifactMode)
		})
		if e != nil {
			return errors.New("source-media storage capacity namespace admission failed")
		}
	}
	if c.Knowledge != nil && c.Knowledge.Enabled {
		s := c.Knowledge.Storage
		ns := s3.CapacityNamespace(s.Mode, s.Endpoint, s.Region, s.Bucket)
		e = repo.EnsureNamespace(ctx, ns, "knowledge/", func(check context.Context) error {
			return s3.VerifyEmptyCapacityPrefix(check, s3.ClientConfig{Region: s.Region, Endpoint: s.Endpoint, AccessKeyID: s.AccessKeyID, SecretAccessKey: s.SecretAccessKey, UsePathStyle: true}, s.Bucket, "knowledge/", s.Mode)
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
