package storeobservations

import (
	"context"
	"encoding/json"
	"task-processor/internal/authidentity"
	o "task-processor/internal/storecenter/observations"
)

func (r *Repository) OrderAmountRecords(ctx context.Context, org, syncID string, q o.OrderAmountQuery, limit int) ([]o.OrderAmountRecord, error) {
	if !authidentity.IsBoundedIdentifier(org) || !o.ValidID(q.StoreID) || !o.ValidID(syncID) || !q.Start.Before(q.End) || limit < 1 || limit > o.MaxOrderAmountCandidates+1 {
		return nil, o.ErrInvalid
	}
	var rows []struct {
		StoreID, SyncID, PlatformID, Site, IssuedAt string
		Amount                                      []byte
	}
	// Keep unknown dates visible. Nonempty invalid native timestamps fail the
	// query rather than silently disappearing; date filtering precedes the cap.
	err := r.db.WithContext(ctx).Raw(`SELECT store_id,sync_id,platform_id,payload->'order'->>'site' AS site,
 COALESCE(payload->'order'->>'issuedAt','') AS issued_at,payload->'order'->'amount' AS amount
 FROM store_observations.records
 WHERE organization_id=? AND store_id=? AND sync_id=? AND kind='orders'
 AND (NULLIF(payload->'order'->>'issuedAt','') IS NULL OR
 (NULLIF(payload->'order'->>'issuedAt','')::timestamptz>=? AND NULLIF(payload->'order'->>'issuedAt','')::timestamptz<?))
 ORDER BY platform_id LIMIT ?`, org, q.StoreID, syncID, q.Start, q.End, limit).Scan(&rows).Error
	if err != nil {
		return nil, o.ErrUnavailable
	}
	result := make([]o.OrderAmountRecord, 0, len(rows))
	for _, row := range rows {
		var price *o.Price
		if len(row.Amount) > 0 && json.Unmarshal(row.Amount, &price) != nil {
			return nil, o.ErrUnavailable
		}
		result = append(result, o.OrderAmountRecord{StoreID: row.StoreID, SyncID: row.SyncID, ID: row.PlatformID, Site: row.Site, IssuedAt: row.IssuedAt, Amount: price})
	}
	return result, nil
}

var _ o.OrderAmountReader = (*Repository)(nil)
