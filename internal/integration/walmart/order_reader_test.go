package walmart

import (
	"context"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	o "task-processor/internal/storecenter/observations"
	auth "task-processor/internal/storecenter/walmart"
	"testing"
	"time"
)

func TestOrdersFixedQueryAndCursorBoundaries(t *testing.T) {
	calls := 0
	c, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox", CallbackURL: "https://example.invalid/callback"}, ClientSecret: "offline", HTTPClient: &http.Client{Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "sandbox.walmartapis.com", r.URL.Host)
		require.Equal(t, "/v3/orders", r.URL.Path)
		require.Equal(t, "GET", r.Method)
		q := r.URL.Query()
		require.Equal(t, "30", q.Get("limit"))
		require.Equal(t, "WFSFulfilled", q.Get("shipNodeType"))
		require.Equal(t, "true", q.Get("replacementInfo"))
		require.Equal(t, "2026-11-01T00:00:00Z", q.Get("createdStartDate"))
		require.Equal(t, "2026-11-02T00:00:00Z", q.Get("createdEndDate"))
		require.Empty(t, q.Get("status"))
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"list":{"meta":{"totalCount":0},"elements":{"order":[]}}}`))}, nil
	})}})
	require.NoError(t, e)
	q := o.WalmartOrderQuery{Start: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), End: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), Partition: 1}
	token := auth.TokenSet{SellerID: "seller", AccessToken: "synthetic"}
	_, e = c.Orders(context.Background(), token, q)
	require.NoError(t, e)
	q.Cursor = "?limit=30&hasMoreElements=true&poIndex=10&soIndex=31&sellerId=seller"
	_, e = c.Orders(context.Background(), token, q)
	require.NoError(t, e)
	for _, bad := range []string{"https://evil.invalid/?poIndex=2", "//evil.invalid/v3/orders", "?status=Created", "?limit=200", "?sellerId=other", "?poIndex=1&poIndex=2", "?createdStartDate=2020-01-01", "?poIndex=%0A"} {
		q.Cursor = bad
		_, e = c.Orders(context.Background(), token, q)
		require.Error(t, e, bad)
	}
	require.Equal(t, 2, calls)
}

func TestOrdersLimitBodyBeforeDecoding(t *testing.T) {
	body := &orderBodyCounter{Reader: strings.NewReader(strings.Repeat(" ", 2<<20) + "{}")}
	c, e := NewOfficialClient(Options{Application: auth.Application{ClientID: "app", Revision: "r1", Environment: "sandbox", CallbackURL: "https://example.invalid/callback"}, ClientSecret: "offline", HTTPClient: &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
	})}})
	require.NoError(t, e)
	_, e = c.Orders(context.Background(), auth.TokenSet{SellerID: "seller", AccessToken: "synthetic"}, o.WalmartOrderQuery{Start: time.Now().Add(-time.Hour), End: time.Now()})
	require.Error(t, e)
	require.Equal(t, (2<<20)+1, body.read)
}

type orderBodyCounter struct {
	io.Reader
	read int
}

func (b *orderBodyCounter) Read(p []byte) (int, error) {
	n, e := b.Reader.Read(p)
	b.read += n
	return n, e
}
func (*orderBodyCounter) Close() error { return nil }
