package main

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"testing"
)

func TestEmptyObservationAlignmentCLI(t *testing.T) {
	if raw := os.Getenv("STORE_ALIGNMENT_TEST_ARGS"); raw != "" {
		var args []string
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			panic(err)
		}
		os.Args = append([]string{"store-center-schema-init"}, args...)
		main()
		return
	}
	for _, tc := range []struct {
		name    string
		args    []string
		message string
		success bool
	}{
		{"help", []string{"-h"}, "align-empty-observations-contract", true},
		{"missing-confirmation", []string{"--align-empty-observations-contract", "--config", "missing"}, "exact database confirmation", false},
		{"mixed-core", []string{"--align-empty-observations-contract", "--align-core-contract"}, "cannot be combined", false},
		{"mixed-install", []string{"--align-empty-observations-contract", "--observations-current-store"}, "cannot be combined", false},
		{"mixed-verify", []string{"--align-empty-observations-contract", "--observations-verify"}, "cannot be combined", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(tc.args)
			require.NoError(t, err)
			cmd := exec.Command(os.Args[0], "-test.run=^TestEmptyObservationAlignmentCLI$")
			cmd.Env = append(os.Environ(), "STORE_ALIGNMENT_TEST_ARGS="+string(args))
			out, err := cmd.CombinedOutput()
			if tc.success {
				require.NoError(t, err, string(out))
			} else {
				require.Error(t, err, string(out))
			}
			require.Contains(t, string(out), tc.message)
		})
	}
}

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
