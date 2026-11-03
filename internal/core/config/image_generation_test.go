package config

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestImageGenerationFixedPriceIsRetiredAndCannotOpenGeneration(t *testing.T) {
	cfg, err := loadConfigFixture(t, []byte("imageagent:\n  generation:\n    priceVersion: retired\n    pointsPerImage: 17\n"))
	require.NoError(t, err)
	require.False(t, cfg.ImageAgent.Generation.Configured())
	cfg, err = loadConfigFixture(t, []byte("imageagent:\n  generation:\n    enabled: true\n    unitVersion: ai-micropoint-v1\n    maximumMicropoints: 17000000\n"))
	require.NoError(t, err)
	require.True(t, cfg.ImageAgent.Generation.Configured())
	require.EqualValues(t, 17000000, cfg.ImageAgent.Generation.MaximumMicropoints)
}
