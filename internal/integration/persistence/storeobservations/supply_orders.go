package storeobservations

import (
	"context"
	"encoding/json"
	o "task-processor/internal/storecenter/observations"
)

func (r *Repository) SupplyOrderRecords(ctx context.Context, org, store, sync string) ([]o.SupplyOrder, error) {
	if !o.ValidID(store) || !o.ValidID(sync) {
		return nil, o.ErrInvalid
	}
	var size int64
	if e := r.db.WithContext(ctx).Raw(`SELECT COALESCE(sum(octet_length(payload::text)),0) FROM store_observations.records WHERE organization_id=? AND store_id=? AND kind='supply-orders' AND sync_id=?`, org, store, sync).Scan(&size).Error; e != nil || size > 4<<20 {
		return nil, o.ErrUnavailable
	}
	var rows []struct {
		PlatformID string
		Payload    []byte
	}
	if e := r.db.WithContext(ctx).Raw(`SELECT platform_id,payload FROM store_observations.records WHERE organization_id=? AND store_id=? AND kind='supply-orders' AND sync_id=? ORDER BY platform_id LIMIT 201`, org, store, sync).Scan(&rows).Error; e != nil || len(rows) > 200 {
		return nil, o.ErrUnavailable
	}
	out := make([]o.SupplyOrder, 0, len(rows))
	for _, row := range rows {
		var value struct {
			SupplyOrder *o.SupplyOrder `json:"supplyOrder"`
		}
		if json.Unmarshal(row.Payload, &value) != nil || value.SupplyOrder == nil || value.SupplyOrder.ID != row.PlatformID || !value.SupplyOrder.Valid() {
			return nil, o.ErrUnavailable
		}
		out = append(out, *value.SupplyOrder)
	}
	return out, nil
}
