package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	schemainit "task-processor/internal/app/runtime/sourceaccountregistryschemainit"
)

func runCommand(args []string, execute func(string) error) error {
	fs := flag.NewFlagSet("source-account-registry-schema-init", flag.ContinueOnError)
	configPath := fs.String("config", "", "database config file path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*configPath) == "" {
		return fmt.Errorf("-config is required; provide an explicit private configuration file")
	}
	return execute(*configPath)
}

func main() {
	err := runCommand(os.Args[1:], func(path string) error {
		if err := schemainit.Run(context.Background(), path); err != nil {
			return err
		}
		fmt.Printf("source account registry schema initialized using %s\n", path)
		return nil
	})
	if err != nil && !errors.Is(err, flag.ErrHelp) {
		_, _ = fmt.Fprintf(os.Stderr, "initialize source account registry schema: %v\n", err)
		os.Exit(1)
	}
}
