package officialpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	submissionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"gorm.io/gorm"
)

func (r *OfficialRepository) ReadAmazonAttempt(ctx context.Context, scope submission.ExecutionScope, target submission.ExecutionTarget) (submission.ExecutionAttempt, error) {
	return r.core.ReadAmazonAttempt(ctx, scope, target)
}

func (r *OfficialRepository) CompleteAmazon(ctx context.Context, claim submission.ExecutionClaim, intent submission.OfficialIntent, receipt submission.AmazonReceipt) error {
	if intent.Amazon == nil || !receipt.Valid() || submission.ValidateOfficialIntent(intent) != nil ||
		claim.Scope.OrganizationID != intent.Owner.OrganizationID {
		return submission.ErrExecutionEvidenceRequired
	}
	ctx, cancel := context.WithTimeout(ctx, submission.ExecutionTimeout)
	defer cancel()
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return submission.ErrExecutionUnavailable
	}
	defer tx.Rollback()
	current, err := readOfficialIntent(tx, intent.Owner.OrganizationID, intent.Key, true)
	if err != nil || current.InputHash != intent.InputHash || current.Amazon == nil {
		return submission.ErrExecutionEvidenceRequired
	}
	finalizer, err := submissionstore.NewTransactionFinalizer(ctx, tx)
	if err != nil {
		return err
	}
	kernel, err := submission.NewExecutionKernel(finalizer)
	if err != nil {
		return err
	}
	attempt, err := kernel.Get(ctx, claim.Scope, claim.AttemptID)
	if err != nil || attempt.IntentKey != intent.Key || attempt.Target != intent.Target || attempt.PayloadFingerprint != intent.PayloadFingerprint {
		return submission.ErrExecutionEvidenceRequired
	}
	if _, err := kernel.Complete(ctx, claim, submission.AmazonReceiptEvidence(attempt, receipt.Accepted, time.Now())); err != nil {
		return err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return submission.ErrExecutionInvalid
	}
	written := tx.Exec(`INSERT INTO public.listing_amazon_receipts (organization_id,attempt_id,body) VALUES (?,?,CAST(? AS jsonb))`,
		attempt.OrganizationID, attempt.AttemptID, string(raw))
	if written.Error != nil || written.RowsAffected != 1 {
		return submission.ErrExecutionUnavailable
	}
	if tx.Commit().Error != nil {
		return submission.ErrExecutionOutcomeUnknown
	}
	return nil
}

func (r *OfficialRepository) ReadAmazonReceipt(ctx context.Context, owner collection.Scope, attempt submission.ExecutionAttempt) (submission.AmazonReceipt, error) {
	if attempt.OrganizationID != owner.OrganizationID || attempt.Target.Platform != "amazon" ||
		(attempt.Status != submission.ExecutionSucceeded && attempt.Status != submission.ExecutionFailedDefinitive) || attempt.Evidence == nil {
		return submission.AmazonReceipt{}, submission.ErrExecutionEvidenceRequired
	}
	var row struct{ Body []byte }
	err := r.db.WithContext(ctx).Table("public.listing_amazon_receipts").Select("body").Where("organization_id=? AND attempt_id=?", owner.OrganizationID, attempt.AttemptID).Take(&row).Error
	var receipt submission.AmazonReceipt
	if err != nil || json.Unmarshal(row.Body, &receipt) != nil || !receipt.Valid() ||
		submission.AmazonReceiptEvidence(attempt, receipt.Accepted, attempt.Evidence.ObservedAt) != *attempt.Evidence {
		return submission.AmazonReceipt{}, submission.ErrExecutionUnavailable
	}
	return receipt, nil
}

func (r *OfficialRepository) SaveAmazonObservation(ctx context.Context, attempt submission.ExecutionAttempt, observation submission.AmazonObservation) error {
	if attempt.Target.Platform != "amazon" || !observation.Valid() {
		return submission.ErrExecutionEvidenceRequired
	}
	raw, err := json.Marshal(observation)
	if err != nil {
		return submission.ErrExecutionInvalid
	}
	err = r.db.WithContext(ctx).Exec(`INSERT INTO public.listing_amazon_observations (organization_id,attempt_id,observed_at,body)
		VALUES (?,?,?,CAST(? AS jsonb))
		ON CONFLICT (organization_id,attempt_id) DO UPDATE SET observed_at=EXCLUDED.observed_at,body=EXCLUDED.body
		WHERE listing_amazon_observations.observed_at<EXCLUDED.observed_at`,
		attempt.OrganizationID, attempt.AttemptID, observation.ObservedAt, string(raw)).Error
	if err != nil {
		return submission.ErrExecutionUnavailable
	}
	return nil
}

func (r *OfficialRepository) ReadAmazonObservation(ctx context.Context, attempt submission.ExecutionAttempt) (*submission.AmazonObservation, error) {
	var row struct {
		Body       []byte
		ObservedAt time.Time
	}
	err := r.db.WithContext(ctx).Table("public.listing_amazon_observations").Select("body, observed_at").
		Where("organization_id=? AND attempt_id=?", attempt.OrganizationID, attempt.AttemptID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	var observation submission.AmazonObservation
	if err != nil || json.Unmarshal(row.Body, &observation) != nil || !observation.Valid() || !row.ObservedAt.Equal(observation.ObservedAt) {
		return nil, submission.ErrExecutionUnavailable
	}
	return &observation, nil
}
