package amazon

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	o "github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter/observations"
	"github.com/stretchr/testify/require"
)

type orderFixtureTransport func(*http.Request) (*http.Response, error)

func (f orderFixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

const amazonTestOrder = `{"orderId":"111-1111111-1111111","createdTime":"2026-11-22T12:00:00Z","lastUpdatedTime":"2026-11-22T13:00:00Z","salesChannel":{"marketplaceId":"ATVPDKIKX0DER"},"orderItems":[{"orderItemId":"12345678","quantityOrdered":2,"product":{"title":"Fixture cup","sellerSku":"US-CUP","customization":{"customizedUrl":"ignored"}}}]}`

func TestIsolatedAmazonOrderRequestAndProjection(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	query := o.AmazonOrderQuery{Start: start, End: start.Add(29 * 24 * time.Hour), Cursor: "original-cursor"}
	requests := 0
	reader, err := NewSyntheticOrderReader(orderFixtureTransport(func(req *http.Request) (*http.Response, error) {
		requests++
		require.Equal(t, http.MethodGet, req.Method)
		require.Equal(t, spAPIOrigin+amazonOrdersPath, req.URL.Scheme+"://"+req.URL.Host+req.URL.Path)
		require.Equal(t, "AAAAfixture", req.Header.Get("x-amz-access-token"))
		require.Equal(t, USMarketplace, req.URL.Query().Get("marketplaceIds"))
		require.Equal(t, start.Format(time.RFC3339), req.URL.Query().Get("createdAfter"))
		require.Equal(t, query.End.Format(time.RFC3339), req.URL.Query().Get("createdBefore"))
		require.Equal(t, query.Cursor, req.URL.Query().Get("paginationToken"))
		require.False(t, req.URL.Query().Has("includedData"))
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"orders":[` + amazonTestOrder + `],"pagination":{"nextToken":"next-cursor"}}`))}, nil
	}))
	require.NoError(t, err)
	page, err := reader.Orders(context.Background(), "AAAAfixture", query)
	require.NoError(t, err)
	require.Equal(t, 1, requests)
	require.False(t, page.Incomplete)
	require.Equal(t, "next-cursor", page.Next)
	require.Len(t, page.Orders, 1)
	require.Empty(t, page.Orders[0].Status)
	require.Equal(t, "US-CUP", page.Orders[0].Items[0].SKU)
	require.Equal(t, 2, page.Orders[0].Items[0].Quantity)
}

func TestIsolatedAmazonOrderMalformedIsNotEmpty(t *testing.T) {
	start := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	query := o.AmazonOrderQuery{Start: start, End: start.Add(29 * 24 * time.Hour)}
	for _, tc := range []struct {
		name, response      string
		incomplete, failure bool
	}{
		{"empty", `{"orders":[]}`, false, false},
		{"missing", `{}`, false, true},
		{"null", `{"orders":null}`, false, true},
		{"broken pagination", `{"orders":[],"pagination":{}}`, true, false},
		{"wrong marketplace", `{"orders":[` + strings.Replace(amazonTestOrder, USMarketplace, "A2EUQ1WTGCTBG2", 1) + `]}`, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := NewSyntheticOrderReader(orderFixtureTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(tc.response))}, nil
			}))
			require.NoError(t, err)
			page, err := reader.Orders(context.Background(), "AAAAfixture", query)
			if tc.failure {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.incomplete, page.Incomplete)
			require.Empty(t, page.Orders)
		})
	}
}
