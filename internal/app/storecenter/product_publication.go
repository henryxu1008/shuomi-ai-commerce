package storecenterapp

import (
	"context"
	"task-processor/internal/marketplace/shein/model"
	"task-processor/internal/storecenter"
)

type publicationProvider interface {
	QueryDocumentState(context.Context, storecenter.OfficialMerchantCredential, string, string) (model.DocumentAudit, error)
	QueryShelfProduct(context.Context, storecenter.OfficialMerchantCredential, string) (model.ShelfProduct, error)
	QuerySKUInventory(context.Context, storecenter.OfficialMerchantCredential, []string, string) (model.InventorySnapshot, error)
	ShelfOn(context.Context, storecenter.OfficialMerchantCredential, string) (model.ShelfResult, error)
}

func (h *MerchantHandle) publicationProvider() (publicationProvider, error) {
	if h == nil || (h.subject.Purpose != storecenter.ProductPurposeStatus && h.subject.Purpose != storecenter.ProductPurposeShelfOn) {
		return nil, ErrProductAccessChanged
	}
	p, ok := h.entry.provider.(publicationProvider)
	if !ok {
		return nil, ErrProductAccessChanged
	}
	return p, nil
}
func (h *MerchantHandle) QueryDocumentState(ctx context.Context, spu, version string) (model.DocumentAudit, error) {
	p, e := h.publicationProvider()
	if e != nil {
		return model.DocumentAudit{}, e
	}
	return callMerchant(ctx, h, "", func(ctx context.Context, k storecenter.OfficialMerchantCredential) (model.DocumentAudit, error) {
		return p.QueryDocumentState(ctx, k, spu, version)
	})
}
func (h *MerchantHandle) QueryShelfProduct(ctx context.Context, spu string) (model.ShelfProduct, error) {
	p, e := h.publicationProvider()
	if e != nil {
		return model.ShelfProduct{}, e
	}
	return callMerchant(ctx, h, "", func(ctx context.Context, k storecenter.OfficialMerchantCredential) (model.ShelfProduct, error) {
		return p.QueryShelfProduct(ctx, k, spu)
	})
}
func (h *MerchantHandle) QuerySKUInventory(ctx context.Context, skus []string, invType string) (model.InventorySnapshot, error) {
	p, e := h.publicationProvider()
	if e != nil {
		return model.InventorySnapshot{}, e
	}
	return callMerchant(ctx, h, "", func(ctx context.Context, k storecenter.OfficialMerchantCredential) (model.InventorySnapshot, error) {
		return p.QuerySKUInventory(ctx, k, skus, invType)
	})
}
func (h *MerchantHandle) ShelfOn(ctx context.Context, skc string) (model.ShelfResult, error) {
	p, e := h.publicationProvider()
	if e != nil {
		return model.ShelfResult{}, e
	}
	return callMerchant(ctx, h, storecenter.ProductPurposeShelfOn, func(ctx context.Context, k storecenter.OfficialMerchantCredential) (model.ShelfResult, error) {
		return p.ShelfOn(ctx, k, skc)
	})
}
