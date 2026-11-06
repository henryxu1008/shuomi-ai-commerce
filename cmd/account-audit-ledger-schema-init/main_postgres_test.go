package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	store "task-processor/internal/aicapability/store"
)

func TestLedgerEmptyDatabasePostgres(t *testing.T) {
	if os.Getenv("ISSUE1075_OWNED_POSTGRES") != "1" {
		t.Skip("requires explicit task-owned temporary PostgreSQL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("issue1075_synthetic"), tcpostgres.WithUsername("issue1075_installer"), tcpostgres.WithPassword("synthetic-ledger-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	parsed.Scheme = "postgresql" // Keep the CLI's existing private input contract.
	dsn = parsed.String()
	open := func(connection string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(connection), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	root := open(dsn)
	for _, role := range []string{"product_agent_owner", "image_agent_owner", "wrong_owner"} {
		require.NoError(t, root.Exec("CREATE ROLE "+role+" LOGIN PASSWORD 'synthetic-ledger-role' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	}
	var rolesBefore, rolesAfter string
	roleSnapshot := "SELECT jsonb_agg(to_jsonb(r) ORDER BY oid)::text FROM pg_roles r"
	require.NoError(t, root.Raw(roleSnapshot).Scan(&rolesBefore).Error)
	fixture := func(name, databaseOwner, login string) (*gorm.DB, string) {
		require.NoError(t, root.Exec("CREATE DATABASE "+pgx.Identifier{name}.Sanitize()+" OWNER "+databaseOwner).Error)
		u, e := url.Parse(dsn)
		require.NoError(t, e)
		u.Path = "/" + name
		u.User = url.UserPassword(login, "synthetic-ledger-role")
		return open(u.String()), u.String()
	}
	t.Run("fresh-current-ledger", func(t *testing.T) {
		db, ownerDSN := fixture("issue1075_fresh", "product_agent_owner", "product_agent_owner")
		out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_fresh")
		require.NoError(t, e, out)
		require.NoError(t, store.VerifyModelPointInvocationSchema(ctx, db))
		var columns, rows int64
		require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='ai_invocations'").Scan(&columns).Error)
		require.EqualValues(t, 52, columns)
		require.NoError(t, db.Table("ai_invocations").Count(&rows).Error)
		require.Zero(t, rows)
		var owner string
		require.NoError(t, db.Raw("SELECT pg_get_userbyid(relowner) FROM pg_class WHERE oid='public.ai_invocations'::regclass").Scan(&owner).Error)
		require.Equal(t, "product_agent_owner", owner)
		var tables []string
		require.NoError(t, db.Raw("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").Scan(&tables).Error)
		require.Equal(t, []string{"ai_invocations"}, tables)
		out, e = ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_fresh")
		require.Error(t, e, "new empty mode must refuse a repeated install")
		require.Contains(t, out, "ledger schema initialization failed")
	})
	t.Run("ddl-failure-rolls-back", func(t *testing.T) {
		db, ownerDSN := fixture("issue1075_rollback", "product_agent_owner", "product_agent_owner")
		u, e := url.Parse(dsn)
		require.NoError(t, e)
		u.Path = "/issue1075_rollback"
		admin := open(u.String())
		// Fail after CREATE TABLE, while the canonical owner creates indexes.
		require.NoError(t, admin.Exec(`CREATE FUNCTION public.fail_synthetic_index() RETURNS event_trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic index failure'; END $$`).Error)
		require.NoError(t, admin.Exec(`CREATE EVENT TRIGGER synthetic_index_failure ON ddl_command_start WHEN TAG IN ('CREATE INDEX') EXECUTE FUNCTION public.fail_synthetic_index()`).Error)
		out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_rollback")
		require.Error(t, e)
		require.Contains(t, out, "verify transaction outcome before any retry")
		require.NotContains(t, out, "synthetic index failure", "database errors stay redacted")
		require.False(t, db.Migrator().HasTable("ai_invocations"), "table DDL must roll back with the failed index")
	})
	for i, setup := range []string{"CREATE TABLE preserved_fact(id int)", "CREATE SEQUENCE preserved_sequence", "CREATE VIEW preserved_view AS SELECT 1 AS value", "CREATE MATERIALIZED VIEW preserved_materialized AS SELECT 1 AS value", "CREATE SCHEMA other_business", "CREATE TABLE ai_invocations(invocation_id text)"} {
		t.Run(fmt.Sprintf("nonempty-%d", i), func(t *testing.T) {
			name := fmt.Sprintf("issue1075_nonempty_%d", i)
			db, ownerDSN := fixture(name, "product_agent_owner", "product_agent_owner")
			require.NoError(t, db.Exec(setup).Error)
			var before, after string
			snapshot := `SELECT jsonb_build_array((SELECT jsonb_agg(to_jsonb(n) ORDER BY oid) FROM pg_namespace n WHERE nspname !~ '^pg_' AND nspname<>'information_schema'),(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'))::text`
			require.NoError(t, db.Raw(snapshot).Scan(&before).Error)
			out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", name)
			require.Error(t, e)
			require.Contains(t, out, "ledger schema initialization failed")
			require.NoError(t, db.Raw(snapshot).Scan(&after).Error)
			require.Equal(t, before, after, "refused database objects and ACL must remain unchanged")
		})
	}
	t.Run("wrong-database", func(t *testing.T) {
		db, ownerDSN := fixture("issue1075_actual", "product_agent_owner", "product_agent_owner")
		out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_other")
		require.Error(t, e)
		require.Contains(t, out, "ledger schema initialization failed")
		require.False(t, db.Migrator().HasTable("ai_invocations"))
	})
	t.Run("wrong-login", func(t *testing.T) {
		db, _ := fixture("issue1075_login", "product_agent_owner", "product_agent_owner")
		u, _ := url.Parse(dsn)
		u.Path = "/issue1075_login"
		out, e := ledgerCLI(t, u.String(), "-namespace", "product", "-confirm-empty-database", "issue1075_login")
		require.Error(t, e)
		require.Contains(t, out, "ledger schema initialization failed")
		require.NotContains(t, out, "synthetic-ledger-only")
		require.False(t, db.Migrator().HasTable("ai_invocations"))
	})
	t.Run("wrong-database-owner", func(t *testing.T) {
		db, ownerDSN := fixture("issue1075_owner", "product_agent_owner", "product_agent_owner")
		require.NoError(t, root.Exec("ALTER DATABASE issue1075_owner OWNER TO wrong_owner").Error)
		out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_owner")
		require.Error(t, e)
		require.Contains(t, out, "ledger schema initialization failed")
		require.False(t, db.Migrator().HasTable("ai_invocations"))
	})
	t.Run("bounded-owner", func(t *testing.T) {
		db, ownerDSN := fixture("issue1075_privilege", "product_agent_owner", "product_agent_owner")
		for _, attribute := range []string{"SUPERUSER", "CREATEDB", "CREATEROLE", "REPLICATION", "BYPASSRLS"} {
			require.NoError(t, root.Exec("ALTER ROLE product_agent_owner "+attribute).Error)
			out, e := ledgerCLI(t, ownerDSN, "-namespace", "product", "-confirm-empty-database", "issue1075_privilege")
			require.NoError(t, root.Exec("ALTER ROLE product_agent_owner NO"+attribute).Error)
			require.Error(t, e, attribute)
			require.Contains(t, out, "ledger schema initialization failed")
			require.False(t, db.Migrator().HasTable("ai_invocations"))
		}
	})
	t.Run("fixed-default-modes", func(t *testing.T) {
		for _, namespace := range []string{"product", "image"} {
			db, ownerDSN := fixture(namespace+"_agent", namespace+"_agent_owner", namespace+"_agent_owner")
			for attempt := 0; attempt < 2; attempt++ {
				out, e := ledgerCLI(t, ownerDSN, "-namespace", namespace)
				require.NoError(t, e, out)
			}
			require.NoError(t, store.VerifyModelPointInvocationSchema(ctx, db))
		}
	})
	require.NoError(t, root.Raw(roleSnapshot).Scan(&rolesAfter).Error)
	require.Equal(t, rolesBefore, rolesAfter, "installer must not alter roles or memberships")
}
