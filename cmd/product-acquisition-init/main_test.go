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

func TestSupplyFeatureCommandRequiresExplicitExclusiveConfirmation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "product-acquisition-init")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	output, err := build.CombinedOutput()
	require.NoError(t, err, "%s", output)

	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"explicit feature", []string{"--enable-supply-chain", "--confirm-database", "retained_product", "--config", filepath.Join(t.TempDir(), "missing.json")}, "supply chain initialization refused"},
		{"empty database mode", []string{"--enable-supply-chain", "--confirm-empty-database", "retained_product"}, "cannot be combined"},
		{"completion mode", []string{"--enable-supply-chain", "--enable-completion-drafts"}, "mutually exclusive"},
		{"plan mode", []string{"--enable-supply-chain", "--enable-acquisition-plans"}, "mutually exclusive"},
		{"trial mixed mode", []string{"--enable-shein-us-trial", "--enable-supply-chain"}, "mutually exclusive"},
		{"trial requires exact Product name", []string{"--enable-shein-us-trial", "--confirm-database", "wrong_product"}, "SHEIN trial ledger initialization refused"},
		{"positional input", []string{"--enable-supply-chain", "retained_product"}, "unexpected positional arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output, err := exec.CommandContext(ctx, binary, tc.args...).CombinedOutput()
			require.Error(t, err, "invalid requests must not reach a database")
			require.Contains(t, string(output), tc.want)
		})
	}
}
