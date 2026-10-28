package image

import (
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestMergeSceneOptionsNormalizesReferencesWithoutAliasingInputs(t *testing.T) {
	t.Parallel()

	base := &SceneOptions{
		SceneStyle:        " studio ",
		BackgroundTone:    " bright ",
		StyleReferenceIDs: []string{"style-1"},
	}
	override := &SceneOptions{
		SceneStyle:        " lifestyle ",
		StyleReferenceIDs: []string{" style-2 ", "style-2", "", "style-3"},
	}

	got, err := MergeSceneOptions(base, override)
	require.NoError(t, err)
	require.Equal(t, "lifestyle", got.SceneStyle)
	require.Equal(t, "bright", got.BackgroundTone)
	require.Equal(t, []string{"style-2", "style-3"}, got.StyleReferenceIDs)

	got.StyleReferenceIDs[0] = "mutated"
	require.Equal(t, []string{"style-1"}, base.StyleReferenceIDs)
	require.Equal(t, " style-2 ", override.StyleReferenceIDs[0])
}

func TestMergeSceneOptionsRejectsReferenceResourceLimitViolations(t *testing.T) {
	t.Parallel()

	tooMany := make([]string, 17)
	for index := range tooMany {
		tooMany[index] = string(rune(0x4000 + index))
	}
	for name, references := range map[string][]string{
		"too many": tooMany,
		"too long": {strings.Repeat("x", 8193)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := MergeSceneOptions(nil, &SceneOptions{StyleReferenceIDs: references})
			require.ErrorIs(t, err, ErrInputInvalid)
		})
	}
}

func TestMergeSceneOptionsPreflightsRawScalarLimitsBeforeNormalization(t *testing.T) {
	t.Parallel()

	const designMaxImageStringBytes = 8 << 10
	const scalarBytes = 8 << 10
	for name, options := range map[string]SceneOptions{
		"overlong whitespace": {SceneStyle: strings.Repeat(" ", designMaxImageStringBytes+1)},
		"aggregate scalar bytes": {
			SceneCategory: strings.Repeat("a", scalarBytes), SceneStyle: strings.Repeat("b", scalarBytes),
			BackgroundTone: strings.Repeat("c", scalarBytes), Composition: strings.Repeat("d", scalarBytes),
			PropsLevel: strings.Repeat("e", scalarBytes), AudienceHint: strings.Repeat("f", scalarBytes),
			CustomSceneHint: strings.Repeat("g", scalarBytes), SlotRole: strings.Repeat("h", scalarBytes),
			SlotBrief: strings.Repeat("i", scalarBytes),
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := MergeSceneOptions(nil, &options)
			require.ErrorIs(t, err, ErrInputInvalid)
			require.Nil(t, got)
		})
	}
}

func TestMergeSceneOptionsRejectsAggregateCreatedAcrossBaseAndOverride(t *testing.T) {
	t.Parallel()

	const scalarBytes = 8 << 10
	base := &SceneOptions{
		SceneCategory: strings.Repeat("a", scalarBytes), SceneStyle: strings.Repeat("b", scalarBytes),
		BackgroundTone: strings.Repeat("c", scalarBytes), Composition: strings.Repeat("d", scalarBytes),
	}
	override := &SceneOptions{
		PropsLevel: strings.Repeat("e", scalarBytes), AudienceHint: strings.Repeat("f", scalarBytes),
		CustomSceneHint: strings.Repeat("g", scalarBytes), SlotRole: strings.Repeat("h", scalarBytes),
		SlotBrief: strings.Repeat("i", scalarBytes),
	}

	got, err := MergeSceneOptions(base, override)
	require.ErrorIs(t, err, ErrInputInvalid)
	require.Nil(t, got)
}
