package storecenter

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCatalogTemuClassificationDoesNotInventOrderCompleteness(t *testing.T) {
	signal := CatalogSignals{Connection: ConnectionStatusConnected, SyncHealthy: true, ProductSync: "completed", Products: CatalogMetric{Complete: true}, Inventory: CatalogMetric{Reason: "threshold_not_set"}, Orders: CatalogMetric{Reason: "observations_unavailable"}}
	require.Equal(t, "normal", classifySignals(PlatformTemu, signal))
	require.False(t, signal.Orders.Complete, "unsupported orders remain unknown")
	require.Equal(t, "unknown", classifySignals(PlatformShein, signal))
	require.Equal(t, "unknown", classifySignals("unsupported", signal))
	signal.Orders.Complete = true
	require.Equal(t, "normal", classifySignals(PlatformShein, signal))
	signal.Orders.Complete = false
	signal.OrderSync = "failed"
	require.Equal(t, "normal", classifySignals(PlatformTemu, signal), "unsupported orders do not affect TEMU health")
	require.Equal(t, "attention", classifySignals(PlatformShein, signal))
	signal.OrderSync = ""
	signal.Products.Complete = false
	require.Equal(t, "unknown", classifySignals(PlatformTemu, signal))
	signal.Products.Complete = true
	signal.SyncHealthy = false
	require.Equal(t, "unknown", classifySignals(PlatformTemu, signal))
	signal.ProductSync = "failed"
	require.Equal(t, "attention", classifySignals(PlatformTemu, signal))
}
