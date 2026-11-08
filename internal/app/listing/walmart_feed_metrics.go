package listingapp

import (
	"context"
	"time"

	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type cockpitSubmissionReader interface {
	ReadCockpitSubmissionAttempt(context.Context, collection.Scope, string) (submission.ExecutionAttempt, error)
	ReadOfficialIntent(context.Context, collection.Scope, string) (submission.OfficialIntent, error)
	ReadOfficial(context.Context, collection.Scope, string) (submission.OfficialReceipt, error)
}

type WalmartFeedMetricsSource interface {
	cockpitSubmissionReader
	ListWalmartFeedAttemptIDs(context.Context, collection.Scope, submission.WalmartFeedReadScope, time.Time, time.Time, int) ([]string, error)
}

type storeSubmissionMetrics struct {
	Status     string                 `json:"status"`
	CapturedAt time.Time              `json:"capturedAt"`
	Counts     *storeSubmissionCounts `json:"counts,omitempty"`
}

type storeSubmissionCounts struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Unknown  int `json:"unknown"`
}

func (a *Application) WalmartFeedMetrics(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (storeSubmissionMetrics, error) {
	result := storeSubmissionMetrics{CapturedAt: now.UTC()}
	if len(filter.Stores) != 1 || !filter.Start.Before(filter.End) {
		return result, record.ErrInvalid
	}
	if a.WalmartFeedSource == nil || a.WalmartFeedMerchant == nil || a.CockpitStoreAccess == nil || a.Authorization == nil || a.Records == nil {
		return result, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return result, err
	}
	storeID := filter.Stores[0]
	readBinding := func() (storecenter.ProductMerchantBinding, error) {
		if err := a.CockpitStoreAccess(ctx, scope, filter.Stores); err != nil {
			return storecenter.ProductMerchantBinding{}, err
		}
		binding, err := a.WalmartFeedMerchant(ctx, scope, storeID)
		if err != nil || !binding.ValidWalmartApplication() || binding.OrganizationID != scope.OrganizationID || binding.StoreID != storeID {
			return storecenter.ProductMerchantBinding{}, record.ErrUnavailable
		}
		return binding, nil
	}
	binding, err := readBinding()
	if err != nil {
		return result, err
	}
	cohort := submission.WalmartFeedReadScope{
		StoreID: binding.StoreID, MerchantIdentityHash: binding.SupplierIdentityHash, Environment: binding.Walmart.Environment,
	}
	ids, err := a.WalmartFeedSource.ListWalmartFeedAttemptIDs(ctx, scope, cohort, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
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
			if err := a.countWalmartFeedAttempt(ctx, scope, binding, filter, id, counts); err != nil {
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
	if !submission.WalmartCompatibleBinding(binding, current) {
		return storeSubmissionMetrics{}, record.ErrForbidden
	}
	return result, nil
}

func (a *Application) countWalmartFeedAttempt(ctx context.Context, scope collection.Scope, binding storecenter.ProductMerchantBinding, filter workbenchmetrics.Filter, id string, counts *storeSubmissionCounts) error {
	attempt, err := a.WalmartFeedSource.ReadCockpitSubmissionAttempt(ctx, scope, id)
	if err != nil || attempt.AttemptID != id || attempt.OrganizationID != scope.OrganizationID || attempt.Target.Platform != "walmart" ||
		attempt.Target.StoreID != binding.StoreID || attempt.Action != submission.WalmartSubmitAction ||
		attempt.CreatedAt.Before(filter.Start) || !attempt.CreatedAt.Before(filter.End) || submission.ValidatePersistedExecutionAttempt(attempt) != nil {
		return record.ErrUnavailable
	}
	intent, err := a.WalmartFeedSource.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil || intent.Owner != scope || intent.Key != attempt.IntentKey || intent.Target != attempt.Target ||
		intent.Kind != "publish" || intent.PayloadFingerprint != attempt.PayloadFingerprint || submission.ValidateOfficialIntent(intent) != nil {
		return record.ErrUnavailable
	}
	if !submission.WalmartCompatibleBinding(intent.Binding, binding) {
		return record.ErrUnavailable
	}
	saved, err := a.Records.ReadTargetRecord(ctx, scope, intent.RecordID)
	if err != nil || !submission.WalmartIntentMatchesRecord(intent, saved) {
		return record.ErrUnavailable
	}
	switch attempt.Status {
	case submission.ExecutionSucceeded:
		receipt, err := a.WalmartFeedSource.ReadOfficial(ctx, scope, id)
		if err != nil || receipt.ID != id || receipt.Owner != scope || receipt.IntentKey != intent.Key || receipt.Target != attempt.Target ||
			receipt.Binding != intent.Binding || receipt.PayloadFingerprint != attempt.PayloadFingerprint || receipt.ResponseHash != attempt.Evidence.Fingerprint ||
			submission.ValidateOfficialReceipt(receipt) != nil || !submission.WalmartReceiptMatchesIntent(receipt, intent) {
			return record.ErrUnavailable
		}
		counts.Accepted++
	case submission.ExecutionFailedDefinitive:
		if !submission.WalmartFeedRejectionMatches(intent, attempt) {
			return record.ErrUnavailable
		}
		counts.Rejected++
	case submission.ExecutionOutcomeUnknown:
		counts.Unknown++
	}
	return nil
}
