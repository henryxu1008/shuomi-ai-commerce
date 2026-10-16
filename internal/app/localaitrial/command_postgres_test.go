//go:build integration

package localaitrial

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	orgresourceadapter "task-processor/internal/integration/orgresource"
	"task-processor/internal/ledger/orgresource"
)

func TestPostgresTrialAIGrantNativeReadOnlyAndConcurrentFirstSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("trial"), tcpostgres.WithUsername("trial"), tcpostgres.WithPassword("controlled-owner"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	owner, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := owner.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = pool.Close() })
	// All schema/role/fixture writes below are confined to this disposable cluster.
	require.NoError(t, orgresourceadapter.AutoMigrate(owner))
	for _, sql := range []string{
		`CREATE ROLE commercial_owner_runtime LOGIN PASSWORD 'controlled-runtime'`,
		`REVOKE CREATE ON SCHEMA public FROM PUBLIC`,
		`GRANT USAGE ON SCHEMA public TO commercial_owner_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON saas_organization_resource_buckets,saas_organization_resource_operations,saas_organization_resource_reservations,saas_organization_resource_debts,saas_member_ai_point_limits,saas_member_ai_point_months,saas_member_resource_positions TO commercial_owner_runtime`,
		`GRANT SELECT,INSERT ON saas_organization_resource_source_claims,saas_organization_resource_events,saas_organization_resource_audit_logs TO commercial_owner_runtime`,
		`GRANT USAGE,SELECT ON SEQUENCE saas_organization_resource_audit_logs_id_seq TO commercial_owner_runtime`,
	} {
		require.NoError(t, owner.Exec(sql).Error)
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	port, err := strconv.Atoi(u.Port())
	require.NoError(t, err)
	cfg := Config{OrganizationID: "trial-org", OperatorID: "trial-operator", ApprovalEvidenceID: "human-trial-approval", Database: DatabaseConfig{Host: u.Hostname(), Port: port, User: orgresourceadapter.RuntimeRole, Password: "controlled-runtime", Database: strings.TrimPrefix(u.Path, "/")}}
	read, err := Run(ctx, cfg, "original-operation", "inspect")
	require.NoError(t, err)
	require.False(t, read.Found)
	for _, table := range []string{"buckets", "debts", "operations", "source_claims", "events", "audit_logs"} {
		assertTrialCount(t, owner, table, 0)
	}
	u.User = url.UserPassword(orgresourceadapter.RuntimeRole, "controlled-runtime")
	runtimeDB, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	runtimePool, err := runtimeDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = runtimePool.Close() })
	repository, err := orgresourceadapter.NewGormRepository(runtimeDB, orgresourceadapter.TransactionConfig{LockTimeout: time.Second, StatementTimeout: 3 * time.Second, TransactionTimeout: 5 * time.Second, TotalRetryBudget: 20 * time.Second, MaxAttempts: 12})
	require.NoError(t, err)
	cfg.GrantEnabled = true
	service, err := orgresource.NewTrialAIGrantService(repository, operatorAuthorizer{cfg})
	require.NoError(t, err)
	const callers = 12
	var wg sync.WaitGroup
	results := make(chan orgresource.TrialAIGrantResult, callers)
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			result, err := service.Grant(ctx, cfg.input(fmt.Sprintf("concurrent-operation-%d", i)))
			results <- result
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	fresh := 0
	for result := range results {
		if !result.Replayed {
			fresh++
		}
		require.Equal(t, "10000", result.Snapshot.BalanceAfter)
	}
	require.Equal(t, 1, fresh)
	result, err := Run(ctx, cfg, "native-recovery-operation", "grant")
	require.NoError(t, err)
	require.True(t, result.Found)
	require.True(t, result.Receipt.Replayed)
	cfg.GrantEnabled = false
	read, err = Run(ctx, cfg, "native-recovery-operation", "inspect")
	require.NoError(t, err)
	require.Equal(t, result.Receipt.Snapshot, read.Receipt.Snapshot)
	for _, table := range []string{"operations", "source_claims", "events", "audit_logs"} {
		assertTrialCount(t, owner, table, 1)
	}
	require.NoError(t, owner.Exec(`REVOKE INSERT ON saas_organization_resource_events FROM commercial_owner_runtime`).Error)
	_, err = Run(ctx, cfg, "native-recovery-operation", "inspect")
	require.ErrorContains(t, err, "runtime permissions unavailable")
	assertTrialCount(t, owner, "events", 1)
}

func assertTrialCount(t *testing.T, db *gorm.DB, suffix string, expected int64) {
	t.Helper()
	var count int64
	require.NoError(t, db.Table("saas_organization_resource_"+suffix).Count(&count).Error)
	require.Equal(t, expected, count, suffix)
}
