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

const GooglePlanTable = "workbench_listing_google_upload_plans"

type googlePlanRow struct {
	OrganizationID, ID, ActorID, MemberID, Plan, Target, ProductHash, InventoryHash string
	Generation                                                                      int64
	SourceID                                                                        string
	CreatedAt                                                                       time.Time
}

func (googlePlanRow) TableName() string { return GooglePlanTable }

func InstallGooglePlanSchema(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`ALTER TABLE public.listing_submission_execution_attempts
			DROP CONSTRAINT listing_submission_execution_attempts_no_send_check,
			ADD CONSTRAINT listing_submission_execution_attempts_no_send_check CHECK ((
				evidence_kind IS DISTINCT FROM 'local_no_send' OR
				(((platform = 'tiktokshop' AND action IN ('tiktok_upload_image','tiktok_create_global','tiktok_publish_us'))
				OR (platform = 'googlemerchant' AND action = 'google_product_insert'))
				AND evidence_reference = attempt_id::text
				AND evidence_reason IN ('preflight_unavailable','facts_changed','access_recheck_failed','image_content_changed','payload_invalid')
				AND finished_at < lease_expires_at)
			) IS TRUE)`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`CREATE TABLE public.workbench_listing_google_upload_plans (
		organization_id VARCHAR(200) NOT NULL, id CHAR(36) NOT NULL,
		actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL,
		plan JSONB NOT NULL CHECK (jsonb_typeof(plan) = 'object'),
		target JSONB NOT NULL CHECK (jsonb_typeof(target) = 'object'),
		product_hash CHAR(64) NOT NULL, inventory_hash CHAR(64) NOT NULL,
		generation BIGINT NOT NULL CHECK (generation > 0),
		source_id CHAR(36) NOT NULL, created_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (organization_id,id))`).Error; err != nil {
			return err
		}
		return tx.Exec(`CREATE INDEX listing_google_upload_by_source ON public.workbench_listing_google_upload_plans
	(organization_id, actor_id, member_id, source_id, created_at DESC, id DESC)`).Error
	})
}

type GooglePlanStore struct{ db *gorm.DB }

func NewGooglePlanStore(ctx context.Context, db *gorm.DB) (*GooglePlanStore, error) {
	if db == nil {
		return nil, submission.ErrExecutionUnavailable
	}
	const readiness = `SELECT coalesce(has_table_privilege(to_regclass('public.workbench_listing_google_upload_plans'),'SELECT')
		AND has_table_privilege(to_regclass('public.workbench_listing_google_upload_plans'),'INSERT'),false)
		AND EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid=to_regclass('public.listing_submission_execution_attempts')
		  AND conname='listing_submission_execution_attempts_no_send_check' AND convalidated AND position('google_product_insert' in pg_get_constraintdef(oid))>0)`
	var installed bool
	if err := db.WithContext(ctx).Raw(readiness).Scan(&installed).Error; err != nil || !installed {
		return nil, submission.ErrExecutionUnavailable
	}
	return &GooglePlanStore{db: db}, nil
}

func (s *GooglePlanStore) Save(ctx context.Context, plan submission.GooglePlan) (submission.GooglePlan, error) {
	if !plan.Valid() {
		return submission.GooglePlan{}, submission.ErrExecutionInvalid
	}
	body, err := json.Marshal(plan)
	if err != nil || len(body) > submission.MaxExecutionPayloadBytes {
		return submission.GooglePlan{}, submission.ErrExecutionInvalid
	}
	target, err := json.Marshal(plan.Target)
	if err != nil {
		return submission.GooglePlan{}, submission.ErrExecutionInvalid
	}
	row := googlePlanRow{OrganizationID: plan.Owner.OrganizationID, ID: plan.ID,
		ActorID: plan.Owner.ActorID, MemberID: plan.Owner.MemberID, Plan: string(body), Target: string(target),
		ProductHash: plan.ProductHash, InventoryHash: plan.InventoryHash, Generation: plan.Generation,
		SourceID: plan.SourceID, CreatedAt: plan.CreatedAt}
	created := s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if created.Error != nil {
		return submission.GooglePlan{}, created.Error
	}
	prior, err := s.Read(ctx, plan.Owner, plan.ID)
	if err != nil {
		return submission.GooglePlan{}, err
	}
	if !plan.SameConfirmation(prior) {
		return submission.GooglePlan{}, submission.ErrExecutionIntentConflict
	}
	return prior, nil
}

func (s *GooglePlanStore) Read(ctx context.Context, scope submission.GooglePlanScope, id string) (submission.GooglePlan, error) {
	var plan submission.GooglePlan
	if scope.Validate() != nil || !submission.ValidGooglePlanID(id) {
		return plan, submission.ErrExecutionInvalid
	}
	var row googlePlanRow
	err := s.db.WithContext(ctx).Where("organization_id = ? AND id = ? AND actor_id = ? AND member_id = ?",
		scope.OrganizationID, id, scope.ActorID, scope.MemberID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return plan, submission.ErrExecutionNotFound
	}
	if err != nil {
		return plan, err
	}
	if json.Unmarshal([]byte(row.Plan), &plan) != nil || json.Unmarshal([]byte(row.Target), &plan.Target) != nil {
		return plan, submission.ErrExecutionUnavailable
	}
	plan.Owner, plan.ProductHash, plan.InventoryHash, plan.Generation = scope, row.ProductHash, row.InventoryHash, row.Generation
	if plan.ID != id || plan.SourceID != row.SourceID || !plan.CreatedAt.Equal(row.CreatedAt) || !plan.Valid() {
		return submission.GooglePlan{}, submission.ErrExecutionUnavailable
	}
	return plan, nil
}

func (s *GooglePlanStore) List(ctx context.Context, scope submission.GooglePlanScope, sourceID string) ([]submission.GoogleRecord, error) {
	if scope.Validate() != nil || !submission.ValidGooglePlanID(sourceID) {
		return nil, submission.ErrExecutionInvalid
	}
	records := make([]submission.GoogleRecord, 0)
	err := s.db.WithContext(ctx).Table(GooglePlanTable).Select("id, created_at").
		Where("organization_id = ? AND actor_id = ? AND member_id = ? AND source_id = ?",
			scope.OrganizationID, scope.ActorID, scope.MemberID, sourceID).
		Order("created_at DESC, id DESC").Limit(20).Scan(&records).Error
	return records, err
}
