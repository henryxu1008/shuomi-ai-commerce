package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"task-processor/internal/app/modelinvocationrecovery"
)

func main() {
	path := flag.String("config", "", "absolute protected configuration path")
	action := flag.String("action", "inspect", "inspect or explicitly approved resolve")
	flag.Parse()
	if *path == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "a protected -config is required")
		os.Exit(2)
	}
	ctx := context.Background()
	cfg, err := modelinvocationrecovery.LoadConfig(ctx, *path)
	var result modelinvocationrecovery.Output
	if err == nil {
		result, err = modelinvocationrecovery.Run(ctx, cfg, *action)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(result) != nil {
		os.Exit(1)
	}
}
