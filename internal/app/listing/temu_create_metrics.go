package listingapp

import (
	"context"
	"time"

	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	"task-processor/internal/product/collection"
)

type TemuCreateMetricsSource interface {
	cockpitSubmissionReader
	ListTemuCreateAttemptIDs(context.Context, collection.Scope, submission.TemuCreateReadScope, time.Time, time.Time, int) ([]string, error)
}

func (a *Application) TemuCreateMetrics(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (publicationMetrics, error) {
	result := publicationMetrics{CapturedAt: now.UTC()}
	if len(filter.Stores) != 1 || !filter.Start.Before(filter.End) {
		return result, record.ErrInvalid
	}
	if a.TemuCreateSource == nil || a.TemuCreateMerchant == nil || a.CockpitStoreAccess == nil || a.Authorization == nil || a.Records == nil {
		return result, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return result, err
	}
	storeID := filter.Stores[0]
	readBinding := func() (submission.TemuCreateReadScope, error) {
		if err := a.CockpitStoreAccess(ctx, scope, filter.Stores); err != nil {
			return submission.TemuCreateReadScope{}, err
		}
		b, err := a.TemuCreateMerchant(ctx, scope, storeID)
		cohort := submission.TemuCreateReadScope{
			StoreID: storeID, ApplicationID: b.ApplicationID, MerchantIdentityHash: b.SupplierIdentityHash,
			Site: b.Site, IdentityPartition: b.IdentityPartition, MerchantMode: b.MerchantMode, RegionID: b.RegionID,
		}
		if err != nil || b.OrganizationID != scope.OrganizationID || !cohort.Valid() || !cohort.MatchesBinding(b) {
			return submission.TemuCreateReadScope{}, record.ErrUnavailable
		}
		return cohort, nil
	}
	cohort, err := readBinding()
	if err != nil {
		return result, err
	}
	ids, err := a.TemuCreateSource.ListTemuCreateAttemptIDs(ctx, scope, cohort, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
	if err != nil {
		return result, record.ErrUnavailable
	}
	result.Status = "limited"
	if len(ids) <= workbenchmetrics.MaxCandidates {
		counts := &publicationCounts{}
		seen := make(map[string]bool, len(ids))
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			if err := a.countTemuCreateAttempt(ctx, scope, cohort, filter, id, counts); err != nil {
				return publicationMetrics{}, err
			}
		}
		result.Status, result.Counts = "complete", counts
	}
	fresh, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return publicationMetrics{}, err
	}
	if fresh != scope {
		return publicationMetrics{}, record.ErrForbidden
	}
	current, err := readBinding()
	if err != nil {
		return publicationMetrics{}, err
	}
	if current != cohort {
		return publicationMetrics{}, record.ErrForbidden
	}
	return result, nil
}

func (a *Application) countTemuCreateAttempt(ctx context.Context, scope collection.Scope, cohort submission.TemuCreateReadScope, filter workbenchmetrics.Filter, id string, counts *publicationCounts) error {
	source := a.TemuCreateSource
	attempt, err := source.ReadCockpitSubmissionAttempt(ctx, scope, id)
	if err != nil || attempt.AttemptID != id || attempt.OrganizationID != scope.OrganizationID ||
		attempt.Target.Platform != "temu" || attempt.Target.StoreID != cohort.StoreID ||
		attempt.Action != submission.TemuPublishAction || attempt.CreatedAt.Before(filter.Start) ||
		!attempt.CreatedAt.Before(filter.End) || submission.ValidatePersistedExecutionAttempt(attempt) != nil {
		return record.ErrUnavailable
	}
	intent, err := source.ReadOfficialIntent(ctx, scope, attempt.IntentKey)
	if err != nil || intent.Owner != scope || intent.Key != attempt.IntentKey || intent.Target != attempt.Target ||
		intent.Kind != "publish" || intent.PayloadFingerprint != attempt.PayloadFingerprint ||
		submission.ValidateOfficialIntent(intent) != nil || !cohort.MatchesBinding(intent.Binding) {
		return record.ErrUnavailable
	}
	saved, err := a.Records.ReadTargetRecord(ctx, scope, intent.RecordID)
	if err != nil || saved.ID != intent.RecordID || saved.Source != intent.Source || saved.Merchant != intent.Binding ||
		collection.Digest(saved) != intent.RecordHash || !record.PublicationReady(saved) {
		return record.ErrUnavailable
	}
	switch attempt.Status {
	case submission.ExecutionOutcomeUnknown:
		counts.Unknown++
	case submission.ExecutionSucceeded:
		receipt, err := source.ReadOfficial(ctx, scope, id)
		if err != nil || receipt.ID != id || receipt.Owner != scope || receipt.IntentKey != intent.Key ||
			receipt.Kind != intent.Kind || receipt.Target != attempt.Target || receipt.Binding != intent.Binding ||
			receipt.RecordID != intent.RecordID || receipt.ProductKey != saved.Source.Source.ProductKey ||
			receipt.PayloadFingerprint != attempt.PayloadFingerprint || receipt.ResponseHash != attempt.Evidence.Fingerprint ||
			submission.ValidateOfficialReceipt(receipt) != nil || receipt.TemuProduct == nil {
			return record.ErrUnavailable
		}
		counts.Confirmed++
	}
	return nil
}
