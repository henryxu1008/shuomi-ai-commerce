package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"task-processor/internal/app/localaitrial"
)

func main() {
	config := flag.String("config", "", "absolute private trial configuration path")
	operation := flag.String("operation-id", "", "stable operation ID retained for replay and recovery")
	action := flag.String("action", "inspect", "inspect (read only) or grant (requires explicit approval)")
	flag.Parse()
	if *config == "" || *operation == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "-config and -operation-id are required; positional arguments are forbidden")
		os.Exit(2)
	}
	ctx := context.Background()
	cfg, err := localaitrial.LoadConfig(ctx, *config)
	if err == nil {
		var result localaitrial.Output
		result, err = localaitrial.Run(ctx, cfg, *operation, *action)
		if err == nil {
			err = json.NewEncoder(os.Stdout).Encode(result)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
