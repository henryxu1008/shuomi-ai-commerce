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
	require.NoError(t, db.Exec(`CREATE ROLE store_center_runtime LOGIN PASSWORD 'synthetic-runtime-password'; REVOKE CREATE ON SCHEMA public FROM PUBLIC; CREATE TABLE public.retained_witness(value text); INSERT INTO public.retained_witness VALUES('preserve-me')`).Error)
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
	require.Error(t, run(path, false, false, false, true, false), "restore cannot install missing schema")
	require.NoError(t, run(path, false, false, true, false, false))
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.goose_store_center_version`).Scan(&after).Error)
	require.Equal(t, before, after)
	var witness string
	require.NoError(t, db.Raw(`SELECT value FROM public.retained_witness`).Scan(&witness).Error)
	require.Equal(t, "preserve-me", witness)
	var baseGrant, observationGrant bool
	require.NoError(t, db.Raw(`SELECT has_table_privilege('store_center_runtime','public.workbench_stores','SELECT'),has_table_privilege('store_center_runtime','store_observations.commands','INSERT')`).Row().Scan(&baseGrant, &observationGrant))
	require.False(t, baseGrant, "no general grants")
	require.True(t, observationGrant)
	require.Error(t, run(path, false, false, true, false, false), "no repeat migration")
	runtimePath := filepath.Join(t.TempDir(), "private-runtime.json")
	b, e = json.Marshal(map[string]any{"host": "127.0.0.1", "port": p, "user": "store_center_runtime", "password": "synthetic-runtime-password", "database": "store_center", "maxConnections": 1, "maxIdleConnections": 0})
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(runtimePath, b, 0600))
	if runtime.GOOS == "windows" {
		out, e := exec.Command("icacls", runtimePath, "/inheritance:r", "/grant:r", os.Getenv("USERNAME")+":F").CombinedOutput()
		require.NoError(t, e, string(out))
	}
	require.NoError(t, run(runtimePath, false, false, false, false, true))
	require.Error(t, run(path, false, false, false, false, true), "verification requires exact serving role")
	require.Error(t, run(runtimePath, false, false, false, true, false), "restoration requires exact owner")
	require.NoError(t, db.Exec(`INSERT INTO store_observations.heads(organization_id,store_id,kind,sequence) VALUES('retained-org','11111111-1111-4111-8111-111111111111','products',1); REVOKE ALL ON store_observations.commands,store_observations.syncs,store_observations.records,store_observations.heads FROM store_center_runtime; REVOKE USAGE ON SCHEMA store_observations FROM store_center_runtime`).Error)
	require.Error(t, run(runtimePath, false, false, false, false, true), "revoked grants cannot pass completed verification")
	require.NoError(t, run(path, false, false, false, true, false))
	require.NoError(t, run(runtimePath, false, false, false, false, true))
	require.NoError(t, db.Raw(`SELECT count(*) FROM public.goose_store_center_version`).Scan(&after).Error)
	require.Equal(t, before, after)
	require.NoError(t, db.Raw(`SELECT value FROM public.retained_witness`).Scan(&witness).Error)
	require.Equal(t, "preserve-me", witness)
	require.NoError(t, db.Raw(`SELECT has_table_privilege('store_center_runtime','public.workbench_stores','SELECT')`).Scan(&baseGrant).Error)
	require.False(t, baseGrant, "restore changes no base grant")
	var retained int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM store_observations.heads WHERE organization_id='retained-org'`).Scan(&retained).Error)
	require.EqualValues(t, 1, retained, "restore preserves observation facts")
	require.NoError(t, db.Exec(`ALTER TABLE store_observations.heads OWNER TO store_center_runtime`).Error)
	require.Error(t, run(path, false, false, false, true, false), "restore rejects ownership drift")
}
