package goods

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/product/asset"
	"testing"
)

func TestTemuImageSlotsBindExactApprovedBytesCategoryUsageAndPosition(t *testing.T) {
	input := DraftInput{Mode: "local", Selection: model.RuleSelection{CatID: 34}, Local: &model.LocalGoodsV2{SKUs: []model.LocalSKU{{}}}, Images: []ImageSlot{{Field: "sku", SKU: 0, Sort: 0, AssetID: "a", Usage: 3}}}
	inventory := asset.ApprovedAssetInventory{Assets: []asset.ApprovedAsset{{ID: "a", URL: "https://images.example/a.png"}}}
	observations := []ImageObservation{{AssetID: "a", SourceURL: inventory.Assets[0].URL, CatID: 34, Usage: 3, Width: 900, Height: 900, Bytes: 123, MediaType: "image/png", ContentHash: strings.Repeat("a", 64)}}
	result, issues := resolveImages(input, inventory, observations, draftPointer(int64(1)))
	require.Empty(t, issues)
	require.Equal(t, []string{inventory.Assets[0].URL}, result.Local.SKUs[0].Images)
	require.Empty(t, input.Local.SKUs[0].Images, "resolution must not mutate editable facts")
	for _, unknown := range []int64{-1, 99} {
		_, issues = resolveImages(input, inventory, observations, &unknown)
		require.NotEmpty(t, issues, "unknown category type cannot default to non-apparel geometry")
	}
	observations[0].SourceURL = "https://images.example/swapped.png"
	_, issues = resolveImages(input, inventory, observations, draftPointer(int64(1)))
	require.NotEmpty(t, issues)
	observations[0].SourceURL = inventory.Assets[0].URL
	input.Images[0].SKU = 2
	_, issues = resolveImages(input, inventory, observations, draftPointer(int64(1)))
	require.NotEmpty(t, issues, "out-of-bounds slots cannot make a record ready")
	input.Images[0].SKU = 0
	input.Images[0].Usage = 2
	_, issues = resolveImages(input, inventory, observations, draftPointer(int64(1)))
	require.NotEmpty(t, issues, "the client cannot change the usage for a field")
	input.Images[0].Usage = 3
	observations[0].ContentHash = strings.Repeat("z", 64)
	_, issues = resolveImages(input, inventory, observations, draftPointer(int64(1)))
	require.NotEmpty(t, issues)
}
