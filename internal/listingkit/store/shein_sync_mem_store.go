package store

import (
	"sync"

	"task-processor/internal/listingkit/sheinsync"
)

type MemSheinSyncRepository struct {
	mu              sync.RWMutex
	nextProductID   int64
	nextJobID       int64
	nextCandidateID int64
	nextRunID       int64
	products        map[string]sheinsync.SheinSyncedProductRecord
	syncJobs        map[int64]sheinsync.SheinSyncJobRecord
	candidates      map[string]sheinsync.SheinActivityCandidateRecord
	enrollmentRuns  map[int64]sheinsync.SheinActivityEnrollmentRunRecord
	enrollmentItems map[string]sheinsync.SheinActivityEnrollmentItemRecord
	sdsCostGroups   map[string]sheinsync.SheinSDSCostGroupRecord
}

func NewMemSheinSyncRepository() sheinsync.SheinSyncRepository {
	return &MemSheinSyncRepository{
		nextProductID:   1,
		nextJobID:       1,
		nextCandidateID: 1,
		nextRunID:       1,
		products:        make(map[string]sheinsync.SheinSyncedProductRecord),
		syncJobs:        make(map[int64]sheinsync.SheinSyncJobRecord),
		candidates:      make(map[string]sheinsync.SheinActivityCandidateRecord),
		enrollmentRuns:  make(map[int64]sheinsync.SheinActivityEnrollmentRunRecord),
		enrollmentItems: make(map[string]sheinsync.SheinActivityEnrollmentItemRecord),
		sdsCostGroups:   make(map[string]sheinsync.SheinSDSCostGroupRecord),
	}
}
