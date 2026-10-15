// product-agent-runtime-prepare qualifies and prepares the two fixed native
// targets. It never runs serving, creates roles, or reads provider credentials.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/app/schema/productagentruntime"
)

func main() {
	target := flag.String("target", "", "fixed target: run or review")
	flag.Parse()
	env := "PRODUCT_AGENT_SCHEMA_DSN"
	if flag.NArg() != 0 || (*target != "run" && *target != "review") {
		fmt.Fprintln(os.Stderr, "requires exactly --target run or --target review")
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
