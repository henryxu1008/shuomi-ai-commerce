package shopify

import (
	"encoding/json"
	"testing"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/stretchr/testify/require"
)

func TestBuildDraftRejectsOptionsShopifyCannotDistinguish(t *testing.T) {
	product := catalog.ProductSnapshot{Title: "Sample", Variants: []catalog.Variant{
		{SKU: "red-a", Attributes: []catalog.Attribute{{Name: "Color", Value: "Red"}}},
		{SKU: "red-b", Attributes: []catalog.Attribute{{Name: "Color", Value: "red"}}},
	}}
	input := Input{Currency: "USD", Prices: []string{"1", "2"}}
	_, err := BuildDraft(input, product, asset.ApprovedAssetInventory{}, "USD")
	require.ErrorIs(t, err, ErrDraft)
	product.Variants[1].Attributes[0].Value = "Blue"
	draft, err := BuildDraft(input, product, asset.ApprovedAssetInventory{}, "USD")
	require.NoError(t, err)
	require.Equal(t, []string{"Red", "Blue"}, draft.Options[0].Values)
	require.Equal(t, "2", draft.Variants[1].Price)
	_, err = BuildDraft(input, product, asset.ApprovedAssetInventory{}, "EUR")
	require.ErrorIs(t, err, ErrDraft)
}

func TestBuildDraftBindsVariantToFrozenApprovedGalleryOrder(t *testing.T) {
	product := catalog.ProductSnapshot{Title: "Sample", Variants: []catalog.Variant{
		{SKU: "red", Attributes: []catalog.Attribute{{Name: "Color", Value: "Red"}}},
		{SKU: "blue", Attributes: []catalog.Attribute{{Name: "Color", Value: "Blue"}}},
	}}
	inventory := asset.ApprovedAssetInventory{Assets: []asset.ApprovedAsset{
		{ID: "red", URL: "https://localhost:36522/red.png"},
		{ID: "blue", URL: "https://localhost:36522/blue.png"},
	}}
	var input Input
	require.NoError(t, json.Unmarshal([]byte(`{"currency":"USD","prices":["1","2"],"imageIds":["blue","red"],"variantImageIds":["red","blue"]}`), &input))
	draft, err := BuildDraft(input, product, inventory, "USD")
	require.NoError(t, err)
	encoded, err := json.Marshal(draft.Variants)
	require.NoError(t, err)
	require.JSONEq(t, `[{"sku":"red","price":"1","values":["Red"],"imageIndex":2},{"sku":"blue","price":"2","values":["Blue"],"imageIndex":1}]`, string(encoded))
	require.NoError(t, json.Unmarshal([]byte(`{"variantImageIds":["outside","blue"]}`), &input))
	_, err = BuildDraft(input, product, inventory, "USD")
	require.ErrorIs(t, err, ErrDraft)
}
