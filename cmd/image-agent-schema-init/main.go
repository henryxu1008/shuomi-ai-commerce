package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	imageagentschemainit "task-processor/internal/app/runtime/imageagentschemainit"
)

type commandOptions struct {
	configPath, currentManifest                                                                         string
	grantImageAgentRuntime, grantImageAgentWorkerRuntime, initOrganizationImageAgent, prepareLedgerOnly bool
}

func runCommand(args []string, execute func(commandOptions) error) error {
	fs := flag.NewFlagSet("image-agent-schema-init", flag.ContinueOnError)
	configPath := fs.String("config", "", "config file path")
	grantImageAgentRuntime := fs.Bool("grant-image-agent-runtime", false, "grant only the pre-provisioned image_agent_runtime role on the existing ImageAgent owner database; no schema migration")
	grantImageAgentWorkerRuntime := fs.Bool("grant-image-agent-worker-runtime", false, "grant only the pre-provisioned image_agent_worker_runtime role on the existing ImageAgent owner database; no schema migration")
	initOrganizationImageAgent := fs.Bool("initialize-organization-image-agent", false, "initialize only a new empty Organization ImageAgent owner database; requires explicit owner config and current manifest")
	prepareLedgerOnly := fs.Bool("prepare-ledger-only-organization-image-agent", false, "explicitly install missing current Image/Asset tables beside a verified current invocation ledger without changing it")
	currentManifest := fs.String("current-application-manifest", "", "absolute private current-application manifest required with -grant-image-agent-runtime")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*configPath) == "" {
		return fmt.Errorf("-config is required; provide an explicit private configuration file")
	}
	selected := 0
	for _, enabled := range []bool{*grantImageAgentRuntime, *grantImageAgentWorkerRuntime, *initOrganizationImageAgent, *prepareLedgerOnly} {
		if enabled {
			selected++
		}
	}
	if selected != 1 {
		return fmt.Errorf("exactly one image agent maintenance operation is required")
	}
	return execute(commandOptions{configPath: *configPath, currentManifest: *currentManifest, grantImageAgentRuntime: *grantImageAgentRuntime, grantImageAgentWorkerRuntime: *grantImageAgentWorkerRuntime, initOrganizationImageAgent: *initOrganizationImageAgent, prepareLedgerOnly: *prepareLedgerOnly})
}

func executeCommand(opts commandOptions) error {
	if opts.prepareLedgerOnly {
		if err := imageagentschemainit.PrepareLedgerOnlyOrganizationImageAgent(context.Background(), opts.configPath, opts.currentManifest); err != nil {
			return fmt.Errorf("prepare current image agent beside retained ledger: %v", err)
		}
		fmt.Println("current organization image tables prepared; invocation ledger preserved")
		return nil
	}
	if opts.initOrganizationImageAgent {
		if err := imageagentschemainit.InitializeOrganizationImageAgent(context.Background(), opts.configPath, opts.currentManifest); err != nil {
			return fmt.Errorf("initialize organization image agent owner: %v", err)
		}
		fmt.Println("organization image agent owner initialized")
		return nil
	}

	if opts.grantImageAgentRuntime {
		if err := imageagentschemainit.GrantImageAgentRuntime(context.Background(), opts.configPath, opts.currentManifest); err != nil {
			return fmt.Errorf("grant image agent runtime permissions: %v", err)
		}
		fmt.Printf("image agent runtime permissions granted using %s\n", opts.configPath)
		return nil
	}
	if opts.grantImageAgentWorkerRuntime {
		if err := imageagentschemainit.GrantImageAgentWorkerRuntime(context.Background(), opts.configPath, opts.currentManifest); err != nil {
			return fmt.Errorf("grant image agent worker runtime permissions: %v", err)
		}
		fmt.Printf("image agent worker runtime permissions granted using %s\n", opts.configPath)
		return nil
	}
	return errors.New("exactly one image agent maintenance operation is required")
}

func main() {
	if err := runCommand(os.Args[1:], executeCommand); err != nil && !errors.Is(err, flag.ErrHelp) {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
