package walmart

import (
	"encoding/json"
	"testing"
)

const matchSchema = `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["MPItemFeedHeader","MPItem"],"properties":{"MPItem":{"type":"array","items":{"type":"object","required":["Item"],"properties":{"Item":{"type":"object","required":["sku","productIdentifiers","price","condition"],"properties":{"condition":{"enum":["New"]},"price":{"type":"number","exclusiveMinimum":0}},"if":{"properties":{"condition":{"const":"Restored"}}},"then":{"required":["restoredProductIdentifier"]}}}}}}}`

func TestApprovedFeedHasStableMerchantSKUAndSchemaValidation(t *testing.T) {
	selection := RuleSelection{FeedType: MatchFeed, Version: "5.0-test", ProductType: "Shirts"}
	rules := RuleSnapshot{Selection: selection, Schema: json.RawMessage(matchSchema), Header: FeedHeader{BusinessUnit: "WALMART_US", Locale: "en", Version: "5.0-test"}}
	in := DraftInput{Selection: selection, SKUs: []SKUInput{{SourceSKUKey: "source-1", Identifier: Identifier{Type: "GTIN", ID: "00012345678905"}, Condition: "New", Price: json.Number("12.50"), ShippingWeight: json.Number("1")}}}
	a, err := BuildDraft(in, rules, "merchant-a", "product-a", []string{"source-1"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := BuildDraft(in, rules, "merchant-a", "product-a", []string{"source-1"})
	if err != nil || a.WireHash != b.WireHash || a.SKUs[0].SellerSKU != b.SKUs[0].SellerSKU {
		t.Fatal("unstable approved feed", err)
	}
	c, err := BuildDraft(in, rules, "merchant-b", "product-a", []string{"source-1"})
	if err != nil || c.SKUs[0].SellerSKU == a.SKUs[0].SellerSKU {
		t.Fatal("merchant namespace collision", err)
	}
	in.SKUs[0].Condition = "Invented"
	if _, err = BuildDraft(in, rules, "merchant-a", "product-a", []string{"source-1"}); err == nil {
		t.Fatal("schema enum bypass")
	}
	in.SKUs[0].Condition = "New"
	if _, err = BuildDraft(in, rules, "merchant-a", "product-a", []string{"source-1", "source-2"}); err == nil {
		t.Fatal("missing source variant")
	}
}

func TestRulesRejectExternalReferencesAndPartialSchemas(t *testing.T) {
	r := RuleSnapshot{Selection: RuleSelection{FeedType: MatchFeed, Version: "5", ProductType: "Shirts"}, Header: FeedHeader{BusinessUnit: "WALMART_US", Locale: "en", Version: "5"}, Schema: json.RawMessage(`{"$schema":"http://json-schema.org/draft-07/schema#","$ref":"file:///etc/passwd"}`)}
	if r.Validate() == nil {
		t.Fatal("external ref accepted")
	}
	r.Schema = json.RawMessage(matchSchema)
	r.Partial = true
	if r.Validate() == nil {
		t.Fatal("partial schema accepted")
	}
}

func TestFeedCompletenessAndRepairNeverResendsSuccessfulOrUnknownSKU(t *testing.T) {
	expected := []string{"a", "b"}
	o := FeedObservation{FeedID: "feed", FeedStatus: "PROCESSED", Received: 2, Succeeded: 1, Failed: 1, Items: []ItemResult{{SKU: "a", IngestionStatus: "SUCCESS", PublicationStatus: "UNPUBLISHED"}, {SKU: "b", IngestionStatus: "DATA_ERROR"}}}
	if !o.CompleteFor(expected) || !o.AllowsRepair(expected, []string{"b"}) || o.AllowsRepair(expected, []string{"a"}) {
		t.Fatal("repair admitted wrong subset")
	}
	o.Items[1].IngestionStatus = "INPROGRESS"
	if o.AllowsRepair(expected, []string{"b"}) {
		t.Fatal("pending repair")
	}
	o.Items[1].SKU = "a"
	if o.CompleteFor(expected) {
		t.Fatal("duplicate SKU accepted")
	}
	o.Items = o.Items[:1]
	if o.CompleteFor(expected) {
		t.Fatal("partial success accepted")
	}
}

func TestFirstReleaseRejectsExemptionAndAutomaticPricingMutations(t *testing.T) {
	selection := RuleSelection{FeedType: MatchFeed, Version: "5"}
	rules := RuleSnapshot{Selection: selection, Schema: []byte(matchSchema), Header: FeedHeader{Locale: "en", Version: "5"}}
	base := SKUInput{SourceSKUKey: "a", Identifier: Identifier{Type: "GTIN", ID: "12345678901234"}, Condition: "New", Price: "10", ShippingWeight: "1"}
	for _, key := range []string{"automatePricing", "automate_pricing", "repricerStrategy", "fulfillmentType"} {
		sku := base
		sku.Attributes = map[string]any{key: "WFS"}
		if _, e := BuildDraft(DraftInput{Selection: selection, SKUs: []SKUInput{sku}}, rules, "merchant", "product", []string{"a"}); e == nil {
			t.Errorf("out of scope mutation %s admitted", key)
		}
	}
	base.Identifier = Identifier{Type: "GTIN", ID: "CUSTOM"}
	if _, e := BuildDraft(DraftInput{Selection: selection, SKUs: []SKUInput{base}}, rules, "merchant", "product", []string{"a"}); e == nil {
		t.Error("GTIN exemption application admitted")
	}
}
