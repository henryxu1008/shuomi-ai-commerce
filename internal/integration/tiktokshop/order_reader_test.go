package tiktokshop

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestOrdersSignedReadOnlySearchUsesOriginalHalfOpenWindow(t *testing.T) {
	calls := 0
	client := newFixtureClient(t, func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "POST", r.Method)
		require.Equal(t, apiOrigin+r.URL.Path, r.URL.Scheme+"://"+r.URL.Host+r.URL.Path)
		require.Equal(t, "/order/202309/orders/search", r.URL.Path)
		q := r.URL.Query()
		require.Equal(t, "100", q.Get("page_size"))
		require.Equal(t, "ASC", q.Get("sort_order"))
		require.Equal(t, "create_time", q.Get("sort_field"))
		require.Equal(t, "fresh-cipher", q.Get("shop_cipher"))
		require.Equal(t, "a?&opaque", q.Get("page_token"))
		require.False(t, q.Has("access_token"))
		require.Equal(t, "synthetic-access", r.Header.Get("x-tts-access-token"))
		body, e := io.ReadAll(r.Body)
		require.NoError(t, e)
		var input map[string]int64
		require.NoError(t, json.Unmarshal(body, &input))
		require.Equal(t, map[string]int64{"create_time_ge": 1793481600, "create_time_lt": 1793654400}, input)
		expected, e := signature(r.URL.Path, q, body, "application/json", clientSecretForOrderTest)
		require.NoError(t, e)
		require.Equal(t, expected, q.Get("sign"))
		return jsonResponse(`{"code":0,"data":{"orders":[],"total_count":0}}`), nil
	})
	_, e := client.Orders(context.Background(), "synthetic-access", "fresh-cipher", o.TikTokOrderQuery{Start: time.Unix(1793481600, 0), End: time.Unix(1793654400, 0), Token: "a?&opaque"})
	require.NoError(t, e)
	require.Equal(t, 1, calls)
	for _, bad := range []o.TikTokOrderQuery{{Start: time.Now(), End: time.Now().Add(time.Hour), Token: "bad\t"}, {Start: time.Now(), End: time.Now().Add(time.Hour), Token: strings.Repeat("a", 8193)}, {Start: time.Now(), End: time.Now()}} {
		_, e = client.Orders(context.Background(), "synthetic-access", "fresh-cipher", bad)
		require.Error(t, e)
	}
	require.Equal(t, 1, calls)
}

const clientSecretForOrderTest = "fixture-secret-never-log"

func TestOrdersBodyLimitedBeforeDecodeAndNeverRetries(t *testing.T) {
	body := &orderReadCounter{Reader: strings.NewReader(strings.Repeat(" ", 2<<20) + "{}")}
	calls := 0
	client := newFixtureClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: body}, nil
	})
	_, e := client.Orders(context.Background(), "synthetic", "cipher", o.TikTokOrderQuery{Start: time.Now().Add(-time.Hour).Truncate(time.Second), End: time.Now().Truncate(time.Second)})
	require.Error(t, e)
	require.Equal(t, (2<<20)+1, body.read)
	require.Equal(t, 1, calls)
}

type orderReadCounter struct {
	io.Reader
	read int
}

func (r *orderReadCounter) Read(p []byte) (int, error) {
	n, e := r.Reader.Read(p)
	r.read += n
	return n, e
}
func (*orderReadCounter) Close() error { return nil }
