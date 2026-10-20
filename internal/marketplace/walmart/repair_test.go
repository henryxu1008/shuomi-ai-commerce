package walmart

import (
	"encoding/json"
	"testing"
)

func TestApprovedRepairWireContainsOnlySelectedFailedSKUs(t *testing.T) {
	s := RuleSelection{FeedType: MatchFeed, Version: "5-test", ProductType: "Shirts"}
	r := RuleSnapshot{Selection: s, Header: FeedHeader{BusinessUnit: "WALMART_US", Locale: "en", Version: s.Version}, Schema: json.RawMessage(matchSchema)}
	in := DraftInput{Selection: s, RepairFrom: "original-attempt", RepairSKUs: []string{SellerSKU("merchant", "product", "b")}, SKUs: []SKUInput{{SourceSKUKey: "a", Identifier: Identifier{Type: "GTIN", ID: "12345678901234"}, Condition: "New", Price: "1", ShippingWeight: "1"}, {SourceSKUKey: "b", Identifier: Identifier{Type: "GTIN", ID: "12345678901235"}, Condition: "New", Price: "2", ShippingWeight: "1"}}}
	d, e := BuildDraft(in, r, "merchant", "product", []string{"a", "b"})
	if e != nil {
		t.Fatal(e)
	}
	if len(d.SKUs) != 1 || d.SKUs[0].SourceSKUKey != "b" {
		t.Fatal("repair included successful SKU")
	}
	in.RepairSKUs = append(in.RepairSKUs, in.RepairSKUs[0])
	if _, e = BuildDraft(in, r, "merchant", "product", []string{"a", "b"}); e == nil {
		t.Fatal("duplicate repair selector")
	}
}
