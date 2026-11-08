package storeobservations

import (
	"context"
	"encoding/json"
	"task-processor/internal/authidentity"
	o "task-processor/internal/storecenter/observations"
)

func (r *Repository) TemuOrderAmountRecords(ctx context.Context, org, sync string, q o.OrderAmountQuery, limit int) ([]o.TemuOrderAmountRecord, error) {
	if !authidentity.IsBoundedIdentifier(org) || !o.ValidID(q.StoreID) || !o.ValidID(sync) || !q.Start.Before(q.End) || limit < 1 || limit > o.MaxOrderAmountCandidates+1 {
		return nil, o.ErrInvalid
	}
	var rows []struct {
		StoreID, SyncID string
		Body            []byte
	}
	e := r.db.WithContext(ctx).Raw(`SELECT store_id,sync_id,payload->'temuOrder' AS body FROM store_observations.records
 WHERE organization_id=? AND store_id=? AND sync_id=? AND kind='orders'
 AND (NULLIF(payload->'temuOrder'->>'date','') IS NULL OR (NULLIF(payload->'temuOrder'->>'date','')::timestamptz>=? AND NULLIF(payload->'temuOrder'->>'date','')::timestamptz<?))
 ORDER BY platform_id LIMIT ?`, org, q.StoreID, sync, q.Start, q.End, limit).Scan(&rows).Error
	if e != nil {
		return nil, o.ErrUnavailable
	}
	result := make([]o.TemuOrderAmountRecord, 0, len(rows))
	for _, row := range rows {
		var order *o.TemuOrder
		if json.Unmarshal(row.Body, &order) != nil || order == nil {
			return nil, o.ErrUnavailable
		}
		result = append(result, o.TemuOrderAmountRecord{StoreID: row.StoreID, SyncID: row.SyncID, Order: *order})
	}
	return result, nil
}
