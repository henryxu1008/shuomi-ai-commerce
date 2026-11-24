package amazonus

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

const syntheticSchema = `{
  "$schema":"https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1",
  "type":"object",
  "required":["item_name","brand","condition_type","externally_assigned_product_identifier","main_product_image_locator","purchasable_offer","fulfillment_availability"],
  "properties":{
    "item_name":{"type":"array","items":{"type":"object","required":["value"],"properties":{"value":{"type":"string","maxUtf8ByteLength":15}}}},
    "fulfillment_availability":{"type":"array","items":{"type":"object","required":["quantity"],"properties":{"quantity":{"type":"integer","minimum":0}}}}
  }
}`

func TestDraftPreservesExplicitZeroAndStableListingIdentifier(t *testing.T) {
	in := fixtureDraftInput()
	a, err := BuildDraft(in, "store-1", "product-1", "https://assets.example.org/main.jpg", []byte(syntheticSchema))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.SKU, "shuomi-") || len(a.SKU) > 40 || a.SKU != SellerSKU("store-1", "product-1") || a.SKU == SellerSKU("store-2", "product-1") {
		t.Fatalf("unexpected SKU %q", a.SKU)
	}
	var attributes struct {
		Availability []struct {
			Quantity int `json:"quantity"`
		} `json:"fulfillment_availability"`
	}
	if err := json.Unmarshal(a.Attributes, &attributes); err != nil || len(attributes.Availability) != 1 || attributes.Availability[0].Quantity != 0 ||
		!strings.Contains(string(a.Attributes), `"value_with_tax":12.30`) || !strings.Contains(string(a.Attributes), `"currency":"USD"`) {
		t.Fatalf("wire payload %s, error %v", a.Attributes, err)
	}
	hash := sha256.Sum256([]byte(syntheticSchema))
	if a.RuleHash != hex.EncodeToString(hash[:]) || a.Validate() != nil {
		t.Fatal("draft is not bound to the verified rule bytes")
	}
	in.PriceCents = 1899
	b, err := BuildDraft(in, "store-1", "product-1", "https://assets.example.org/main.jpg", []byte(syntheticSchema))
	if err != nil || b.SKU != a.SKU || b.WireHash == a.WireHash {
		t.Fatal("changing sales terms must retain the SKU and invalidate the old confirmation")
	}
}

func TestDraftRejectsMissingOrUnsafeProductData(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*DraftInput)
	}{
		{"missing quantity", func(in *DraftInput) { in.Quantity = nil }},
		{"missing price", func(in *DraftInput) { in.PriceCents = 0 }},
		{"invalid GTIN checksum", func(in *DraftInput) { in.GTIN = "036000291451" }},
		{"missing brand", func(in *DraftInput) { in.Brand = "" }},
		{"invalid UTF8 bytes in title", func(in *DraftInput) { in.Title = "测试六个字符" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := fixtureDraftInput()
			tc.edit(&in)
			if _, err := BuildDraft(in, "store-1", "product-1", "https://assets.example.org/main.jpg", []byte(syntheticSchema)); err == nil {
				t.Fatal("invalid product was accepted")
			}
		})
	}
	for _, schema := range []string{
		`{"$ref":"https://external.example.org/path"}`,
		`{"$schema":"https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1","$ref":"https://external.example.org/path"}`,
		`{"$schema":"https://schemas.amazon.com/selling-partners/definitions/product-types/meta-schema/v1","required":["missing_attribute"]}`,
	} {
		if _, err := BuildDraft(fixtureDraftInput(), "store-1", "product-1", "https://assets.example.org/main.jpg", []byte(schema)); err == nil {
			t.Fatal("unsafe or unmatched rule document was accepted")
		}
	}
}

func fixtureDraftInput() DraftInput {
	quantity := 0
	return DraftInput{ProductType: "LUGGAGE", Title: "Travel pouch", Brand: "Fixture", GTIN: "036000291452", PriceCents: 1230, Quantity: &quantity, MainImageID: "fixture-image"}
}
