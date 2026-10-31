package image

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPolicySceneDefaultsRetainsAllValuesAndOwnsCollections(t *testing.T) {
	input := PolicySceneDefaults{SceneCategory: " general ", SceneStyle: " minimal ", BackgroundTone: " white ", Composition: " centered ", PropsLevel: " none ", AudienceHint: " all ", CustomSceneHint: " hint ", SlotRole: " main ", SlotBrief: " brief ", StyleReferenceIDs: []string{" a ", "a", "b"}}
	got, err := NormalizePolicySceneDefaults(input)
	require.NoError(t, err)
	require.Equal(t, PolicySceneDefaults{SceneCategory: "general", SceneStyle: "minimal", BackgroundTone: "white", Composition: "centered", PropsLevel: "none", AudienceHint: "all", CustomSceneHint: "hint", SlotRole: "main", SlotBrief: "brief", StyleReferenceIDs: []string{"a", "b"}}, got)
	got.StyleReferenceIDs[0] = "changed"
	require.Equal(t, " a ", input.StyleReferenceIDs[0])
	data, err := json.Marshal(PolicySceneDefaults{})
	require.NoError(t, err)
	require.JSONEq(t, `{"SceneCategory":"","SceneStyle":"","BackgroundTone":"","Composition":"","PropsLevel":"","AudienceHint":"","CustomSceneHint":"","SlotRole":"","SlotBrief":"","StyleReferenceIDs":null}`, string(data))
	_, err = NormalizePolicySceneDefaults(PolicySceneDefaults{SlotBrief: strings.Repeat("x", maxImageStringBytes+1)})
	require.ErrorIs(t, err, ErrInputInvalid)
	_, err = NormalizePolicySceneDefaults(PolicySceneDefaults{StyleReferenceIDs: make([]string, maxStyleReferences+1)})
	require.ErrorIs(t, err, ErrInputInvalid)
}
