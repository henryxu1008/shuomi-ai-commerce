// Package shopdiagnosis consumes current native operating facts for advisory
// analysis. It owns no financial facts, provider, grants or business mutation.
package shopdiagnosis

import (
	"context"
	"errors"
	"task-processor/internal/agent"
	c "task-processor/internal/operationscockpit"
	o "task-processor/internal/storecenter/observations"
)

type Boundary interface {
	Current(context.Context, c.Scope) (c.Access, error)
	ReadStores(context.Context, c.Scope, []string) error
	ListStores(context.Context, c.Scope) ([]c.StoreReference, error)
}
type SnapshotReader interface {
	Snapshot(context.Context, c.Scope, c.Query) (c.Snapshot, error)
}
type ObservationReader interface {
	List(context.Context, o.Scope, o.Query) (o.Result, error)
}

// NativeReader shares the original Cockpit repository and authorization ports.
// It cannot sync or call a platform provider; optional TEMU reads are excluded.
type NativeReader struct {
	Repository   SnapshotReader
	Boundary     Boundary
	Observations ObservationReader
}
type NativeFacts struct {
	Store             c.StoreReference
	Current, Previous c.StoreAggregate
	Observations      [2]SavedObservation
}
type SavedObservation struct {
	State           string `json:"state"`
	Complete        bool   `json:"complete"`
	Sources, Latest []c.ObservationEvidence
	Counts          SavedCounts `json:"counts"`
}
type SavedCounts struct{ Total, Active, OffShelf, Pending, Transit, Exceptional, Unknown int }

func (r NativeReader) Authorize(ctx context.Context, id agent.ExecutionIdentity, storeID string) error {
	if ctx == nil || ctx.Err() != nil || r.Boundary == nil || !c.UUID(storeID) {
		return agent.ErrInvalid
	}
	scope := c.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	if !scope.Valid() || !id.Valid() {
		return c.ErrForbidden
	}
	access, e := r.Boundary.Current(ctx, scope)
	if e != nil {
		return e
	}
	if !access.StoresRead || !access.AdviceRead {
		return c.ErrForbidden
	}
	return r.Boundary.ReadStores(ctx, scope, []string{storeID})
}
func (r NativeReader) Read(ctx context.Context, id agent.ExecutionIdentity, storeID string, period c.Period) (NativeFacts, error) {
	var out NativeFacts
	if !period.Valid() || r.Repository == nil {
		return out, agent.ErrInvalid
	}
	if e := r.Authorize(ctx, id, storeID); e != nil {
		return out, e
	}
	scope := c.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID}
	shops, e := r.Boundary.ListStores(ctx, scope)
	if e != nil {
		return out, e
	}
	for _, s := range shops {
		if s.ID == storeID {
			out.Store = s
			break
		}
	}
	if out.Store.ID == "" || out.Store.Status != "active" {
		return NativeFacts{}, c.ErrForbidden
	}
	snapshot, e := r.Repository.Snapshot(ctx, scope, c.Query{Module: "advice", Period: period, StoreIDs: []string{storeID}})
	if e != nil {
		return NativeFacts{}, e
	}
	out.Current = snapshot.Stores[storeID]
	out.Previous = snapshot.Previous[storeID]
	if snapshot.Period != period || out.Current.StoreID != storeID || out.Current.Period != period {
		return NativeFacts{}, c.ErrUnavailable
	}
	for n, kind := range []o.Kind{o.Products, o.Orders} {
		saved := SavedObservation{State: "unavailable", Sources: []c.ObservationEvidence{}, Latest: []c.ObservationEvidence{}}
		// Access.Open for TEMU queries QueryStore even for saved reads. Never enter
		// that path from this offline consumer; manual financial facts still work.
		if out.Store.Platform == "temu" {
			saved.State = "offline_unavailable"
		} else if out.Store.Platform != "shein" {
			saved.State = "unsupported"
		} else if r.Observations != nil {
			view, err := r.Observations.List(ctx, o.Scope{OrganizationID: id.OrganizationID, ActorID: id.ActorID, MemberID: id.MemberID}, o.Query{Kind: kind, Stores: []string{storeID}, Limit: 1, RecordByteLimit: 1024})
			if errors.Is(err, o.ErrForbidden) || errors.Is(err, o.ErrNotFound) {
				saved.State = "permission_required"
			} else if err == nil {
				if len(view.Syncs) > 1 || len(view.Latest) > 1 {
					return NativeFacts{}, c.ErrUnavailable
				}
				saved.State = "available"
				saved.Complete = view.Complete
				if !view.Complete {
					saved.State = "incomplete"
				}
				s := view.Summary
				saved.Counts = SavedCounts{s.Total, s.Active, s.OffShelf, s.Pending, s.Transit, s.Exceptional, s.Unknown}
				for _, v := range []int{s.Total, s.Active, s.OffShelf, s.Pending, s.Transit, s.Exceptional, s.Unknown} {
					if v < 0 {
						return NativeFacts{}, c.ErrUnavailable
					}
				}
				for _, pair := range []struct {
					values []o.Sync
					target *[]c.ObservationEvidence
				}{{view.Syncs, &saved.Sources}, {view.Latest, &saved.Latest}} {
					for _, s := range pair.values {
						if s.StoreID != storeID || s.Kind != kind || !o.ValidID(s.ID) {
							return NativeFacts{}, c.ErrUnavailable
						}
						ev := c.ObservationEvidence{StoreID: s.StoreID, SyncID: s.ID, Status: s.Status, ObservedAt: s.ObservedAt, ErrorCode: s.ErrorCode}
						if s.Range != nil {
							ev.Start = &s.Range.Start
							ev.End = &s.Range.End
						}
						*pair.target = append(*pair.target, ev)
						if s.Status == "failed" || s.Status == "suspended" || s.Status == "partial" {
							saved.State = "latest_incomplete"
							saved.Complete = false
						}
					}
				}
				if len(view.Syncs) == 0 {
					saved.State = "unavailable"
					saved.Complete = false
					saved.Counts = SavedCounts{}
				}
			}
		}
		out.Observations[n] = saved
	}
	// Do not disclose a snapshot after grants were revoked during the read.
	if e = r.Authorize(ctx, id, storeID); e != nil {
		return NativeFacts{}, e
	}
	return out, nil
}
