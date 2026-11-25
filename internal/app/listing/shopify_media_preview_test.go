package listingapp

import (
	"testing"

	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
)

func TestShopifyMediaPreviewRequiresExactVariantsAndUniqueMain(t *testing.T) {
	local := []catalog.Variant{
		{SKU: "BLK-S", Attributes: []catalog.Attribute{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "S"}}},
		{SKU: "BLK-M", Attributes: []catalog.Attribute{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "M"}}},
	}
	remote := []wire.DraftMediaVariant{
		{SKU: "BLK-M", MediaIDs: []string{"media"}, Options: []wire.DraftMediaOption{{Name: "Size", Value: "M"}, {Name: "Color", Value: "Black"}}},
		{SKU: "BLK-S", Options: []wire.DraftMediaOption{{Name: "Color", Value: "Black"}, {Name: "Size", Value: "S"}}},
	}
	images := []ShopifyMediaPreviewImage{{ID: "main", Color: "Black", Role: asset.RoleMain, State: wire.DraftMediaPresent, mediaID: "media"}}
	got, ok := previewShopifyVariants(local, remote, images)
	if !ok || len(got) != 2 || got[0].SKU != "BLK-S" || got[0].AssetID != "main" ||
		got[0].State != "different" || got[1].AssetID != "main" || got[1].State != "same" {
		t.Fatalf("same-color variants did not share their exact approved main: %+v, %t", got, ok)
	}
	ambiguous := append(images, ShopifyMediaPreviewImage{ID: "second", Color: "Black", Role: asset.RoleMain, State: wire.DraftMediaPresent})
	if got, ok := previewShopifyVariants(local, remote, ambiguous); !ok || got[0].State != "unverified" || got[1].AssetID != "" {
		t.Fatalf("multiple available mains must remain unverified: %+v, %t", got, ok)
	}
	remote[0].Options[0].Value = "S"
	if _, ok := previewShopifyVariants(local, remote, images); ok {
		t.Fatal("matching SKU with a different option was accepted")
	}
}
