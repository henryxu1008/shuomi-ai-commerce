package storeobservations

import (
	"context"
	"encoding/json"
	"task-processor/internal/authidentity"
	o "task-processor/internal/storecenter/observations"
)

func (r *Repository) VisitSupplyOrders(ctx context.Context, org, store, sync string, visit func(o.SupplyOrder) error) error {
	if ctx == nil || !authidentity.IsBoundedIdentifier(org) || !o.ValidID(store) || !o.ValidID(sync) || visit == nil {
		return o.ErrInvalid
	}
	// One MVCC statement, at most 200 original orders. Decode and release each
	// schema-bounded record before reading the next; do not materialize a batch.
	rows, err := r.db.WithContext(ctx).Raw(`SELECT platform_id,payload FROM store_observations.records WHERE organization_id=? AND store_id=? AND kind='supply-orders' AND sync_id=? ORDER BY platform_id LIMIT 201`, org, store, sync).Rows()
	if err != nil {
		return o.ErrUnavailable
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
		var id string
		var payload []byte
		if count > 200 || rows.Scan(&id, &payload) != nil || len(payload) > 262144 {
			return o.ErrUnavailable
		}
		var value struct {
			SupplyOrder *o.SupplyOrder `json:"supplyOrder"`
		}
		if json.Unmarshal(payload, &value) != nil || value.SupplyOrder == nil || value.SupplyOrder.ID != id || !value.SupplyOrder.Valid() {
			return o.ErrUnavailable
		}
		if err := visit(*value.SupplyOrder); err != nil {
			return err
		}
	}
	if rows.Err() != nil || ctx.Err() != nil {
		return o.ErrUnavailable
	}
	return nil
}
