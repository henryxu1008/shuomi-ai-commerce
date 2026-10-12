package storecenterapp

import (
	"context"
	"sync/atomic"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/storecenter"
)

type TemuMutationProvider interface {
	CreateLocalGoods(context.Context, storecenter.OfficialMerchantCredential, model.LocalGoodsV2) (temugoods.CreatedGoods, error)
	UploadLocalImage(context.Context, storecenter.OfficialMerchantCredential, model.LocalImageUpload) (model.LocalImageUploaded, string, error)
}
type TemuSupplierMutationProvider interface {
	CreateSupplierGoods(context.Context, storecenter.OfficialMerchantCredential, model.SupplierGoods) (temugoods.CreatedGoods, error)
	UploadSupplierImage(context.Context, storecenter.OfficialMerchantCredential, model.SupplierImageUpload) (model.SupplierImageUploaded, string, error)
}

// Native mutations use the original caller's explicit purpose and current
// private Store proof. The original Submission owner controls send permits.
type TemuMutationHandle struct {
	authority    *temuMerchantAuthority
	mutationSent atomic.Bool
}

func validTemuProductPurpose(purpose string) bool {
	return purpose == storecenter.ProductPurposeRules || purpose == storecenter.ProductPurposePublish || purpose == storecenter.ProductPurposeImage
}

func (a *OfficialProductAccess) AuthorizeTemuMutation(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *MerchantBinding) (*TemuMutationHandle, error) {
	if subject.Purpose != storecenter.ProductPurposePublish && subject.Purpose != storecenter.ProductPurposeImage {
		return nil, ErrProductAccessChanged
	}
	proof, err := a.authorizeTemu(ctx, subject, storeID, expected)
	if err != nil {
		return nil, err
	}
	return &TemuMutationHandle{authority: proof}, nil
}

func (h *TemuMutationHandle) Binding() MerchantBinding {
	if h == nil || h.authority == nil {
		return MerchantBinding{}
	}
	return h.authority.binding
}

func callTemuMutation[T any](ctx context.Context, h *TemuMutationHandle, purpose string, local bool, call func(context.Context, storecenter.OfficialMerchantCredential) (T, error)) (T, error) {
	var zero T
	if h == nil || h.authority == nil || ctx == nil || h.authority.subject.Purpose != purpose || (h.authority.binding.MerchantMode == storecenter.MerchantLocal) != local {
		return zero, ErrProductAccessChanged
	}
	ctx, cancel := context.WithDeadline(ctx, h.authority.expiresAt)
	defer cancel()
	credential, err := h.authority.credential(ctx)
	if err != nil {
		return zero, err
	}
	if !h.mutationSent.CompareAndSwap(false, true) {
		return zero, ErrProductAccessChanged
	}
	// Once dispatched, confirmation is retained by the original completion UoW
	// even if membership changes afterwards. No second provider call is allowed.
	return call(ctx, credential)
}

func (h *TemuMutationHandle) CreateLocalGoods(ctx context.Context, input model.LocalGoodsV2) (temugoods.CreatedGoods, error) {
	return callTemuMutation(ctx, h, storecenter.ProductPurposePublish, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (temugoods.CreatedGoods, error) {
		return h.authority.entry.provider.(TemuMutationProvider).CreateLocalGoods(ctx, c, input)
	})
}
func (h *TemuMutationHandle) UploadLocalImage(ctx context.Context, input model.LocalImageUpload) (model.LocalImageUploaded, string, error) {
	type response struct {
		value model.LocalImageUploaded
		hash  string
	}
	result, err := callTemuMutation(ctx, h, storecenter.ProductPurposeImage, true, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (response, error) {
		value, hash, err := h.authority.entry.provider.(TemuMutationProvider).UploadLocalImage(ctx, c, input)
		return response{value, hash}, err
	})
	return result.value, result.hash, err
}

func (h *TemuMutationHandle) CreateSupplierGoods(ctx context.Context, input model.SupplierGoods) (temugoods.CreatedGoods, error) {
	return callTemuMutation(ctx, h, storecenter.ProductPurposePublish, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (temugoods.CreatedGoods, error) {
		provider, ok := h.authority.entry.provider.(TemuSupplierMutationProvider)
		if !ok {
			return temugoods.CreatedGoods{}, ErrProductAccessChanged
		}
		return provider.CreateSupplierGoods(ctx, c, input)
	})
}
func (h *TemuMutationHandle) UploadSupplierImage(ctx context.Context, input model.SupplierImageUpload) (model.SupplierImageUploaded, string, error) {
	type response struct {
		value model.SupplierImageUploaded
		hash  string
	}
	result, err := callTemuMutation(ctx, h, storecenter.ProductPurposeImage, false, func(ctx context.Context, c storecenter.OfficialMerchantCredential) (response, error) {
		provider, ok := h.authority.entry.provider.(TemuSupplierMutationProvider)
		if !ok {
			return response{}, ErrProductAccessChanged
		}
		value, hash, err := provider.UploadSupplierImage(ctx, c, input)
		return response{value, hash}, err
	})
	return result.value, result.hash, err
}
