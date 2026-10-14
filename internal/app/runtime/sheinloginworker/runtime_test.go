package sheinloginworker

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"task-processor/internal/core/config"
	"task-processor/internal/listingadmin"
	"task-processor/internal/sheinlogin"
	sheinloginbootstrap "task-processor/internal/sheinlogin/bootstrap"
)

type fakeSheinLoginWorkerStoreAPI struct {
	listingadmin.StoreAPI
}

func TestSheinLoginWorkerDefaultServiceBuilderRetainsAccountRepository(t *testing.T) {
	redis := miniredis.RunT(t)
	host, portText, err := net.SplitHostPort(redis.Addr())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "stores.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	repository := listingadmin.NewGormStoreRepository(db)
	cfg := &config.Config{Platforms: config.PlatformsConfig{Shein: config.PlatformConfig{
		CookieRedis: config.RedisConfig{Host: host, Port: port},
	}}}
	runtime, err := buildSheinLoginWorkerRuntimeWithDependencies("worker.yaml", sheinLoginWorkerRuntimeDependencies{
		LoadConfig: func(string) (*config.Config, error) { return cfg, nil },
		BuildDatabaseStoreAPI: func(*config.Config) (listingadmin.StoreAPI, listingadmin.StoreRepository, func() error, error) {
			return listingadmin.NewGormStoreAPI(repository), repository, sqlDB.Close, nil
		},
		BuildLoginService: buildSheinLoginWorkerService,
	})
	if err != nil {
		t.Fatalf("real default login builder failed with isolated database and Redis: %v", err)
	}
	t.Cleanup(func() { _ = runtime.Close() })
	health := runtime.result.Service.Health(context.Background())
	if !health.Initialized || !health.RedisReady || !health.ManagementReady {
		t.Fatalf("real service dependencies are incomplete: %+v", health)
	}
}

func TestLoadSheinLoginWorkerConfigAcceptsDatabaseAndRedisOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "worker.yaml")
	contents := []byte(`database:
  host: database.internal
  port: 5432
  user: listingkit
  password: test-only
  database: listingkit
platforms:
  shein:
    loginService:
      baseURL: https://login.example.invalid
      sharedKey: must-not-reach-worker
      tenantID: legacy-tenant
      identifier: legacy-store
      merchantName: legacy-merchant
      username: legacy-user
      password: legacy-password
      maxConcurrentLogins: 3
    cookieRedis:
      host: redis.internal
      port: 6379
      password: test-only
      db: 9
`)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatalf("write worker config: %v", err)
	}

	cfg, err := loadSheinLoginWorkerConfig(path)
	if err != nil {
		t.Fatalf("load worker config: %v", err)
	}
	if cfg.Database == nil || cfg.Database.Host != "database.internal" {
		t.Fatalf("database config = %#v", cfg.Database)
	}
	redis := cfg.EffectiveSheinCookieRedis()
	if redis.Host != "redis.internal" || redis.DB != 9 {
		t.Fatalf("cookie redis config = %#v", redis)
	}
	login := cfg.Platforms.Shein.LoginService
	if login.BaseURL != "" || login.SharedKey != "" || login.TenantID != "" || login.Identifier != "" || login.MerchantName != "" || login.Username != "" || login.Password != "" {
		t.Fatalf("worker retained unused login client/account values: %#v", login)
	}
	if login.MaxConcurrentLogins != 3 {
		t.Fatalf("worker max concurrent logins = %d, want 3", login.MaxConcurrentLogins)
	}
}

func TestBuildSheinLoginWorkerRuntimeWithDependenciesUsesOnlyScopedFactories(t *testing.T) {
	cfg := &config.Config{Database: &config.DatabaseConfig{Host: "database.internal"}}
	storeAPI := &fakeSheinLoginWorkerStoreAPI{}
	repository := listingadmin.NewGormStoreRepository(nil)
	service := &sheinlogin.Service{}
	var calls []string

	runtime, err := buildSheinLoginWorkerRuntimeWithDependencies("worker.yaml", sheinLoginWorkerRuntimeDependencies{
		LoadConfig: func(path string) (*config.Config, error) {
			calls = append(calls, "load-config:"+path)
			return cfg, nil
		},
		BuildDatabaseStoreAPI: func(got *config.Config) (listingadmin.StoreAPI, listingadmin.StoreRepository, func() error, error) {
			calls = append(calls, "build-database-store")
			if got != cfg || got.Database.Host != "database.internal" {
				t.Fatalf("store factory config = %#v", got)
			}
			return storeAPI, repository, func() error {
				calls = append(calls, "close-database-store")
				return nil
			}, nil
		},
		BuildLoginService: func(gotConfig *config.Config, gotStoreAPI listingadmin.StoreAPI, gotRepository listingadmin.StoreRepository) (*sheinloginbootstrap.BuildResult, func() error, error) {
			calls = append(calls, "build-login-service")
			if gotConfig != cfg || gotStoreAPI != storeAPI || gotRepository != repository {
				t.Fatalf("login dependencies = %#v / %#v", gotConfig, gotStoreAPI)
			}
			return &sheinloginbootstrap.BuildResult{Service: service}, func() error {
				calls = append(calls, "close-login-service")
				return nil
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("build worker runtime: %v", err)
	}
	if runtime == nil || runtime.result == nil || runtime.result.Service != service {
		t.Fatalf("worker runtime = %#v", runtime)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("close worker runtime: %v", err)
	}
	wantCalls := []string{
		"load-config:worker.yaml",
		"build-database-store",
		"build-login-service",
		"close-login-service",
		"close-database-store",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("worker construction calls = %v, want %v", calls, wantCalls)
	}
}

func TestBuildSheinLoginWorkerRuntimeWithDependenciesClosesStoreWhenLoginUnavailable(t *testing.T) {
	var calls []string
	_, err := buildSheinLoginWorkerRuntimeWithDependencies("worker.yaml", sheinLoginWorkerRuntimeDependencies{
		LoadConfig: func(string) (*config.Config, error) {
			calls = append(calls, "load-config")
			return &config.Config{}, nil
		},
		BuildDatabaseStoreAPI: func(*config.Config) (listingadmin.StoreAPI, listingadmin.StoreRepository, func() error, error) {
			calls = append(calls, "build-database-store")
			return &fakeSheinLoginWorkerStoreAPI{}, nil, func() error {
				calls = append(calls, "close-database-store")
				return nil
			}, nil
		},
		BuildLoginService: func(*config.Config, listingadmin.StoreAPI, listingadmin.StoreRepository) (*sheinloginbootstrap.BuildResult, func() error, error) {
			calls = append(calls, "build-login-service")
			return &sheinloginbootstrap.BuildResult{}, func() error {
				calls = append(calls, "close-login-service")
				return nil
			}, nil
		},
	})
	if err == nil || !errors.Is(err, errSheinLoginWorkerUnavailable) {
		t.Fatalf("build error = %v, want unavailable", err)
	}
	wantCalls := []string{
		"load-config",
		"build-database-store",
		"build-login-service",
		"close-login-service",
		"close-database-store",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("failure cleanup calls = %v, want %v", calls, wantCalls)
	}
}
