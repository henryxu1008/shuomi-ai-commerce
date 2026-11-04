package imageagent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVIPQuoteAndIntentRequireMediumWithoutChangingOldRoutes(t *testing.T) {
	old := generationTestIntent()
	before, err := NewGenerationFact(old)
	require.NoError(t, err)
	require.Equal(t, "77f77ead89afbb68c947a15707581c15cd3bbc8c0b80ae81d2de8e20bfd56779", before.Fingerprint, "current official-price intent has a frozen identity; old fixed-point facts are not reinterpreted")
	i := old
	i.Model, i.Quality = "gpt-image-2-vip", "medium"
	i.Billing = syntheticImageBilling(i.Model, i.PriceVersion, i.Points)
	q := ImageGenerationQuote{Billing: syntheticImageBilling(i.Model, i.PriceVersion, i.Points), Provider: i.Provider, Model: i.Model, Protocol: i.Protocol, Resolution: i.Resolution, Quality: i.Quality, Points: i.Points, PriceVersion: i.PriceVersion, RouteReference: i.RouteReference, CredentialReference: i.CredentialReference, ConfigurationVersion: i.ConfigurationVersion}
	require.True(t, q.Valid(), "VIP medium quote must be admitted")
	f, err := NewGenerationFact(i)
	require.NoError(t, err)
	require.Equal(t, "medium", f.Intent.Quality)
	for _, quality := range []string{"", "auto", "high"} {
		q.Quality, i.Quality = quality, quality
		require.False(t, q.Valid(), quality)
		_, err = NewGenerationFact(i)
		require.Error(t, err, quality)
	}
	for _, model := range []string{"gpt-image-2.5", "qwen-image-3.0", "qwen-image-3.0-pro"} {
		i = old
		i.Model = model
		i.Billing = syntheticImageBilling(i.Model, i.PriceVersion, i.Points)
		if model != "gpt-image-2.5" {
			i.Provider, i.Protocol = "bailian", "dashscope-image-sync-v1"
		}
		_, err = NewGenerationFact(i)
		require.NoError(t, err, model)
		i.Quality = "medium"
		_, err = NewGenerationFact(i)
		require.Error(t, err, model)
	}
}
