package storecenterapp

import (
	"context"
	"time"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

func (a *OfficialProductAccess) ShopifyProductBinding(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string) (MerchantBinding, error) {
	_, binding, err := a.shopifyProductMaterial(ctx, subject, storeID)
	return binding, err
}

func (a *OfficialProductAccess) shopifyProductMaterial(ctx context.Context, subject storecenter.ProductExecutionSubject, storeID string) (storecenter.ShopifyBinding, MerchantBinding, error) {
	if a == nil || !a.applications.ShopifyConfigured() || ctx == nil ||
		(subject.Purpose != storecenter.ProductPurposeRules && subject.Purpose != storecenter.ProductPurposePublish && subject.Purpose != storecenter.ProductPurposeStatus) {
		return storecenter.ShopifyBinding{}, MerchantBinding{}, ErrProductAccessChanged
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	material, err := a.reader.ReadProductExecution(ctx, subject, storeID, a.authorization, a.now())
	if err != nil || material.Platform != storecenter.PlatformShopify || material.Shopify == nil || material.Shopify.AppID != a.applications.shopify.AppID {
		return storecenter.ShopifyBinding{}, MerchantBinding{}, ErrProductAccessChanged
	}
	value := *material.Shopify
	binding := MerchantBinding{Platform: storecenter.PlatformShopify, OrganizationID: subject.OrganizationID, StoreID: storeID,
		Site: "shopify", StoreVersion: material.StoreVersion, ConnectionRevision: value.Generation, ApplicationID: value.AppID,
		ApplicationRevision: "shopify-admin-2026-10", SupplierIdentityHash: shopifyDigestOf(value.ShopKey), ServiceExpiresAt: material.ServiceExpiresAt,
		Shopify: storecenter.ShopifyMerchantBinding{ShopKey: value.ShopKey, Currency: value.Currency}}
	if !binding.ValidShopifyApplication() {
		return storecenter.ShopifyBinding{}, MerchantBinding{}, ErrProductAccessChanged
	}
	return value, binding, nil
}

// ShopifyPlanDomain returns the approved installation's shop domain without contacting Shopify.
func (a *OfficialProductAccess) ShopifyPlanDomain(ctx context.Context, subject storecenter.ProductExecutionSubject, expected MerchantBinding) (string, error) {
	if subject.Purpose != storecenter.ProductPurposeRules {
		return "", ErrProductAccessChanged
	}
	value, current, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID)
	if err != nil || current != expected {
		return "", ErrProductAccessChanged
	}
	return value.Domain, nil
}

type ShopifyDraftWrite struct {
	transport *wire.DraftWrite
}

func (w *ShopifyDraftWrite) Create(ctx context.Context, draft shopify.Draft) (shopify.ProductObservation, error) {
	input := wire.Draft{SourceKey: draft.SourceKey, Title: draft.Title, DescriptionHTML: draft.DescriptionHTML, Currency: draft.Currency}
	for _, option := range draft.Options {
		input.Options = append(input.Options, wire.DraftOption{Name: option.Name, Values: option.Values})
	}
	for _, variant := range draft.Variants {
		input.Variants = append(input.Variants, wire.DraftVariant{SKU: variant.SKU, Price: variant.Price, Values: variant.Values, ImageIndex: variant.ImageIndex})
	}
	for _, image := range draft.Images {
		input.Images = append(input.Images, wire.DraftImage{URL: image.URL, Alt: image.Alt})
	}
	observation, err := w.transport.Create(ctx, input)
	return shopifyObservation(observation), err
}

func (w *ShopifyDraftWrite) StageImages(ctx context.Context, contents [][]byte) ([]string, error) {
	return w.transport.StageImages(ctx, contents)
}

func shopifyObservation(value wire.ProductObservation) shopify.ProductObservation {
	return shopify.ProductObservation{ID: value.ID, Status: value.Status, Media: value.Media}
}

// VerifyShopifyDraftWrite checks original member and Store access again after
// the exact installation was verified. A Listing send permit is still required.
func (a *OfficialProductAccess) VerifyShopifyDraftWrite(ctx context.Context, subject storecenter.ProductExecutionSubject, expected MerchantBinding) (*ShopifyDraftWrite, error) {
	if subject.Purpose != storecenter.ProductPurposePublish {
		return nil, ErrProductAccessChanged
	}
	value, binding, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID)
	if err != nil || binding != expected {
		return nil, ErrProductAccessChanged
	}
	app := a.applications.shopify
	token, err := app.Issuer.Issue(ctx, value.Domain)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	writer, err := app.Admin.VerifyDraftWrite(ctx, shopifyWireIdentity(value.ShopifyInstallation), token.Value)
	if err != nil {
		return nil, ErrProductAccessChanged
	}
	if _, current, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID); err != nil || current != expected {
		return nil, ErrProductAccessChanged
	}
	return &ShopifyDraftWrite{transport: writer}, nil
}

type ShopifyDraftReadback struct {
	Product  shopify.ProductObservation
	AdminURL string
}

func (a *OfficialProductAccess) ReadShopifyDraft(ctx context.Context, subject storecenter.ProductExecutionSubject, expected MerchantBinding, productID, sourceKey string) (ShopifyDraftReadback, error) {
	if subject.Purpose != storecenter.ProductPurposeStatus || (productID == "") == (sourceKey == "") {
		return ShopifyDraftReadback{}, ErrProductAccessChanged
	}
	value, binding, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID)
	if err != nil || binding != expected {
		return ShopifyDraftReadback{}, ErrProductAccessChanged
	}
	app := a.applications.shopify
	token, err := app.Issuer.Issue(ctx, value.Domain)
	if err != nil {
		return ShopifyDraftReadback{}, ErrProductAccessChanged
	}
	var observation wire.ProductObservation
	if productID != "" {
		observation, err = app.Admin.FindProductByID(ctx, shopifyWireIdentity(value.ShopifyInstallation), token.Value, productID)
	} else {
		observation, err = app.Admin.FindDraftBySource(ctx, shopifyWireIdentity(value.ShopifyInstallation), token.Value, sourceKey)
	}
	if err != nil {
		return ShopifyDraftReadback{}, err
	}
	if _, current, err := a.shopifyProductMaterial(ctx, subject, expected.StoreID); err != nil || current != expected {
		return ShopifyDraftReadback{}, ErrProductAccessChanged
	}
	return ShopifyDraftReadback{Product: shopifyObservation(observation), AdminURL: wire.ProductAdminURL(value.Domain, observation.ID)}, nil
}
