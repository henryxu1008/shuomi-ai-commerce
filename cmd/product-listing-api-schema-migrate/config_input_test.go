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
	err := runCommand([]string{"-config", "private-fixture.yaml"}, func(arg commandOptions) error {
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
