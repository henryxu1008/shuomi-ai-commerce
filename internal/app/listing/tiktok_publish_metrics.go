package listingapp

import (
	"context"
	"encoding/json"
	"time"

	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
)

type TikTokPublishMetricsSource interface {
	cockpitSubmissionReader
	ListTikTokGlobalPublishAttemptIDs(context.Context, collection.Scope, submission.TikTokGlobalPublishReadScope, time.Time, time.Time, int) ([]string, error)
	ReadTikTokGlobalReference(context.Context, collection.Scope, string) (submission.OfficialReceipt, error)
}

func (a *Application) TikTokPublishMetrics(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (storeSubmissionMetrics, error) {
	result := storeSubmissionMetrics{CapturedAt: now.UTC()}
	if len(filter.Stores) != 1 || !filter.Start.Before(filter.End) {
		return result, record.ErrInvalid
	}
	if a.TikTokPublishSource == nil || a.TikTokPublishMerchant == nil || a.CockpitStoreAccess == nil || a.Authorization == nil || a.Records == nil {
		return result, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return result, err
	}
	storeID := filter.Stores[0]
	readBinding := func() (submission.TikTokGlobalPublishReadScope, error) {
		if err := a.CockpitStoreAccess(ctx, scope, filter.Stores); err != nil {
			return submission.TikTokGlobalPublishReadScope{}, err
		}
		binding, err := a.TikTokPublishMerchant(ctx, scope, storeID)
		if err != nil || !binding.Valid() || binding.StoreID != storeID {
			return submission.TikTokGlobalPublishReadScope{}, record.ErrUnavailable
		}
		return binding, nil
	}
	binding, err := readBinding()
	if err != nil {
		return result, err
	}
	ids, err := a.TikTokPublishSource.ListTikTokGlobalPublishAttemptIDs(ctx, scope, binding, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
	if err != nil {
		return result, record.ErrUnavailable
	}
	result.Status = "limited"
	if len(ids) <= workbenchmetrics.MaxCandidates {
		counts := &storeSubmissionCounts{}
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if err := a.countTikTokPublishAttempt(ctx, scope, binding, filter, id, counts); err != nil {
				return storeSubmissionMetrics{}, err
			}
		}
		result.Status, result.Counts = "complete", counts
	}
	fresh, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return storeSubmissionMetrics{}, err
	}
	if fresh != scope {
		return storeSubmissionMetrics{}, record.ErrForbidden
	}
	current, err := readBinding()
	if err != nil {
		return storeSubmissionMetrics{}, err
	}
	if current != binding {
		return storeSubmissionMetrics{}, record.ErrForbidden
	}
	return result, nil
}

func (a *Application) countTikTokPublishAttempt(ctx context.Context, scope collection.Scope, binding submission.TikTokGlobalPublishReadScope, filter workbenchmetrics.Filter, id string, counts *storeSubmissionCounts) error {
	source := a.TikTokPublishSource
	attempt, err := source.ReadCockpitSubmissionAttempt(ctx, scope, id)
	if err != nil || attempt.AttemptID != id || attempt.OrganizationID != scope.OrganizationID || attempt.Target.Platform != "tiktokshop" ||
		attempt.Target.StoreID != binding.StoreID || attempt.Action != submission.TikTokPublishAction ||
		attempt.CreatedAt.Before(filter.Start) || !attempt.CreatedAt.Before(filter.End) || submission.ValidatePersistedExecutionAttempt(attempt) != nil {
		return record.ErrUnavailable
	}
	intent, err := source.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil || intent.Owner != scope || intent.Key != attempt.IntentKey || intent.Target != attempt.Target ||
		intent.Kind != submission.TikTokPublishStage || intent.TikTok == nil || intent.TikTok.Global == nil ||
		intent.PayloadFingerprint != attempt.PayloadFingerprint || !binding.MatchesBinding(intent.Binding) {
		return record.ErrUnavailable
	}
	saved, err := a.Records.ReadTargetRecord(ctx, scope, intent.RecordID)
	if err != nil || saved.ID != intent.RecordID || saved.Source != intent.Source || saved.Merchant != intent.Binding ||
		collection.Digest(saved) != intent.RecordHash || saved.Input.TikTok == nil ||
		saved.Input.TikTok.Selection.Method != marketplace.GlobalPublishing || saved.Input.TikTok.SaveMode != "LISTING" {
		return record.ErrUnavailable
	}
	global, err := source.ReadTikTokGlobalReference(ctx, scope, intent.Key)
	if err != nil {
		return record.ErrUnavailable
	}
	payload, effect, err := submission.ResolveTikTokPayload(saved, submission.TikTokPublishStage, intent.Key, nil, nil, &global)
	if err != nil || effect.Global == nil || *effect.Global != *intent.TikTok.Global ||
		effect.DraftHash != intent.TikTok.DraftHash || !submission.TikTokPayloadMatchesIntent(intent, payload) {
		return record.ErrUnavailable
	}
	switch attempt.Status {
	case submission.ExecutionOutcomeUnknown:
		counts.Unknown++
	case submission.ExecutionSucceeded:
		receipt, err := source.ReadOfficial(ctx, scope, id)
		if err != nil || receipt.ID != id || receipt.Owner != scope || receipt.IntentKey != intent.Key ||
			receipt.Kind != intent.Kind || receipt.Target != attempt.Target || receipt.Binding != intent.Binding ||
			receipt.PayloadFingerprint != attempt.PayloadFingerprint || receipt.ResponseHash != attempt.Evidence.Fingerprint ||
			submission.ValidateOfficialReceipt(receipt) != nil || !submission.TikTokReceiptMatchesIntent(receipt, intent) ||
			receipt.TikTok.Publication == nil {
			return record.ErrUnavailable
		}
		var sent marketplace.PublishGlobalRequest
		if json.Unmarshal(payload, &sent) != nil || !marketplace.CorrelatesGlobalPublication(*receipt.TikTok.Publication, sent, binding.ShopID) {
			return record.ErrUnavailable
		}
		switch receipt.TikTok.Publication.State {
		case marketplace.SubmittedForReview:
			counts.Accepted++
		case marketplace.PublicationFailed, marketplace.RemoteDraft:
			counts.Rejected++
		default:
			return record.ErrUnavailable
		}
	}
	return nil
}
