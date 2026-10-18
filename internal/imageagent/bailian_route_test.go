package imageagent

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQwenRouteIsBoundAcrossQuoteAndIntent(t *testing.T) {
	i := generationTestIntent()
	i.Provider = "bailian"
	i.Model = "qwen-image-3.0"
	i.Protocol = "dashscope-image-sync-v1"
	_, err := NewGenerationFact(i)
	require.NoError(t, err)
	q := ImageGenerationQuote{Provider: i.Provider, Model: i.Model, Protocol: i.Protocol, Resolution: i.Resolution, Quality: i.Quality, PriceVersion: i.PriceVersion, Points: i.Points, RouteReference: i.RouteReference, CredentialReference: i.CredentialReference, ConfigurationVersion: i.ConfigurationVersion}
	require.True(t, q.Valid())
	for _, field := range []string{"provider", "model", "protocol"} {
		wrong := i
		switch field {
		case "provider":
			wrong.Provider = "grsai"
		case "model":
			wrong.Model = "gpt-image-2.5"
		case "protocol":
			wrong.Protocol = "grsai-json-sync-v1"
		}
		_, err = NewGenerationFact(wrong)
		require.Error(t, err, field)
	}
}
