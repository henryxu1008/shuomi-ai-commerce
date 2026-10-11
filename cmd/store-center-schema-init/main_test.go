package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestObservationsOnlyRejectsBroaderInstallationBeforeOpeningDatabase(t *testing.T) {
	for _, flags := range [][2]bool{{true, false}, {false, true}, {true, true}} {
		err := run("missing-private-config", flags[0], flags[1], true)
		require.ErrorContains(t, err, "cannot be combined")
	}
}
