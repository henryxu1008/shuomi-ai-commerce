package officialpersistence

import (
	"context"
	"strconv"
	"time"

	submissionstore "task-processor/internal/integration/persistence/listing/submission"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

func (r *OfficialRepository) ListTemuCreateAttemptIDs(ctx context.Context, scope collection.Scope, cohort submission.TemuCreateReadScope, start, end time.Time, limit int) ([]string, error) {
	if r == nil || r.db == nil || scope.Validate() != nil || !cohort.Valid() || !start.Before(end) || limit < 1 {
		return nil, submission.ErrExecutionInvalid
	}
	var ids []string
	err := r.db.WithContext(ctx).Table(submissionstore.AttemptTable+" AS a").
		Joins("JOIN listing_submission_official_intents i ON i.organization_id=a.organization_id AND i.intent_key=a.intent_key").
		Where("a.organization_id=? AND i.actor_id=? AND i.member_id=? AND a.platform='temu' AND a.store_id=? AND a.action=? AND i.kind='publish'",
			scope.OrganizationID, scope.ActorID, scope.MemberID, cohort.StoreID, submission.TemuPublishAction).
		Where("a.created_at>=? AND a.created_at<?", start, end).
		Where("a.status IN ?", []submission.ExecutionStatus{submission.ExecutionSucceeded, submission.ExecutionOutcomeUnknown}).
		Where("i.body_json#>>'{binding,site}'=? AND i.body_json#>>'{binding,application_id}'=? AND i.body_json#>>'{binding,supplier_identity_hash}'=?",
			cohort.Site, cohort.ApplicationID, cohort.MerchantIdentityHash).
		Where("i.body_json#>>'{binding,identity_partition}'=? AND i.body_json#>>'{binding,merchant_mode}'=? AND COALESCE(i.body_json#>>'{binding,region_id}','0')=?",
			cohort.IdentityPartition, cohort.MerchantMode, strconv.FormatInt(cohort.RegionID, 10)).
		Distinct("a.attempt_id").Limit(limit).Pluck("a.attempt_id", &ids).Error
	if err != nil {
		return nil, submission.ErrExecutionUnavailable
	}
	return ids, nil
}
