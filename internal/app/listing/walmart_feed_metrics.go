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

type WalmartFeedMetricsSource interface {
	ListWalmartFeedAttemptIDs(context.Context, collection.Scope, storecenter.ProductMerchantBinding, time.Time, time.Time, int) ([]string, error)
	ReadWalmartFeedAttempt(context.Context, collection.Scope, string) (submission.ExecutionAttempt, error)
	ReadOfficialIntent(context.Context, collection.Scope, string) (submission.OfficialIntent, error)
	ReadOfficial(context.Context, collection.Scope, string) (submission.OfficialReceipt, error)
}

type walmartFeedMetrics struct {
	Status     string                   `json:"status"`
	CapturedAt time.Time                `json:"capturedAt"`
	Counts     *walmartFeedMetricCounts `json:"counts,omitempty"`
}

type walmartFeedMetricCounts struct {
	Accepted int `json:"accepted"`
	Rejected int `json:"rejected"`
	Unknown  int `json:"unknown"`
}

func (a *Application) WalmartFeedMetrics(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (walmartFeedMetrics, error) {
	result := walmartFeedMetrics{CapturedAt: now.UTC()}
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
	ids, err := a.WalmartFeedSource.ListWalmartFeedAttemptIDs(ctx, scope, binding, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
	if err != nil {
		return result, record.ErrUnavailable
	}
	result.Status = "limited"
	if len(ids) <= workbenchmetrics.MaxCandidates {
		counts := &walmartFeedMetricCounts{}
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if err := a.countWalmartFeedAttempt(ctx, scope, binding, filter, id, counts); err != nil {
				return walmartFeedMetrics{}, err
			}
		}
		result.Status, result.Counts = "complete", counts
	}
	fresh, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return walmartFeedMetrics{}, err
	}
	if fresh != scope {
		return walmartFeedMetrics{}, record.ErrForbidden
	}
	current, err := readBinding()
	if err != nil {
		return walmartFeedMetrics{}, err
	}
	if !submission.WalmartCompatibleBinding(binding, current) {
		return walmartFeedMetrics{}, record.ErrForbidden
	}
	return result, nil
}

func (a *Application) countWalmartFeedAttempt(ctx context.Context, scope collection.Scope, binding storecenter.ProductMerchantBinding, filter workbenchmetrics.Filter, id string, counts *walmartFeedMetricCounts) error {
	attempt, err := a.WalmartFeedSource.ReadWalmartFeedAttempt(ctx, scope, id)
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
