package listingapp

import (
	"context"
	"errors"
	"time"

	storeapp "github.com/henryxu1008/shuomi-ai-commerce/internal/app/storecenter"
	amazonapi "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/amazon"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	amazonmarket "github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/amazon"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shein/goods"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
)

type AmazonTargetPreparer struct {
	Access *storeapp.AmazonProductAccess
	Images record.TargetImageReader
}

func (a AmazonTargetPreparer) PrepareAmazon(ctx context.Context, scope collection.Scope, storeID string, input amazonmarket.DraftInput, product catalog.PublishedSnapshot, inventory asset.ApprovedAssetInventory) (storecenter.ProductMerchantBinding, record.AmazonTarget, error) {
	if a.Access == nil || a.Images == nil {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, record.ErrUnavailable
	}
	if len(product.Snapshot.Variants) > 1 {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, record.ErrNotReady
	}
	observed, err := a.amazonImage(ctx, input.MainImageID, inventory)
	if err != nil {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, err
	}
	subject := amazonSubject(scope)
	conn, err := a.Access.Open(ctx, subject, storeID, nil)
	if err != nil {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, record.ErrNotReady
	}
	draft, err := a.amazonPreview(ctx, conn, product.Identity.ProductKey, input, observed.SourceURL)
	if err != nil {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, err
	}
	binding := conn.Binding()
	if !a.Access.Current(ctx, subject, binding) {
		return storecenter.ProductMerchantBinding{}, record.AmazonTarget{}, record.ErrConflict
	}
	validUntil := time.Now().UTC().Truncate(time.Microsecond).Add(record.AmazonReviewLifetime)
	if binding.CredentialExpiresAt.Before(validUntil) {
		validUntil = binding.CredentialExpiresAt
	}
	return binding, record.AmazonTarget{Draft: draft, ValidUntil: validUntil, ImageHash: observed.ContentHash}, nil
}

func (a AmazonTargetPreparer) amazonImage(ctx context.Context, imageID string, inventory asset.ApprovedAssetInventory) (goods.OfficialImageObservation, error) {
	for _, approved := range inventory.Assets {
		if imageID == "" || approved.ID != imageID {
			continue
		}
		observation, err := a.Images.Probe(ctx, approved, 1)
		if err != nil || observation.AssetID != imageID || observation.SourceURL != approved.URL || !goods.ValidImageObservation(observation) {
			return goods.OfficialImageObservation{}, record.ErrNotReady
		}
		return observation, nil
	}
	return goods.OfficialImageObservation{}, record.ErrNotReady
}

func (a AmazonTargetPreparer) amazonPreview(ctx context.Context, conn *storeapp.AmazonProductSession, productKey string, input amazonmarket.DraftInput, imageURL string) (amazonmarket.Draft, error) {
	definition, err := conn.GetDefinition(ctx, input.ProductType)
	if err != nil {
		return amazonmarket.Draft{}, record.ErrUnavailable
	}
	draft, err := amazonmarket.BuildDraft(input, conn.Binding().StoreID, productKey, imageURL, definition.Schema)
	if errors.Is(err, amazonmarket.ErrInvalidRules) {
		return amazonmarket.Draft{}, record.ErrUnavailable
	}
	if err != nil {
		return amazonmarket.Draft{}, record.ErrNotReady
	}
	preview, err := conn.Preview(ctx, draft.SKU, amazonListingInput(draft))
	if err != nil {
		return amazonmarket.Draft{}, record.ErrUnavailable
	}
	if !preview.Valid {
		return amazonmarket.Draft{}, record.ErrNotReady
	}
	for _, issue := range preview.Issues {
		if issue.Severity == "ERROR" {
			return amazonmarket.Draft{}, record.ErrNotReady
		}
	}
	return draft, nil
}

func amazonListingInput(draft amazonmarket.Draft) amazonapi.ListingInput {
	return amazonapi.ListingInput{ProductType: draft.ProductType, Attributes: draft.Attributes}
}

func amazonSubject(scope collection.Scope) storecenter.ProductExecutionSubject {
	return storecenter.ProductExecutionSubject{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID, MemberID: scope.MemberID}
}
