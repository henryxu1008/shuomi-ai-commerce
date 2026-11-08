package officialpersistence

import (
	"context"
	"time"

	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func (r *OfficialRepository) ListTikTokGlobalPublishAttemptIDs(ctx context.Context, scope collection.Scope, cohort submission.TikTokGlobalPublishReadScope, start, end time.Time, limit int) ([]string, error) {
	if r == nil || r.db == nil || scope.Validate() != nil || !cohort.Valid() || !start.Before(end) || limit < 1 {
		return nil, submission.ErrExecutionInvalid
	}
	var ids []string
	err := r.db.WithContext(ctx).Table(submissionstore.AttemptTable+" AS a").
		Joins("JOIN listing_submission_official_intents i ON i.organization_id=a.organization_id AND i.intent_key=a.intent_key").
		Where("a.organization_id=? AND i.actor_id=? AND i.member_id=? AND a.platform='tiktokshop' AND a.store_id=? AND a.action=? AND i.kind='publish'",
			scope.OrganizationID, scope.ActorID, scope.MemberID, cohort.StoreID, submission.TikTokPublishAction).
		Where("a.created_at>=? AND a.created_at<?", start, end).
		Where("a.status IN ?", []submission.ExecutionStatus{submission.ExecutionSucceeded, submission.ExecutionOutcomeUnknown}).
		Where("i.body_json#>>'{binding,application_id}'=? AND i.body_json#>>'{binding,supplier_identity_hash}'=? AND i.body_json#>>'{binding,tiktok,shop_id}'=?",
			cohort.ApplicationID, cohort.MerchantIdentityHash, cohort.ShopID).
		Where("i.body_json#>>'{tiktok,global,id}' IS NOT NULL").
		Distinct("a.attempt_id").Limit(limit).Pluck("a.attempt_id", &ids).Error
	if err != nil {
		return nil, submission.ErrExecutionUnavailable
	}
	return ids, nil
}

// ReadTikTokGlobalReference follows the exact prerequisite of an owned final intent.
// The original publish path permits a compatible global creation by another member.
func (r *OfficialRepository) ReadTikTokGlobalReference(ctx context.Context, scope collection.Scope, finalKey string) (submission.OfficialReceipt, error) {
	ctx, cancel := context.WithTimeout(ctx, submission.ExecutionTimeout)
	defer cancel()
	intent, err := r.ReadOfficialIntent(ctx, scope, finalKey)
	if err != nil || intent.Kind != submission.TikTokPublishStage || intent.TikTok == nil || intent.TikTok.Global == nil {
		return submission.OfficialReceipt{}, submission.ErrExecutionUnavailable
	}
	ref := *intent.TikTok.Global
	receipt, err := readOfficialReceipt(r.db.WithContext(ctx), scope.OrganizationID, ref.ID)
	if err != nil || receipt.Kind != submission.TikTokGlobalCreateStage || receipt.TikTok == nil || receipt.TikTok.Global == nil ||
		submission.TikTokReceiptRef(receipt) != ref || !submission.TikTokCompatibleBinding(receipt.Binding, intent.Binding) ||
		receipt.ProductKey != intent.Source.Source.ProductKey || receipt.TikTok.DraftHash != intent.TikTok.DraftHash {
		return submission.OfficialReceipt{}, submission.ErrExecutionUnavailable
	}
	if err := r.validateCompleted(ctx, receipt); err != nil {
		return submission.OfficialReceipt{}, err
	}
	return receipt, nil
}
