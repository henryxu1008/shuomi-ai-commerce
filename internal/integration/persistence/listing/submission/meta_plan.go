package submissionpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type metaPlanRow struct {
	OrganizationID, ID, ActorID, MemberID, SourceID, Plan, ProductHash, InventoryHash string
	Version                                                                           uint64
	Generation                                                                        int64
	CreatedAt                                                                         time.Time
}

func (metaPlanRow) TableName() string { return "workbench_listing_meta_simulation_plans" }

func InstallMetaPlanSchema(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TABLE public.workbench_listing_meta_simulation_plans (
   organization_id VARCHAR(200) NOT NULL, id CHAR(36) NOT NULL,
   actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL,
   source_id CHAR(36) NOT NULL, created_at TIMESTAMPTZ NOT NULL,
   plan JSONB NOT NULL CHECK (jsonb_typeof(plan)='object'),
   product_hash CHAR(64) NOT NULL, inventory_hash CHAR(64) NOT NULL,
   generation BIGINT NOT NULL CHECK (generation>0), version BIGINT NOT NULL CHECK (version>0),
   PRIMARY KEY (organization_id,id)
  )`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE INDEX listing_meta_simulation_by_source ON public.workbench_listing_meta_simulation_plans
   (organization_id,actor_id,member_id,source_id,created_at DESC,id DESC)`).Error
	})
}

type MetaPlanStore struct{ db *gorm.DB }

func NewMetaPlanStore(ctx context.Context, db *gorm.DB) (*MetaPlanStore, error) {
	if db == nil {
		return nil, submission.ErrExecutionUnavailable
	}
	var ready bool
	if err := db.WithContext(ctx).Raw("SELECT to_regclass('public.workbench_listing_meta_simulation_plans') IS NOT NULL").Scan(&ready).Error; err != nil || !ready {
		return nil, submission.ErrExecutionUnavailable
	}
	return &MetaPlanStore{db}, nil
}

func (s *MetaPlanStore) Save(ctx context.Context, plan submission.MetaPlan) (submission.MetaPlan, error) {
	if !plan.Valid() {
		return submission.MetaPlan{}, submission.ErrExecutionInvalid
	}
	data, err := json.Marshal(plan)
	if err != nil {
		return submission.MetaPlan{}, err
	}
	row := metaPlanRow{
		OrganizationID: plan.Owner.OrganizationID, ID: plan.ID, ActorID: plan.Owner.ActorID, MemberID: plan.Owner.MemberID, SourceID: plan.SourceID,
		Plan: string(data), ProductHash: plan.ProductHash, InventoryHash: plan.InventoryHash, Version: plan.Version, Generation: plan.Generation, CreatedAt: plan.CreatedAt,
	}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return submission.MetaPlan{}, err
	}
	saved, err := s.Read(ctx, plan.Owner, plan.ID)
	if err != nil {
		return saved, err
	}
	if !plan.SameConfirmation(saved) {
		return submission.MetaPlan{}, submission.ErrExecutionIntentConflict
	}
	return saved, nil
}

func (s *MetaPlanStore) Read(ctx context.Context, scope submission.MetaPlanScope, id string) (submission.MetaPlan, error) {
	var row metaPlanRow
	var plan submission.MetaPlan
	err := s.db.WithContext(ctx).Where("organization_id = ? AND actor_id = ? AND member_id = ? AND id = ?", scope.OrganizationID, scope.ActorID, scope.MemberID, id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return plan, submission.ErrExecutionNotFound
	}
	if err != nil {
		return plan, err
	}
	if json.Unmarshal([]byte(row.Plan), &plan) != nil {
		return plan, submission.ErrExecutionUnavailable
	}
	plan.Owner, plan.Generation, plan.Version, plan.ProductHash, plan.InventoryHash = scope, row.Generation, row.Version, row.ProductHash, row.InventoryHash
	if plan.ID != id || plan.SourceID != row.SourceID || !plan.CreatedAt.Equal(row.CreatedAt) || !plan.Valid() {
		return submission.MetaPlan{}, submission.ErrExecutionUnavailable
	}
	return plan, nil
}

func (s *MetaPlanStore) List(ctx context.Context, scope submission.MetaPlanScope, sourceID string) ([]submission.MetaRecord, error) {
	records := []submission.MetaRecord{}
	err := s.db.WithContext(ctx).Model(&metaPlanRow{}).Select("id, created_at").
		Where("organization_id = ? AND actor_id = ? AND member_id = ? AND source_id = ?", scope.OrganizationID, scope.ActorID, scope.MemberID, sourceID).
		Order("created_at DESC, id DESC").Limit(20).Scan(&records).Error
	return records, err
}
