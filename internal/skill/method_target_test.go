package skill

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMethodMarkerProjectsOnlyExplicitTargetWithoutReservingExistingMetadata(t *testing.T) {
	for _, tc := range []struct{ name, metadata, target, display string }{
		{"original metadata", "  target: arbitrary-current-title-metadata\n", "", ""},
		{"unknown marker", "  listingkit-method-v1: other-future-value\n", "", ""},
		{"image", "  listingkit-method-v1: product-image\n  listingkit-display-name: 标准商品套图\n", "product-image", "标准商品套图"},
		{"title", "  listingkit-method-v1: product-title\n", "product-title", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := []File{{Path: "SKILL.md", Text: "---\nname: valid-current-method\ndescription: A bounded current method.\nmetadata:\n" + tc.metadata + "---\nUse only saved facts.\n"}}
			p, err := ValidatePackage(files)
			require.NoError(t, err)
			require.Equal(t, files, p.Files)
			raw, err := json.Marshal(p)
			require.NoError(t, err)
			var projected map[string]any
			require.NoError(t, json.Unmarshal(raw, &projected))
			if tc.target == "" {
				require.NotContains(t, projected, "target")
			} else {
				require.Equal(t, tc.target, projected["target"])
			}
			if tc.display == "" {
				require.NotContains(t, projected, "displayName")
			} else {
				require.Equal(t, tc.display, projected["displayName"])
			}
		})
	}
}

func TestChatTextPackageCannotEnterProductConsumers(t *testing.T) {
	files := []File{{Path: "SKILL.md", Text: "---\nname: action-items\ndescription: Organize handoff notes.\nmetadata:\n  listingkit-method-v1: chat-text\n---\nReturn a list of actions.\n"}}
	pkg, err := ValidatePackage(files)
	require.NoError(t, err)
	require.True(t, pkg.Supports("chat-text"))
	require.False(t, pkg.Supports(TitleTarget))
	require.False(t, pkg.Supports(ImageTarget))
	require.False(t, pkg.Supports(AuthoringTarget))

	_, err = ValidatePackage(append(files, File{Path: "references/example.md", Text: "An example"}))
	require.ErrorIs(t, err, ErrInvalid)
}
