package openai

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestFlareReferenceUsageAcceptsExplicitTotalsWithoutInventingPartitions(t *testing.T) {
	for _, raw := range []string{
		`{"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150}}`,
		`{"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150,"input_tokens_details":{"text_tokens":30}}}`,
		`{"usage":{"input_tokens":0,"output_tokens":0,"total_tokens":0}}`,
	} {
		u, err := ObserveFlareImageUsage([]byte(raw))
		require.NoError(t, err)
		require.Equal(t, u.TotalTokens, u.InputTokens+u.OutputTokens)
		require.False(t, u.InputImageKnown)
		require.Zero(t, u.InputImageTokens)
		require.NoError(t, u.Validate())
	}
}
func TestFlareReferenceUsageRejectsMissingAmbiguousOrContradictoryFacts(t *testing.T) {
	for _, raw := range []string{
		`{}`, `{"usage":{}}`,
		`{"usage":{"input_tokens":0,"output_tokens":0}}`,
		`{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":14}}`,
		`{"usage":{"input_tokens":10,"input_tokens":11,"output_tokens":5,"total_tokens":15}}`,
		`{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15,"input_tokens_details":{"text_tokens":8,"image_tokens":8}}}`,
		`{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15,"input_tokens_details":{"image_tokens":-1}}}`,
		`{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15,"output_tokens_details":{"text_tokens":1,"image_tokens":4}}}`,
		`{"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15,"output_tokens_details":{"audio_tokens":1}}}`,
	} {
		_, err := ObserveFlareImageUsage([]byte(raw))
		require.Error(t, err, raw)
	}
}
