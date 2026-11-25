package listingapp

import (
	"testing"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
)

func TestShopifyMediaPreviewRequiresExactVariantsAndUniqueMain(t *testing.T) {
	local := []catalog.Variant{
		{SKU: "BLK-S", Attributes: []catalog.Attribute{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "S"}}, Images: []catalog.Image{{URL: "black"}}},
		{SKU: "BLK-M", Attributes: []catalog.Attribute{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "M"}}, Images: []catalog.Image{{URL: "black"}}},
		{SKU: "WHT-M", Attributes: []catalog.Attribute{{Name: "Color", Value: "White"}, {Name: "Size", Value: "M"}}, Images: []catalog.Image{{URL: "white"}}},
	}
	remote := []wire.DraftMediaVariant{
		{SKU: "BLK-M", MediaIDs: []string{"media"}, Options: []wire.DraftMediaOption{{Name: "Size", Value: "M"}, {Name: "Color", Value: "Black"}}},
		{SKU: "BLK-S", Options: []wire.DraftMediaOption{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "S"}}},
		{SKU: "WHT-M", Options: []wire.DraftMediaOption{{Name: "Color", Value: "White"}, {Name: "Size", Value: "M"}}},
	}
	images := []ShopifyMediaPreviewImage{
		{ID: "main", Color: "Black", Role: asset.RoleMain, State: wire.DraftMediaPresent, mediaID: "media", sourceID: "black-original"},
		{ID: "white", Color: "White", Role: asset.RoleMain, State: wire.DraftMediaMissing, sourceID: "white-original"},
	}
	sources := []asset.SourceImage{{ID: "black-original", URL: "black"}, {ID: "white-original", URL: "white"}}
	got, ok := previewShopifyVariants(local, remote, images, sources)
	if !ok || len(got) != 3 || got[0].SKU != "BLK-S" || got[0].AssetID != "main" ||
		got[0].State != "different" || got[1].AssetID != "main" || got[1].State != "same" {
		t.Fatalf("same-color variants did not share their exact approved main: %+v, %t", got, ok)
	}
	ambiguous := append(images, ShopifyMediaPreviewImage{ID: "second", Color: "Black", Role: asset.RoleMain, State: wire.DraftMediaPresent, sourceID: "black-original"})
	if got, ok := previewShopifyVariants(local, remote, ambiguous, sources); !ok || got[0].State != "unverified" || got[1].AssetID != "" {
		t.Fatalf("multiple available mains must remain unverified: %+v, %t", got, ok)
	}
	remote[0].MediaIDs = []string{"another-media", "media"}
	if got, ok := previewShopifyVariants(local, remote, images, sources); !ok || got[1].State != "unverified" || got[1].CurrentAssetID != "" {
		t.Fatalf("multiple remote media cannot identify a primary image: %+v, %t", got, ok)
	}
	if got, ok := previewShopifyVariants(local, remote, images[:1], sources); !ok || got[0].AssetID != "" {
		t.Fatalf("an incomplete color dimension was guessed: %+v, %t", got, ok)
	}
	remote[0].Options[0].Value = "S"
	if _, ok := previewShopifyVariants(local, remote, images, sources); ok {
		t.Fatal("matching SKU with a different option was accepted")
	}
}
