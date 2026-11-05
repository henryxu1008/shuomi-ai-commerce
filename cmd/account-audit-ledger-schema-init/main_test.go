package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Exercise the actual CLI without producing or installing a separate binary.
func TestLedgerCLIProcess(t *testing.T) {
	if os.Getenv("ISSUE1075_LEDGER_CLI_PROCESS") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			break
		}
	}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	os.Exit(0)
}

func ledgerCLI(t *testing.T, dsn string, args ...string) (string, error) {
	t.Helper()
	file := filepath.Join(t.TempDir(), "synthetic-owner-dsn")
	require.NoError(t, os.WriteFile(file, []byte(dsn), 0600))
	commandArgs := append([]string{"-test.run=^TestLedgerCLIProcess$", "--", "-dsn-file", file}, args...)
	command := exec.Command(os.Args[0], commandArgs...)
	command.Env = append(os.Environ(), "ISSUE1075_LEDGER_CLI_PROCESS=1")
	out, err := command.CombinedOutput()
	return string(out), err
}

func TestEmptyDatabaseModeRejectsInvalidArgumentsBeforeConnecting(t *testing.T) {
	for _, name := range []string{"", "product_agent", " leading", "trailing ", "Upper", "a-b", "a/b", "1db", strings.Repeat("a", 64)} {
		t.Run(name, func(t *testing.T) {
			out, err := ledgerCLI(t, "postgresql://synthetic:never-connect@127.0.0.1:1/unused", "-namespace", "product", "-confirm-empty-database", name)
			require.Error(t, err)
			require.Contains(t, out, "invalid confirmed empty database")
			require.NotContains(t, out, "never-connect")
		})
	}
	out, err := ledgerCLI(t, "postgresql://synthetic:never-connect@127.0.0.1:1/unused", "-namespace", "image", "-confirm-empty-database", "new_run")
	require.Error(t, err)
	require.Contains(t, out, "invalid confirmed empty database")
}
