package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"gorm.io/gorm"

	storeschema "task-processor/internal/app/schema/storecenter"
	coreconfig "task-processor/internal/core/config"
	observationschema "task-processor/internal/integration/persistence/storeobservations"
	platformdatabase "task-processor/internal/platform/database"
	"task-processor/internal/storecenter"
)

func main() {
	path := flag.String("config", "", "absolute private Store schema-owner JSON config")
	observations := flag.Bool("observations", false, "install readonly platform observations on an authorized fresh empty Store instance")
	observationsOnly := flag.Bool("observations-only", false, "verify retained Store schema and install only observations; no base migrations or general grants")
	grantsOnly := flag.Bool("observations-grants-only", false, "explicitly restore original observation grants on an existing owner-controlled schema")
	verify := flag.Bool("observations-verify", false, "read-only verification using the private store_center_runtime config")
	cockpit := flag.Bool("operations-cockpit", false, "install manual financial facts and goals on an authorized fresh empty Store instance")
	tiktok := flag.Bool("tiktok-shop", false, "grant admitted TikTok OAuth tables on an authorized fresh empty Store instance")
	shein := flag.Bool("shein-login-worker", false, "install optional current SHEIN login execution fence and narrow worker role on an authorized fresh Store")
	walmart := flag.Bool("walmart-us", false, "grant admitted Walmart US OAuth tables on an authorized fresh empty Store instance")
	align := flag.Bool("align-core-contract", false, "align only the four admitted core CHECKs on a retained Store database; no grants or platform installation")
	confirmed := flag.String("confirm-database", "", "exact existing database name required for --align-core-contract")
	flag.Parse()
	confirmationSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "confirm-database" {
			confirmationSet = true
		}
	})
	err := func() error {
		if flag.NArg() != 0 {
			return errors.New("unexpected positional arguments")
		}
		if *align {
			if *observations || *cockpit || *observationsOnly || *grantsOnly || *verify || *tiktok || *walmart || *shein {
				return errors.New("core alignment cannot be combined with any installation or observation mode")
			}
			return alignCoreContract(*path, *confirmed)
		}
		if confirmationSet {
			return errors.New("--confirm-database requires --align-core-contract")
		}
		if *shein && (*observationsOnly || *grantsOnly || *verify) {
			return errors.New("SHEIN installation cannot be combined with observation maintenance")
		}
		if *shein {
			return runWithCapabilities(*path, *observations, *cockpit, *tiktok, *walmart, true)
		}
		return runWithTikTok(*path, *observations, *cockpit, *observationsOnly, *grantsOnly, *verify, *tiktok, *walmart)
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *align {
		fmt.Println("Store core contract and schema version verified; existing facts and permissions retained")
	} else if *verify {
		fmt.Println("Store observation schema and serving privileges verified; no schema operation")
	} else if *grantsOnly {
		fmt.Println("Original Store observation grants restored; schema and facts retained")
	} else if *observationsOnly {
		fmt.Println("Store observations installed; base schema, facts and permissions retained")
	} else {
		fmt.Println("current Store schema installed and narrow runtime roles granted")
	}
}

func alignCoreContract(path, confirmed string) error {
	if confirmed == "" {
		return errors.New("core contract alignment refused: exact database confirmation required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := loadOwnerConfig(path, "store_center_owner")
	if err != nil || cfg.Database != confirmed {
		return errors.New("core contract alignment refused: private owner config and exact database confirmation required")
	}
	db, err := platformdatabase.OpenExistingWritableContext(ctx, cfg)
	if err != nil {
		return errors.New("core contract alignment refused: owner database unavailable")
	}
	defer platformdatabase.Close(db)
	if err := storeschema.AlignCoreContract(ctx, db, confirmed); err != nil {
		return fmt.Errorf("core contract alignment refused: %w", err)
	}
	return nil
}

func run(storePath string, observations, cockpit, observationsOnly, grantsOnly, verify bool) error {
	return runWithTikTok(storePath, observations, cockpit, observationsOnly, grantsOnly, verify, false)
}
func runWithTikTok(storePath string, observations, cockpit, observationsOnly, grantsOnly, verify, tiktok bool, walmartOption ...bool) error {
	walmart := len(walmartOption) == 1 && walmartOption[0]
	narrowModes := 0
	for _, enabled := range []bool{observationsOnly, grantsOnly, verify} {
		if enabled {
			narrowModes++
		}
	}
	if narrowModes > 1 || narrowModes > 0 && (observations || cockpit || tiktok || walmart) {
		return errors.New("observation narrow modes cannot be combined with each other, base or Cockpit installation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	role := "store_center_owner"
	if verify {
		role = "store_center_runtime"
	}
	storeCfg, err := loadOwnerConfig(storePath, role)
	if err != nil {
		return err
	}
	open := platformdatabase.OpenExistingWritableContext
	if verify {
		open = platformdatabase.OpenExistingReadOnlyContext
	}
	records, err := open(ctx, storeCfg)
	if err != nil {
		return errors.New("open private Store role database failed")
	}
	defer platformdatabase.Close(records)
	if verify {
		return observationschema.VerifyRuntime(ctx, records)
	}
	if grantsOnly {
		if err := storecenter.VerifyCurrentSchema(ctx, records); err != nil {
			return fmt.Errorf("verify retained Store schema: %w", err)
		}
		return storeschema.RestoreObservationsGrants(ctx, records)
	}
	if observationsOnly {
		if err := storecenter.VerifyCurrentSchema(ctx, records); err != nil {
			return fmt.Errorf("verify retained Store schema: %w", err)
		}
		return storeschema.InstallObservations(ctx, records)
	}
	if err := storeschema.Migrate(ctx, records); err != nil {
		return fmt.Errorf("initialize current Store schema: %w", err)
	}
	if err := grantStoreRuntime(ctx, records); err != nil {
		return err
	}
	if tiktok {
		if err := records.WithContext(ctx).Exec(`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_tiktok_oauth_grants,public.workbench_store_tiktok_oauth_attempts,public.workbench_store_tiktok_shop_bindings,public.workbench_store_tiktok_oauth_refresh_operations TO store_center_runtime`).Error; err != nil {
			return err
		}
		if err := records.WithContext(ctx).Exec(`GRANT SELECT,INSERT ON public.workbench_store_tiktok_authorization_events TO store_center_runtime`).Error; err != nil {
			return err
		}
	}
	if walmart {
		if err := records.WithContext(ctx).Exec(`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_walmart_oauth_grants,public.workbench_store_walmart_oauth_attempts TO store_center_runtime`).Error; err != nil {
			return err
		}
	}
	if observations {
		if err := storeschema.InstallObservations(ctx, records); err != nil {
			return fmt.Errorf("initialize Store observation schema: %w", err)
		}
	}
	if cockpit {
		if err := storeschema.InstallOperationsCockpit(ctx, records); err != nil {
			return fmt.Errorf("initialize Cockpit schema: %w", err)
		}
	}
	return nil
}

func runWithCapabilities(path string, observations, cockpit, tiktok, walmart, shein bool) error {
	if err := runWithTikTok(path, observations, cockpit, false, false, false, tiktok, walmart); err != nil {
		return err
	}
	if !shein {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := loadOwnerConfig(path, "store_center_owner")
	if err != nil {
		return err
	}
	db, err := platformdatabase.OpenExistingWritableContext(ctx, cfg)
	if err != nil {
		return errors.New("Store schema-owner unavailable")
	}
	defer platformdatabase.Close(db)
	return storecenter.InstallSheinLoginCapability(ctx, db)
}

func grantStoreRuntime(ctx context.Context, db *gorm.DB) error {
	for _, statement := range []string{
		`DO $$ BEGIN EXECUTE format('GRANT CONNECT ON DATABASE %I TO store_center_runtime',current_database()); END $$`,
		`GRANT USAGE ON SCHEMA public TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON public.workbench_stores TO store_center_runtime`,
		`GRANT SELECT,INSERT ON public.workbench_store_audit_logs TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_member_grants TO store_center_runtime`,
		`GRANT SELECT,INSERT ON public.workbench_store_member_grant_operations TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_service_operations TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_connections,public.workbench_store_connection_attempts TO store_center_runtime`,
		`GRANT SELECT,INSERT ON public.workbench_store_merchant_bindings TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON public.workbench_store_groups,public.workbench_store_profiles TO store_center_runtime`,
		`GRANT SELECT,INSERT ON public.workbench_store_metadata_operations TO store_center_runtime`,
	} {
		if err := db.WithContext(ctx).Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}

func loadOwnerConfig(path, role string) (*platformdatabase.Config, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("schema-owner config must be an absolute private file")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > 16384 {
		return nil, errors.New("schema-owner config must be a bounded regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("schema-owner config must be private")
	}
	if runtime.GOOS == "windows" && coreconfig.VerifyPrivateFiles(context.Background(), []string{path}) != nil {
		return nil, errors.New("schema-owner config must be private")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var config struct {
		Host               string `json:"host"`
		Port               int    `json:"port"`
		User               string `json:"user"`
		Password           string `json:"password"`
		Database           string `json:"database"`
		MaxConnections     int    `json:"maxConnections"`
		MaxIdleConnections int    `json:"maxIdleConnections"`
	}
	decoder := json.NewDecoder(io.LimitReader(f, 16385))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, errors.New("invalid schema-owner JSON")
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return nil, errors.New("schema-owner config must contain one object")
	}
	if config.Host != "127.0.0.1" || config.Port < 1 || config.Port > 65535 || config.User != role || config.Password == "" || len(config.Password) > 1024 || strings.ContainsAny(config.Password, " =\\'\t\r\n\v\f\x00") || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]{0,62}$`).MatchString(config.Database) || config.MaxConnections < 1 || config.MaxConnections > 2 || config.MaxIdleConnections < 0 || config.MaxIdleConnections > config.MaxConnections {
		return nil, errors.New("schema-owner requires explicit loopback target, owner role and bounded credentials")
	}
	return &platformdatabase.Config{Host: config.Host, Port: config.Port, User: config.User, Password: config.Password, Database: config.Database, MaxConnections: config.MaxConnections, MaxIdleConnections: config.MaxIdleConnections}, nil
}
