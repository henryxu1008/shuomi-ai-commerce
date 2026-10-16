package listingkitschemamigrate

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"task-processor/internal/core/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	_ "modernc.org/sqlite"
)

func TestRetiredTaskSchemaScopesRejectBeforeConfigurationOrDatabaseAccess(t *testing.T) {
	for _, scope := range []string{"", "all", "shein-sync"} {
		t.Run(scope, func(t *testing.T) {
			loaded := false
			err := runWithDependencies(context.Background(), Options{Scope: scope}, runtimeDependencies{
				LoadConfig: func(string) (*config.Config, error) {
					loaded = true
					return nil, errors.New("configuration must not be read")
				},
			})
			if loaded || err == nil || !strings.Contains(err.Error(), "retired") {
				t.Fatalf("legacy scope %q: loaded=%v err=%v; want retirement rejection before I/O", scope, loaded, err)
			}
		})
	}
}

func TestDefaultLoaderAcceptsDatabaseOnlyConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "database-only.yaml")
	contents := []byte("database:\n  host: database.internal\n  port: 5432\n  user: listingkit\n  password: test-only\n  database: listingkit\n")
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := defaultRuntimeDependencies().LoadConfig(path)
	if err != nil {
		t.Fatalf("load database-only config: %v", err)
	}
	if cfg.Database == nil || cfg.Database.Host != "database.internal" {
		t.Fatalf("database config = %#v", cfg.Database)
	}
}

func TestResolveConfigPathAndParseFlags(t *testing.T) {
	if got := ResolveConfigPath(""); got != "config/config-dev.yaml" {
		t.Fatalf("default config path = %q", got)
	}
	if got := ResolveConfigPath("config/custom.yaml"); got != "config/custom.yaml" {
		t.Fatalf("config path precedence = %q", got)
	}

	fs := flag.NewFlagSet("listingkit-schema-migrate", flag.ContinueOnError)
	opts := ParseFlagsFrom(fs,
		"--config", "config/runtime.yaml",
		"--log-level", "debug",
		"--scope", "workbench",
	)
	if opts.Config != "config/runtime.yaml" || opts.LogLevel != "debug" || opts.Scope != "workbench" {
		t.Fatalf("unexpected parsed options: %+v", opts)
	}
}

func TestRunDispatchesCommercialScopeToCommercialDatabase(t *testing.T) {
	commercial := &gorm.DB{}
	openedCommercial, migratedCommercial, closedCommercial := false, false, false
	err := runWithDependencies(context.Background(), Options{Config: "config/test.yaml", LogLevel: "error", Scope: "commercial"}, runtimeDependencies{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{CommercialDatabase: &config.DatabaseConfig{}}, nil
		},
		OpenDB: func(*config.DatabaseConfig) (*gorm.DB, error) {
			t.Fatal("primary database must not be opened for commercial scope")
			return nil, nil
		},
		OpenCommercial: func(*config.DatabaseConfig) (*gorm.DB, error) {
			openedCommercial = true
			return commercial, nil
		},
		CloseDB: func(got *gorm.DB) error {
			if got != commercial {
				t.Fatal("closed unexpected database")
			}
			closedCommercial = true
			return nil
		},
		MigrateCommercial: func(got *gorm.DB) error {
			if got != commercial {
				t.Fatal("migrated unexpected database")
			}
			migratedCommercial = true
			return nil
		},
	})
	if err != nil {
		t.Fatalf("runWithDependencies returned error: %v", err)
	}
	if !openedCommercial || !migratedCommercial || !closedCommercial {
		t.Fatalf("expected commercial open/migrate/close, got open=%v migrate=%v close=%v", openedCommercial, migratedCommercial, closedCommercial)
	}
}

func TestMigrationScopeDispatchesExactMigratorsInOrder(t *testing.T) {
	db := &gorm.DB{}
	for _, test := range []struct {
		name      string
		scope     string
		wantOrder []string
		wantErr   error
	}{
		{name: "workbench", scope: "workbench", wantOrder: []string{"workbench"}},
		{name: "unknown", scope: "unknown", wantErr: flag.ErrHelp},
	} {
		t.Run(test.name, func(t *testing.T) {
			var order []string
			err := runMigration(db, test.scope, runtimeDependencies{

				MigrateWorkbench: func(got *gorm.DB) error {
					if got != db {
						t.Fatal("MigrateWorkbench received a different database")
					}
					order = append(order, "workbench")
					return nil
				},
			})
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("runMigration() error = %v, want %v", err, test.wantErr)
			}
			if !reflect.DeepEqual(order, test.wantOrder) {
				t.Fatalf("migration order = %v, want %v", order, test.wantOrder)
			}
		})
	}
}

func TestMigrationUnknownScopeReturnsHelpBeforeAnyRuntimeSideEffect(t *testing.T) {
	calls := make([]string, 0, 6)
	err := runWithDependencies(context.Background(), Options{Scope: "unknown"}, runtimeDependencies{
		LoadConfig: func(string) (*config.Config, error) {
			calls = append(calls, "load-config")
			return &config.Config{Database: &config.DatabaseConfig{}}, nil
		},
		OpenDB: func(*config.DatabaseConfig) (*gorm.DB, error) {
			calls = append(calls, "open-db")
			return &gorm.DB{}, nil
		},
		CloseDB: func(*gorm.DB) error {
			calls = append(calls, "close-db")
			return nil
		},

		MigrateWorkbench: func(*gorm.DB) error {
			calls = append(calls, "migrate-workbench")
			return nil
		},
	})
	if !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("runWithDependencies() error = %v, want flag.ErrHelp", err)
	}
	if len(calls) != 0 {
		t.Fatalf("unknown scope caused runtime side effects: %v", calls)
	}
}

func TestMigrationScopeWorkbenchClosesDatabaseOnMigrationFailure(t *testing.T) {
	db := &gorm.DB{}
	wantErr := errors.New("workbench migration failed")
	closed := false
	err := runWithDependencies(context.Background(), Options{Config: "config/test.yaml", LogLevel: "error", Scope: "workbench"}, runtimeDependencies{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{Database: &config.DatabaseConfig{}}, nil
		},
		OpenDB: func(*config.DatabaseConfig) (*gorm.DB, error) { return db, nil },
		CloseDB: func(got *gorm.DB) error {
			if got != db {
				t.Fatal("closed a different database")
			}
			closed = true
			return nil
		},

		MigrateWorkbench: func(*gorm.DB) error { return wantErr },
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("runWithDependencies() error = %v, want wrapped %v", err, wantErr)
	}
	if !closed {
		t.Fatal("database was not closed after Workbench migration failure")
	}
}

func TestMigrationScopeWorkbenchDefaultsToOwnedMigrator(t *testing.T) {
	db, err := gorm.Open(sqlite.Dialector{DriverName: "sqlite", DSN: ":memory:"}, &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	err = runWithDependencies(context.Background(), Options{Config: "config/test.yaml", LogLevel: "error", Scope: "workbench"}, runtimeDependencies{
		LoadConfig: func(string) (*config.Config, error) {
			return &config.Config{Database: &config.DatabaseConfig{}}, nil
		},
		OpenDB:  func(*config.DatabaseConfig) (*gorm.DB, error) { return db, nil },
		CloseDB: func(*gorm.DB) error { return nil },
	})
	if err != nil {
		t.Fatalf("runWithDependencies() error = %v", err)
	}
	if !db.Migrator().HasTable("workbench_stores") {
		t.Fatal("default Workbench migrator did not create the owned schema")
	}
}

func TestMigrationScopeFlagHelpAdvertisesAllSupportedScopes(t *testing.T) {
	var output bytes.Buffer
	fs := flag.NewFlagSet("listingkit-schema-migrate", flag.ContinueOnError)
	fs.SetOutput(&output)
	ParseFlagsFrom(fs, "--help")
	fs.PrintDefaults()
	for _, scope := range []string{"commercial", "workbench"} {
		if !strings.Contains(output.String(), scope) {
			t.Fatalf("scope help does not advertise %q: %s", scope, output.String())
		}
	}
}
