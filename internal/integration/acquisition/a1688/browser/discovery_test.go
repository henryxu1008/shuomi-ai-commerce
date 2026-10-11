package browser

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"task-processor/internal/product/sourcing"
	"testing"
	"time"
)

func TestDiscoveryUsesAcquisitionSlots(t *testing.T) {
	c := New(Options{ExecutablePath: "test", MaxConcurrent: 1})
	// Validation must run first; use the actual test executable as a harmless valid path.
	c.opts.ExecutablePath, _ = os.Executable()
	c.slots <- struct{}{}
	input, e := sourcing.Canonical1688Discovery("keyword", "水杯")
	require.NoError(t, e)
	_, e = c.Discover(context.Background(), input)
	require.ErrorIs(t, e, ErrCapacity)
}

func TestDiscoveryReadsRenderedCurrentPageBelowViewport(t *testing.T) {
	page := `<!doctype html><html><body>
<a href="https://detail.1688.com/offer/123.html">first</a>
<div style="height:3000px"></div>
<a href="https://detail.1688.com/offer/456.html">below viewport</a>
<a style="display:none" href="https://detail.1688.com/offer/789.html">hidden</a>
<a style="visibility:hidden" href="https://detail.1688.com/offer/987.html">hidden</a>
</body></html>`
	srv := serveFixture(t, page)
	c := newTestClient(Options{ExecutablePath: fixtureBrowserPath(t), Headless: true, AllowedOrigins: []string{srv.URL}})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	raw, err := c.collectPage(ctx, srv.URL, false, nil, discoveryScript)
	require.NoError(t, err)
	input, err := sourcing.Canonical1688Discovery("keyword", "fixture")
	require.NoError(t, err)
	result, err := decodeDiscovery(input, raw)
	require.NoError(t, err)
	require.Equal(t, []string{"https://detail.1688.com/offer/123.html", "https://detail.1688.com/offer/456.html"}, result.Products)
	require.False(t, result.Truncated)
}

func TestDiscoveryReportsLimitOnRenderedLinksBelowViewport(t *testing.T) {
	var page strings.Builder
	page.WriteString(`<!doctype html><html><body><div style="height:3000px"></div>`)
	for i := 1; i <= 201; i++ {
		fmt.Fprintf(&page, `<a href="https://detail.1688.com/offer/%d.html">product</a><br>`, i)
	}
	page.WriteString(`</body></html>`)
	srv := serveFixture(t, page.String())
	c := newTestClient(Options{ExecutablePath: fixtureBrowserPath(t), Headless: true, AllowedOrigins: []string{srv.URL}})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	raw, err := c.collectPage(ctx, srv.URL, false, nil, discoveryScript)
	require.NoError(t, err)
	input, err := sourcing.Canonical1688Discovery("keyword", "fixture")
	require.NoError(t, err)
	result, err := decodeDiscovery(input, raw)
	require.NoError(t, err)
	require.Len(t, result.Products, 200)
	require.True(t, result.Truncated)
}
func TestDiscoveryDecodesOnlyCanonicalVisibleLinks(t *testing.T) {
	input, e := sourcing.Canonical1688Discovery("keyword", "水杯")
	require.NoError(t, e)
	_, e = decodeDiscovery(input, map[string]any{"products": []any{"https://evil.test/offer/123.html"}, "truncated": false})
	require.Error(t, e)
	result, e := decodeDiscovery(input, map[string]any{"products": []any{"https://detail.1688.com/offer/123.html"}, "truncated": true})
	require.NoError(t, e)
	require.True(t, result.Truncated)
	_, e = decodeDiscovery(input, map[string]any{"products": []any{}, "truncated": false})
	require.ErrorIs(t, e, ErrUnsupported)
}
