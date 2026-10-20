package imageagent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVIPQuoteAndIntentRequireMediumWithoutChangingOldRoutes(t *testing.T) {
	old := generationTestIntent()
	before, err := NewGenerationFact(old)
	require.NoError(t, err)
	require.Equal(t, "419cdce766de9ebdcead7aa64d8f93023ea5c6a66d84ec51a0222f003dce95ec", before.Fingerprint, "original auto intent bytes and fingerprint must remain stable")
	i := old
	i.Model, i.Quality = "gpt-image-2-vip", "medium"
	q := ImageGenerationQuote{Provider: i.Provider, Model: i.Model, Protocol: i.Protocol, Resolution: i.Resolution, Quality: i.Quality, Points: i.Points, PriceVersion: i.PriceVersion, RouteReference: i.RouteReference, CredentialReference: i.CredentialReference, ConfigurationVersion: i.ConfigurationVersion}
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
