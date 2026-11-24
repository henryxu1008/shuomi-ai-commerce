package shopify

import (
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
