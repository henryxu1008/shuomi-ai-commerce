// product-agent-runtime-prepare qualifies and prepares fixed native targets and
// explicit schema capabilities. It never serves or reads provider credentials.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/app/schema/productagentruntime"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	target := flag.String("target", "", "fixed target: run, review, invocation-provider-rejection, invocation-billing-mode, managed-model-channels, owned-grsai-product-image, or owned-toapis-product-image")
	flag.Parse()
	env := "PRODUCT_AGENT_SCHEMA_DSN"
	if flag.NArg() != 0 || (*target != "run" && *target != "review" && *target != "invocation-provider-rejection" && *target != "invocation-billing-mode" && *target != "managed-model-channels" && *target != "owned-grsai-product-image" && *target != "owned-toapis-product-image") {
		fmt.Fprintln(os.Stderr, "requires --target run, review, invocation-provider-rejection, invocation-billing-mode, managed-model-channels, owned-grsai-product-image, or owned-toapis-product-image")
		os.Exit(2)
	}
	if *target == "review" {
		env = "PRODUCT_REVIEW_SCHEMA_DSN"
	}
	dsn := os.Getenv(env)
	if dsn == "" {
		fmt.Fprintln(os.Stderr, env+" required through private environment")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableAutomaticPing: true})
	if err != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	pool, err := db.DB()
	if err != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	defer pool.Close()
	pool.SetMaxOpenConns(1)
	pool.SetMaxIdleConns(1)
	if err = pool.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	var result productagentruntime.Result
	if *target == "run" {
		result, err = productagentruntime.PrepareRun(ctx, db)
	} else if *target == "managed-model-channels" {
		result, err = productagentruntime.PrepareManagedModelChannels(ctx, db)
	} else if *target == "owned-grsai-product-image" {
		result, err = productagentruntime.PrepareOwnedGRSaiProductImage(ctx, db)
	} else if *target == "owned-toapis-product-image" {
		result, err = productagentruntime.PrepareOwnedToAPIsProductImage(ctx, db)
	} else if *target == "invocation-provider-rejection" {
		result, err = productagentruntime.PrepareProviderRejectionProof(ctx, db)
	} else if *target == "invocation-billing-mode" {
		result, err = productagentruntime.PrepareInvocationBillingMode(ctx, db)
	} else {
		result, err = productagentruntime.PrepareReview(ctx, db)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "native preparation failed:", err)
		fmt.Fprintln(os.Stderr, "retain current configuration; verify transaction outcome before any retry")
		os.Exit(1)
	}
	if json.NewEncoder(os.Stdout).Encode(result) != nil {
		os.Exit(1)
	}
}
