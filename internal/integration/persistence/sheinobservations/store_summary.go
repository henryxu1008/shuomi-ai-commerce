package sheinobservations

import (
	"context"
	"database/sql"
	"strings"

	"gorm.io/gorm"
	"task-processor/internal/authidentity"
	o "task-processor/internal/marketplace/shein/observations"
)

func (r *Repository) StoreAggregates(ctx context.Context, org string, kind o.Kind, sources map[string]string, inventory bool) (out map[string]o.StoreAggregate, err error) {
	out = map[string]o.StoreAggregate{}
	if !authidentity.IsBoundedIdentifier(org) || !kind.Valid() || len(sources) > 500 {
		return nil, o.ErrInvalid
	}
	if len(sources) == 0 {
		return out, nil
	}
	conditions, args := []string{}, []any{org, kind}
	for store, sync := range sources {
		if !o.ValidID(store) || !o.ValidID(sync) {
			return nil, o.ErrInvalid
		}
		conditions = append(conditions, "(r.store_id=? AND r.sync_id=?)")
		args = append(args, store, sync)
		out[store] = o.StoreAggregate{}
	}
	base := ` FROM shein_observations.records r`
	if kind == o.Products {
		base += ` CROSS JOIN LATERAL jsonb_array_elements(r.payload->'product'->'skcs') k`
	}
	where := ` WHERE r.organization_id=? AND r.kind=? AND (` + strings.Join(conditions, " OR ") + `)`
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := `SELECT r.store_id, count(*) FILTER(WHERE r.payload->'order'->>'status' IN('1','2','3')) AS pending,count(*) FILTER(WHERE r.payload->'order'->>'status' IS NULL OR r.payload->'order'->>'status' NOT IN('1','2','3','4','5','6','7','8','9')) AS status_unknown`
		if kind == o.Products {
			query = `SELECT r.store_id,count(*) FILTER(WHERE k->>'siteStatus'='1') AS active,count(*) FILTER(WHERE k->>'siteStatus' IS NULL OR k->>'siteStatus' NOT IN('0','1')) AS status_unknown`
		}
		var totals []struct {
			StoreID                        string
			Active, Pending, StatusUnknown int64
		}
		if err := tx.Raw(query+base+where+` GROUP BY r.store_id`, args...).Scan(&totals).Error; err != nil {
			return err
		}
		for _, total := range totals {
			out[total.StoreID] = o.StoreAggregate{Active: total.Active, Pending: total.Pending, StatusUnknown: total.StatusUnknown}
		}
		if kind != o.Products || !inventory {
			return nil
		}
		// Transfer only inventory identities and quantities, never full Product detail.
		rows, err := tx.Raw(`SELECT r.store_id,coalesce(sku->>'id','missing:'||r.platform_id||':'||(k->>'id')),coalesce(stock->>'warehouseId',''),CASE WHEN jsonb_typeof(stock->'quantity')='number' THEN (stock->>'quantity')::bigint ELSE NULL END`+base+` LEFT JOIN LATERAL jsonb_array_elements(k->'skus') sku ON true LEFT JOIN LATERAL jsonb_array_elements(sku->'inventory') stock ON true`+where, args...).Rows()
		if err != nil {
			return err
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			count++
			if count > 100000 {
				return o.ErrUnavailable
			}
			var store string
			var fact o.InventoryFact
			if err = rows.Scan(&store, &fact.SKU, &fact.Warehouse, &fact.Quantity); err != nil {
				return err
			}
			value := out[store]
			value.Inventory = append(value.Inventory, fact)
			out[store] = value
		}
		return rows.Err()
	}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return nil, o.ErrUnavailable
	}
	return out, nil
}
