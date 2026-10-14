package observations

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTemuCursorExhaustionAndCyclesNeverUseOrdinalCompletion(t *testing.T) {
	sync := Sync{Kind: Products, Progress: Checkpoint{Page: 1}}
	next := "opaque-next"
	c, status, err := advanceTemuCursor(sync, 10, 10, &next)
	require.NoError(t, err)
	require.Equal(t, "running", status)
	require.Equal(t, next, c.TemuPageToken)
	sync.Progress = c
	c, status, err = advanceTemuCursor(sync, 0, 10, nil)
	require.NoError(t, err)
	require.Equal(t, "completed", status)
	_, status, err = advanceTemuCursor(sync, 10, 20, &next)
	require.NoError(t, err)
	require.Equal(t, "partial", status)
	_, status, err = advanceTemuCursor(Sync{Kind: Products, Progress: Checkpoint{Page: 1}}, 1, 2, nil)
	require.NoError(t, err)
	require.Equal(t, "partial", status)
}

func TestNativeProviderCursorRemainsOnlyInOriginalCheckpoint(t *testing.T) {
	s := Sync{Progress: Checkpoint{TemuPageToken: "opaque-provider-cursor", TemuTokenHashes: []string{"exact-digest"}}}
	stored, err := json.Marshal(s.Progress)
	require.NoError(t, err)
	require.Contains(t, string(stored), "opaque-provider-cursor")
	public, err := json.Marshal(s)
	require.NoError(t, err)
	require.NotContains(t, string(public), "opaque-provider-cursor")
	require.NotContains(t, string(public), "exact-digest")
}
