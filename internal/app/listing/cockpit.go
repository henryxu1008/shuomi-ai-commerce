package listingapp

import (
	"context"
	"errors"
	"slices"
	"time"

	"task-processor/internal/app/workbenchmetrics"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type listingMetrics struct {
	Status     string               `json:"status"`
	CapturedAt time.Time            `json:"capturedAt"`
	Counts     *listingMetricCounts `json:"counts,omitempty"`
}

type listingMetricCounts struct {
	Pending int `json:"pending"`
	Review  int `json:"review"`
}

type targetCohortReader interface {
	ListNewTargetIDs(context.Context, collection.Scope, []string, time.Time, time.Time, int) ([]string, error)
}

func (a *Application) CockpitMetrics(ctx context.Context, filter workbenchmetrics.Filter, now time.Time) (listingMetrics, error) {
	result := listingMetrics{CapturedAt: now.UTC()}
	cohort, ok := a.Records.(targetCohortReader)
	if !ok || a.CockpitStoreAccess == nil || a.TargetMerchants == nil || a.StageProjection.Facts == nil || a.Authorization == nil {
		return result, record.ErrUnavailable
	}
	scope, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return result, err
	}
	checkStores := func() error { return a.CockpitStoreAccess(ctx, scope, filter.Stores) }
	if err = checkStores(); err != nil {
		return result, err
	}
	ids, err := cohort.ListNewTargetIDs(ctx, scope, filter.Stores, filter.Start, filter.End, workbenchmetrics.MaxCandidates+1)
	if err != nil {
		return result, err
	}
	counts := &listingMetricCounts{}
	bindings := make(map[string]storecenter.ProductMerchantBinding)
	if len(ids) > workbenchmetrics.MaxCandidates {
		result.Status = "limited"
	} else {
		for _, id := range ids {
			saved, err := a.Records.ReadTargetHead(ctx, scope, id)
			if err != nil {
				return result, err
			}
			storeID := saved.Input.StoreID
			if !slices.Contains(filter.Stores, storeID) || saved.TargetID != id {
				return result, record.ErrUnavailable
			}
			binding, found := bindings[storeID]
			if !found {
				binding, err = a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
				if err != nil {
					return result, err
				}
				if !validTargetMerchant(scope, storeID, binding) {
					return result, record.ErrForbidden
				}
				bindings[storeID] = binding
			}
			if !sameMetricsMerchant(saved.Merchant, binding) {
				continue
			}
			publication, err := a.Publication(ctx, saved.Source.ID, storeID)
			if err != nil {
				return result, err
			}
			if publication.ReceiptID != "" && !sameMetricsMerchant(publication.binding, binding) {
				return result, record.ErrForbidden
			}
			if publication.RecordID == saved.ID && (publication.TikTok == nil || publication.TikTok.Publication == nil ||
				publication.TikTok.Publication.State != marketplace.PublicationFailed) {
				continue
			}
			pending, review, err := a.StageProjection.metricQueues(ctx, scope, saved)
			if err != nil {
				return result, err
			}
			if pending {
				counts.Pending++
			}
			if review {
				counts.Review++
			}
		}
		result.Status, result.Counts = "complete", counts
	}
	current, err := a.Authorization.Authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return listingMetrics{}, err
	}
	if current != scope {
		return listingMetrics{}, record.ErrForbidden
	}
	if err = checkStores(); err != nil {
		return listingMetrics{}, err
	}
	for storeID, binding := range bindings {
		current, err := a.TargetMerchants.ReadTargetMerchant(ctx, scope, storeID)
		if err != nil {
			return listingMetrics{}, err
		}
		if !validTargetMerchant(scope, storeID, current) || !sameMetricsMerchant(binding, current) {
			return listingMetrics{}, record.ErrForbidden
		}
	}
	return result, nil
}

// The current reader enforces the live credential and service rules. Compare
// the historical target against stable merchant identity, not token expiry.
func sameMetricsMerchant(saved, current storecenter.ProductMerchantBinding) bool {
	switch current.Platform {
	case storecenter.PlatformWalmart:
		return submission.WalmartCompatibleBinding(saved, current)
	case storecenter.PlatformTikTokShop:
		return submission.TikTokCompatibleBinding(saved, current)
	}
	return saved.OrganizationID == current.OrganizationID && saved.StoreID == current.StoreID &&
		saved.Site == current.Site && saved.Platform == current.Platform &&
		saved.ApplicationID == current.ApplicationID && saved.ApplicationType == current.ApplicationType &&
		saved.SupplierIdentityHash == current.SupplierIdentityHash &&
		saved.IdentityPartition == current.IdentityPartition && saved.MerchantMode == current.MerchantMode &&
		saved.RegionID == current.RegionID
}

func (p ReviewProjection) metricQueues(ctx context.Context, scope collection.Scope, saved record.TargetRecord) (pending, review bool, err error) {
	item, err := p.Facts.LatestOptimization(ctx, scope, saved.Source.ID, saved.Input.StoreID)
	if errors.Is(err, preparation.ErrNotFound) {
		return true, false, nil
	}
	if err != nil {
		return false, false, err
	}
	if item.RecordID == "" {
		return item.Status != preparation.ItemPending && item.Status != preparation.ItemRunning && item.Status != preparation.ItemUnknown, false, nil
	}
	if item.RecordID != saved.ID || item.RecordRevision != saved.Revision {
		return true, false, nil
	}
	switch item.Status {
	case preparation.ItemPending, preparation.ItemRunning, preparation.ItemUnknown:
		return false, false, nil
	case preparation.ItemReview:
		state, err := p.reviewState(ctx, scope, item, saved.Source.Source.ProductKey, saved.EffectiveVersion)
		return state == "rejected", state == "pending", err
	default:
		return true, false, nil
	}
}
