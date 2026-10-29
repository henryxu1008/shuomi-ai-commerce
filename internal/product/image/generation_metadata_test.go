package image

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

var (
	metadataResultSink GenerationMetadata
	metadataErrorSink  error
)

func TestNormalizeGenerationMetadataDefensivelyCopiesAndCanonicalizesValues(t *testing.T) {
	t.Parallel()

	input := GenerationMetadata{
		Capability:      " render_scene ",
		ModelFamily:     " image-model ",
		InvocationID:    " invocation-1 ",
		PromptReference: " scene/default ",
		PromptVersion:   " v2 ",
		Values:          map[string]string{"seed": " 42 "},
	}

	got, err := NormalizeGenerationMetadata(input)
	require.NoError(t, err)
	require.Equal(t, "render_scene", got.Capability)
	require.Equal(t, "image-model", got.ModelFamily)
	require.Equal(t, map[string]string{"seed": "42"}, got.Values)

	input.Values["seed"] = "changed"
	require.Equal(t, "42", got.Values["seed"])
}

func TestNormalizeGenerationMetadataRejectsOversizedValues(t *testing.T) {
	t.Parallel()

	values := make(map[string]string, 129)
	for i := 0; i < 129; i++ {
		values[string(rune(0x2000+i))] = "value"
	}
	_, err := NormalizeGenerationMetadata(GenerationMetadata{Capability: "render_scene", Values: values})
	require.ErrorIs(t, err, ErrInputInvalid)
}

func TestNormalizeGenerationMetadataRejectsKeysThatConflictAfterNormalization(t *testing.T) {
	t.Parallel()

	_, err := NormalizeGenerationMetadata(GenerationMetadata{
		Capability: "render_scene",
		Values: map[string]string{
			" seed ": "41",
			"seed":   "42",
		},
	})
	require.ErrorIs(t, err, ErrInputInvalid)
}

func TestNormalizeGenerationMetadataPreflightsRawStringsBeforeAllocationOrTrimming(t *testing.T) {
	const designMaxImageStringBytes = 8 << 10
	const designMaxImageInputBytes = 64 << 10

	aggregateValues := make(map[string]string, 9)
	for index := 0; index < 9; index++ {
		aggregateValues[string(rune('a'+index))] = strings.Repeat(string(rune('a'+index)), designMaxImageStringBytes-1)
	}
	for name, metadata := range map[string]GenerationMetadata{
		"overlong fixed field before trim": {Capability: strings.Repeat(" ", designMaxImageStringBytes+1)},
		"overlong map key":                 {Values: map[string]string{strings.Repeat("k", designMaxImageStringBytes+1): "value"}},
		"overlong map value":               {Values: map[string]string{"key": strings.Repeat("v", designMaxImageStringBytes+1)}},
		"raw aggregate":                    {Values: aggregateValues},
	} {
		t.Run(name, func(t *testing.T) {
			allocations := testing.AllocsPerRun(10, func() {
				metadataResultSink, metadataErrorSink = NormalizeGenerationMetadata(metadata)
			})
			require.ErrorIs(t, metadataErrorSink, ErrInputInvalid)
			require.Equal(t, GenerationMetadata{}, metadataResultSink)
			require.Zero(t, allocations, "raw resource limits must be checked before result maps or sort keys are allocated")
		})
	}
	require.Greater(t, 9*(designMaxImageStringBytes-1), designMaxImageInputBytes)
}

func TestNormalizeGenerationMetadataRejectsBlankCanonicalMapEntries(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]map[string]string{
		"blank key":   {" ": "value"},
		"blank value": {"key": " "},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := NormalizeGenerationMetadata(GenerationMetadata{Values: values})
			require.ErrorIs(t, err, ErrInputInvalid)
			require.Equal(t, GenerationMetadata{}, got)
		})
	}
}
