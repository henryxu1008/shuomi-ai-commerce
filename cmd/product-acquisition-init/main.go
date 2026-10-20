package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"task-processor/internal/app/productsourcing"
)

func main() {
	manifest := flag.String("config", "", "absolute private initialization manifest path")
	confirmed := flag.String("confirm-empty-database", "", "exact pre-created empty database name")
	feature := flag.Bool("enable-acquisition-plans", false, "explicitly add current acquisition-plan and preparation tables; preserve existing Product facts")
	completion := flag.Bool("enable-completion-drafts", false, "explicitly add unbound completion drafts and minimum grants; preserve existing Product facts")
	supply := flag.Bool("enable-supply-chain", false, "explicitly add current supply tables and exact grants to a confirmed nonempty Product database")
	featureDatabase := flag.String("confirm-database", "", "exact existing dedicated Product database name for feature initialization")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var err error
	features := 0
	for _, enabled := range []bool{*feature, *completion, *supply} {
		if enabled {
			features++
		}
	}
	if flag.NArg() != 0 {
		err = fmt.Errorf("unexpected positional arguments")
	} else if features > 1 {
		err = fmt.Errorf("feature initialization switches are mutually exclusive")
	} else if features == 1 {
		if *confirmed != "" {
			err = fmt.Errorf("empty-database confirmation cannot be combined with feature initialization")
		} else if *supply {
			err = productsourcing.EnableSupplyChain(ctx, *manifest, *featureDatabase)
		} else if *completion {
			err = productsourcing.EnableCompletionDrafts(ctx, *manifest, *featureDatabase)
		} else {
			err = productsourcing.EnableAcquisitionPlans(ctx, *manifest, *featureDatabase)
		}
	} else {
		if *featureDatabase != "" {
			err = fmt.Errorf("feature confirmation requires a feature initialization switch")
		} else {
			err = productsourcing.InitializeAcquisitionDatabase(ctx, *manifest, *confirmed)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("Product acquisition schema and runtime grants initialized")
}
