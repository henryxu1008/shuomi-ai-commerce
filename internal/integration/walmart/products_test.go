package walmart

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	marketplace "task-processor/internal/marketplace/walmart"
	auth "task-processor/internal/storecenter/walmart"
	"testing"
)

func TestSubmitFeedSeparatesDefinitiveRejectionFromUncertainResponse(t *testing.T) {
	selection := marketplace.RuleSelection{FeedType: marketplace.MatchFeed, Version: "5-synthetic"}
	rules := marketplace.RuleSnapshot{Selection: selection, Header: marketplace.FeedHeader{Locale: "en", Version: selection.Version}, Schema: []byte(`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["MPItem"],"properties":{"MPItem":{"type":"array","minItems":1,"items":{"type":"object","required":["Item"]}}}}`)}
	draft, err := marketplace.BuildDraft(marketplace.DraftInput{Selection: selection, SKUs: []marketplace.SKUInput{{SourceSKUKey: "source", Identifier: marketplace.Identifier{Type: "GTIN", ID: "12345678901234"}, Condition: "New", Price: "10", ShippingWeight: "1"}}}, rules, "merchant", "product", []string{"source"})
	require.NoError(t, err)
	for _, status := range []int{401, 403, 400, 429, 500, 200, 0} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			calls := 0
			client, err := NewOfficialClient(Options{Application: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox", CallbackURL: "https://app.example/callback"}, ClientSecret: "synthetic", MatchVersion: selection.Version, HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				if status == 0 {
					return nil, errors.New("response lost")
				}
				return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(`{}`)), Header: make(http.Header)}, nil
			})}})
			require.NoError(t, err)
			accepted, err := client.SubmitFeed(context.Background(), auth.TokenSet{AccessToken: "synthetic", SellerID: "seller"}, draft, uuid.NewString())
			expected := marketplace.ErrUnknown
			if status == 401 || status == 403 {
				expected = marketplace.ErrRejected
			}
			require.ErrorIs(t, err, expected)
			require.Empty(t, accepted.FeedID)
			require.Equal(t, 1, calls, "mutation is never retried")
		})
	}
}

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
