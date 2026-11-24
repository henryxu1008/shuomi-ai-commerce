package submissionpersistence

import (
	"context"
	"strings"
	"time"

	recordstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"gorm.io/gorm"
)

func requireAmazonEffect(ctx context.Context, tx *gorm.DB, reservation submission.ExecutionReservation) error {
	intent, partition, err := reservation.Amazon.Read(ctx)
	claim := reservation.Attempt
	if err != nil || claim.IntentKey != intent.Key || claim.Target != intent.Target ||
		claim.OrganizationID != intent.Owner.OrganizationID || claim.PayloadFingerprint != intent.PayloadFingerprint ||
		claim.Action != submission.AmazonSubmitAction {
		return submission.ErrExecutionEvidenceRequired
	}
	var reference struct{ TargetID string }
	owner := intent.Owner
	if err := tx.Raw(`SELECT target_id FROM public.listing_target_records WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?`,
		owner.OrganizationID, owner.ActorID, owner.MemberID, intent.RecordID).Scan(&reference).Error; err != nil || reference.TargetID == "" {
		return submission.ErrExecutionEvidenceRequired
	}
	if err := tx.Exec(`SELECT 1 FROM public.listing_preparation_targets WHERE organization_id=? AND actor_id=? AND member_id=? AND id=? FOR UPDATE`,
		owner.OrganizationID, owner.ActorID, owner.MemberID, reference.TargetID).Error; err != nil {
		return submission.ErrExecutionUnavailable
	}
	records, err := recordstore.NewRepository(ctx, tx)
	if err != nil {
		return err
	}
	saved, err := records.ReadTargetHead(ctx, owner, reference.TargetID)
	if err != nil || !submission.AmazonIntentMatchesRecord(intent, saved) || !time.Now().Before(saved.Amazon.ValidUntil) {
		return submission.ErrExecutionIntentConflict
	}
	env := "production"
	if intent.Binding.AmazonBeta {
		env = "beta"
	}
	productKey := intent.Source.Source.ProductKey
	gtin := saved.Input.Amazon.GTIN
	if len(gtin) < 12 || len(gtin) > 14 {
		return submission.ErrExecutionEvidenceRequired
	}
	normalizedGTIN := strings.Repeat("0", 14-len(gtin)) + gtin
	if err := tx.Exec(`INSERT INTO public.listing_amazon_sku_occupancy
		(environment, marketplace, seller_partition, identity_kind, identity_value, organization_id, store_id, product_key)
		VALUES (?, 'ATVPDKIKX0DER', ?, 'gtin', ?, ?, ?, ?), (?, 'ATVPDKIKX0DER', ?, 'sku', ?, ?, ?, ?)
		ON CONFLICT DO NOTHING`, env, partition, normalizedGTIN, owner.OrganizationID, intent.Target.StoreID, productKey,
		env, partition, intent.Amazon.SKU, owner.OrganizationID, intent.Target.StoreID, productKey).Error; err != nil {
		return submission.ErrExecutionUnavailable
	}
	var sameOwner bool
	if err := tx.Raw(`SELECT count(*)=2 FROM public.listing_amazon_sku_occupancy
		WHERE environment=? AND marketplace='ATVPDKIKX0DER' AND seller_partition=?
		AND ((identity_kind='gtin' AND identity_value=?) OR (identity_kind='sku' AND identity_value=?))
		AND organization_id=? AND store_id=? AND product_key=?`,
		env, partition, normalizedGTIN, intent.Amazon.SKU, owner.OrganizationID, intent.Target.StoreID, productKey).Scan(&sameOwner).Error; err != nil || !sameOwner {
		return submission.ErrExecutionTargetClaimed
	}
	body, bodyHash, err := submission.EncodeOfficialIntent(intent)
	if err != nil {
		return err
	}
	written := tx.Exec(`INSERT INTO public.listing_submission_official_intents
		(organization_id, actor_id, member_id, intent_key, record_id, kind, input_hash, body_hash, body_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, CAST(? AS jsonb))`, owner.OrganizationID, owner.ActorID, owner.MemberID,
		intent.Key, intent.RecordID, intent.Kind, intent.InputHash, bodyHash, string(body))
	if written.Error != nil || written.RowsAffected != 1 {
		return submission.ErrExecutionUnavailable
	}
	return nil
}

func (r *Repository) ReadAmazonAttempt(ctx context.Context, scope submission.ExecutionScope, target submission.ExecutionTarget) (submission.ExecutionAttempt, error) {
	if r == nil || ctx == nil || !submission.ValidAmazonTarget(target) {
		return submission.ExecutionAttempt{}, submission.ErrExecutionInvalid
	}
	fence, found, err := r.findTarget(ctx, r.db, scope.OrganizationID, target, false)
	if err != nil {
		return submission.ExecutionAttempt{}, err
	}
	if !found {
		return submission.ExecutionAttempt{}, submission.ErrExecutionNotFound
	}
	attempt, err := r.Get(ctx, scope, fence.CurrentAttemptID)
	if err != nil || attempt.Target != target || attempt.FenceEpoch != fence.Epoch || string(attempt.Status) != fence.CurrentStatus {
		return submission.ExecutionAttempt{}, submission.ErrExecutionUnavailable
	}
	return attempt, nil
}
