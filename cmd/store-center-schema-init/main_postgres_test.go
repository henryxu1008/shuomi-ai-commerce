//go:build integration

package main

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	storeschema "task-processor/internal/app/schema/storecenter"
)

func TestObservationsOnlyPreservesBaseMigrationsFactsAndGrants(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c, e := pg.Run(ctx, "postgres:16-alpine", pg.WithDatabase("store_center"), pg.WithUsername("store_center_owner"), pg.WithPassword("synthetic-test-password"), pg.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })
	dsn, e := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, e)
	pool, e := db.DB()
	require.NoError(t, e)
	t.Cleanup(func() { _ = pool.Close() })
	require.NoError(t, storeschema.Migrate(ctx, db))
	require.NoError(t, db.Exec(`CREATE ROLE store_center_runtime; REVOKE CREATE ON SCHEMA public FROM PUBLIC; CREATE TABLE public.retained_witness(value text); INSERT INTO public.retained_witness VALUES('preserve-me')`).Error)
	// Do not grant base permissions: observations-only must not broaden them.
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	_, port, e := net.SplitHostPort(u.Host)
	require.NoError(t, e)
	p, e := strconv.Atoi(port)
	require.NoError(t, e)
	path := filepath.Join(t.TempDir(), "private-owner.json")
	b, e := json.Marshal(map[string]any{"host": "127.0.0.1", "port": p, "user": "store_center_owner", "password": "synthetic-test-password", "database": "store_center", "maxConnections": 1, "maxIdleConnections": 0})
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(path, b, 0600))
	if runtime.GOOS == "windows" {
		out, e := exec.Command("icacls", path, "/inheritance:r", "/grant:r", os.Getenv("USERNAME")+":F").CombinedOutput()
		require.NoError(t, e, string(out))
	}
	var before, after int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.goose_store_center_version`).Scan(&before).Error)
	require.NoError(t, run(path, false, false, true))
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.goose_store_center_version`).Scan(&after).Error)
	require.Equal(t, before, after)
	var witness string
	require.NoError(t, db.Raw(`SELECT value FROM public.retained_witness`).Scan(&witness).Error)
	require.Equal(t, "preserve-me", witness)
	var baseGrant, observationGrant bool
	require.NoError(t, db.Raw(`SELECT has_table_privilege('store_center_runtime','public.workbench_stores','SELECT'),has_table_privilege('store_center_runtime','shein_observations.commands','INSERT')`).Row().Scan(&baseGrant, &observationGrant))
	require.False(t, baseGrant, "no general grants")
	require.True(t, observationGrant)
	require.Error(t, run(path, false, false, true), "no repeat migration")
}
