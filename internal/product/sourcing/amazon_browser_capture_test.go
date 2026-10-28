package sourcing

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func amazonFixture() []byte {
	return []byte(`{"captureVersion":2,"source":"amazon","evidence":{"schemaVersion":2,"site":"us","sourceID":"B012345678","sourceURL":"https://www.amazon.com/dp/B012345678","title":"Cotton bag","description":null,"attributes":[],"variants":[{"sourceID":"B012345678","sku":null,"title":null,"attributes":[{"name":"Color","value":"Blue"}],"price":{"amount":"19.99","currency":"USD"},"images":[{"url":"https://m.media-amazon.com/images/I/blue.jpg","role":"main"}]}],"priceFacts":[],"images":[],"observation":{"scope":"selected_product","parentASIN":null},"capturedAt":"2026-10-23T00:00:00Z","contentSHA256":"` + strings.Repeat("a", 64) + `","parserVersion":"amazon-browser-dom/v1","warnings":[],"missingFacts":[{"field":"variants","reason":"only selected product observed"}]}}`)
}

func TestAmazonConditionQuotesPreserveSourceWithoutProjectingUnknownOrUsed(t *testing.T) {
	for _, tc := range []struct {
		name, price string
		want        float64
	}{
		{"unknown", `{"amount":"19.99","currency":"USD"}`, 0},
		{"used", `{"amount":"19.99","currency":"USD","condition":"used","conditionNote":"Very Good"}`, 0},
		{"new with used", `{"amount":"19.99","currency":"USD","condition":"new","alternative":{"amount":"15.99","currency":null,"condition":"used"}}`, 19.99},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := strings.Replace(string(amazonFixture()), `{"amount":"19.99","currency":"USD"}`, tc.price, 1)
			capture, err := ParseAmazonBrowserCapture([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			env, err := capture.Envelope("capture")
			if err != nil {
				t.Fatal(err)
			}
			v := env.ProductCandidate.Variants[0]
			if v.Price != tc.want || !strings.Contains(v.Attributes["source.price.fact"], `"amount":"19.99"`) {
				t.Fatalf("lost or misprojected price: %+v", v)
			}
			if tc.name == "new with used" && !strings.Contains(v.Attributes["source.price.fact"], `"condition":"used"`) {
				t.Fatal("lost used quote")
			}
			if tc.name == "new with used" {
				quotes, err := ProjectSourceQuotes(map[string]string{"price_fact.0": v.Attributes["source.price.fact"]})
				if err != nil || len(quotes) != 1 || quotes[0].Alternative == nil || quotes[0].Alternative.Amount != "15.99" {
					t.Fatalf("lost source quote readback: %+v %v", quotes, err)
				}
			}
		})
	}
}

func TestAmazonConditionQuoteStrictContractAndOldDigest(t *testing.T) {
	old, err := ParseAmazonBrowserCapture(amazonFixture())
	if err != nil || old.PayloadSHA256 != "e26f59b6e15c97bd6f44bef8afb029ef4d7fca8c09715a7fd26631f84ecbacb1" {
		t.Fatalf("old digest changed: %s %v", old.PayloadSHA256, err)
	}
	rawNew := strings.Replace(string(amazonFixture()), `{"amount":"19.99","currency":"USD"}`, `{"amount":"35.98","currency":"USD","condition":"new","alternative":{"amount":"31.99","currency":null,"condition":"used","conditionNote":"Very Good"}}`, 1)
	newCapture, err := ParseAmazonBrowserCapture([]byte(rawNew))
	if err != nil {
		t.Fatal(err)
	}
	if newCapture.PayloadSHA256 != "da9e2e942f912cc06c7b124676149456b602739e98d3243ae1ebc14f504eb3f9" {
		t.Fatalf("new digest changed: %s", newCapture.PayloadSHA256)
	}
	for _, bad := range []string{
		`{"amount":"19.99","currency":"USD","condition":null}`,
		`{"amount":"19.99","currency":"USD","condition":"new","conditionNote":"Good"}`,
		`{"amount":"19.99","currency":"USD","condition":"new","alternative":{"amount":"15.00","currency":"USD","condition":"used","condition":"used"}}`,
		`{"amount":"19.99","currency":"USD","condition":"new","alternative":{"amount":"15.00","currency":"USD","condition":"new"}}`,
		`{"amount":"19.99","currency":"USD","condition":"used","alternative":{"amount":"15.00","currency":"USD","condition":"used"}}`,
	} {
		raw := strings.Replace(string(amazonFixture()), `{"amount":"19.99","currency":"USD"}`, bad, 1)
		if _, err := ParseAmazonBrowserCapture([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid quote %s", bad)
		}
	}
}

func TestAmazon171DualOffersRespectAggregateBudget(t *testing.T) {
	var input map[string]any
	if err := json.Unmarshal(amazonFixture(), &input); err != nil {
		t.Fatal(err)
	}
	e := input["evidence"].(map[string]any)
	e["observation"] = map[string]any{"scope": "partial_family", "parentASIN": "B012345679"}
	variants := make([]any, 0, 171)
	for i := 0; i < 171; i++ {
		id := fmt.Sprintf("B%09d", i)
		if i == 0 {
			id = "B012345678"
		}
		variants = append(variants, map[string]any{"sourceID": id, "sku": nil, "title": nil, "attributes": []any{map[string]any{"name": "Color", "value": fmt.Sprintf("c%d", i)}}, "images": []any{map[string]any{"url": "https://m.media-amazon.com/images/I/blue.jpg", "role": "main"}}, "price": map[string]any{"amount": "35.98", "currency": "USD", "condition": "new", "alternative": map[string]any{"amount": "31.99", "currency": "USD", "condition": "used"}}})
	}
	e["variants"] = variants
	e["missingFacts"] = []any{}
	raw, _ := json.Marshal(input)
	capture, err := ParseAmazonBrowserCapture(raw)
	if err != nil {
		t.Fatalf("171 offers rejected: %v", err)
	}
	env, err := capture.Envelope("capture")
	if err != nil || len(env.ProductCandidate.Variants) != 171 {
		t.Fatalf("lost offers: %v", err)
	}
	attributes := make([]any, 256)
	for i := range attributes {
		attributes[i] = map[string]any{"name": fmt.Sprintf("a%d", i), "value": "x"}
	}
	e["attributes"] = attributes
	for i := 0; i < 85; i++ {
		variants[i].(map[string]any)["attributes"] = append(variants[i].(map[string]any)["attributes"].([]any), map[string]any{"name": "Size", "value": fmt.Sprint(i)})
	}
	raw, _ = json.Marshal(input)
	if _, err = ParseAmazonBrowserCapture(raw); err == nil {
		t.Fatal("accepted aggregate over limit")
	}
}

func TestAmazonCaptureStrictMappingAndDigest(t *testing.T) {
	capture, err := ParseAmazonBrowserCapture(amazonFixture())
	if err != nil {
		t.Fatal(err)
	}
	env, err := capture.Envelope("capture")
	if err != nil {
		t.Fatal(err)
	}
	if env.Identity.SourcePlatform != "amazon" || env.Identity.SourceID != "us:B012345678" || env.ProductCandidate.Variants[0].Images[0].URL != "https://m.media-amazon.com/images/I/blue.jpg" {
		t.Fatalf("lost source relationship: %+v", env)
	}
	reordered := map[string]any{}
	_ = json.Unmarshal(amazonFixture(), &reordered)
	raw, _ := json.Marshal(reordered)
	again, err := ParseAmazonBrowserCapture(raw)
	if err != nil || again.PayloadSHA256 != capture.PayloadSHA256 {
		t.Fatal("object order changed canonical digest", err)
	}
	for _, pair := range [][2]string{{`"source":"amazon"`, `"source":"1688"`}, {`"site":"us"`, `"site":"fr"`}, {`"sourceID":"B012345678"`, `"sourceID":"B012345679"`}, {`"title":"Cotton bag"`, `"Title":"Cotton bag"`}, {`"captureVersion":2`, `"captureVersion":2,"captureVersion":2`}, {`"role":"main"`, `"role":"approved"`}, {`https://m.media-amazon.com/images/I/blue.jpg`, `http://127.0.0.1/a.jpg`}} {
		if _, err := ParseAmazonBrowserCapture([]byte(strings.Replace(string(amazonFixture()), pair[0], pair[1], 1))); err == nil {
			t.Fatalf("accepted invalid input %s", pair[1])
		}
	}
}

func TestAmazonCaptureDoesNotGuessVariantRelationship(t *testing.T) {
	raw := strings.Replace(string(amazonFixture()), `"parentASIN":null`, `"parentASIN":"B012345679"`, 1)
	if _, err := ParseAmazonBrowserCapture([]byte(raw)); err == nil {
		t.Fatal("selected product claimed family relationship")
	}
	var input map[string]any
	_ = json.Unmarshal(amazonFixture(), &input)
	e := input["evidence"].(map[string]any)
	v := e["variants"].([]any)[0]
	e["variants"] = []any{v, v}
	rawBytes, _ := json.Marshal(input)
	if _, err := ParseAmazonBrowserCapture(rawBytes); err == nil {
		t.Fatal("duplicate variant accepted")
	}
}

func TestAmazonPartialFamilyKeepsAnchorAndEachChildFacts(t *testing.T) {
	for _, currency := range []any{"USD", nil} {
		var input map[string]any
		_ = json.Unmarshal(amazonFixture(), &input)
		e := input["evidence"].(map[string]any)
		e["observation"] = map[string]any{"scope": "partial_family", "parentASIN": "B012345679"}
		anchor := e["variants"].([]any)[0]
		anchor.(map[string]any)["price"].(map[string]any)["condition"] = "new"
		child := map[string]any{"sourceID": "B012345680", "sku": nil, "title": nil,
			"attributes": []any{map[string]any{"name": "Color", "value": "Beige"}},
			"price":      map[string]any{"amount": "44.98", "currency": currency, "condition": "new"},
			"images":     []any{map[string]any{"url": "https://m.media-amazon.com/images/I/beige.jpg", "role": "main"}}}
		e["variants"] = []any{anchor, child}
		raw, _ := json.Marshal(input)
		capture, err := ParseAmazonBrowserCapture(raw)
		if err != nil {
			t.Fatal(err)
		}
		env, err := capture.Envelope("capture")
		if err != nil {
			t.Fatal(err)
		}
		variants := env.ProductCandidate.Variants
		if env.Identity.SourceID != "us:B012345678" || len(variants) != 2 || variants[0].Price != 19.99 || variants[1].SourceID != "B012345680" || variants[1].SKU != "" || variants[1].Images[0].URL != "https://m.media-amazon.com/images/I/beige.jpg" {
			t.Fatalf("mixed anchor or child facts: %+v", env)
		}
		if currency == nil {
			if variants[1].Currency != "" || variants[1].Price != 0 || variants[1].Attributes["source.price.amount"] != "44.98" {
				t.Fatalf("missing currency inherited anchor facts: %+v", variants[1])
			}
		} else if variants[1].Price != 44.98 || variants[1].Currency != "USD" {
			t.Fatalf("lost independent child price: %+v", variants[1])
		}
	}
}

func TestAmazonPartialFamilyFolds171UnknownChildFactsWithinExistingReadBounds(t *testing.T) {
	for _, field := range []string{"price.currency", "price", "attributes.rendered"} {
		var input map[string]any
		_ = json.Unmarshal(amazonFixture(), &input)
		e := input["evidence"].(map[string]any)
		e["observation"] = map[string]any{"scope": "partial_family", "parentASIN": "B012345679"}
		variants, missing := []any{}, []any{}
		for i := 0; i < 171; i++ {
			id := fmt.Sprintf("B%09d", i)
			if i == 0 {
				id = "B012345678"
			}
			var price any = map[string]any{"amount": "39.98", "currency": nil}
			reason := "source currency not explicit"
			if field == "price" {
				price, reason = nil, "source price not observed"
			}
			if field == "attributes.rendered" {
				price = map[string]any{"amount": "39.98", "currency": "USD", "condition": "new"}
				reason = "rendered dimension selection not observed; values supplied by current-ASIN static data"
			}
			variants = append(variants, map[string]any{"sourceID": id, "sku": nil, "title": nil,
				"attributes": []any{map[string]any{"name": "Color", "value": fmt.Sprintf("c%d", i)}}, "price": price,
				"images": []any{map[string]any{"url": "https://m.media-amazon.com/images/I/" + id + ".jpg", "role": "main"}}})
			missing = append(missing, map[string]any{"field": fmt.Sprintf("variants.%d.%s", i, field), "reason": reason})
		}
		e["variants"], e["missingFacts"] = variants, missing
		raw, _ := json.Marshal(input)
		capture, err := ParseAmazonBrowserCapture(raw)
		if err != nil {
			t.Fatalf("%s family rejected: %v", field, err)
		}
		env, err := capture.Envelope("capture")
		if err != nil {
			t.Fatal(err)
		}
		snapshot, err := ToSnapshot(env)
		if err != nil || len(snapshot.Variants) != 171 || snapshot.Review == nil || len(snapshot.Review.Reasons) > 128 {
			t.Fatalf("%s family unreadable: %v", field, err)
		}
		for _, variant := range env.ProductCandidate.Variants {
			if field == "attributes.rendered" {
				if variant.Price != 39.98 || variant.Currency != "USD" {
					t.Fatalf("changed observed quote: %+v", variant)
				}
				continue
			}
			if variant.Price != 0 || variant.Currency != "" {
				t.Fatalf("invented unknown %s: %+v", field, variant)
			}
		}
	}
}

func TestAmazonCaptureRejectsFactsOutsideExistingCollectionReadBounds(t *testing.T) {
	badWarnings := map[string]any{}
	_ = json.Unmarshal(amazonFixture(), &badWarnings)
	e := badWarnings["evidence"].(map[string]any)
	e["warnings"] = []any{map[string]any{"code": strings.Repeat("c", 129), "field": "title"}}
	raw, _ := json.Marshal(badWarnings)
	if _, err := ParseAmazonBrowserCapture(raw); err == nil {
		t.Fatal("accepted unreadable warning code")
	}
	e["warnings"] = []any{}
	missing := []any{}
	for i := 0; i < 129; i++ {
		missing = append(missing, map[string]any{"field": "source", "reason": fmt.Sprintf("missing fact %d", i)})
	}
	e["missingFacts"] = missing
	raw, _ = json.Marshal(badWarnings)
	if _, err := ParseAmazonBrowserCapture(raw); err == nil {
		t.Fatal("accepted unreadable derived review reasons")
	}
	for _, change := range [][2]string{{`"Cotton bag"`, `"` + strings.Repeat("a", 2001) + `"`}, {`"sku":null`, `"sku":"` + strings.Repeat("a", 201) + `"`}, {`https://m.media-amazon.com/images/I/blue.jpg`, `https://m.media-amazon.com/` + strings.Repeat("a", 2049)}} {
		_, err := ParseAmazonBrowserCapture([]byte(strings.Replace(string(amazonFixture()), change[0], change[1], 1)))
		if err == nil {
			t.Fatal("accepted capture that existing Collection cannot read")
		}
	}
	if _, err := ParseAmazonBrowserCapture([]byte(strings.Replace(string(amazonFixture()), "Cotton bag", `Cotton\u0000bag`, 1))); err == nil {
		t.Fatal("accepted NUL in title")
	}
}
