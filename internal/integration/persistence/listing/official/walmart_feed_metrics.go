package officialpersistence

import (
	"context"
	"time"

	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func (r *OfficialRepository) ListWalmartFeedAttemptIDs(ctx context.Context, scope collection.Scope, cohort submission.WalmartFeedReadScope, start, end time.Time, limit int) ([]string, error) {
	if r == nil || r.db == nil || scope.Validate() != nil || !cohort.Valid() || !start.Before(end) || limit < 1 {
		return nil, submission.ErrExecutionInvalid
	}
	var ids []string
	err := r.db.WithContext(ctx).Table(submissionstore.AttemptTable+" AS a").
		Joins("JOIN listing_submission_official_intents i ON i.organization_id=a.organization_id AND i.intent_key=a.intent_key").
		Where("a.organization_id=? AND i.actor_id=? AND i.member_id=? AND a.platform='walmart' AND a.store_id=? AND a.action=? AND i.kind='publish'",
			scope.OrganizationID, scope.ActorID, scope.MemberID, cohort.StoreID, submission.WalmartSubmitAction).
		Where("a.created_at>=? AND a.created_at<?", start, end).
		Where("i.body_json->'binding'->>'supplier_identity_hash'=? AND i.body_json#>>'{binding,walmart,environment}'=?",
			cohort.MerchantIdentityHash, cohort.Environment).
		Distinct("a.attempt_id").Limit(limit).Pluck("a.attempt_id", &ids).Error
	if err != nil {
		return nil, submission.ErrExecutionUnavailable
	}
	return ids, nil
}

func (r *OfficialRepository) ReadWalmartFeedAttempt(ctx context.Context, scope collection.Scope, id string) (submission.ExecutionAttempt, error) {
	if r == nil || r.core == nil || scope.Validate() != nil || !collection.ValidID(id) {
		return submission.ExecutionAttempt{}, submission.ErrExecutionInvalid
	}
	return r.core.Get(ctx, submission.ExecutionScope{OrganizationID: scope.OrganizationID}, id)
}
