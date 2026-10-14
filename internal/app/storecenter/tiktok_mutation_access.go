package storecenterapp

import (
	"context"
	"sync/atomic"

	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/storecenter"
	authorization "task-processor/internal/storecenter/tiktokshop"
)

type TikTokMutationProvider interface {
	UploadProductImage(context.Context, string, string, string, []byte) (marketplace.UploadedImage, error)
	CreateGlobalProduct(context.Context, string, marketplace.CreateGlobalRequest) (marketplace.CreatedGlobalProduct, error)
	CreateLocalProduct(context.Context, string, string, marketplace.CreateLocalRequest) (marketplace.CreatedLocalProduct, error)
	PublishGlobalProduct(context.Context, string, string, marketplace.PublishGlobalRequest, marketplace.GlobalPublishExpectation) (marketplace.GlobalPublishResult, error)
}
type TikTokMutationHandle struct {
	authority *TikTokRulesHandle
	sent      atomic.Bool
}

func (a *OfficialProductAccess) AuthorizeTikTokMutation(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string, expected *MerchantBinding) (*TikTokMutationHandle, error) {
	if subject.Purpose != storecenter.ProductPurposePublish && subject.Purpose != storecenter.ProductPurposeImage || a == nil || !a.applications.TikTokConfigured() {
		return nil, ErrProductAccessChanged
	}
	if _, ok := a.applications.tiktokProvider.(TikTokMutationProvider); !ok {
		return nil, ErrProductAccessChanged
	}
	authority, err := a.authorizeTikTok(ctx, subject, storeID, expected)
	if err != nil {
		return nil, err
	}
	return &TikTokMutationHandle{authority: authority}, nil
}
func (h *TikTokMutationHandle) Binding() MerchantBinding {
	if h == nil {
		return MerchantBinding{}
	}
	return h.authority.Binding()
}

func (h *TikTokMutationHandle) consume(ctx context.Context, purpose, scope string) (authorization.TokenSet, authorization.Shop, error) {
	if h == nil || h.authority == nil || h.authority.subject.Purpose != purpose {
		return authorization.TokenSet{}, authorization.Shop{}, ErrProductAccessChanged
	}
	a := h.authority
	tokens, err := a.credential(ctx, scope)
	if err != nil {
		return authorization.TokenSet{}, authorization.Shop{}, err
	}
	shops, err := a.owner.applications.tiktokProvider.AuthorizedShops(ctx, tokens.AccessToken)
	if err != nil {
		return authorization.TokenSet{}, authorization.Shop{}, ErrProductAccessChanged
	}
	shop, err := authorization.SelectUSCrossBorderShop(shops, a.material.Binding.Shop.Code)
	if err != nil || shop.ID != a.binding.TikTok.ShopID {
		return authorization.TokenSet{}, authorization.Shop{}, ErrProductAccessChanged
	}
	if _, err = a.credential(ctx, scope); err != nil || !h.sent.CompareAndSwap(false, true) {
		return authorization.TokenSet{}, authorization.Shop{}, ErrProductAccessChanged
	}

	return tokens, shop, nil
}
func (h *TikTokMutationHandle) UploadImage(ctx context.Context, useCase, mediaType string, content []byte) (marketplace.UploadedImage, error) {
	tokens, _, err := h.consume(ctx, storecenter.ProductPurposeImage, "seller.product.basic")
	if err != nil {
		return marketplace.UploadedImage{}, err
	}
	return h.authority.owner.applications.tiktokProvider.(TikTokMutationProvider).UploadProductImage(ctx, tokens.AccessToken, useCase, mediaType, content)
}
func (h *TikTokMutationHandle) CreateGlobal(ctx context.Context, input marketplace.CreateGlobalRequest) (marketplace.CreatedGlobalProduct, error) {
	tokens, _, err := h.consume(ctx, storecenter.ProductPurposePublish, "seller.global_product.write")
	if err != nil {
		return marketplace.CreatedGlobalProduct{}, err
	}
	return h.authority.owner.applications.tiktokProvider.(TikTokMutationProvider).CreateGlobalProduct(ctx, tokens.AccessToken, input)
}
func (h *TikTokMutationHandle) CreateLocal(ctx context.Context, input marketplace.CreateLocalRequest) (marketplace.CreatedLocalProduct, error) {
	tokens, shop, err := h.consume(ctx, storecenter.ProductPurposePublish, "seller.product.write")
	if err != nil {
		return marketplace.CreatedLocalProduct{}, err
	}
	return h.authority.owner.applications.tiktokProvider.(TikTokMutationProvider).CreateLocalProduct(ctx, tokens.AccessToken, shop.Cipher, input)
}
func (h *TikTokMutationHandle) PublishUS(ctx context.Context, globalID string, input marketplace.PublishGlobalRequest, expected marketplace.GlobalPublishExpectation) (marketplace.GlobalPublishResult, error) {
	if expected.ShopID != h.Binding().TikTok.ShopID {
		return marketplace.GlobalPublishResult{}, ErrProductAccessChanged
	}
	tokens, _, err := h.consume(ctx, storecenter.ProductPurposePublish, "seller.global_product.write")
	if err != nil {
		return marketplace.GlobalPublishResult{}, err
	}
	return h.authority.owner.applications.tiktokProvider.(TikTokMutationProvider).PublishGlobalProduct(ctx, tokens.AccessToken, globalID, input, expected)
}
