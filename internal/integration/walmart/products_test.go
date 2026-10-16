package walmart

import (
	"context"
	"io"
	"net/http"
	"strings"
	auth "task-processor/internal/storecenter/walmart"
	"testing"
)

func TestProductPageUsesBoundedOffsetAndMissingBodyIsNotAnEmptyShop(t *testing.T) {
	raw := `{"totalItems":11,"ItemResponse":[{"sku":"last","publishedStatus":"UNPUBLISHED"}]}`
	c, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "app", Revision: "r1", Environment: "production", CallbackURL: "https://app.example/callback"}, ClientSecret: "secret", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Query().Get("offset") != "10" || r.URL.Query().Get("limit") != "10" {
			t.Errorf("wrong offset %s", r.URL.RawQuery)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(raw)), Header: make(http.Header)}, nil
	})}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = c.Products(context.Background(), auth.TokenSet{AccessToken: "token", SellerID: "seller"}, 2); e != nil {
		t.Fatal(e)
	}
	raw = `{}`
	if _, e = c.Products(context.Background(), auth.TokenSet{AccessToken: "token", SellerID: "seller"}, 2); e == nil {
		t.Fatal("missing fields reported empty shop")
	}
}

func TestSKUAbsenceRequiresExplicitCompleteResponse(t *testing.T) {
	raw := "{}"
	c, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "app", Revision: "r1", Environment: "production", CallbackURL: "https://app.example/callback"}, ClientSecret: "secret", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(raw)), Header: make(http.Header)}, nil
	})}})
	if e != nil {
		t.Fatal(e)
	}
	for _, invalid := range []string{`{}`, `{"totalItems":0}`, `{"ItemResponse":[]}`, `{"totalItems":1,"ItemResponse":[{"sku":"other"}]}`} {
		raw = invalid
		if _, e = c.ProductBySKU(context.Background(), auth.TokenSet{AccessToken: "token", SellerID: "seller"}, "own-sku"); e == nil {
			t.Fatalf("unsafe absence: %s", raw)
		}
	}
	raw = `{"totalItems":0,"ItemResponse":[]}`
	p, e := c.ProductBySKU(context.Background(), auth.TokenSet{AccessToken: "token", SellerID: "seller"}, "own-sku")
	if e != nil || p != nil {
		t.Fatalf("explicit empty %v %v", p, e)
	}
}
