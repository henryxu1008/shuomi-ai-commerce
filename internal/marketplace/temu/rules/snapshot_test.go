package rules

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOfficialSnapshotRevisionIsStableAcrossCheckoutLineEndings(t *testing.T) {
	raw, err := officialData.ReadFile("data/local-specifications.tsv")
	require.NoError(t, err)
	rows, revision, err := decodeSnapshotRows(raw, 15, 39)
	require.NoError(t, err)
	windows := strings.ReplaceAll(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n", "\r\n")
	windowsRows, windowsRevision, err := decodeSnapshotRows([]byte(windows), 15, 39)
	require.NoError(t, err)
	require.Equal(t, rows, windowsRows)
	require.Equal(t, revision, windowsRevision, "the same official rules cannot invalidate approvals solely due to a checkout OS")
	changed := strings.Replace(string(raw), "[2.00,99999999.99]", "[3.00,99999999.99]", 1)
	_, changedRevision, err := decodeSnapshotRows([]byte(changed), 15, 39)
	require.NoError(t, err)
	require.NotEqual(t, revision, changedRevision, "a changed official bound must change the rule revision")
}
