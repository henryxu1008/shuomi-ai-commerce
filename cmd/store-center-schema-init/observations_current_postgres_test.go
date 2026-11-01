package main

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"
	storeschema "task-processor/internal/app/schema/storecenter"
	observations "task-processor/internal/integration/persistence/storeobservations"
	"task-processor/internal/storecenter"
)

func TestCurrentStoreObservationsInstall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	c, err := pg.Run(ctx, "postgres:17.2-alpine", pg.WithDatabase("isolated_install_tests"), pg.WithUsername("test_admin"), pg.WithPassword("isolated-admin"), pg.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	admin := openCoreAlignmentDB(t, dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE store_center_owner LOGIN PASSWORD 'isolated-owner' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE store_center_runtime LOGIN PASSWORD 'isolated-runtime' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS`).Error)
	fixture := func(t *testing.T) (owner, serving, setup *gorm.DB, name string) {
		t.Helper()
		name = "store_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, admin.Exec("CREATE DATABASE "+name+" OWNER store_center_owner").Error)
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		u.Path = "/" + name
		setup = openCoreAlignmentDB(t, u.String())
		require.NoError(t, setup.Exec("REVOKE ALL ON DATABASE "+name+" FROM PUBLIC; REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)
		u.User = url.UserPassword("store_center_owner", "isolated-owner")
		owner = openCoreAlignmentDB(t, u.String())
		require.NoError(t, storeschema.Migrate(ctx, owner))
		require.NoError(t, grantStoreRuntime(ctx, owner))
		seedCoreAlignmentFacts(t, owner)
		u.User = url.UserPassword("store_center_runtime", "isolated-runtime")
		serving = openCoreAlignmentDB(t, u.String())
		return
	}
	baseSnapshot := func(t *testing.T, db *gorm.DB) string {
		t.Helper()
		// Existing snapshot covers every public row/ACL/column/default/index and
		// history. Exclude only the newly installed observation namespace.
		var publicTables []string
		require.NoError(t, db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`).Scan(&publicTables).Error)
		parts := []string{}
		for _, table := range publicTables {
			var value string
			require.NoError(t, db.Raw(`SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM "public"."`+table+`" t`).Scan(&value).Error)
			parts = append(parts, table+":"+value)
		}
		for _, query := range []string{
			`SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid)::text FROM pg_class c WHERE c.relnamespace='public'::regnamespace`,
			`SELECT jsonb_agg(to_jsonb(a) ORDER BY a.attrelid,a.attnum)::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid WHERE c.relnamespace='public'::regnamespace`,
			`SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid)::text FROM pg_constraint c WHERE connamespace='public'::regnamespace`,
			`SELECT jsonb_agg(to_jsonb(i) ORDER BY i.indexrelid)::text FROM pg_index i JOIN pg_class c ON c.oid=i.indrelid WHERE c.relnamespace='public'::regnamespace`,
			`SELECT jsonb_agg(to_jsonb(d) ORDER BY d.oid)::text FROM pg_attrdef d JOIN pg_class c ON c.oid=d.adrelid WHERE c.relnamespace='public'::regnamespace`,
			`SELECT jsonb_build_array((SELECT datacl FROM pg_database WHERE datname=current_database()),(SELECT nspacl FROM pg_namespace WHERE nspname='public'),(SELECT coalesce(jsonb_agg(to_jsonb(d)),'[]') FROM pg_default_acl d))::text`,
		} {
			var value string
			require.NoError(t, db.Raw(query).Scan(&value).Error)
			parts = append(parts, value)
		}
		return strings.Join(parts, "\n")
	}
	absent := func(t *testing.T, db *gorm.DB) {
		t.Helper()
		var missing bool
		require.NoError(t, db.Raw(`SELECT to_regnamespace('store_observations') IS NULL`).Scan(&missing).Error)
		require.True(t, missing)
	}
	t.Run("current facts and 19 native tables preserved", func(t *testing.T) {
		owner, serving, _, name := fixture(t)
		before := baseSnapshot(t, owner)
		require.Error(t, storeschema.InstallObservations(ctx, owner), "fresh path must still refuse business facts")
		require.NoError(t, storeschema.InstallCurrentStoreObservations(ctx, owner, name))
		require.Equal(t, before, baseSnapshot(t, owner))
		require.NoError(t, observations.VerifyRuntime(ctx, serving))
		require.NoError(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, serving, storecenter.RuntimeCapabilities{Observations: true}))
		require.Error(t, storeschema.InstallCurrentStoreObservations(ctx, owner, name), "repeat does not rebuild")
		require.NoError(t, owner.Exec(`INSERT INTO store_observations.heads VALUES('org','00000000-0000-4000-8000-000000000001','orders',1); REVOKE USAGE ON SCHEMA store_observations FROM store_center_runtime`).Error)
		require.Error(t, storeschema.InstallCurrentStoreObservations(ctx, owner, name), "repeat cannot restore revoked grants")
		var usage bool
		require.NoError(t, owner.Raw(`SELECT has_schema_privilege('store_center_runtime','store_observations','USAGE')`).Scan(&usage).Error)
		require.False(t, usage)
		var rows int64
		require.NoError(t, owner.Table("store_observations.heads").Count(&rows).Error)
		require.EqualValues(t, 1, rows)
		require.Equal(t, before, baseSnapshot(t, owner))
	})
	for _, test := range []string{"wrong database", "wrong owner", "core drift", "runtime privilege drift", "foreign table", "history drift", "existing schema", "retired schema"} {
		t.Run(test, func(t *testing.T) {
			owner, _, setup, name := fixture(t)
			selected, confirmed := owner, name
			switch test {
			case "wrong database":
				confirmed = "another_store"
			case "wrong owner":
				selected = setup
			case "core drift":
				require.NoError(t, owner.Exec(`ALTER TABLE workbench_stores ADD COLUMN unsupported text`).Error)
			case "runtime privilege drift":
				require.NoError(t, owner.Exec(`GRANT DELETE ON workbench_stores TO store_center_runtime`).Error)
			case "foreign table":
				require.NoError(t, owner.Exec(`CREATE TABLE unsupported(value text)`).Error)
			case "history drift":
				require.NoError(t, owner.Exec(`INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(9999999999,true)`).Error)
			case "existing schema":
				require.NoError(t, owner.Exec(`CREATE SCHEMA store_observations`).Error)
			case "retired schema":
				require.NoError(t, owner.Exec(`CREATE SCHEMA shein_observations`).Error)
			}
			before := baseSnapshot(t, owner)
			require.Error(t, storeschema.InstallCurrentStoreObservations(ctx, selected, confirmed))
			require.Equal(t, before, baseSnapshot(t, owner))
			if test != "existing schema" {
				absent(t, owner)
			}
		})
	}
	t.Run("grant failure rolls back creation", func(t *testing.T) {
		owner, _, setup, name := fixture(t)
		require.NoError(t, setup.Exec(`CREATE FUNCTION public.reject_observation_grant() RETURNS event_trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'synthetic grant failure'; END$$;
CREATE EVENT TRIGGER reject_observation_grant ON ddl_command_end WHEN TAG IN ('GRANT') EXECUTE FUNCTION public.reject_observation_grant()`).Error)
		before := baseSnapshot(t, owner)
		err := storeschema.InstallCurrentStoreObservations(ctx, owner, name)
		require.ErrorContains(t, err, "synthetic grant failure")
		absent(t, owner)
		require.Equal(t, before, baseSnapshot(t, owner))
	})
	t.Run("concurrent single install", func(t *testing.T) {
		owner, serving, _, name := fixture(t)
		before := baseSnapshot(t, owner)
		var group sync.WaitGroup
		out := make(chan error, 2)
		for i := 0; i < 2; i++ {
			group.Add(1)
			go func() { defer group.Done(); out <- storeschema.InstallCurrentStoreObservations(ctx, owner, name) }()
		}
		group.Wait()
		close(out)
		var successes int
		for err := range out {
			if err == nil {
				successes++
			}
		}
		require.Equal(t, 1, successes)
		require.NoError(t, observations.VerifyRuntime(ctx, serving))
		require.Equal(t, before, baseSnapshot(t, owner))
	})
}
