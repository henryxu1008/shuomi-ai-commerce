package storecenterapp

import (
	"context"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

type TemuSupplierSizeProvider interface {
	QuerySupplierSizeClass(context.Context, storecenter.OfficialMerchantCredential, int64) (model.SupplierSizeRequirement, error)
	QuerySupplierSizeCharts(context.Context, storecenter.OfficialMerchantCredential, int64, int64) (model.SupplierSizePage, error)
	QuerySupplierSizeSettings(context.Context, storecenter.OfficialMerchantCredential, int64, int64) (model.SupplierSizeSettings, error)
}

func (h *TemuRulesHandle) SupplierSizeSettings(ctx context.Context, cat, class int64) (model.SupplierSizeSettings, error) {
	return callTemuRule(ctx, h, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.SupplierSizeSettings, error) {
		p, ok := h.entry.provider.(TemuSupplierSizeProvider)
		if !ok {
			return model.SupplierSizeSettings{}, ErrProductAccessChanged
		}
		return p.QuerySupplierSizeSettings(ctx, c, cat, class)
	})
}

func (h *TemuRulesHandle) SupplierSizeClass(ctx context.Context, cat int64) (model.SupplierSizeRequirement, error) {
	return callTemuRule(ctx, h, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.SupplierSizeRequirement, error) {
		p, ok := h.entry.provider.(TemuSupplierSizeProvider)
		if !ok {
			return model.SupplierSizeRequirement{}, ErrProductAccessChanged
		}
		return p.QuerySupplierSizeClass(ctx, c, cat)
	})
}
func (h *TemuRulesHandle) SupplierSizeCharts(ctx context.Context, cat, offset int64) (model.SupplierSizePage, error) {
	return callTemuRule(ctx, h, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (model.SupplierSizePage, error) {
		p, ok := h.entry.provider.(TemuSupplierSizeProvider)
		if !ok {
			return model.SupplierSizePage{}, ErrProductAccessChanged
		}
		return p.QuerySupplierSizeCharts(ctx, c, cat, offset)
	})
}
