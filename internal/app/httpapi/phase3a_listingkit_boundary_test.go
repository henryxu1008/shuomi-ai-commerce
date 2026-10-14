package httpapi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListingKitTemporalWorkerEntrypointFileStaysRetired(t *testing.T) {
	t.Parallel()

	_, err := os.Stat("listingkit_temporal_worker.go")
	require.True(t, os.IsNotExist(err), "listingkit_temporal_worker.go should stay retired; Temporal runtime ownership belongs in internal/listingkit/httpapi")
}
