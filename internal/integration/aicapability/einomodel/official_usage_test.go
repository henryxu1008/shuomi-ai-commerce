package einomodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialTextUsagePreservesCacheAndDoesNotDoubleCountReasoningOrImages(t *testing.T) {
	usage, err := ObserveOfficialOpenAITextUsage([]byte(`{"choices":[],"usage":{"prompt_tokens":1000,"completion_tokens":100,"total_tokens":1100,"prompt_tokens_details":{"cached_tokens":200},"completion_tokens_details":{"reasoning_tokens":40}}}`))
	require.NoError(t, err)
	require.True(t, usage.Complete)
	require.EqualValues(t, 1000, usage.InputText)
	require.EqualValues(t, 200, usage.CachedInputText)
	require.EqualValues(t, 100, usage.OutputText)
	require.Zero(t, usage.InputImage, "Mini image tokens already belong to prompt_tokens")
	observed := observeUsage([]byte(`{"usage":{"prompt_tokens":1000,"completion_tokens":100,"total_tokens":1100,"prompt_tokens_details":{"cached_tokens":200}}}`), AdapterOpenAICompatible)
	require.NotNil(t, observed.BillableUsage, "the original guarded transport must carry the parsed partitions")
	require.Equal(t, usage, *observed.BillableUsage)
}

func TestOfficialTextUsageMissingCacheIsNotZeroAndDuplicateCountersAreUnknown(t *testing.T) {
	for _, raw := range []string{
		`{"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
		`{"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7,"prompt_tokens_details":{"cached_tokens":6}}}`,
		`{"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":8,"prompt_tokens_details":{"cached_tokens":0}}}`,
		`{"usage":{"prompt_tokens":5,"prompt_tokens":6,"completion_tokens":2,"total_tokens":8,"prompt_tokens_details":{"cached_tokens":0}}}`,
		`{"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7,"prompt_tokens_details":{"cached_tokens":0},"completion_tokens_details":{"reasoning_tokens":3}}}`,
		`{"usage":{"prompt_tokens":null,"completion_tokens":2,"total_tokens":2,"prompt_tokens_details":{"cached_tokens":0}}}`,
	} {
		usage, err := ObserveOfficialOpenAITextUsage([]byte(raw))
		require.ErrorIs(t, err, ErrUsageUnknown, raw)
		require.False(t, usage.Complete)
	}
}

func TestOfficialDirectImageUsageRequiresFullPartitionsAndCountsOutputOnce(t *testing.T) {
	usage, err := ObserveOfficialOpenAIImageUsage([]byte(`{"data":[{"b64_json":"synthetic"}],"usage":{"input_tokens":70,"input_tokens_details":{"text_tokens":20,"image_tokens":50},"output_tokens":196,"total_tokens":266}}`))
	require.NoError(t, err)
	require.True(t, usage.Complete)
	require.EqualValues(t, 20, usage.InputText)
	require.EqualValues(t, 50, usage.InputImage)
	require.EqualValues(t, 196, usage.OutputImage)
	require.Zero(t, usage.OutputText)
	require.Zero(t, usage.CachedInputImage, "direct Images API has no cached price discount")
	for _, raw := range []string{
		`{"usage":{"input_tokens":70,"output_tokens":196,"total_tokens":266}}`,
		`{"usage":{"input_tokens":70,"input_tokens_details":{"text_tokens":20},"output_tokens":196,"total_tokens":266}}`,
		`{"usage":{"input_tokens":70,"input_tokens_details":{"text_tokens":21,"image_tokens":50},"output_tokens":196,"total_tokens":266}}`,
		`{"usage":{"input_tokens":70,"input_tokens_details":{"text_tokens":20,"image_tokens":50},"output_tokens":196,"total_tokens":266,"output_tokens_details":{"image_tokens":195}}}`,
		`{"usage":{"input_tokens":0,"input_tokens_details":{"text_tokens":0,"image_tokens":0},"output_tokens":-1,"total_tokens":-1}}`,
	} {
		_, err := ObserveOfficialOpenAIImageUsage([]byte(raw))
		require.ErrorIs(t, err, ErrUsageUnknown, raw)
	}
}

func TestOfficialObservedZeroDoesNotProveNoSendOrSuccessfulOutput(t *testing.T) {
	usage, err := ObserveOfficialOpenAITextUsage([]byte(`{"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0,"prompt_tokens_details":{"cached_tokens":0}}}`))
	require.NoError(t, err)
	require.True(t, usage.Complete, "outcome validation remains the original invocation owner's responsibility")
	_, err = ObserveOfficialOpenAITextUsage([]byte(`{}`))
	require.ErrorIs(t, err, ErrUsageUnknown)
}
