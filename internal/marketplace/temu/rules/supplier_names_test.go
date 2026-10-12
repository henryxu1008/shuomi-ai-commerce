package rules

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestSupplierNamesUseExactRootCategoryAndWeightedOfficialLimit(t *testing.T) {
	require.True(t, SupplierNameAllowed(strings.Repeat("A", 320), 999))
	require.False(t, SupplierNameAllowed(strings.Repeat("A", 251), 653))
	require.True(t, SupplierNameAllowed(strings.Repeat("货", 125), 653))
	require.False(t, SupplierNameAllowed(strings.Repeat("货", 126), 653))
	require.False(t, SupplierNameAllowed(strings.Repeat("A", 101), 1))
	require.True(t, SupplierNameAllowed(strings.Repeat("A", 350), 26205))
	require.False(t, SupplierNameAllowed("Explicit", 0))
	source, err := SupplierNameSource()
	require.NoError(t, err)
	require.Equal(t, 20, source.IncludedRows)
	require.Len(t, source.ContentHash, 64)
}
