package observations

import (
	"context"
	"errors"
	"time"
)

type Metric struct {
	Value    *int64 `json:"value"`
	Unknown  int64  `json:"unknown"`
	Complete bool   `json:"complete"`
	Reason   string `json:"reason"`
}
type InventoryFact struct {
	SKU, Warehouse string
	Quantity       *int64
}

func LowInventory(rows []InventoryFact, threshold *int64) Metric {
	if threshold == nil {
		return Metric{Reason: "threshold_not_set"}
	}
	type key struct{ sku, warehouse string }
	seen := map[key]InventoryFact{}
	bad := map[key]bool{}
	for _, row := range rows {
		id := key{row.SKU, row.Warehouse}
		prior, exists := seen[id]
		if row.SKU == "" || row.Warehouse == "" || row.Quantity == nil || *row.Quantity < 0 {
			bad[id] = true
		}
		if exists && prior.Quantity != nil && row.Quantity != nil && *prior.Quantity != *row.Quantity {
			bad[id] = true
		}
		seen[id] = row
	}
	count, unknown := int64(0), int64(0)
	for id, row := range seen {
		if bad[id] {
			unknown++
			continue
		}
		if *row.Quantity <= *threshold {
			count++
		}
	}
	result := Metric{Value: &count, Unknown: unknown, Complete: unknown == 0}
	if unknown > 0 {
		result.Reason = "inventory_unknown"
	}
	return result
}

type StoreAggregate struct {
	Active, Pending, StatusUnknown int64
	Inventory                      []InventoryFact
}

// Narrow owner projection: no records, publication or provider mutation handle.
type StoreAggregateReader interface {
	StoreAggregates(context.Context, string, Kind, map[string]string, bool) (map[string]StoreAggregate, error)
}
type StoreSummary struct {
	StoreID    string
	Kind       Kind
	Metric     Metric
	Inventory  Metric
	SyncStatus string
	ObservedAt *time.Time
	Complete   bool
	Reason     string
}

func (s *Service) StoreSummaries(ctx context.Context, scope Scope, kind Kind, stores []string, thresholds map[string]*int64) (map[string]StoreSummary, error) {
	repository, ok := s.Repository.(StoreAggregateReader)
	if !s.available() || !ok {
		return nil, ErrUnavailable
	}
	stores, err := s.allowedStores(ctx, scope, kind, false, stores)
	if err != nil {
		return nil, err
	}
	heads, latest, err := s.Repository.Heads(ctx, scope.OrganizationID, kind, stores)
	if err != nil {
		return nil, err
	}
	out := map[string]StoreSummary{}
	sources := map[string]string{}
	handles := []Merchant{}
	for _, id := range stores {
		out[id] = StoreSummary{StoreID: id, Kind: kind, Reason: "not_synced", Metric: Metric{Reason: "not_synced"}, Inventory: Metric{Reason: "not_synced"}}
	}
	headHandles := make([]Merchant, len(heads))
	if err = forStores(ctx, len(heads), func(ctx context.Context, i int) error {
		head := heads[i]
		m, e := s.Access.Open(ctx, scope, head.StoreID, kind, false, &head.Binding)
		if e != nil {
			if errors.Is(e, ErrUnavailable) {
				return e
			}
			return nil
		}
		headHandles[i] = m
		return nil
	}); err != nil {
		return nil, err
	}
	for i, head := range heads {
		m := headHandles[i]
		if m == nil {
			continue
		}
		handles = append(handles, m)
		sources[head.StoreID] = head.ID
		complete := head.Status == "completed" && !head.Progress.Incomplete
		reason := ""
		if !complete {
			reason = "sync_incomplete"
		}
		out[head.StoreID] = StoreSummary{StoreID: head.StoreID, Kind: kind, Complete: complete, SyncStatus: head.Status, ObservedAt: head.ObservedAt, Reason: reason}
	}
	// Later failed/partial/pending attempts are visible only under the same current
	// binding, including original rejected local attempts without a binding.
	latestHandles := make([]Merchant, len(latest))
	if err = forStores(ctx, len(latest), func(ctx context.Context, i int) error {
		sync := latest[i]
		if sync.Binding.ApplicationID == "" {
			return nil
		}
		m, e := s.Access.Open(ctx, scope, sync.StoreID, kind, false, &sync.Binding)
		if e != nil {
			if errors.Is(e, ErrUnavailable) {
				return e
			}
			return nil
		}
		latestHandles[i] = m
		return nil
	}); err != nil {
		return nil, err
	}
	for i, sync := range latest {
		if latestHandles[i] == nil && !(sync.Binding.ApplicationID == "" && sync.Status == "suspended" && (sync.ErrorCode == "unsupported_application" || sync.ErrorCode == "store_unavailable")) {
			continue
		}
		if latestHandles[i] != nil {
			handles = append(handles, latestHandles[i])
		}
		value := out[sync.StoreID]
		value.SyncStatus = sync.Status
		if sync.Status != "completed" || sync.Progress.Incomplete {
			value.Complete = false
			value.Reason = "sync_incomplete"
		}
		out[sync.StoreID] = value
	}
	aggregates, err := repository.StoreAggregates(ctx, scope.OrganizationID, kind, sources, kind == Products)
	if err != nil {
		return nil, err
	}
	for id, source := range sources {
		_ = source
		value := out[id]
		aggregate := aggregates[id]
		count := aggregate.Active
		if kind == Orders {
			count = aggregate.Pending
		}
		value.Metric = Metric{Unknown: aggregate.StatusUnknown, Complete: value.Complete && aggregate.StatusUnknown == 0, Reason: value.Reason}
		if value.Metric.Complete {
			value.Metric.Value = &count
		} else if aggregate.StatusUnknown > 0 {
			value.Metric.Reason = "status_unknown"
		}
		if kind == Products {
			value.Inventory = LowInventory(aggregate.Inventory, thresholds[id])
			if !value.Complete {
				value.Inventory.Complete = false
				value.Inventory.Reason = "sync_incomplete"
			}
		}
		out[id] = value
	}
	if err = forStores(ctx, len(handles), func(ctx context.Context, i int) error { return handles[i].Check(ctx) }); err != nil {
		return nil, err
	}
	if _, err = s.allowedStores(ctx, scope, kind, false, stores); err != nil {
		return nil, err
	}
	return out, nil
}
