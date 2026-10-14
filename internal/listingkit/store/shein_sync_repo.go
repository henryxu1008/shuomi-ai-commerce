package store

import (
	"gorm.io/gorm"

	"task-processor/internal/listingkit/sheinsync"
)

type GormSheinSyncRepository struct {
	db *gorm.DB
}

func NewSheinSyncRepository(db *gorm.DB) sheinsync.SheinSyncRepository {
	return &GormSheinSyncRepository{db: db}
}

func AutoMigrateSheinSyncRepository(db *gorm.DB) error {
	return db.AutoMigrate(
		&sheinsync.SheinSyncedProductRecord{},
		&sheinsync.SheinSDSCostGroupRecord{},
		&sheinsync.SheinSyncJobRecord{},
		&sheinsync.SheinActivityCandidateRecord{},
		&sheinsync.SheinActivityEnrollmentRunRecord{},
		&sheinsync.SheinActivityEnrollmentItemRecord{},
	)
}
