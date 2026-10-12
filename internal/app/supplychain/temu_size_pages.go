package supplychainapp

import (
	"context"
	"sort"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/marketplace/temu/model"
)

func readTemuSupplierSizePages(ctx context.Context, merchant TemuSupplierSizeMerchant, selection model.RuleSelection, observe func(string, string, any, any)) (model.SupplierSizePage, []model.SupplierSizePage, error) {
	offsets := map[int64]bool{selection.SupplierSizeOffset: true, 0: true}
	for _, offset := range selection.SupplierSizeOffsets {
		if offset < 0 || offset > 1000000 {
			return model.SupplierSizePage{}, nil, record.ErrInvalid
		}
		offsets[offset] = true
	}
	ordered := make([]int64, 0, len(offsets))
	for offset := range offsets {
		ordered = append(ordered, offset)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	var current model.SupplierSizePage
	others := []model.SupplierSizePage{}
	seen := map[int64]bool{}
	for _, offset := range ordered {
		if err := ctx.Err(); err != nil {
			return model.SupplierSizePage{}, nil, err
		}
		page, err := merchant.SupplierSizeCharts(ctx, selection.CatID, offset)
		if err != nil {
			return model.SupplierSizePage{}, nil, err
		}
		if page.Offset == nil || *page.Offset != offset || page.PageSize == nil || *page.PageSize != 50 || page.Total == nil || *page.Total < 0 || page.Charts == nil || len(page.Charts) > 50 || offset+int64(len(page.Charts)) > *page.Total {
			return model.SupplierSizePage{}, nil, record.ErrNotReady
		}
		for _, chart := range page.Charts {
			if chart.ID == nil || *chart.ID < 1 || seen[*chart.ID] {
				return model.SupplierSizePage{}, nil, record.ErrNotReady
			}
			seen[*chart.ID] = true
		}
		observe("E47", "bg.glo.goods.sizecharts.get", struct{ CatID, Offset int64 }{selection.CatID, offset}, page)
		if offset == selection.SupplierSizeOffset {
			current = page
		} else {
			others = append(others, page)
		}
	}
	return current, others, nil
}
