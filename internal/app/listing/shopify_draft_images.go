package listingapp

import (
	"context"
	"slices"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

type shopifyDraftImageStager interface {
	StageImages(context.Context, [][]byte) ([]string, error)
}

type shopifyDraftImages struct {
	contents  [][]byte
	positions []int
}

func (s *UploadService) readShopifyDraftImages(ctx context.Context, scope collection.Scope, saved record.TargetRecord,
	inventory asset.ApprovedAssetInventory, draft shopify.Draft,
) (shopifyDraftImages, error) {
	var prepared shopifyDraftImages
	read := s.dependencies.ShopifyGeneratedImages
	source := asset.SourceSelection{TenantID: scope.OrganizationID, ProductKey: inventory.Scope.ProductKey,
		EffectiveCatalogVersion: inventory.Scope.SourceSnapshotVersion, TargetPlatform: inventory.Scope.TargetPlatform}
	for i, image := range draft.Images {
		if wire.ValidPublicDraftImageURL(image.URL) {
			continue
		}
		if read == nil {
			return shopifyDraftImages{}, record.ErrNotReady
		}
		id := saved.Input.Shopify.ImageIDs[i]
		index := slices.IndexFunc(inventory.Assets, func(item asset.ApprovedAsset) bool { return item.ID == id })
		if index < 0 || inventory.Assets[index].URL != image.URL || inventory.Assets[index].GenerationEvidence == nil {
			return shopifyDraftImages{}, record.ErrNotReady
		}
		content, err := read(ctx, source, inventory.Assets[index], wire.MaxStagedDraftImageBytes)
		if err != nil {
			return shopifyDraftImages{}, err
		}
		prepared.contents = append(prepared.contents, content)
		prepared.positions = append(prepared.positions, i)
	}
	return prepared, nil
}

func (images shopifyDraftImages) stage(ctx context.Context, writer ShopifyDraftWriter, draft *shopify.Draft) error {
	stager, ok := writer.(shopifyDraftImageStager)
	if !ok {
		return record.ErrNotReady
	}
	resources, err := stager.StageImages(ctx, images.contents)
	if err != nil || len(resources) != len(images.positions) {
		return record.ErrNotReady
	}
	draft.Images = slices.Clone(draft.Images)
	for i, position := range images.positions {
		draft.Images[position].URL = resources[i]
	}
	return nil
}
