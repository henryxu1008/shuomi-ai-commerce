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

	store "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/aiworkbench"
)

func main() {
	role := flag.String("runtime-role", "", "existing restricted PostgreSQL runtime role")
	modeOnly := flag.Bool("execution-mode-only", false, "add only nullable execution-mode metadata; retain historical proposals and grants")
	dailyOnly := flag.Bool("store-daily-report-only", false, "install the bounded deterministic report extension; retain all existing plans and grants")
	deletionOnly := flag.Bool("schedule-deletion-only", false, "add only the scheduled-plan deletion marker and its bounded runtime grant")
	editPermissions := flag.String("schedule-edit-permissions", "", "enable or disable only installed scheduled-plan editing column grants")
	authoringOnly := flag.Bool("skill-authoring-only", false, "extend only the installed personal skill authoring table and its bounded grants")
	chatTurnsOnly := flag.Bool("chat-turn-only", false, "install only Chat skill handoff references and bounded grants; retain existing requests")
	emptyScopeOnly := flag.Bool("empty-scope-only", false, "upgrade only the recognized empty Workbench baseline")
	expectedDB := flag.String("expected-database", "", "explicit approved database name")
	expectedAddress := flag.String("expected-server-address", "", "explicit approved PostgreSQL server IP")
	expectedPort := flag.Int("expected-server-port", 0, "explicit approved PostgreSQL server port")
	flag.Parse()
	selected := 0
	for _, enabled := range []bool{*modeOnly, *dailyOnly, *deletionOnly, *emptyScopeOnly, *authoringOnly, *chatTurnsOnly, *editPermissions != ""} {
		if enabled {
			selected++
		}
	}
	if selected > 1 || *editPermissions != "" && *editPermissions != "enable" && *editPermissions != "disable" {
		fmt.Fprintln(os.Stderr, "select one explicit owner operation")
		os.Exit(2)
	}
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
	if *authoringOnly || *chatTurnsOnly {
		if *role != "ai_workbench_runtime" || *expectedDB == "" || *expectedAddress == "" || *expectedPort < 1 {
			fmt.Fprintln(os.Stderr, "skill authoring upgrade requires exact target and ai_workbench_runtime")
			os.Exit(2)
		}
		var matches bool
		if db.WithContext(ctx).Raw("SELECT current_database()=? AND host(inet_server_addr())=? AND inet_server_port()=?", *expectedDB, *expectedAddress, *expectedPort).Scan(&matches).Error != nil || !matches {
			fmt.Fprintln(os.Stderr, "skill authoring target does not match")
			os.Exit(1)
		}
		install := store.InstallSkillAuthoringUpgrade
		if *chatTurnsOnly {
			install = store.InstallChatTurnSchema
		}
		if err := install(ctx, db, *role); err != nil {
			fmt.Fprintln(os.Stderr, "skill authoring upgrade not installed")
			os.Exit(1)
		}
		fmt.Println("Selected Chat schema extension verified; existing drafts and receipts retained")
		return
	}
	if *editPermissions != "" {
		if *role != "ai_workbench_runtime" || *expectedDB == "" || *expectedAddress == "" || *expectedPort < 1 {
			fmt.Fprintln(os.Stderr, "scheduled editing permissions require exact target and ai_workbench_runtime")
			os.Exit(2)
		}
		var matches bool
		if db.WithContext(ctx).Raw("SELECT current_database()=? AND inet_server_addr()::text=? AND inet_server_port()=?", *expectedDB, *expectedAddress, *expectedPort).Scan(&matches).Error != nil || !matches {
			fmt.Fprintln(os.Stderr, "scheduled editing target does not match")
			os.Exit(1)
		}
		if err = store.SetScheduledEditPermissions(db.WithContext(ctx), *role, *editPermissions == "enable"); err != nil {
			fmt.Fprintln(os.Stderr, "scheduled editing permissions not changed")
			os.Exit(1)
		}
		fmt.Println("Scheduled editing permissions verified; plans, tasks and results unchanged")
		return
	}
	if *deletionOnly {
		if *modeOnly || *dailyOnly || *emptyScopeOnly || *role != "ai_workbench_runtime" || *expectedDB == "" || *expectedAddress == "" || *expectedPort < 1 {
			fmt.Fprintln(os.Stderr, "scheduled deletion requires exact target and existing ai_workbench_runtime role")
			os.Exit(2)
		}
		var matches bool
		if db.WithContext(ctx).Raw("SELECT current_database()=? AND inet_server_addr()::text=? AND inet_server_port()=?", *expectedDB, *expectedAddress, *expectedPort).Scan(&matches).Error != nil || !matches {
			fmt.Fprintln(os.Stderr, "scheduled deletion target does not match")
			os.Exit(1)
		}
		if err = store.InstallScheduledDeletionSchema(db.WithContext(ctx), *role); err != nil {
			fmt.Fprintln(os.Stderr, "scheduled deletion extension not installed")
			os.Exit(1)
		}
		fmt.Println("Scheduled deletion extension installed; existing plans, tasks and results retained")
		return
	}
	if *dailyOnly {
		if *modeOnly || *emptyScopeOnly || *role != "ai_workbench_runtime" || *expectedDB == "" || *expectedAddress == "" || *expectedPort < 1 {
			fmt.Fprintln(os.Stderr, "daily-report extension requires exact target and existing ai_workbench_runtime role")
			os.Exit(2)
		}
		var matches bool
		if db.WithContext(ctx).Raw("SELECT current_database()=? AND inet_server_addr()::text=? AND inet_server_port()=?", *expectedDB, *expectedAddress, *expectedPort).Scan(&matches).Error != nil || !matches {
			fmt.Fprintln(os.Stderr, "daily-report target does not match")
			os.Exit(1)
		}
		err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if e := store.InstallDailyReportSchema(tx); e != nil {
				return e
			}
			return store.GrantDailyReportRuntime(tx, *role)
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "daily-report extension not installed")
			os.Exit(1)
		}
		fmt.Println("Daily-report extension installed; existing plans and tasks unchanged; serving readiness must verify both owners")
		return
	}
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
		fmt.Println("Exact empty-scope preparation verified; no conversation or task created; current full serving schema readiness is not implied")
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
