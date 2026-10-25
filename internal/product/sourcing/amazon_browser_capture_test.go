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
		child := map[string]any{"sourceID": "B012345680", "sku": nil, "title": nil,
			"attributes": []any{map[string]any{"name": "Color", "value": "Beige"}},
			"price":      map[string]any{"amount": "44.98", "currency": currency},
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
