package agent

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const selectionBindingWire = `{"ContextKind":"product-selection-v1","ContextID":"550e8400-e29b-41d4-a716-446655440000","ProductKey":"","CatalogVersion":"","PublicationID":"","TargetPlatform":"product","Selection":[{"ItemID":"550e8400-e29b-41d4-a716-446655440001","ItemRevision":"1","ProductKey":"product-a","CatalogVersion":"2","PublicationID":"publication-a"},{"ItemID":"550e8400-e29b-41d4-a716-446655440002","ItemRevision":"3","ProductKey":"product-b","CatalogVersion":"1","PublicationID":"publication-b"}]}`

func TestSelectionBindingRequiresEveryExactCandidate(t *testing.T) {
	var binding Binding
	require.NoError(t, json.Unmarshal([]byte(selectionBindingWire), &binding))
	require.True(t, binding.Valid(), "explicit multi-product binding must be valid without inventing a single Product")
	for _, mutation := range []struct{ from, to string }{
		{`"ContextKind":"product-selection-v1"`, `"ContextKind":"acquisition"`},
		{`"ProductKey":""`, `"ProductKey":"first-product"`},
		{`"ItemRevision":"3"`, `"ItemRevision":"03"`},
		{`"CatalogVersion":"2"`, `"CatalogVersion":"0"`},
		{`"TargetPlatform":"product"`, `"TargetPlatform":"shein"`},
		{`440002`, `440001`},
		{`"product-b"`, `"product-a"`},
		{`"PublicationID":"publication-b"`, `"PublicationID":""`},
	} {
		var changed Binding
		require.NoError(t, json.Unmarshal([]byte(strings.Replace(selectionBindingWire, mutation.from, mutation.to, 1)), &changed))
		require.False(t, changed.Valid(), "mutation: %s", mutation.from)
	}
	changed := strings.Replace(selectionBindingWire, `"product-b"`, `"product-c"`, 1)
	var other Binding
	require.NoError(t, json.Unmarshal([]byte(changed), &other))
	require.True(t, other.Valid())
	a, _ := json.Marshal(binding)
	b, _ := json.Marshal(other)
	require.NotEqual(t, sha256.Sum256(a), sha256.Sum256(b), "all candidate identities enter the original fingerprint")
}

func TestEmptySelectionPreservesOriginalSingleProductBindingBytes(t *testing.T) {
	const original = `{"ContextKind":"acquisition","ContextID":"original-context","ProductKey":"original-product","CatalogVersion":"1","PublicationID":"original-publication","TargetPlatform":"shein"}`
	var binding Binding
	require.NoError(t, json.Unmarshal([]byte(original), &binding))
	require.True(t, binding.Valid())
	wire, err := json.Marshal(binding)
	require.NoError(t, err)
	require.Equal(t, original, string(wire))
	require.Equal(t, sha256.Sum256([]byte(original)), sha256.Sum256(wire))
}
