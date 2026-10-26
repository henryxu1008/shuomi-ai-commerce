package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCoreAlignmentCommandIsExplicitAndExclusive(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "store-center-schema-init")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, ".").CombinedOutput()
	require.NoError(t, err, "%s", output)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"explicit mode", []string{"--align-core-contract", "--confirm-database", "retained_store", "--config", filepath.Join(t.TempDir(), "missing.json")}, "core contract alignment refused"},
		{"missing confirmation", []string{"--align-core-contract"}, "exact database confirmation"},
		{"confirmation without mode", []string{"--confirm-database", "retained_store"}, "requires --align-core-contract"},
		{"empty confirmation without mode", []string{"--confirm-database="}, "requires --align-core-contract"},
		{"positional arguments", []string{"--align-core-contract", "retained_store"}, "unexpected positional arguments"},
		{"retained observations without alignment", []string{"--retained-observations"}, "require --align-core-contract"},
		{"retained false without alignment", []string{"--retained-observations=false"}, "require --align-core-contract"},
		{"retained cockpit without alignment", []string{"--retained-operations-cockpit"}, "require --align-core-contract"},
		{"retained with browser mode", []string{"--browser-workbench", "--retained-operations-cockpit"}, "require --align-core-contract"},
		{"explicit retained capabilities", []string{"--align-core-contract", "--retained-observations", "--retained-operations-cockpit", "--confirm-database", "retained_store", "--config", filepath.Join(t.TempDir(), "missing.json")}, "core contract alignment refused"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := exec.CommandContext(ctx, binary, tc.args...).CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), tc.want)
		})
	}
	for _, mode := range []string{"--observations", "--observations-only", "--observations-grants-only", "--observations-verify", "--operations-cockpit", "--tiktok-shop", "--walmart-us"} {
		t.Run(mode, func(t *testing.T) {
			output, err := exec.CommandContext(ctx, binary, "--align-core-contract", mode).CombinedOutput()
			require.Error(t, err)
			require.Contains(t, string(output), "cannot be combined")
		})
	}
}
