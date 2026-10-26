package draft

import (
	"task-processor/internal/marketplace/shein/offlinepackage"
	"testing"
)

func TestLocalVariantGroupRetainsOrderAndCommonAttributes(t *testing.T) {
	if got := buildLocalVariantGroups(nil, nil); got != nil {
		t.Fatalf("empty group: %#v", got)
	}
	variants := []offlinepackage.Variant{
		{SKU: "first", Attributes: map[string]string{"color": "Blue", "material": "Cotton"}},
		{SKU: "second", Attributes: map[string]string{"color": "Red", "material": "Cotton"}},
	}
	got := buildLocalVariantGroups(variants, &offlinepackage.ImageSet{MainImage: "https://controlled.invalid/main.jpg"})
	if len(got) != 1 || got[0].skcName != "first" || got[0].supplierCode != "first" || got[0].mainImageURL != "https://controlled.invalid/main.jpg" || len(got[0].skus) != 2 || got[0].skus[1].SKU != "second" {
		t.Fatalf("default group changed: %#v", got)
	}
	if len(got[0].attributes) != 1 || got[0].attributes["material"] != "Cotton" {
		t.Fatalf("common attributes changed: %#v", got[0].attributes)
	}
	got[0].skus[0].SKU = "mutated"
	if variants[0].SKU != "first" {
		t.Fatal("group slice shares source storage")
	}
}
