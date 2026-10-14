package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreDiagnosisBindingHasAnExactNonProductIdentity(t *testing.T) {
	wire := `{"ContextKind":"shop-operations-diagnosis-v1","ContextID":"11111111-1111-4111-8111-111111111111","TargetPlatform":"business","StoreDiagnosis":{"StoreID":"22222222-2222-4222-8222-222222222222","StartDate":"2026-10-01","EndDate":"2026-10-07","EvidenceDigest":"` + strings.Repeat("a", 64) + `"}}`
	var binding Binding
	require.NoError(t, json.Unmarshal([]byte(wire), &binding))
	require.True(t, binding.Valid(), "do not invent a Product identity for shop facts")
	for _, change := range []struct{ from, to string }{
		{`"TargetPlatform":"business"`, `"TargetPlatform":"product"`},
		{`"2026-10-07"`, `"2026-09-30"`},
		{`"2026-10-01"`, `"2026-02-30"`},
		{strings.Repeat("a", 64), strings.Repeat("A", 64)},
		{`"shop-operations-diagnosis-v1"`, `"acquisition"`},
	} {
		var invalid Binding
		require.NoError(t, json.Unmarshal([]byte(strings.Replace(wire, change.from, change.to, 1)), &invalid))
		require.False(t, invalid.Valid(), change.from)
	}
	raw, err := json.Marshal(binding)
	require.NoError(t, err)
	var changed Binding
	require.NoError(t, json.Unmarshal([]byte(strings.Replace(wire, strings.Repeat("a", 64), strings.Repeat("b", 64), 1)), &changed))
	next, err := json.Marshal(changed)
	require.NoError(t, err)
	require.NotEqual(t, string(raw), string(next), "frozen disclosure participates in native request identity")
	var product Binding
	require.NoError(t, json.Unmarshal([]byte(`{"ContextKind":"acquisition","ContextID":"context","ProductKey":"product","CatalogVersion":"1","PublicationID":"pub","TargetPlatform":"shein"}`), &product))
	raw, err = json.Marshal(product)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "StoreDiagnosis", "zero new carrier cannot rewrite existing bytes/hash")
}
