package httpapi

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestContentSkillSelectionIsExactAndStartOnly(t *testing.T) {
	raw := `{"skillId":"11111111-1111-4111-8111-111111111111","version":1,"digest":"` + strings.Repeat("a", 64) + `"}`
	b := productAgentRequestBody{SkillSelection: json.RawMessage(raw)}
	s, e := b.skillSelection("start")
	require.NoError(t, e)
	require.Equal(t, uint64(1), s.Version)
	for _, action := range []string{"resume", "review", "read"} {
		_, e = b.skillSelection(action)
		require.Error(t, e)
	}
	for _, invalid := range []string{"null", `{}`, strings.Replace(raw, `"version":1`, `"version":0`, 1), strings.Replace(raw, `"skillId"`, `"owner"`, 1), strings.TrimSuffix(raw, "}") + `,"ownerId":"another-user"}`} {
		_, e = (productAgentRequestBody{SkillSelection: json.RawMessage(invalid)}).skillSelection("start")
		require.Error(t, e)
	}
	s, e = (productAgentRequestBody{}).skillSelection("start")
	require.NoError(t, e)
	require.Nil(t, s)
}
