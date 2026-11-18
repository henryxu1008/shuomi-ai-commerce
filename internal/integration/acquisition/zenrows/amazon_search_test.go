package zenrows

import (
	"context"
	"crypto/sha256"
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
	got := amazonSearchEvidence(body, at)
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
