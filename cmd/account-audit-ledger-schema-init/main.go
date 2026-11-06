// Installs the existing invocation ledger in one fresh Agent owner database.
// Serving reads it through a different, SELECT-only role.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	store "task-processor/internal/aicapability/store"
)

func main() {
	path := flag.String("dsn-file", "", "private owner DSN file")
	namespace := flag.String("namespace", "", "existing invocation owner: image or product")
	confirmed := flag.String("confirm-empty-database", "", "exact other fresh product database; refuses existing objects")
	flag.Parse()
	if flag.NArg() != 0 || *path == "" || (*namespace != "image" && *namespace != "product") {
		fmt.Fprintln(os.Stderr, "bounded invocation owner and private DSN file required")
		os.Exit(2)
	}
	emptyMode := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "confirm-empty-database" {
			emptyMode = true
		}
	})
	if emptyMode && (*namespace != "product" || *confirmed == "product_agent" || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(*confirmed)) {
		fmt.Fprintln(os.Stderr, "invalid confirmed empty database")
		os.Exit(2)
	}
	data, err := os.ReadFile(*path)
	if err != nil || len(data) > 4096 || !strings.HasPrefix(strings.TrimSpace(string(data)), "postgresql://") {
		fmt.Fprintln(os.Stderr, "owner DSN unavailable")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := gorm.Open(postgres.Open(strings.TrimSpace(string(data))), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "ledger owner unavailable")
		os.Exit(1)
	}
	raw, err := db.DB()
	if err != nil {
		os.Exit(1)
	}
	defer raw.Close()
	raw.SetMaxOpenConns(1)
	if err := raw.PingContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "ledger owner unavailable")
		os.Exit(1)
	}
	if emptyMode {
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return installEmptyProductLedger(ctx, tx, *confirmed) }); err != nil {
			fmt.Fprintln(os.Stderr, "ledger schema initialization failed")
			fmt.Fprintln(os.Stderr, "retain target; verify transaction outcome before any retry")
			os.Exit(1)
		}
		return
	}
	var databaseName, roleName string
	if err := db.WithContext(ctx).Raw("SELECT current_database(), current_user").Row().Scan(&databaseName, &roleName); err != nil || databaseName != *namespace+"_agent" || roleName != *namespace+"_agent_owner" {
		fmt.Fprintln(os.Stderr, "wrong invocation ledger owner")
		os.Exit(1)
	}
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return store.AutoMigrateInvocationLedger(tx) }); err != nil {
		fmt.Fprintln(os.Stderr, "ledger schema initialization failed")
		os.Exit(1)
	}
}

// The explicit new-database mode never qualifies or repairs an existing ledger.
// All qualification and DDL share the original bounded transaction and owner.
func installEmptyProductLedger(ctx context.Context, tx *gorm.DB, confirmed string) error {
	refused := errors.New("confirmed empty product target unavailable")
	var admitted bool
	if err := tx.Raw(`SELECT current_database() = ? AND current_user = 'product_agent_owner'
		AND session_user = 'product_agent_owner'
		AND pg_catalog.pg_get_userbyid(d.datdba) = 'product_agent_owner'
		AND NOT (r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
		FROM pg_catalog.pg_database d JOIN pg_catalog.pg_roles r ON r.rolname = current_user
		WHERE d.datname = current_database()`, confirmed).Scan(&admitted).Error; err != nil || !admitted {
		return refused
	}
	for _, setting := range []string{"SET LOCAL search_path=public,pg_catalog", "SET LOCAL lock_timeout='5s'", "SET LOCAL statement_timeout='25s'"} {
		if err := tx.Exec(setting).Error; err != nil {
			return refused
		}
	}
	var empty bool
	if err := tx.Raw(`SELECT current_schema() = 'public'
		AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace WHERE left(nspname,3) <> 'pg_' AND nspname NOT IN ('public','information_schema'))
		AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
			WHERE left(n.nspname,3) <> 'pg_' AND n.nspname <> 'information_schema' AND c.relkind IN ('r','p','v','m','f','S'))`).Scan(&empty).Error; err != nil || !empty {
		return refused
	}
	if err := store.AutoMigrateInvocationLedger(tx); err != nil {
		return refused
	}
	if err := store.VerifyModelPointInvocationSchema(ctx, tx); err != nil {
		return refused
	}
	var owned bool
	if err := tx.Raw(`SELECT pg_catalog.pg_get_userbyid(relowner) = 'product_agent_owner' FROM pg_catalog.pg_class WHERE oid = 'public.ai_invocations'::regclass`).Scan(&owned).Error; err != nil || !owned {
		return refused
	}
	return nil
}
