package httpapi

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	"testing"
)

func TestImageAnalysisConfirmationBindsWholePreparedImageModelAndFee(t *testing.T) {
	s := agentconfig.Snapshot{ID: "prepared-id", Digest: strings.Repeat("a", 64), ExecutionModelProfile: aicapability.ModelProfile{ModelID: aicapability.SingleImageMiniModel}, ImageAnalysis: &agentconfig.ImageAnalysisConfiguration{QuoteDigest: strings.Repeat("b", 64), MaximumMicropoints: 1234, Image: aicapability.SingleImageInput{SelectionDigest: strings.Repeat("c", 64), ContentSHA256: strings.Repeat("d", 64)}}}
	c := imageAnalysisConfirmBody{PreparationID: s.ID, ConfigurationDigest: s.Digest, QuoteDigest: s.ImageAnalysis.QuoteDigest, SelectionDigest: s.ImageAnalysis.Image.SelectionDigest, ContentSHA256: s.ImageAnalysis.Image.ContentSHA256, ModelID: s.ExecutionModelProfile.ModelID, MaximumMicropoints: "1234", Confirmed: true}
	require.True(t, c.matches(s))
	for _, field := range []string{"confirmed", "image", "selection", "quote", "configuration", "model", "fee", "preparation"} {
		other := c
		switch field {
		case "confirmed":
			other.Confirmed = false
		case "image":
			other.ContentSHA256 = strings.Repeat("e", 64)
		case "selection":
			other.SelectionDigest = strings.Repeat("e", 64)
		case "quote":
			other.QuoteDigest = strings.Repeat("e", 64)
		case "configuration":
			other.ConfigurationDigest = strings.Repeat("e", 64)
		case "model":
			other.ModelID = "other"
		case "fee":
			other.MaximumMicropoints = "1235"
		case "preparation":
			other.PreparationID = "other"
		}
		require.False(t, other.matches(s), field)
	}
}
