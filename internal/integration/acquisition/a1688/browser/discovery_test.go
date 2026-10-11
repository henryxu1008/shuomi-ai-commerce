package browser

import (
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"task-processor/internal/product/sourcing"
	"testing"
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
