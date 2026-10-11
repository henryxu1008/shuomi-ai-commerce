package main

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestObservationsOnlyRejectsBroaderInstallationBeforeOpeningDatabase(t *testing.T) {
	for _, flags := range [][2]bool{{true, false}, {false, true}, {true, true}} {
		err := run("missing-private-config", flags[0], flags[1], true, false, false)
		require.ErrorContains(t, err, "cannot be combined")
	}
}

func TestObservationNarrowModesRejectCombinationsBeforeOpeningDatabase(t *testing.T) {
	for _, modes := range [][3]bool{{true, true, false}, {true, false, true}, {false, true, true}} {
		require.ErrorContains(t, run("missing-private-config", false, false, modes[0], modes[1], modes[2]), "cannot be combined")
	}
	for _, modes := range [][3]bool{{false, true, false}, {false, false, true}} {
		for _, broad := range [][2]bool{{true, false}, {false, true}} {
			require.ErrorContains(t, run("missing-private-config", broad[0], broad[1], modes[0], modes[1], modes[2]), "cannot be combined")
		}
	}
}
