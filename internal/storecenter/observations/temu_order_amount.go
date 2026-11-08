package observations

import (
	"context"
	"reflect"
	"task-processor/internal/storecenter"
)

type TemuOrderAmountRecord struct {
	StoreID, SyncID string
	Order           TemuOrder
}
type TemuOrderAmountReader interface {
	TemuOrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]TemuOrderAmountRecord, error)
}
type SavedTemuOrderAccess interface {
	ReadSavedTemuOrderBinding(context.Context, Scope, string) (Binding, error)
}

func sameSavedTemuOrderBinding(local, captured Binding) bool {
	return local.ValidTemuOrderApplication() && local == captured
}

func (s *Service) SavedTemuOrderAmounts(ctx context.Context, scope Scope, q OrderAmountQuery) (OrderAmountMetrics, error) {
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
	access, ok := s.Access.(SavedTemuOrderAccess)
	if !ok {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	reader, ok := s.Repository.(TemuOrderAmountReader)
	if !ok {
		return OrderAmountMetrics{}, ErrUnavailable
	}
	readBinding := func() (Binding, error) {
		if e := s.Access.Authorize(ctx, scope, Orders, false); e != nil {
			return Binding{}, e
		}
		b, e := access.ReadSavedTemuOrderBinding(ctx, scope, q.StoreID)
		if e != nil {
			return Binding{}, e
		}
		if b.Platform != storecenter.PlatformTemu || b.Site != "temu-order-us" || !b.ValidTemuOrderApplication() || b.OrganizationID != scope.OrganizationID || b.StoreID != q.StoreID {
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
		if !ValidID(h.ID) || h.StoreID != q.StoreID || h.Owner.OrganizationID != scope.OrganizationID || h.Kind != Orders || !sameSavedTemuOrderBinding(binding, h.Binding) || h.Binding.Platform != storecenter.PlatformTemu {
			return OrderAmountMetrics{}, ErrUnavailable
		}
		result.Source = &OrderAmountSource{SyncID: h.ID, ObservedAt: h.ObservedAt, Range: h.Range}
		result.Reason = "sync_incomplete"
		if h.Status == "completed" && !h.Progress.Incomplete && h.ObservedAt != nil && !h.ObservedAt.IsZero() && h.Progress.TemuOrders != nil && h.Progress.TemuOrders.Complete {
			result.Reason = "newer_sync_incomplete"
			if len(latest) == 1 && reflect.DeepEqual(h, latest[0]) {
				result.Reason = "range_not_covered"
				if h.Range != nil && !h.Range.Start.After(q.Start) && !h.Range.End.Before(q.End) {
					values, err := reader.TemuOrderAmountRecords(ctx, scope.OrganizationID, h.ID, q, MaxOrderAmountCandidates+1)
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
						result, err = aggregateOrderAmountsWithDecimal(result, rows, q, temuOrderDecimal, "temu-order-us")
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
