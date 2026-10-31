package main

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestConfigurationIsRequiredBeforeExecution(t *testing.T) {
	for _, args := range [][]string{nil, {"-config", " \t"}} {
		called := false
		err := runCommand(args, func(arg commandOptions) error { called = true; return nil })
		if called || err == nil || !strings.Contains(err.Error(), "-config is required") {
			t.Fatalf("args=%v called=%v err=%v", args, called, err)
		}
	}
}
func TestExplicitConfigurationAndHelp(t *testing.T) {
	called := false
	sentinel := errors.New("synthetic execution result")
	err := runCommand([]string{"-config", "private-fixture.yaml", "-grant-image-agent-runtime"}, func(arg commandOptions) error {
		called = true
		if arg.configPath != "private-fixture.yaml" {
			t.Fatal("path changed")
		}
		return sentinel
	})
	if !called || !errors.Is(err, sentinel) {
		t.Fatalf("called=%v err=%v", called, err)
	}
	called = false
	err = runCommand([]string{"-help"}, func(arg commandOptions) error { called = true; return nil })
	if called || !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("help called=%v err=%v", called, err)
	}
}

func TestExactlyOneImageMaintenanceActionBeforeExecution(t *testing.T) {
	flags := []string{"-grant-image-agent-runtime", "-grant-image-agent-worker-runtime", "-initialize-organization-image-agent", "-prepare-ledger-only-organization-image-agent"}
	for mask := 0; mask < 1<<len(flags); mask++ {
		args := []string{"-config", "private-fixture.yaml", "-current-application-manifest", "current-fixture.json"}
		selected := 0
		for i, name := range flags {
			if mask&(1<<i) != 0 {
				args = append(args, name)
				selected++
			}
		}
		called := false
		sentinel := errors.New("synthetic execution result")
		err := runCommand(args, func(opts commandOptions) error {
			called = true
			if opts.configPath != "private-fixture.yaml" || opts.currentManifest != "current-fixture.json" {
				t.Fatal("private input changed")
			}
			got := []bool{opts.grantImageAgentRuntime, opts.grantImageAgentWorkerRuntime, opts.initOrganizationImageAgent, opts.prepareLedgerOnly}
			for i, enabled := range got {
				if enabled != (mask&(1<<i) != 0) {
					t.Fatalf("action %d changed", i)
				}
			}
			return sentinel
		})
		if selected == 1 {
			if !called || !errors.Is(err, sentinel) {
				t.Fatalf("mask=%d called=%v err=%v", mask, called, err)
			}
		} else if called || err == nil || !strings.Contains(err.Error(), "exactly one") {
			t.Fatalf("mask=%d called=%v err=%v", mask, called, err)
		}
	}
}
