package dataacquisition

import "testing"

func TestZenRowsSearchEvidenceKeepsPriceAndCurrencyPaired(t *testing.T) {
	e := Evidence{
		Site: "us", ASIN: "B000123456", Title: "Lamp", MainImage: "https://m.media-amazon.com/images/lamp.jpg",
		Availability: "unknown", CapturedAt: "2026-11-18T12:49:30Z",
	}
	for _, tc := range []struct {
		name, version, currency string
		price                   float64
		missing                 []string
		valid                   bool
	}{
		{"saved v1", amazonZenRowsSearchV1, "", 0, []string{"price", "currency"}, true},
		{"v1 cannot claim a price", amazonZenRowsSearchV1, "USD", 1, nil, false},
		{"v2 missing", AmazonZenRowsSearchVersion, "", 0, []string{"price", "currency"}, true},
		{"v2 verified pair", AmazonZenRowsSearchVersion, "USD", 1, nil, true},
		{"price only", AmazonZenRowsSearchVersion, "", 1, []string{"currency"}, false},
		{"currency only", AmazonZenRowsSearchVersion, "USD", 0, []string{"price"}, false},
		{"other currency", AmazonZenRowsSearchVersion, "CAD", 1, nil, false},
		{"conflicting price missing", AmazonZenRowsSearchVersion, "USD", 1, []string{"price"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fact := e
			fact.ParserVersion, fact.Price, fact.Currency = tc.version, tc.price, tc.currency
			fact.Missing = append([]string{"availability"}, tc.missing...)
			if err := fact.Validate(); (err == nil) != tc.valid {
				t.Fatalf("expected valid=%v, got %v", tc.valid, err)
			}
		})
	}
}

func TestQueryRejectsUnknownSiteAndCrossSiteURL(t *testing.T) {
	for _, q := range []Query{
		{Site: "zz", Mode: "asin", ASINs: []string{"B000123456"}, Limit: 1},
		{Site: "us", Mode: "asin", ASINs: []string{"https://www.amazon.co.jp/dp/B000123456"}, Limit: 1},
		{Site: "us", Mode: "asin", ASINs: []string{"https://www.amazon.com.evil.test/dp/B000123456"}, Limit: 1},
		{Site: "us", Mode: "keyword", Keyword: "chair", Limit: 201},
		{Site: "us", Mode: "category", CategoryNode: "12&url=http://127.0.0.1", Limit: 1},
		{Site: "us", Mode: "asin", ASINs: []string{"B000123456"}, Limit: 1, Fields: []string{"cookie"}},
	} {
		if _, err := NormalizeQuery(q); err == nil {
			t.Fatalf("accepted unsafe query: %#v", q)
		}
	}
}

func TestQuerySupportsAllInputsAndDeduplicates(t *testing.T) {
	for _, site := range Sites() {
		q, err := NormalizeQuery(Query{Site: site.Code, Mode: "asin", ASINs: []string{"b000123456", "B000123456"}, Limit: 2})
		if err != nil || len(q.ASINs) != 1 {
			t.Fatalf("site %s: %#v %v", site.Code, q, err)
		}
		if _, err := NormalizeQuery(Query{Site: site.Code, Mode: "keyword", Keyword: "desk chair", CategoryNode: "123456", Limit: 10}); err != nil {
			t.Fatal(err)
		}
		if _, err := NormalizeQuery(Query{Site: site.Code, Mode: "category", CategoryNode: "123456", Limit: 10}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEvidenceDoesNotTreatUnknownAvailabilityAsUnavailable(t *testing.T) {
	e := Evidence{Site: "us", ASIN: "B000123456", Title: "Chair", MainImage: "https://m.media-amazon.com/images/a.jpg", CapturedAt: "2026-10-09T12:00:00Z", ParserVersion: "amazon-v1"}
	if e.Validate() == nil {
		t.Fatal("unknown availability was accepted")
	}
	e.Availability = "available"
	if e.Validate() == nil {
		t.Fatal("available without price accepted")
	}
	e.Availability = "unavailable"
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	e.Price = -1
	if e.Validate() == nil {
		t.Fatal("negative source price accepted")
	}
	e.Price, e.Currency = 1, "123"
	if e.Validate() == nil {
		t.Fatal("numeric currency accepted")
	}
}

func TestEnvelopePreservesMissingFields(t *testing.T) {
	e := Evidence{Site: "us", ASIN: "B000123456", Title: "Chair", MainImage: "https://m.media-amazon.com/images/a.jpg", CapturedAt: "2026-10-09T12:00:00Z", ParserVersion: "amazon-v1", Availability: "unavailable", Missing: []string{"price", "currency"}}
	envelope, err := e.Envelope("0511e1d2-b555-4970-b323-3b629a901901")
	if err != nil || len(envelope.Warnings) != 2 {
		t.Fatalf("missing field reasons lost: %#v %v", envelope.Warnings, err)
	}
}

func TestUnknownAvailabilityIsSavedAsMissingWithoutFabricatingSalePrice(t *testing.T) {
	e := Evidence{Site: "us", ASIN: "B000123456", Title: "Chair", MainImage: "https://m.media-amazon.com/images/a.jpg", CapturedAt: "2026-10-09T12:00:00Z", ParserVersion: "amazon-v1", Availability: "unknown", Missing: []string{"availability", "price", "currency"}}
	if err := e.Validate(); err != nil {
		t.Fatalf("valid incomplete evidence rejected: %v", err)
	}
	envelope, err := e.Envelope("0511e1d2-b555-4970-b323-3b629a901901")
	if err != nil {
		t.Fatal(err)
	}
	if len(envelope.ProductCandidate.Variants) != 0 || envelope.ProductCandidate.Attributes["availability"] != "unknown" || len(envelope.Warnings) != 3 {
		t.Fatalf("unknown state or missing reasons changed: %#v", envelope)
	}
	e.Missing = nil
	if e.Validate() == nil {
		t.Fatal("unknown availability must carry its missing reason")
	}
	e.Availability = "available"
	if e.Validate() == nil {
		t.Fatal("explicitly available without real price must still fail")
	}
}
