// browser-kernel-schema-init installs a fresh, independently owned metadata
// schema. It is never invoked by the application or deployment startup.
package main

import (
	"context"
	"flag"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	store "task-processor/internal/integration/persistence/browserkernel"
	"time"
)

func main() {
	role := flag.String("runtime-role", "", "existing browser_kernel_runtime role")
	flag.Parse()
	if flag.NArg() != 0 || *role != "browser_kernel_runtime" {
		fmt.Fprintln(os.Stderr, "requires --runtime-role browser_kernel_runtime")
		os.Exit(2)
	}
	dsn := os.Getenv("BROWSER_KERNEL_SCHEMA_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "BROWSER_KERNEL_SCHEMA_DSN required for the fresh browser_kernel database")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	raw, e := db.DB()
	if e != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	defer raw.Close()
	e = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if e := store.InstallSchema(tx); e != nil {
			return e
		}
		return store.GrantRuntime(tx, *role)
	})
	if e != nil {
		fmt.Fprintln(os.Stderr, "fresh browser kernel schema initialization failed")
		os.Exit(1)
	}
	fmt.Println("browser kernel schema ready; no release metadata or artifacts published")
}
