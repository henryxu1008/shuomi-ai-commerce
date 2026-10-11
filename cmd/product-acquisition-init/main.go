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
	featureDatabase := flag.String("confirm-database", "", "exact existing dedicated Product database name for feature initialization")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var err error
	if *feature {
		if *confirmed != "" {
			err = fmt.Errorf("empty-database confirmation cannot be combined with feature initialization")
		} else {
			err = productsourcing.EnableAcquisitionPlans(ctx, *manifest, *featureDatabase)
		}
	} else {
		if *featureDatabase != "" {
			err = fmt.Errorf("feature confirmation requires enable-acquisition-plans")
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
