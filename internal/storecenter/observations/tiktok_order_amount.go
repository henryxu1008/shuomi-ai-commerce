package observations

import (
	"context"
	"reflect"
	"task-processor/internal/storecenter"
)

type TikTokOrderAmountRecord struct {
	StoreID, SyncID string
	Order           TikTokOrder
}
type TikTokOrderAmountReader interface {
	TikTokOrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]TikTokOrderAmountRecord, error)
}
type SavedTikTokOrderAccess interface {
	ReadSavedTikTokOrderBinding(context.Context, Scope, string) (Binding, error)
}

// Only this saved-order read excludes the captured online scope digest.
// Current grant/shop/application and every other binding field remain fenced.
func sameSavedTikTokOrderBinding(local, captured Binding) bool {
	if !captured.ValidTikTokApplication() || local.ScopesHash != "" {
		return false
	}
	captured.ScopesHash = ""
	return local == captured
}

func (s *Service) SavedTikTokOrderAmounts(ctx context.Context, scope Scope, q OrderAmountQuery) (OrderAmountMetrics, error) {
	result := OrderAmountMetrics{Status: "coverage_incomplete", Reason: "no_saved_sync"}
	if !s.available() {
		return result, ErrUnavailable
	}
	result.CapturedAt = s.now()
	if !scope.Valid() {
		return OrderAmountMetrics{}, ErrForbidden
	}
	if !q.valid(s.now()) {
		return OrderAmountMetrics{}, ErrInvalid
	}
	access, ok := s.Access.(SavedTikTokOrderAccess)
	if !ok {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	reader, ok := s.Repository.(TikTokOrderAmountReader)
	if !ok {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	readBinding := func() (Binding, error) {
		if e := s.Access.Authorize(ctx, scope, Orders, false); e != nil {
			return Binding{}, e
		}
		b, e := access.ReadSavedTikTokOrderBinding(ctx, scope, q.StoreID)
		if e != nil {
			return Binding{}, e
		}
		if b.Platform != storecenter.PlatformTikTokShop || b.Site != "tiktokshop-us" || b.ScopesHash != "" || b.OrganizationID != scope.OrganizationID || b.StoreID != q.StoreID {
			return Binding{}, ErrUnavailable
		}
		return b, nil
	}
	binding, e := readBinding()
	if e != nil {
		return OrderAmountMetrics{}, e
	}
	heads, latest, e := s.Repository.Heads(ctx, scope.OrganizationID, Orders, []string{q.StoreID})
	if e != nil || len(heads) > 1 || len(latest) > 1 {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	heads, latest = freezeOrderAmountSources(heads), freezeOrderAmountSources(latest)
	if len(heads) == 1 {
		h := heads[0]
		if !ValidID(h.ID) || h.StoreID != q.StoreID || h.Owner.OrganizationID != scope.OrganizationID || h.Kind != Orders || !sameSavedTikTokOrderBinding(binding, h.Binding) || h.Binding.Platform != storecenter.PlatformTikTokShop {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		result.Source = &OrderAmountSource{SyncID: h.ID, ObservedAt: h.ObservedAt, Range: h.Range}
		result.Reason = "sync_incomplete"
		if h.Status == "completed" && !h.Progress.Incomplete && h.ObservedAt != nil && !h.ObservedAt.IsZero() && h.Progress.TikTokOrders != nil && h.Progress.TikTokOrders.Complete {
			result.Reason = "newer_sync_incomplete"
			if len(latest) == 1 && reflect.DeepEqual(h, latest[0]) {
				result.Reason = "range_not_covered"
				if h.Range != nil && !h.Range.Start.After(q.Start) && !h.Range.End.Before(q.End) {
					values, err := reader.TikTokOrderAmountRecords(ctx, scope.OrganizationID, h.ID, q, MaxOrderAmountCandidates+1)
					if err != nil {
						return OrderAmountMetrics{}, ErrUnavailable
					}
					if len(values) > MaxOrderAmountCandidates {
						result.Status = "limited"
						result.Reason = ""
					} else {
						rows := make([]OrderAmountRecord, 0, len(values))
						for _, v := range values {
							if v.StoreID != q.StoreID || v.SyncID != h.ID || !v.Order.Valid() {
								return OrderAmountMetrics{}, ErrUnavailable
							}
							rows = append(rows, OrderAmountRecord{StoreID: v.StoreID, SyncID: v.SyncID, ID: v.Order.ID, Site: v.Order.Site, IssuedAt: v.Order.Date, Amount: v.Order.Amount})
						}
						result, err = aggregateOrderAmountsWithDecimal(result, rows, q, tikTokOrderDecimal, "tiktokshop-us")
						if err != nil {
							return OrderAmountMetrics{}, err
						}
					}
				}
			}
		}
	}
	fresh, e := readBinding()
	if e != nil {
		return OrderAmountMetrics{}, e
	}
	if fresh != binding || ctx.Err() != nil {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	current, currentLatest, e := s.Repository.Heads(ctx, scope.OrganizationID, Orders, []string{q.StoreID})
	if e != nil || !reflect.DeepEqual(heads, current) || !reflect.DeepEqual(latest, currentLatest) {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	return result, nil
}
