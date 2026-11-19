package bravesearch

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestOnlyOneBraveWebPageAndUsableLinks(t *testing.T) {
	calls := 0
	client := New(testTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "GET", r.Method)
		require.Equal(t, "api.search.brave.com", r.URL.Host)
		require.Equal(t, "/res/v1/web/search", r.URL.Path)
		require.Equal(t, "fixture-key", r.Header.Get("X-Subscription-Token"))
		require.Equal(t, "public rule", r.URL.Query().Get("q"))
		require.Equal(t, "5", r.URL.Query().Get("count"))
		require.Empty(t, r.URL.Query().Get("offset"))
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(
			"{" + "\"web\":{\"results\":[{\"title\":\"Official\",\"url\":\"https://example.org/rules\",\"description\":\"A rule\"},{\"title\":\"Unsafe\",\"url\":\"javascript:alert(1)\"}]}}"))}, nil
	}))
	result, err := client.Search(context.Background(), "fixture-key", "public rule", "US", "en")
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Len(t, result, 1)
	require.Equal(t, "https://example.org/rules", result[0].URL)
}

func TestPublicLinksRejectBrowserNormalizedPrivateAddresses(t *testing.T) {
	for _, value := range []string{
		"http://127.0.0.1", "http://0177.0.0.1", "https://127.0.0.0x1", "http://[::ffff:127.0.0.1]/", "http://[fe80::3%25example]/",
		"https://router.local", "https://foo.localhost", "https://who@example.org",
	} {
		require.False(t, publicLink(value), value)
	}
	require.True(t, publicLink("https://example.org/article"))
}
