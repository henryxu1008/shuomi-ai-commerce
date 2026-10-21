// agent-configuration-schema-init installs the admitted fresh-install schema in
// the existing ProductAgentDB. Serving never performs DDL or auto-enables agents.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"regexp"
	store "task-processor/internal/integration/persistence/agentconfig"
	routestore "task-processor/internal/integration/persistence/routeconfig"
	skillstore "task-processor/internal/integration/persistence/skill"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	role := flag.String("runtime-role", "", "existing runtime PostgreSQL role to grant bounded privileges")
	modesOnly := flag.Bool("execution-modes-only", false, "add only execution mode tables and their bounded grants; retain existing facts and grants")
	contentOnly := flag.Bool("content-configuration-only", false, "add only the content parameter carrier and exact checks; retain existing facts, activations and grants")
	materialsOnly := flag.Bool("materials-configuration-only", false, "add only exact registered platform materials checks; retain existing facts, activations and grants")
	skillsOnly := flag.Bool("skills-only", false, "add bounded Skill storage and content personal-Skill admission; do not enable agents or import templates")
	artworkOnly := flag.Bool("artwork-configuration-only", false, "add only exact POD creative configuration checks; retain existing facts, activations and grants")
	modelRoutesOnly := flag.Bool("model-route-pricing-only", false, "install only route revision/current and credential metadata projection; no current config or credential changes")
	writerRole := flag.String("model-route-writer-role", "", "existing dedicated bounded model route writer")
	writerConnections := flag.Int("model-route-writer-connections", 0, "explicit existing writer role connection limit: 2 or 4; omitted leaves unchanged")
	flag.Parse()
	selected := 0
	for _, v := range []bool{*modesOnly, *contentOnly, *materialsOnly, *skillsOnly, *artworkOnly, *modelRoutesOnly} {
		if v {
			selected++
		}
	}
	if (*modelRoutesOnly && *writerRole != "model_route_config_writer") || (!*modelRoutesOnly && (*writerRole != "" || *writerConnections != 0)) || (*writerConnections != 0 && *writerConnections != 2 && *writerConnections != 4) || *writerRole == *role || flag.NArg() != 0 || selected > 1 || !regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`).MatchString(*role) {
		fmt.Fprintln(os.Stderr, "requires --runtime-role with an existing role")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dsn := os.Getenv("AGENT_CONFIGURATION_SCHEMA_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "AGENT_CONFIGURATION_SCHEMA_DSN required")
		os.Exit(2)
	}
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		fmt.Fprintln(os.Stderr, "schema database unavailable")
		os.Exit(1)
	}
	raw, e := db.DB()
	if e != nil {
		os.Exit(1)
	}
	defer raw.Close()
	e = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if *modelRoutesOnly {
			if e := routestore.ConfigureWriterConnections(ctx, tx, *writerConnections); e != nil {
				return e
			}
			if e := routestore.InstallSchema(ctx, tx); e != nil {
				return e
			}
			return tx.Exec(routestore.GrantSQL(*role, false) + routestore.GrantSQL(*writerRole, true)).Error
		}
		if *skillsOnly {
			if e := skillstore.InstallSchema(ctx, tx); e != nil {
				return e
			}
			return skillstore.GrantRuntime(ctx, tx, *role)
		}
		if *artworkOnly {
			return store.UpdateArtworkConfigurationSchema(ctx, tx, *role)
		}
		if *materialsOnly {
			return store.UpdateMaterialsConfigurationSchema(ctx, tx, *role)
		}
		if *contentOnly {
			return store.UpdateContentConfigurationSchema(ctx, tx, *role)
		}
		if *modesOnly {
			return store.InstallExecutionModes(tx, *role)
		}
		if e := store.InstallSchema(tx); e != nil {
			return e
		}
		if e := store.GrantRuntime(tx, *role); e != nil {
			return e
		}
		if e := skillstore.InstallSchema(ctx, tx); e != nil {
			return e
		}
		return skillstore.GrantRuntime(ctx, tx, *role)
	})
	if e != nil {
		fmt.Fprintln(os.Stderr, "agent configuration schema initialization failed")
		os.Exit(1)
	}
	fmt.Println("agent configuration schema ready; no enterprise activation created")
}
