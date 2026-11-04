package routeconfig

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestReferenceImageProfileIsFrozenByTheOriginalPriceVersion(t *testing.T) {
	now := time.Date(2026, 11, 4, 0, 0, 0, 0, time.UTC)
	var in PlatformPricesInput
	require.NoError(t, json.Unmarshal([]byte(`{"prices":[{"id":"flare-reference","provider":"openai","model":"gpt-image-2.5-flare-2026-09-08","region":"global","serviceType":"image-sync","channel":"openai-direct","currency":"CNY","usageVersion":"reference-images-totals-v1","rates":{"inputText":10000,"inputImage":20000,"outputImage":10000},"sourceUrl":"https://developers.openai.com/api/docs/","verifiedAt":"2026-11-04T00:00:00Z","effectiveFrom":"2026-11-04T00:00:00Z","effectiveUntil":"2026-11-05T00:00:00Z","image":{"width":1024,"height":1024,"quality":"auto","referenceEstimate":{"inputTokens":100,"outputTokens":100,"operationScope":"product-image-effects-v1","size":"1024x1024","quality":"auto","allowAutoSize":true,"maximumPromptBytes":65536,"maximumReferenceImages":3,"maximumReferenceBytes":16777216,"maximumReferencePixels":64000000}}}]}`), &in))
	frozen, err := FreezePlatformPrices(in, "reference-1", nil, now)
	require.NoError(t, err)
	encoded, err := json.Marshal(frozen)
	require.NoError(t, err)
	require.Contains(t, string(encoded), `"referenceEstimate"`)
	// Mutating the caller's price shape cannot rewrite an already frozen snapshot.
	in.Prices[0].Image.Width = 512
	require.Equal(t, 1024, frozen.Prices[0].Image.Width)
}
