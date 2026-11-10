//go:build integration

package storecenter

import (
	"context"
	_ "embed"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	observations "task-processor/internal/integration/persistence/storeobservations"
	current "task-processor/internal/storecenter"
)

//go:embed testdata/empty-observations-baseline.sql
var emptyObservationBaseline string

func emptyObservationFixture(t *testing.T) (context.Context, *gorm.DB, *gorm.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	t.Cleanup(cancel)
	c, err := pg.Run(ctx, "postgres:17-alpine", pg.WithDatabase("store_center"), pg.WithUsername("fixture_admin"), pg.WithPassword(uuid.NewString()), pg.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Terminate(context.Background()) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(d string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(d), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	admin := open(dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE store_center_owner LOGIN PASSWORD 'synthetic-owner'; CREATE ROLE store_center_runtime LOGIN PASSWORD 'synthetic-runtime'; ALTER DATABASE store_center OWNER TO store_center_owner; ALTER SCHEMA public OWNER TO store_center_owner; REVOKE ALL ON DATABASE store_center FROM PUBLIC; REVOKE ALL ON SCHEMA public FROM PUBLIC; GRANT CONNECT ON DATABASE store_center TO store_center_runtime`).Error)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword("store_center_owner", "synthetic-owner")
	owner := open(u.String())
	require.NoError(t, Migrate(ctx, owner))
	for _, sql := range []string{
		`GRANT USAGE ON SCHEMA public TO store_center_runtime`,
		`GRANT SELECT,INSERT,UPDATE ON workbench_stores,workbench_store_member_grants,workbench_store_service_operations,workbench_store_connections,workbench_store_connection_attempts,workbench_store_groups,workbench_store_profiles TO store_center_runtime`,
		`GRANT SELECT,INSERT ON workbench_store_audit_logs,workbench_store_member_grant_operations,workbench_store_merchant_bindings,workbench_store_metadata_operations TO store_center_runtime`,
	} {
		require.NoError(t, owner.Exec(sql).Error)
	}
	// Native fresh B must match the same immutable projection used for alignment.
	require.NoError(t, observations.Install(ctx, owner))
	require.NoError(t, observations.GrantRuntime(ctx, owner, "store_center_runtime"))
	pool, _ := owner.DB()
	tx, err := pool.BeginTx(ctx, nil)
	require.NoError(t, err)
	b, err := observations.InspectEmptyContractTx(ctx, tx)
	require.NoError(t, err)
	require.True(t, b)
	require.NoError(t, tx.Rollback())
	resetEmptyObservationBaseline(t, owner)
	return ctx, owner, admin
}

func resetEmptyObservationBaseline(t *testing.T, owner *gorm.DB) {
	t.Helper()
	// Only this test's disposable database, never a retained environment.
	require.NoError(t, owner.Exec(`DROP SCHEMA store_observations CASCADE`).Error)
	require.NoError(t, owner.Exec(emptyObservationBaseline).Error)
	require.NoError(t, observations.GrantRuntime(context.Background(), owner, "store_center_runtime"))
}

func observationCheckIdentity(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT jsonb_agg(jsonb_build_array(c.relname,k.oid,k.conname,pg_get_constraintdef(k.oid)) ORDER BY c.relname,k.conname)::text FROM pg_constraint k JOIN pg_class c ON c.oid=k.conrelid WHERE c.relnamespace='store_observations'::regnamespace`).Scan(&s).Error)
	return s
}

func TestEmptyObservationAlignmentPostgres(t *testing.T) {
	ctx, owner, admin := emptyObservationFixture(t)
	require.NoError(t, admin.Exec(`CREATE ROLE unexpected_reader NOLOGIN`).Error)
	pool, _ := owner.DB()
	t.Run("IT1-old-chain-refuses-A-new-chain-preserves", func(t *testing.T) {
		tx, e := pool.BeginTx(ctx, nil)
		require.NoError(t, e)
		require.Error(t, current.VerifyCurrentObservationsInstallBoundary(ctx, tx, "store_center", true), "B-only original consumer stays strict")
		require.NoError(t, current.VerifyEmptyObservationsMaintenanceBoundary(ctx, tx, "store_center"))
		before, e := observationPreservation(ctx, tx)
		require.NoError(t, e)
		require.NoError(t, tx.Rollback())
		require.NoError(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
		tx, e = pool.BeginTx(ctx, nil)
		require.NoError(t, e)
		after, e := observationPreservation(ctx, tx)
		require.NoError(t, e)
		require.Equal(t, before, after)
		require.NoError(t, tx.Rollback())
		identity := observationCheckIdentity(t, owner)
		require.NoError(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
		require.Equal(t, identity, observationCheckIdentity(t, owner), "zero DDL reentry, also after lost acknowledgement")
	})
	t.Run("simultaneous-maintainers", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		var wg sync.WaitGroup
		errs := make(chan error, 6)
		for range 6 {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- AlignEmptyObservationsContract(ctx, owner, "store_center") }()
		}
		wg.Wait()
		close(errs)
		for e := range errs {
			require.NoError(t, e)
		}
	})
	t.Run("refuse-drift-before-writes", func(t *testing.T) {
		for _, tc := range []struct{ name, sql string }{
			{"missing", `ALTER TABLE store_observations.heads DROP CONSTRAINT heads_kind_check`},
			{"weak", `ALTER TABLE store_observations.heads DROP CONSTRAINT heads_kind_check; ALTER TABLE store_observations.heads ADD CONSTRAINT heads_kind_check CHECK(true)`},
			{"mixed", `ALTER TABLE store_observations.heads DROP CONSTRAINT heads_kind_check; ALTER TABLE store_observations.heads ADD CONSTRAINT heads_kind_check CHECK(kind IN('products','orders','supply-orders'))`},
			{"not-valid", `ALTER TABLE store_observations.heads DROP CONSTRAINT heads_kind_check; ALTER TABLE store_observations.heads ADD CONSTRAINT heads_kind_check CHECK(kind IN('products','orders')) NOT VALID`},
			{"extra", `ALTER TABLE store_observations.heads ADD CONSTRAINT extra CHECK(sequence>0)`},
			{"privilege", `GRANT DELETE ON store_observations.heads TO store_center_runtime`},
			{"column-grant", `GRANT UPDATE(actor_id) ON store_observations.commands TO store_center_runtime`},
			{"unexpected-reader", `GRANT SELECT ON store_observations.records TO unexpected_reader`},
			{"unexpected-schema-reader", `GRANT USAGE ON SCHEMA store_observations TO unexpected_reader`},
			{"unexpected-column-reader", `GRANT SELECT(payload) ON store_observations.records TO unexpected_reader`},
			{"unexpected-default-privileges", `ALTER DEFAULT PRIVILEGES IN SCHEMA store_observations GRANT SELECT ON TABLES TO unexpected_reader`},
			{"nonempty", `INSERT INTO store_observations.heads VALUES('org-a','d6f6ca0a-27e2-4c4a-b1aa-4505110ae635','products',1,NULL)`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resetEmptyObservationBaseline(t, owner)
				require.NoError(t, owner.Exec(tc.sql).Error)
				before := observationCheckIdentity(t, owner)
				require.Error(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
				require.Equal(t, before, observationCheckIdentity(t, owner))
			})
		}
	})
	t.Run("misconfirmed-database", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		require.Error(t, AlignEmptyObservationsContract(ctx, owner, "another"))
	})
	t.Run("non-owner", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		before := observationCheckIdentity(t, owner)
		require.Error(t, AlignEmptyObservationsContract(ctx, admin, "store_center"))
		require.Equal(t, before, observationCheckIdentity(t, owner))
	})
	t.Run("unexpected-role-member", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		require.NoError(t, admin.Exec(`GRANT store_center_runtime TO unexpected_reader`).Error)
		defer admin.Exec(`REVOKE store_center_runtime FROM unexpected_reader`)
		require.Error(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
	})
	t.Run("DDL-failure-after-first-replacement", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		before := observationCheckIdentity(t, owner)
		require.NoError(t, admin.Exec(`CREATE FUNCTION public.fixture_reject_alignment() RETURNS event_trigger LANGUAGE plpgsql AS $$ BEGIN IF EXISTS(SELECT 1 FROM pg_event_trigger_ddl_commands() WHERE objid='store_observations.records'::regclass) THEN RAISE EXCEPTION 'synthetic injected DDL failure'; END IF; END $$; CREATE EVENT TRIGGER fixture_reject_alignment ON ddl_command_end WHEN TAG IN('ALTER TABLE') EXECUTE FUNCTION public.fixture_reject_alignment()`).Error)
		defer admin.Exec(`DROP EVENT TRIGGER fixture_reject_alignment; DROP FUNCTION public.fixture_reject_alignment()`)
		require.ErrorContains(t, AlignEmptyObservationsContract(ctx, owner, "store_center"), "synthetic injected DDL failure")
		require.Equal(t, before, observationCheckIdentity(t, owner), "first successful CHECK replacement must also roll back")
	})
	t.Run("transaction-rollback", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		before := observationCheckIdentity(t, owner)
		tx, e := pool.BeginTx(ctx, nil)
		require.NoError(t, e)
		require.NoError(t, observations.AlignEmptyContractTx(ctx, tx))
		require.NoError(t, tx.Rollback())
		require.Equal(t, before, observationCheckIdentity(t, owner))
	})
	t.Run("writer-commits-first", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		tx, e := pool.BeginTx(ctx, nil)
		require.NoError(t, e)
		_, e = tx.ExecContext(ctx, `INSERT INTO store_observations.heads VALUES('org-a','d6f6ca0a-27e2-4c4a-b1aa-4505110ae635','products',1,NULL)`)
		require.NoError(t, e)
		done := make(chan error, 1)
		go func() { done <- AlignEmptyObservationsContract(ctx, owner, "store_center") }()
		select {
		case e := <-done:
			t.Fatalf("maintenance crossed writer lock: %v", e)
		case <-time.After(100 * time.Millisecond):
		}
		require.NoError(t, tx.Commit())
		require.ErrorContains(t, <-done, "nonempty")
	})
	t.Run("maintenance-locks-exclude-writer-and-cancel", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		tx, e := pool.BeginTx(ctx, nil)
		require.NoError(t, e)
		_, e = tx.ExecContext(ctx, `LOCK TABLE store_observations.commands,store_observations.heads,store_observations.records,store_observations.syncs IN ACCESS EXCLUSIVE MODE`)
		require.NoError(t, e)
		writer, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancel()
		e = owner.WithContext(writer).Exec(`INSERT INTO store_observations.heads VALUES('org-a','d6f6ca0a-27e2-4c4a-b1aa-4505110ae635','products',1,NULL)`).Error
		require.Error(t, e)
		maintenance, cancelM := context.WithTimeout(ctx, 150*time.Millisecond)
		defer cancelM()
		require.Error(t, AlignEmptyObservationsContract(maintenance, owner, "store_center"))
		require.NoError(t, tx.Rollback())
		require.NoError(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
	})
	t.Run("IT1-retained-cockpit-is-verified-and-preserved", func(t *testing.T) {
		resetEmptyObservationBaseline(t, owner)
		require.NoError(t, InstallOperationsCockpit(ctx, owner))
		require.NoError(t, AlignEmptyObservationsContract(ctx, owner, "store_center"))
		resetEmptyObservationBaseline(t, owner)
		require.NoError(t, owner.Exec(`GRANT SELECT ON operations_cockpit.commands TO unexpected_reader`).Error)
		require.Error(t, AlignEmptyObservationsContract(ctx, owner, "store_center"), "other retained capability cannot carry an unknown reader")
		require.NoError(t, owner.Exec(`REVOKE SELECT ON operations_cockpit.commands FROM unexpected_reader`).Error)
		require.NoError(t, owner.Exec(`GRANT DELETE ON operations_cockpit.commands TO store_center_runtime`).Error)
		require.Error(t, AlignEmptyObservationsContract(ctx, owner, "store_center"), "existing cockpit is not a privilege bypass")
		require.NoError(t, owner.Exec(`REVOKE DELETE ON operations_cockpit.commands FROM store_center_runtime; ALTER TABLE operations_cockpit.commands DROP CONSTRAINT commands_pkey`).Error)
		require.Error(t, AlignEmptyObservationsContract(ctx, owner, "store_center"), "unchanged cockpit must match its entire native contract")
	})
}
