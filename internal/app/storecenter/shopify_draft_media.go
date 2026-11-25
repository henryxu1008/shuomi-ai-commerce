package storecenterapp

import (
	"context"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

func (a *OfficialProductAccess) InspectShopifyDraftMedia(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID, selected string, proofs []wire.DraftMediaFingerprint) (wire.DraftMediaObservation, string, error) {
	if subject.Purpose != storecenter.ProductPurposeStatus {
		return wire.DraftMediaObservation{}, "", ErrProductAccessChanged
	}
	value, binding, err := a.shopifyProductMaterial(ctx, subject, storeID)
	if err != nil {
		return wire.DraftMediaObservation{}, "", ErrProductAccessChanged
	}
	id, err := wire.ParseDraftMediaProduct(selected, value.Domain)
	if err != nil {
		return wire.DraftMediaObservation{}, "", err
	}
	app := a.applications.shopify
	token, err := app.Issuer.Issue(ctx, value.Domain)
	if err != nil {
		return wire.DraftMediaObservation{}, "", ErrProductAccessChanged
	}
	draft, err := app.Admin.InspectDraftMedia(ctx, shopifyWireIdentity(value.ShopifyInstallation), token.Value, id, proofs)
	if err != nil {
		return wire.DraftMediaObservation{}, "", err
	}
	if _, current, err := a.shopifyProductMaterial(ctx, subject, storeID); err != nil || binding != current {
		return wire.DraftMediaObservation{}, "", ErrProductAccessChanged
	}
	return draft, wire.ProductAdminURL(value.Domain, draft.ID), nil
}
