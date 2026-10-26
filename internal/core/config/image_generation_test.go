package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImageGenerationPriceIsExplicitAndHasNoDefault(t *testing.T) {
	cfg, err := loadConfigFixture(t, []byte("openai:\n  apiKey: test-only\nimageagent:\n  generation:\n    priceVersion: image-price-v2\n    pointsPerImage: 17\n"))
	require.NoError(t, err)
	require.Equal(t, "image-price-v2", cfg.ImageAgent.Generation.PriceVersion)
	require.EqualValues(t, 17, cfg.ImageAgent.Generation.PointsPerImage)
	cfg, err = loadConfigFixture(t, []byte("openai:\n  apiKey: test-only\n"))
	require.NoError(t, err)
	require.Equal(t, ImageAgentGenerationConfig{}, cfg.ImageAgent.Generation)
}
