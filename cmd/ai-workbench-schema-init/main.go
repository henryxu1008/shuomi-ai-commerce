// ai-workbench-schema-init installs the fresh-install Workbench schema in the
// Product Agent database, then grants the pre-created bounded serving role.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	store "task-processor/internal/integration/persistence/aiworkbench"
)

func main() {
	role := flag.String("runtime-role", "", "existing restricted PostgreSQL runtime role")
	modeOnly := flag.Bool("execution-mode-only", false, "add only nullable execution-mode metadata; retain historical proposals and grants")
	emptyScopeOnly := flag.Bool("empty-scope-only", false, "upgrade only the recognized empty Workbench baseline")
	expectedDB := flag.String("expected-database", "", "explicit approved database name")
	expectedAddress := flag.String("expected-server-address", "", "explicit approved PostgreSQL server IP")
	expectedPort := flag.Int("expected-server-port", 0, "explicit approved PostgreSQL server port")
	flag.Parse()
	if flag.NArg() != 0 || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(*role) {
		fmt.Fprintln(os.Stderr, "requires --runtime-role with an existing role")
		os.Exit(2)
	}
	dsn := os.Getenv("AI_WORKBENCH_SCHEMA_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "AI_WORKBENCH_SCHEMA_DSN required")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	raw, err := db.DB()
	if err != nil {
		os.Exit(1)
	}
	defer raw.Close()
	if *emptyScopeOnly {
		if *modeOnly || *role != "ai_workbench_runtime" || *expectedDB == "" || *expectedAddress == "" || *expectedPort <= 0 {
			fmt.Fprintln(os.Stderr, "empty-scope mode requires exact target and ai_workbench_runtime role")
			os.Exit(2)
		}
		runtimeDSN := os.Getenv("AI_WORKBENCH_RUNTIME_DSN")
		if runtimeDSN == "" {
			fmt.Fprintln(os.Stderr, "AI_WORKBENCH_RUNTIME_DSN required")
			os.Exit(2)
		}
		runtime, openErr := gorm.Open(postgres.Open(runtimeDSN), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if openErr != nil {
			fmt.Fprintln(os.Stderr, "runtime database unavailable")
			os.Exit(1)
		}
		runtimeRaw, openErr := runtime.DB()
		if openErr != nil {
			os.Exit(1)
		}
		defer runtimeRaw.Close()
		err = store.InstallEmptyScope(ctx, db, runtime, store.EmptyScopeTarget{Database: *expectedDB, Address: *expectedAddress, Port: *expectedPort})
		if err != nil {
			fmt.Fprintln(os.Stderr, "empty-scope preparation not verified; do not enable Chat, inspect committed state read-only")
			os.Exit(1)
		}
		fmt.Println("AI Workbench schema verified as runtime; no conversation or task created")
		return
	}
	if *expectedDB != "" || *expectedAddress != "" || *expectedPort != 0 {
		fmt.Fprintln(os.Stderr, "target flags require --empty-scope-only")
		os.Exit(2)
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if *modeOnly {
			return store.InstallObservedExecutionMode(tx, *role)
		}
		if err := store.InstallSchema(tx); err != nil {
			return err
		}
		return store.GrantRuntime(tx, *role)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "AI Workbench schema initialization failed")
		os.Exit(1)
	}
	fmt.Println("AI Workbench schema ready; no conversation or task created")
}
