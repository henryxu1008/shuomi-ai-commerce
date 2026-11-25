package listingapp

import (
	"context"
	"slices"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
)

type shopifyDraftImageStager interface {
	StageImages(context.Context, [][]byte) ([]string, error)
}

func (s *UploadService) stageShopifyDraftImages(ctx context.Context, scope collection.Scope, saved record.TargetRecord,
	inventory asset.ApprovedAssetInventory, writer ShopifyDraftWriter, draft *shopify.Draft,
) (bool, error) {
	var contents [][]byte
	var positions []int
	read := s.dependencies.ShopifyGeneratedImages
	stager, canStage := writer.(shopifyDraftImageStager)
	for i, image := range draft.Images {
		if _, err := httpimage.ValidatePublicHTTPSURL(image.URL); err == nil {
			continue
		}
		if read == nil || !canStage {
			return false, record.ErrNotReady
		}
		id := saved.Input.Shopify.ImageIDs[i]
		index := slices.IndexFunc(inventory.Assets, func(item asset.ApprovedAsset) bool { return item.ID == id })
		if index < 0 || inventory.Assets[index].URL != image.URL || inventory.Assets[index].GenerationEvidence == nil {
			return false, record.ErrNotReady
		}
		source := asset.SourceSelection{TenantID: scope.OrganizationID, ProductKey: inventory.Scope.ProductKey,
			EffectiveCatalogVersion: inventory.Scope.SourceSnapshotVersion, TargetPlatform: inventory.Scope.TargetPlatform}
		content, err := read(ctx, source, inventory.Assets[index], wire.MaxStagedDraftImageBytes)
		if err != nil {
			return false, err
		}
		contents = append(contents, content)
		positions = append(positions, i)
	}
	if len(positions) == 0 {
		return false, nil
	}
	resources, err := stager.StageImages(ctx, contents)
	if err != nil || len(resources) != len(positions) {
		return false, record.ErrNotReady
	}
	for i, position := range positions {
		draft.Images[position].URL = resources[i]
	}
	return true, nil
}
