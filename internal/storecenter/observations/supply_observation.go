package observations

import (
	"context"
	"errors"
	"reflect"
	"time"
)

type SupplyRecordReader interface {
	VisitSupplyOrders(context.Context, string, string, string, func(SupplyOrder) error) error
}
type SupplyObservationSource struct {
	SyncID     string     `json:"syncId"`
	Status     string     `json:"status"`
	OrderIDs   []string   `json:"orderIds"`
	MissingIDs []string   `json:"missingIds"`
	Notes      []string   `json:"notes"`
	StartedAt  time.Time  `json:"startedAt"`
	ObservedAt *time.Time `json:"observedAt"`
}
type SupplyObservation struct {
	StoreID       string                   `json:"storeId"`
	Status        string                   `json:"status"`
	CanSync       bool                     `json:"canSync"`
	Source        *SupplyObservationSource `json:"source,omitempty"`
	Summary       *SupplySummary           `json:"summary,omitempty"`
	SavedOrderIDs []string                 `json:"savedOrderIds"`
}

func (s *Service) SupplyObservation(ctx context.Context, scope Scope, store string) (SupplyObservation, error) {
	result := SupplyObservation{StoreID: store, Status: "not_synced", SavedOrderIDs: []string{}}
	if !s.available() || !scope.Valid() || !ValidID(store) {
		return result, ErrInvalid
	}
	if e := s.Access.Authorize(ctx, scope, SupplyOrders, false); e != nil {
		return result, e
	}
	handle, e := s.Access.Open(ctx, scope, store, SupplyOrders, false, nil)
	if errors.Is(e, ErrUnsupported) {
		result.Status = "not_applicable"
		return result, nil
	}
	if e != nil {
		return result, e
	}
	if e = s.Access.Authorize(ctx, scope, SupplyOrders, true); e == nil {
		result.CanSync = true
	} else if !errors.Is(e, ErrForbidden) {
		return result, e
	}
	_, latest, e := s.Repository.Heads(ctx, scope.OrganizationID, SupplyOrders, []string{store})
	if e != nil {
		return result, e
	}
	if len(latest) == 0 {
		return result, handle.Check(ctx)
	}
	if len(latest) != 1 {
		return result, ErrUnavailable
	}
	sync, e := s.Repository.ReadSync(ctx, scope.OrganizationID, latest[0].ID)
	if e != nil || sync.Kind != SupplyOrders || sync.StoreID != store || sync.Owner.OrganizationID != scope.OrganizationID || sync.Revision != latest[0].Revision || !ValidSupplyProgress(sync.Progress.SupplyOrders) {
		return result, ErrUnavailable
	}
	if sync.Binding != handle.Binding() {
		return result, ErrConflict
	}
	p := sync.Progress.SupplyOrders
	result.Source = &SupplyObservationSource{SyncID: sync.ID, Status: sync.Status, OrderIDs: append([]string{}, p.OrderIDs...), MissingIDs: append([]string{}, p.MissingIDs...), Notes: append([]string{}, sync.Progress.Notes...), StartedAt: sync.CreatedAt, ObservedAt: sync.ObservedAt}
	result.Status = "incomplete"
	if !sync.Terminal() {
		result.Status = "running"
	} else if sync.Status == "completed" || sync.Status == "partial" {
		reader, ok := s.Repository.(SupplyRecordReader)
		if !ok {
			return result, ErrUnavailable
		}
		allowed, seen := map[string]bool{}, map[string]bool{}
		for _, id := range p.OrderIDs[:p.Next] {
			allowed[id] = true
		}
		for _, id := range p.MissingIDs {
			delete(allowed, id)
		}
		var accumulator supplySummaryAccumulator
		err := reader.VisitSupplyOrders(ctx, scope.OrganizationID, store, sync.ID, func(order SupplyOrder) error {
			if !order.Valid() || !allowed[order.ID] || seen[order.ID] {
				return ErrUnavailable
			}
			seen[order.ID] = true
			result.SavedOrderIDs = append(result.SavedOrderIDs, order.ID)
			accumulator.add(order)
			return nil
		})
		if err != nil {
			return result, err
		}
		if len(seen) != len(allowed) {
			return result, ErrUnavailable
		}
		summary := accumulator.result()
		result.Summary = &summary
		if sync.Status == "completed" && p.Next == len(p.OrderIDs) && len(p.MissingIDs) == 0 && !sync.Progress.Incomplete {
			result.Status = "saved"
		}
	}
	_, fresh, e := s.Repository.Heads(ctx, scope.OrganizationID, SupplyOrders, []string{store})
	if e != nil || len(fresh) != 1 || fresh[0].ID != sync.ID || fresh[0].Revision != sync.Revision {
		return result, ErrUnavailable
	}
	after, e := s.Repository.ReadSync(ctx, scope.OrganizationID, sync.ID)
	if e != nil || !reflect.DeepEqual(after, sync) {
		return result, ErrUnavailable
	}
	if e = handle.Check(ctx); e != nil {
		return SupplyObservation{}, e
	}
	return result, nil
}
