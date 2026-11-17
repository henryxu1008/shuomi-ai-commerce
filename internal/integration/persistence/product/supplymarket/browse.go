package supplymarketpersistence

import (
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/supplymarket"
)

type selectionReference struct {
	ReleaseID string
	Receipt   collection.Receipt
}

func selectionRefs(ctx context.Context, db *gorm.DB, scope collection.Scope, ids []string) ([]selectionReference, error) {
	p, actor, member, e := principal(scope, "")
	if e != nil {
		return nil, e
	}
	var rows []struct {
		ReleaseID      string
		CollectionJSON []byte
	}
	if e = db.WithContext(ctx).Raw(`SELECT DISTINCT receipt_json->>'releaseId' AS release_id,receipt_json->'collection' AS collection_json FROM supply_market_commands WHERE principal=? AND actor_id=? AND member_id=? AND receipt_json->>'releaseId' IN ? AND jsonb_typeof(receipt_json->'collection')='object' ORDER BY release_id,collection_json LIMIT 1001`, p, actor, member, ids).Scan(&rows).Error; e != nil {
		return nil, e
	}
	if len(rows) > 1000 {
		return nil, supplymarket.ErrUnavailable
	}
	refs := make([]selectionReference, 0, len(rows))
	for _, row := range rows {
		var receipt collection.Receipt
		if len(row.CollectionJSON) > 65536 || json.Unmarshal(row.CollectionJSON, &receipt) != nil || !collection.ValidID(receipt.ItemID) || !collection.ValidID(receipt.OperationID) {
			return nil, supplymarket.ErrUnavailable
		}
		refs = append(refs, selectionReference{row.ReleaseID, receipt})
	}
	return refs, nil
}
func (r *Repository) ReadMarketData(ctx context.Context, scope collection.Scope, ids []string) (map[string]supplymarket.MarketData, error) {
	result := map[string]supplymarket.MarketData{}
	if scope.Validate() != nil || len(ids) > 50 {
		return nil, supplymarket.ErrInvalid
	}
	for _, id := range ids {
		if !collection.ValidID(id) {
			return nil, supplymarket.ErrInvalid
		}
		result[id] = supplymarket.MarketData{Prices: []supplymarket.PriceRange{}, Added: new(false)}
	}
	if len(ids) == 0 {
		return result, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []struct {
			ID           string
			Prices       []byte
			CategoryJSON []byte
		}
		query := `SELECT r.id,COALESCE((SELECT jsonb_agg(jsonb_build_object('currency',currency,'minimum',minimum,'maximum',maximum) ORDER BY currency) FROM (SELECT variant->>'currency' AS currency, min(CASE WHEN jsonb_typeof(variant->'price')='number' THEN (variant->>'price')::numeric END) AS minimum,max(CASE WHEN jsonb_typeof(variant->'price')='number' THEN (variant->>'price')::numeric END) AS maximum FROM jsonb_array_elements(COALESCE(release_json->'product'->'variants','[]'::jsonb)) AS variant WHERE jsonb_typeof(variant->'price')='number' AND jsonb_typeof(variant->'currency')='string' AND variant->>'currency'<>'' GROUP BY variant->>'currency') ranges),'[]'::jsonb) AS prices, CASE WHEN c.id IS NOT NULL THEN jsonb_build_object('parent', jsonb_build_object('id',p.id,'name',p.name), 'child', jsonb_build_object('id',c.id,'name',c.name,'parentId',p.id)) END AS category_json FROM supply_market_releases r LEFT JOIN supply_market_categories c ON c.id=r.category_id LEFT JOIN supply_market_categories p ON p.id=c.parent_id WHERE r.id IN ?`
		if e := tx.Raw(query, ids).Scan(&rows).Error; e != nil {
			return e
		}
		for _, row := range rows {
			data := result[row.ID]
			if json.Unmarshal(row.Prices, &data.Prices) != nil {
				return supplymarket.ErrUnavailable
			}
			if len(row.CategoryJSON) > 0 {
				if json.Unmarshal(row.CategoryJSON, &data.Category) != nil || data.Category == nil ||
					!collection.ValidID(data.Category.Parent.ID) || !collection.ValidID(data.Category.Child.ID) {
					return supplymarket.ErrUnavailable
				}
			}
			result[row.ID] = data
		}
		refs, e := selectionRefs(ctx, tx, scope, ids)
		if e != nil {
			return e
		}
		receipts := make([]collection.Receipt, 0, len(refs))
		for _, ref := range refs {
			receipts = append(receipts, ref.Receipt)
		}
		receiver, e := r.deps.Receiver(tx)
		if e != nil || receiver == nil {
			return supplymarket.ErrUnavailable
		}
		visible, e := receiver.Visible(ctx, scope, receipts, false)
		if errors.Is(e, collection.ErrForbidden) || errors.Is(e, supplymarket.ErrForbidden) {
			for id, data := range result {
				data.Added = nil
				result[id] = data
			}
			return nil
		}
		if e != nil {
			return e
		}
		for _, ref := range refs {
			if _, ok := visible[ref.Receipt.ItemID]; ok {
				data := result[ref.ReleaseID]
				data.Added = new(true)
				result[ref.ReleaseID] = data
			}
		}
		return nil
	})
	return result, err
}
