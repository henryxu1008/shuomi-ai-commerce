package storecenterapp

import (
	"context"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

// ReadPairedShopifyTrialMedia verifies the exact dedicated pairing with live
// Publish authorization. It returns only the image and variant evidence.
func (a *OfficialProductAccess) ReadPairedShopifyTrialMedia(ctx context.Context, subject storecenter.ProductExecutionSubject, expected storecenter.ShopifyBinding, product string) (wire.DraftMediaEvidence, error) {
	token, err := a.pairedShopifyMediaToken(ctx, subject, expected)
	if err != nil {
		return wire.DraftMediaEvidence{}, err
	}
	evidence, err := a.applications.shopify.Admin.ReadDraftMediaEvidence(ctx, shopifyWireIdentity(expected.ShopifyInstallation), token.Value, product)
	if err != nil {
		return wire.DraftMediaEvidence{}, err
	}
	if actual, _, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID); err != nil || actual != expected {
		return wire.DraftMediaEvidence{}, ErrProductAccessChanged
	}
	return evidence, nil
}

// VerifyPairedShopifyMediaWrite grants a one-shot wire writer. Listing is still
// responsible for obtaining the recorded dispatch permit before using it.
func (a *OfficialProductAccess) VerifyPairedShopifyMediaWrite(ctx context.Context, subject storecenter.ProductExecutionSubject, expected storecenter.ShopifyBinding, product string) (*wire.DraftMediaWrite, error) {
	token, err := a.pairedShopifyMediaToken(ctx, subject, expected)
	if err != nil {
		return nil, err
	}
	writer, err := a.applications.shopify.Admin.VerifyDraftMediaWrite(ctx, shopifyWireIdentity(expected.ShopifyInstallation), token.Value, product)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	if actual, _, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID); err != nil || actual != expected {
		return nil, ErrProductAccessChanged
	}
	return writer, nil
}

func (a *OfficialProductAccess) CheckPairedShopifyMediaBinding(ctx context.Context, subject storecenter.ProductExecutionSubject, expected storecenter.ShopifyBinding) error {
	if subject.Purpose != storecenter.ProductPurposePublish {
		return ErrProductAccessChanged
	}
	actual, _, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID)
	if err != nil || actual != expected {
		return ErrProductAccessChanged
	}
	return nil
}

func (a *OfficialProductAccess) pairedShopifyMediaToken(ctx context.Context, subject storecenter.ProductExecutionSubject, expected storecenter.ShopifyBinding) (wire.AccessToken, error) {
	if err := a.CheckPairedShopifyMediaBinding(ctx, subject, expected); err != nil {
		return wire.AccessToken{}, err
	}
	token, err := a.applications.shopify.Issuer.Issue(ctx, expected.Domain)
	if err != nil {
		return wire.AccessToken{}, ErrProductAccessChanged
	}
	return token, nil
}
