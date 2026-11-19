package zenrows

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
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
	const describedPrice = `<span class="a-price" data-a-size="xl" data-a-color="base"><span class="a-offscreen">$19.95</span><span aria-hidden="true"><span class="a-price-symbol">$</span><span class="a-price-whole">19<span class="a-price-decimal">.</span></span><span class="a-price-fraction">95</span></span></span>`
	const countPrice = ` <span class="a-size-base a-color-base">(<span class="a-price a-text-price" data-a-size="b" data-a-color="base"><span class="a-offscreen">$9.98</span><span aria-hidden="true">$9.98</span></span>/count)</span>`
	const listPrice = ` <span class="a-offscreen">List: $29.95</span><div aria-hidden="List: $29.95" class="a-section aok-inline-block"><span class="a-size-base a-color-secondary">List: </span><span class="a-price a-text-price" data-a-size="b" data-a-strike="true" data-a-color="secondary"><span class="a-offscreen">$29.95</span><span aria-hidden="true">$29.95</span></span></div>`
	const promotion = `<div class="a-row a-size-base a-color-secondary"><span class="a-size-base s-highlighted-text-padding s-promotion-highlight-color aok-inline-block">Save 5%</span><span class="a-color-base"> on 2 select item(s)</span></div>`
	describedOffer := func(linkContent, after string) string {
		return `<div class="a-row a-size-base a-color-base"><div class="a-row"><span id="price-link" class="aok-offscreen">Price, product page</span><a aria-describedby="price-link">` +
			linkContent + `</a></div><div class="a-row"></div></div>` + after
	}
	for _, tc := range []struct {
		name, markup string
		price        float64
	}{
		{"described primary price", describedOffer(describedPrice, ""), 19.95},
		{"described primary with separate count and list", describedOffer(describedPrice+countPrice+listPrice, ""), 19.95},
		{"described primary with separate promotion", describedOffer(describedPrice, promotion), 19.95},
		{"membership condition in a promotion row", describedOffer(describedPrice, strings.ReplaceAll(strings.ReplaceAll(promotion, "Save 5%", "Prime exclusive"), "on 2 select item(s)", "Price shown requires Prime membership")), 0},
		{"discount with additional qualification", describedOffer(describedPrice, strings.Replace(promotion, "on 2 select item(s)", "on 2 select item(s) with Prime", 1)), 0},
		{"conditional price description", strings.Replace(describedOffer(describedPrice, ""), "Price, product page", "Price, product page with a membership", 1), 0},
		{"duplicate description target", describedOffer(describedPrice, "") + `<span id="price-link" class="aok-offscreen">Price, product page</span>`, 0},
		{"missing description target", strings.Replace(describedOffer(describedPrice, ""), `id="price-link"`, `id="other-link"`, 1), 0},
		{"hidden described link", strings.Replace(describedOffer(describedPrice, ""), `<a aria-describedby`, `<a hidden aria-describedby`, 1), 0},
		{"additional accessible qualification", strings.Replace(describedOffer(describedPrice, ""), `<a aria-describedby`, `<a aria-label="From $19.95" aria-describedby`, 1), 0},
		{"described primary without display evidence", describedOffer(offer("$19.95"), ""), 0},
		{"described primary with conflicting display", describedOffer(strings.Replace(describedPrice, "fraction\">95", "fraction\">94", 1), ""), 0},
		{"unknown condition in described link", describedOffer(describedPrice+`<span>with a membership</span>`, ""), 0},
		{"unrecognized neighboring condition", describedOffer(describedPrice, `<span>with a membership</span>`), 0},
		{"mislabeled reference amount", describedOffer(describedPrice+strings.Replace(listPrice, `data-a-strike="true"`, "", 1), ""), 0},
		{"count and list only", describedOffer(countPrice+listPrice, ""), 0},
		{"promotion only", describedOffer("", promotion), 0},
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

func TestAmazonSearchKeepsSourceCardForQuoteDiagnosis(t *testing.T) {
	const html = `<article data-component-type="s-search-result" data-asin="B000000001">
		<img class="s-image" src="https://m.media-amazon.com/images/I/first.jpg">
		<output class="new-price" aria-label="Current price">$42.15</output>
		<s style="text-decoration: line-through; background: url(secret-style)">$52.15</s>
		<span style="display: none"><b>$62.15</b></span>
		<a href="https://example.test/private-url" onclick="private-handler()">Seller</a>
		<script>private-script</script><input value="private-input">
		<div data-component-type="s-search-result" data-asin="B000000009"><span>$99.15</span></div>
	</article><div data-component-type="s-search-result" data-asin="B000000002">
		<img class="s-image" src="https://m.media-amazon.com/images/I/second.jpg"><span>See buying options</span>
	</div>`
	body, err := json.Marshal(map[string]any{
		"html": html,
		"parsed": map[string]any{"results": []map[string]any{
			{"asin": "B000000001", "title": "First"},
			{"asin": "B000000002", "title": "Second", "price": 12.15},
		}},
	})
	require.NoError(t, err)
	facts := DecodeAmazonFirstPage(body, time.Now().UTC())
	require.Len(t, facts, 2)
	for _, fact := range facts {
		require.Zero(t, fact.Price)
		require.Empty(t, fact.Currency)
	}
	type savedEvidence struct {
		QuoteCard struct {
			Status string
			Markup string
		}
	}
	raw, err := json.Marshal(facts)
	require.NoError(t, err)
	var stored []savedEvidence
	require.NoError(t, json.Unmarshal(raw, &stored))
	first := stored[0].QuoteCard
	require.Equal(t, "captured", first.Status)
	require.Contains(t, first.Markup, `data-asin="B000000001"`)
	require.Contains(t, first.Markup, `<output class="new-price" aria-label="Current price">$42.15</output>`)
	require.Contains(t, first.Markup, `text-decoration: line-through`)
	require.Contains(t, first.Markup, `display: none`)
	for _, secret := range []string{"private-url", "private-handler", "private-script", "private-input", "secret-style", "first.jpg", "$99.15", "B000000009", "B000000002"} {
		require.NotContains(t, first.Markup, secret)
	}
	second := stored[1].QuoteCard
	require.Equal(t, "captured", second.Status)
	require.Contains(t, second.Markup, "See buying options")
	require.NotContains(t, second.Markup, "$")
}

func TestAmazonFirstPageSenderRejectsKeyDecodedFromQuoteCard(t *testing.T) {
	const key = "loopback-secret&'only"
	const encodedKey = "loopback&#45;secret&amp;&#39;only"
	for _, tc := range []struct {
		name, markup string
	}{
		{"card text", "<span>" + encodedKey + "</span>"},
		{"card attribute", " <span aria-label='" + encodedKey + "'>See current price</span>"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			approval := dataacquisition.AmazonFirstPageApproval{
				ID: uuid.NewString(), ProjectID: uuid.NewString(), Keyword: "lamp",
				Scope:      collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"},
				AccountRef: uuid.NewString(), BudgetRef: uuid.NewString(), KeyDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(key))),
				BudgetExpiresAt: time.Now().Add(time.Minute), ExpiresAt: time.Now().Add(time.Minute),
				AvailableCredits: 25, MaximumCredits: 25, NoOverageCharges: true, Funding: orgresource.FundingEnterprise,
				Price: dataacquisition.PriceConfirmation{Version: "test"},
			}
			const image = `<img class="s-image" src="https://m.media-amazon.com/images/I/lamp.jpg">`
			html := `<div data-component-type="s-search-result" data-asin="B000000001">` + image + tc.markup + `</div>` +
				`<div data-component-type="s-search-result" data-asin="B000000002">` + image + `</div>`
			body, err := json.Marshal(map[string]any{
				"html": html,
				"parsed": map[string]any{"results": []map[string]any{
					{"asin": "B000000001", "title": "First"}, {"asin": "B000000002", "title": "Second"},
				}},
			})
			require.NoError(t, err)
			require.NotContains(t, string(body), key)
			require.Len(t, DecodeAmazonFirstPage(body, time.Now().UTC()), 2)
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("X-Request-Id", "vendor-receipt")
				w.Header().Set("X-Request-Credits", "25")
				w.Header().Set("Zr-Final-Url", approval.TargetURL())
				_, _ = w.Write(body)
			}))
			t.Cleanup(server.Close)
			send := amazonFirstPageFetch(server.Client(), server.URL, func(context.Context) (string, error) { return key, nil })
			got := send(context.Background(), approval)
			require.Equal(t, dataacquisition.AmazonFirstPageUnknown, got.Receipt.State)
			require.Equal(t, "vendor-receipt", got.Receipt.RequestID)
			require.Empty(t, got.Evidence)
			require.EqualValues(t, 1, requests.Load())
		})
	}
}
