package imageagent

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColorImageSelectionRequiresExactCompleteOptionAndDistinctOriginals(t *testing.T) {
	options := []ImageColorOption{{Name: "Shade", Groups: []ImageColorGroup{
		{Value: "Black", OriginalIDs: []string{"black"}},
		{Value: "Gray", OriginalIDs: []string{"gray"}},
		{Value: "White", OriginalIDs: []string{"white"}},
	}}}
	selection := ColorImageSelection{Option: "Shade", Groups: []ColorImageOriginal{
		{Color: "Black", OriginalID: "black"}, {Color: "Gray", OriginalID: "gray"}, {Color: "White", OriginalID: "white"},
	}}
	require.True(t, selection.Valid())
	require.True(t, selection.Matches(options))
	selection.Groups[0].OriginalID, selection.Groups[1].OriginalID = "gray", "black"
	require.True(t, selection.Valid())
	require.False(t, selection.Matches(options))
	selection.Groups[0].OriginalID = "black"
	require.False(t, selection.Valid())
	selection.Groups = []ColorImageOriginal{{Color: "Black", OriginalID: "black"}, {Color: "Gray", OriginalID: "gray"}}
	require.True(t, selection.Valid())
	require.False(t, selection.Matches(options))
}

func TestGroupedImagePromptUsesOnlyFrozenColor(t *testing.T) {
	base := ProductContextRef{Title: "Test jacket", Attributes: map[string]string{"color": "black, gray, white", "size": "M", "material": "wool"}}
	recipe := &ImageSlotRecipe{Placement: ImagePlacement{Color: "White"}}
	value := imageTextPrompt(base, "detail", recipe)
	require.Contains(t, value, "Selected color: White")
	require.Contains(t, value, "Confirmed material: wool")
	require.NotContains(t, value, "black, gray, white")
	require.NotContains(t, value, "Confirmed size")
	recipe.Placement.Color = ""
	require.Contains(t, imageTextPrompt(base, "detail", recipe), "Confirmed color: black, gray, white")
}
