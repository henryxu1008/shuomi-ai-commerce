package zenrows

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

func TestAmazonSearchUsesSamePageEvidence(t *testing.T) {
	at := time.Date(2026, 11, 18, 5, 0, 0, 0, time.UTC)
	body := []byte(`{"html":"<div class='s-main-slot'><div data-component-type='s-search-result' data-asin='B000000001'><img class='s-image' src='https://m.media-amazon.com/images/I/sample.jpg'></div><div data-component-type='s-search-result' data-asin='B000000002'><img class='s-image' src='https://example.org/bad.jpg'></div></div>","parsed":{"results":[{"asin":"B000000001","title":"Lamp","price":11.99},{"asin":"B000000001","title":"Duplicate"},{"asin":"B000000002","title":"Untrusted image"},{"asin":"B000000003","title":"No original card"}],"pagination":{"next_page_url":"/s?k=lamp&page=2"}}}`)
	got := DecodeAmazonFirstPage(body, at)
	require.Len(t, got, 1)
	require.Equal(t, "B000000001", got[0].ASIN)
	require.Equal(t, "Lamp", got[0].Title)
	require.Equal(t, "unknown", got[0].Availability)
	require.Empty(t, got[0].Currency)
	require.Zero(t, got[0].Price)
	require.Contains(t, got[0].Missing, "price")
	require.Equal(t, dataacquisition.AmazonZenRowsSearchVersion, got[0].ParserVersion)
	require.NoError(t, got[0].Validate())
}

func TestAmazonSearchPreservesOnlyUnambiguousSameCardUSPrices(t *testing.T) {
	const image = `<img class="s-image" src="https://m.media-amazon.com/images/I/lamp.jpg">`
	offer := func(text string) string {
		return `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">` + text + `</span></span>`
	}
	card := func(id, prices string) string {
		return fmt.Sprintf(`<div data-component-type="s-search-result" data-asin="%s">%s<div data-cy="price-recipe">%s</div></div>`, id, image, prices)
	}
	for _, tc := range []struct {
		name, markup string
		price        float64
	}{
		{"current exact price", offer("$19.95") + `<span class="a-price a-text-price"><span class="a-offscreen">$29.95</span></span>`, 19.95},
		{"grouped amount", offer("US$1,299.00"), 1299},
		{"currency code", offer("USD 19.95"), 19.95},
		{"whole amount", offer("$20"), 20},
		{"none despite structured amount", "", 0},
		{"unknown currency", offer("19.95"), 0},
		{"foreign currency", offer("CA$19.95"), 0},
		{"range", offer("$19.95 - $29.95"), 0},
		{"multiple main offers", offer("$19.95") + offer("$29.95"), 0},
		{"conditional", offer("From $19.95"), 0},
		{"adjacent lower bound", "<span>From</span>" + offer("$19.95"), 0},
		{"adjacent condition", offer("$19.95") + "<span>with a membership</span>", 0},
		{"adjacent foreign currency", "<span>CAD</span>" + offer("$19.95"), 0},
		{"adjacent second amount", offer("$19.95") + "<span>– $29.95</span>", 0},
		{"conditional accessible label", "<div aria-label=\"From $19.95\">" + offer("$19.95") + "</div>", 0},
		{"split visual copy", `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$19.95</span><span aria-hidden="true"><span class="a-price-symbol">$</span><span class="a-price-whole">19<span class="a-price-decimal">.</span></span><span class="a-price-fraction">95</span></span></span>`, 19.95},
		{"conflicting visual copy", `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$19.95</span><span aria-hidden="true">$29.95</span></span>`, 0},
		{"struck-out visual copy", `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$19.95</span><span aria-hidden="true"><del>$19.95</del></span></span>`, 0},
		{"hidden visual copy", `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$19.95</span><span aria-hidden="true" style="display: none">$19.95</span></span>`, 0},
		{"unit price", `<span class="a-price" data-a-size="mini"><span class="a-offscreen">$0.25</span></span>`, 0},
		{"list price only", `<span class="a-price a-text-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$29.95</span></span>`, 0},
		{"struck-out price attribute", `<span class="a-price" data-a-size="xl" data-a-color="base" data-a-strike="true"><span class="a-offscreen">$29.95</span></span>`, 0},
		{"struck-out ancestor", "<del>" + offer("$29.95") + "</del>", 0},
		{"hidden offer", "<span hidden>" + offer("$19.95") + "</span>", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			html := `<div class="s-main-slot">` + card("B000000001", tc.markup) + card("B000000002", offer("$39.99")) + "</div>"
			body, err := json.Marshal(map[string]any{
				"html": html,
				"parsed": map[string]any{"results": []map[string]any{
					{"asin": "B000000001", "title": "First", "price": 19.95},
					{"asin": "B000000002", "title": "Second"},
				}},
			})
			require.NoError(t, err)
			got := DecodeAmazonFirstPage(body, time.Now().UTC())
			require.Len(t, got, 2)
			require.Equal(t, "B000000001", got[0].ASIN)
			require.Equal(t, tc.price, got[0].Price)
			require.Equal(t, "unknown", got[0].Availability)
			if tc.price > 0 {
				require.Equal(t, "USD", got[0].Currency)
				require.NotContains(t, got[0].Missing, "price")
				require.NotContains(t, got[0].Missing, "currency")
			} else {
				require.Empty(t, got[0].Currency)
				require.Contains(t, got[0].Missing, "price")
				require.Contains(t, got[0].Missing, "currency")
			}
		})
	}
}

func TestAmazonSearchRejectsQualifierOnPriceAreaAncestor(t *testing.T) {
	body := []byte(`{"html":"<div data-component-type='s-search-result' data-asin='B000000001'><img class='s-image' src='https://m.media-amazon.com/images/I/lamp.jpg'><div aria-label='From $19.95'><div data-cy='price-recipe'><span class='a-price' data-a-size='xl' data-a-color='base'><span class='a-offscreen'>$19.95</span></span></div></div></div>","parsed":{"results":[{"asin":"B000000001","title":"Lamp"}]}}`)
	got := DecodeAmazonFirstPage(body, time.Now().UTC())
	require.Len(t, got, 1)
	require.Zero(t, got[0].Price)
	require.Empty(t, got[0].Currency)
}

func TestAmazonFirstPageSenderUsesFixedRequestAndNeverRedirects(t *testing.T) {
	const key = "loopback-test-only"
	var calls atomic.Int32
	approval := dataacquisition.AmazonFirstPageApproval{
		ID: uuid.NewString(), ProjectID: uuid.NewString(), Keyword: "camp light + stand",
		Scope:      collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"},
		AccountRef: uuid.NewString(), BudgetRef: uuid.NewString(), KeyDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(key))),
		BudgetExpiresAt: time.Now().Add(time.Minute), ExpiresAt: time.Now().Add(time.Minute),
		AvailableCredits: 25, MaximumCredits: 25, NoOverageCharges: true, Funding: orgresource.FundingEnterprise,
		Price: dataacquisition.PriceConfirmation{Version: "test"},
	}
	test := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := calls.Add(1)
		query := r.URL.Query()
		if r.URL.Path != "/v1/" || r.Method != http.MethodGet || r.ProtoMajor != 1 || !r.Close ||
			len(query) != 6 || query.Get("apikey") != key || query.Get("url") != approval.TargetURL() ||
			query.Get("extract") != "auto" || query.Get("proxy_country") != "us" ||
			query.Get("js_render") != "true" || query.Get("premium_proxy") != "true" {
			t.Error("unexpected Fetch request")
			http.Error(w, "invalid request", 400)
			return
		}
		w.Header().Set("X-Request-Id", "vendor-receipt")
		w.Header().Set("Zr-Final-Url", approval.TargetURL())
		if call == 1 {
			http.Redirect(w, r, "/second-send", http.StatusTemporaryRedirect)
			return
		}
		w.Header().Set("X-Request-Credits", "26")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(test.Close)
	client := oneRequestClient(time.Second)
	client.Transport.(*http.Transport).TLSClientConfig = test.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	send := amazonFirstPageFetch(client, test.URL+"/v1/", func(context.Context) (string, error) { return key, nil })
	require.Equal(t, dataacquisition.AmazonFirstPageRejected, send(context.Background(), approval).Receipt.State)
	require.EqualValues(t, 1, calls.Load())
	receipt := send(context.Background(), approval).Receipt
	require.Equal(t, dataacquisition.AmazonFirstPageOverLimit, receipt.State)
	require.NotNil(t, receipt.Credits)
	require.Equal(t, 26, *receipt.Credits)
	require.EqualValues(t, 2, calls.Load())
	approval.MaximumCredits = 24
	require.Equal(t, dataacquisition.AmazonFirstPageUnknown, send(context.Background(), approval).Receipt.State)
	require.EqualValues(t, 2, calls.Load())
}
