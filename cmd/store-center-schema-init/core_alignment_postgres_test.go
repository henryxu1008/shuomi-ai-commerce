package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	pg "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	storeschema "task-processor/internal/app/schema/storecenter"
	cockpit "task-processor/internal/integration/persistence/operationscockpit"
	observations "task-processor/internal/integration/persistence/storeobservations"
	platformmigration "task-processor/internal/platform/database/migration"
	"task-processor/internal/storecenter"
)

type coreAlignmentFixture struct {
	owner, serving *gorm.DB
	name, config   string
}

func TestCoreAlignmentPreservesRetainedStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	c, err := pg.Run(ctx, "postgres:17.2-alpine", pg.WithDatabase("isolated_store_tests"), pg.WithUsername("test_admin"), pg.WithPassword("isolated-admin"), pg.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	admin := openCoreAlignmentDB(t, dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE store_center_owner LOGIN PASSWORD 'isolated-owner' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE store_center_runtime LOGIN PASSWORD 'isolated-runtime' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE retained_reader NOLOGIN`).Error)
	newFixture := func(t *testing.T) coreAlignmentFixture {
		t.Helper()
		name := "store_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, admin.Exec("CREATE DATABASE "+name+" OWNER store_center_owner").Error)
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		u.Path, u.User = "/"+name, url.UserPassword("test_admin", "isolated-admin")
		setup := openCoreAlignmentDB(t, u.String())
		require.NoError(t, setup.Exec("REVOKE ALL ON DATABASE "+name+" FROM PUBLIC; REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)
		u.User = url.UserPassword("store_center_owner", "isolated-owner")
		owner := openCoreAlignmentDB(t, u.String())
		pool, err := owner.DB()
		require.NoError(t, err)
		// Reuse the existing owner and original Goose IDs. Only the test fixture
		// recreates the precisely observed two-platform baseline.
		runner, err := platformmigration.NewWithVersionTable(goose.DialectPostgres, pool, "public.goose_store_center_version",
			goose.NewGoMigration(2026092801, &goose.GoFunc{RunTx: storecenter.InstallCoreSchemaTx}, nil),
			goose.NewGoMigration(2026101101, &goose.GoFunc{RunTx: storecenter.InstallMetadataSchemaTx}, nil))
		require.NoError(t, err)
		_, err = runner.Up(ctx)
		require.NoError(t, err)
		require.NoError(t, owner.Exec(`DROP TABLE workbench_store_tiktok_authorization_events,workbench_store_tiktok_oauth_refresh_operations,workbench_store_tiktok_shop_bindings,workbench_store_tiktok_oauth_attempts,workbench_store_tiktok_oauth_grants,workbench_store_walmart_oauth_attempts,workbench_store_walmart_oauth_grants;
ALTER TABLE workbench_stores DROP CONSTRAINT store_platform, DROP CONSTRAINT store_tiktok_identity, DROP CONSTRAINT store_walmart_market, ADD CONSTRAINT store_platform CHECK(platform IN ('shein','temu'));
ALTER TABLE workbench_store_merchant_bindings DROP CONSTRAINT store_merchant_binding_shape, ADD CONSTRAINT store_merchant_binding_shape CHECK(octet_length(identity_hash)=64 AND app_id<>'' AND ((platform='shein' AND open_key_id<>'') OR (platform='temu' AND open_key_id='')));
GRANT SELECT ON workbench_stores TO retained_reader`).Error)
		require.NoError(t, grantStoreRuntime(ctx, owner))
		seedCoreAlignmentFacts(t, owner)

		port, err := strconv.Atoi(u.Port())
		require.NoError(t, err)
		config := filepath.Join(t.TempDir(), "private-owner.json")
		body, err := json.Marshal(map[string]any{"host": "127.0.0.1", "port": port, "user": "store_center_owner", "password": "isolated-owner", "database": name, "maxConnections": 2, "maxIdleConnections": 0})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(config, body, 0600))
		if runtime.GOOS == "windows" {
			out, err := exec.Command("icacls", config, "/inheritance:r", "/grant:r", os.Getenv("USERNAME")+":F").CombinedOutput()
			require.NoError(t, err, "%s", out)
		}
		u.User = url.UserPassword("store_center_runtime", "isolated-runtime")
		return coreAlignmentFixture{owner, openCoreAlignmentDB(t, u.String()), name, config}
	}

	installCapabilities := func(t *testing.T, f coreAlignmentFixture, caps storecenter.RuntimeCapabilities) {
		t.Helper()
		// Reuse the native capability owners on synthetic data. The retained
		// maintenance path under test must never call either installer or GRANT.
		if caps.Observations {
			require.NoError(t, observations.Install(ctx, f.owner))
			require.NoError(t, observations.GrantRuntime(ctx, f.owner, "store_center_runtime"))
			require.NoError(t, f.owner.Exec(`INSERT INTO store_observations.commands VALUES('org','00000000-0000-4000-8000-000000000010','actor','member','00000000-0000-4000-8000-000000000011',repeat('a',64),'{}',now())`).Error)
		}
		if caps.OperationsCockpit {
			require.NoError(t, cockpit.Install(ctx, f.owner))
			require.NoError(t, cockpit.GrantRuntime(ctx, f.owner, "store_center_runtime"))
			require.NoError(t, f.owner.Exec(`INSERT INTO operations_cockpit.facts VALUES('org','00000000-0000-4000-8000-000000000010','00000000-0000-4000-8000-000000000001',1,'2026-01-01','2026-01-31','{}','retained','actor',now())`).Error)
		}
	}
	for _, caps := range []storecenter.RuntimeCapabilities{{}, {Observations: true}, {OperationsCockpit: true}, {Observations: true, OperationsCockpit: true}} {
		t.Run(fmt.Sprintf("installed observations=%t cockpit=%t", caps.Observations, caps.OperationsCockpit), func(t *testing.T) {
			f := newFixture(t)
			installCapabilities(t, f, caps)
			require.Error(t, storeschema.InstallBrowserWorkbench(ctx, f.owner, f.name), "2501 cannot precede core alignment")
			before := coreAlignmentFullSnapshot(t, f.owner)
			if caps.Observations || caps.OperationsCockpit {
				require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}), "capabilities must be explicit")
			}
			require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
			before = coreAlignmentSnapshot(t, f.owner)
			require.NoError(t, alignCoreContract(f.config, f.name, caps))
			require.Equal(t, before, coreAlignmentSnapshot(t, f.owner))
			require.NoError(t, storecenter.VerifyRuntimePermissionsForCapabilities(ctx, f.serving, caps))
			after := coreAlignmentFullSnapshot(t, f.owner)
			require.NoError(t, alignCoreContract(f.config, f.name, caps))
			require.Equal(t, after, coreAlignmentFullSnapshot(t, f.owner))
			// The next native owner still requires 1601 before browser 2501.
			require.NoError(t, storeschema.InstallBrowserWorkbench(ctx, f.owner, f.name))
			var versions []int64
			require.NoError(t, f.owner.Raw(`SELECT version_id FROM goose_store_center_version ORDER BY id`).Scan(&versions).Error)
			require.Equal(t, []int64{0, 2026092801, 2026101101, 2026101601, 2026102501}, versions)
		})
	}
	for _, tc := range []struct{ name, drift string }{
		{"unknown empty schema", `CREATE SCHEMA unexpected`},
		{"unknown schema table", `CREATE SCHEMA unexpected; CREATE TABLE unexpected.facts(id integer)`},
		{"unknown optional table", `CREATE TABLE operations_cockpit.unexpected(id integer)`},
		{"optional view", `CREATE VIEW store_observations.unexpected AS SELECT 1 AS id`},
		{"optional sequence", `CREATE SEQUENCE store_observations.unexpected`},
		{"optional function", `CREATE FUNCTION operations_cockpit.unexpected() RETURNS int LANGUAGE sql AS 'SELECT 1'`},
		{"optional type", `CREATE TYPE operations_cockpit.unexpected AS ENUM ('x')`},
		{"same count wrong table", `ALTER TABLE operations_cockpit.fact_versions RENAME TO unexpected`},
		{"missing optional table", `DROP TABLE operations_cockpit.fact_versions`},
		{"missing required privilege", `REVOKE INSERT ON operations_cockpit.fact_versions FROM store_center_runtime`},
		{"PUBLIC extra effective privilege", `GRANT DELETE ON operations_cockpit.facts TO PUBLIC`},
		{"column extra effective privilege", `GRANT UPDATE(receipt) ON operations_cockpit.commands TO store_center_runtime`},
		{"optional grant option", `GRANT SELECT ON store_observations.records TO store_center_runtime WITH GRANT OPTION`},
		{"column type drift", `ALTER TABLE operations_cockpit.facts ALTER COLUMN note TYPE varchar(1000)`},
		{"column nullability drift", `ALTER TABLE store_observations.records ALTER COLUMN payload DROP NOT NULL`},
		{"default drift", `ALTER TABLE store_observations.records ALTER COLUMN window_key SET DEFAULT 'changed'`},
		{"check drift", `ALTER TABLE operations_cockpit.facts DROP CONSTRAINT facts_revision_check, ADD CONSTRAINT facts_revision_check CHECK(revision>=0)`},
		{"index drift", `DROP INDEX operations_cockpit.facts_scope_period`},
		{"extra optional check", `ALTER TABLE operations_cockpit.facts ADD CONSTRAINT unexpected CHECK(note<>'')`},
		{"extra optional index", `CREATE INDEX unexpected ON operations_cockpit.facts(note)`},
		{"foreign key drift", `ALTER TABLE operations_cockpit.goal_versions DROP CONSTRAINT goal_versions_organization_id_goal_id_fkey`},
		{"unknown history with capabilities", `INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(2026101501,true)`},
		{"2501 before 1601", `INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(2026102501,true)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			caps := storecenter.RuntimeCapabilities{Observations: true, OperationsCockpit: true}
			installCapabilities(t, f, caps)
			require.NoError(t, f.owner.Exec(tc.drift).Error)
			before := coreAlignmentFullSnapshot(t, f.owner)
			require.Error(t, alignCoreContract(f.config, f.name, caps))
			require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner), "refusal preserves optional objects, facts and ACLs too")
		})
	}
	t.Run("declared but missing capability", func(t *testing.T) {
		f := newFixture(t)
		before := coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{Observations: true}))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
	})
	t.Run("browser still requires the aligned core and exact history", func(t *testing.T) {
		f := newFixture(t)
		require.NoError(t, f.owner.Exec(coreAlignmentTargetChecks).Error)
		before := coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, storeschema.InstallBrowserWorkbench(ctx, f.owner, f.name), "missing 1601")
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
		require.NoError(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
		require.NoError(t, f.owner.Exec(`ALTER TABLE workbench_stores ALTER COLUMN name TYPE varchar(100)`).Error)
		before = coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, storeschema.InstallBrowserWorkbench(ctx, f.owner, f.name), "core drift")
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
	})

	t.Run("two concurrent entries, preserved facts and ACLs, verified replay", func(t *testing.T) {
		f := newFixture(t)
		before := coreAlignmentSnapshot(t, f.owner)
		require.Error(t, alignCoreContract(f.config, "wrong_database", storecenter.RuntimeCapabilities{}))
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				errs <- alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{})
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, before, coreAlignmentSnapshot(t, f.owner))
		require.NoError(t, storecenter.VerifyCurrentSchemaForCapabilities(ctx, f.serving, storecenter.RuntimeCapabilities{}))
		require.NoError(t, storecenter.VerifyRuntimePermissions(ctx, f.serving))
		require.Error(t, storecenter.VerifyRuntimePermissions(ctx, f.owner))
		var count int64
		require.NoError(t, f.owner.Raw(`SELECT count(*) FROM goose_store_center_version WHERE version_id=2026101601 AND is_applied`).Scan(&count).Error)
		require.EqualValues(t, 1, count)
		complete := coreAlignmentFullSnapshot(t, f.owner)
		require.NoError(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}), "a lost success response is resolved by verified replay")
		require.Equal(t, complete, coreAlignmentFullSnapshot(t, f.owner))
	})

	t.Run("already current without version only appends Goose record", func(t *testing.T) {
		f := newFixture(t)
		require.NoError(t, f.owner.Exec(coreAlignmentTargetChecks).Error)
		before := coreAlignmentSnapshot(t, f.owner)
		var definitions string
		require.NoError(t, f.owner.Raw(`SELECT string_agg(oid::text||':'||pg_get_constraintdef(oid),',' ORDER BY conname) FROM pg_constraint WHERE connamespace='public'::regnamespace`).Scan(&definitions).Error)
		require.NoError(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
		require.Equal(t, before, coreAlignmentSnapshot(t, f.owner))
		var after string
		require.NoError(t, f.owner.Raw(`SELECT string_agg(oid::text||':'||pg_get_constraintdef(oid),',' ORDER BY conname) FROM pg_constraint WHERE connamespace='public'::regnamespace`).Scan(&after).Error)
		require.Equal(t, definitions, after, "B is not rewritten")
	})

	for _, tc := range []struct{ name, drift string }{
		{"mixed checks", `ALTER TABLE workbench_stores ADD CONSTRAINT store_walmart_market CHECK(platform<>'walmart' OR region='US')`},
		{"weak platform", `ALTER TABLE workbench_stores DROP CONSTRAINT store_platform, ADD CONSTRAINT store_platform CHECK(platform<>'')`},
		{"unvalidated platform", `ALTER TABLE workbench_stores DROP CONSTRAINT store_platform, ADD CONSTRAINT store_platform CHECK(platform IN ('shein','temu')) NOT VALID`},
		{"weak merchant", `ALTER TABLE workbench_store_merchant_bindings DROP CONSTRAINT store_merchant_binding_shape, ADD CONSTRAINT store_merchant_binding_shape CHECK(app_id<>'')`},
		{"column drift", `ALTER TABLE workbench_stores ALTER COLUMN name TYPE varchar(100)`},
		{"default drift", `ALTER TABLE workbench_stores ALTER COLUMN name SET DEFAULT 'changed'`},
		{"index drift", `DROP INDEX idx_workbench_stores_org_platform_region`},
		{"extra index", `CREATE INDEX unexpected_index ON workbench_stores(name)`},
		{"extra check", `ALTER TABLE workbench_stores ADD CONSTRAINT unexpected_check CHECK(name<>'')`},
		{"foreign key drift", `ALTER TABLE workbench_store_profiles DROP CONSTRAINT store_profile_store, ADD CONSTRAINT store_profile_store FOREIGN KEY(organization_id,store_id) REFERENCES workbench_stores(organization_id,id) ON DELETE CASCADE`},
		{"other check drift", `ALTER TABLE workbench_store_member_grants DROP CONSTRAINT store_member_grant_version, ADD CONSTRAINT store_member_grant_version CHECK(version>=0)`},
		{"missing history", `DELETE FROM goose_store_center_version WHERE version_id=2026092801`},
		{"unknown history", `INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(2026101501,true)`},
		{"false completed history", `INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(2026101601,true)`},
		{"grant missing", `REVOKE INSERT ON workbench_store_connections FROM store_center_runtime`},
		{"forbidden column grant", `GRANT UPDATE(app_id) ON workbench_store_merchant_bindings TO store_center_runtime`},
		{"grant option", `GRANT SELECT ON workbench_stores TO store_center_runtime WITH GRANT OPTION`},
		{"column grant option", `GRANT SELECT(name) ON workbench_stores TO store_center_runtime WITH GRANT OPTION`},
		{"unexpected table", `CREATE TABLE unexpected(id integer)`},
		{"row security", `ALTER TABLE workbench_stores ENABLE ROW LEVEL SECURITY`},
		{"applied state drift", coreAlignmentTargetChecks + `; INSERT INTO goose_store_center_version(version_id,is_applied) VALUES(2026101601,true); REVOKE INSERT ON workbench_store_connections FROM store_center_runtime`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			require.NoError(t, f.owner.Exec(tc.drift).Error)
			before := coreAlignmentFullSnapshot(t, f.owner)
			require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
			require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner), "refusal cannot repair schema, facts, history or privileges")
		})
	}

	t.Run("only original owner may maintain and runtime cannot inherit a role", func(t *testing.T) {
		f := newFixture(t)
		before := coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, storeschema.AlignCoreContract(ctx, f.serving, f.name, storecenter.RuntimeCapabilities{}))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
		require.NoError(t, admin.Exec(`GRANT retained_reader TO store_center_runtime`).Error)
		t.Cleanup(func() { require.NoError(t, admin.Exec(`REVOKE retained_reader FROM store_center_runtime`).Error) })
		require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
	})

	t.Run("full fresh installation keeps original entry and records new verifier once", func(t *testing.T) {
		f := newFixture(t)
		// Remove only this test fixture's objects, then exercise the real fresh
		// owner twice. There are no retained environment connections here.
		require.NoError(t, f.owner.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`).Error)
		require.NoError(t, storeschema.Migrate(ctx, f.owner))
		require.NoError(t, storeschema.Migrate(ctx, f.owner))
		var versions []int64
		require.NoError(t, f.owner.Raw(`SELECT version_id FROM goose_store_center_version ORDER BY id`).Scan(&versions).Error)
		require.Equal(t, []int64{0, 2026092801, 2026101101, 2026101601}, versions)
		require.NoError(t, storecenter.VerifyCurrentSchema(ctx, f.owner))
		require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}), "core maintenance cannot broaden into optional platform tables")
	})

	t.Run("default fresh mode refuses retained core A and B", func(t *testing.T) {
		f := newFixture(t)
		for range 2 {
			before := coreAlignmentFullSnapshot(t, f.owner)
			require.Error(t, storeschema.Migrate(ctx, f.owner))
			require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
			require.NoError(t, f.owner.Exec(coreAlignmentTargetChecks).Error)
		}
	})

	t.Run("late Goose record failure rolls back all checks and can retry", func(t *testing.T) {
		f := newFixture(t)
		require.NoError(t, f.owner.Exec(`CREATE FUNCTION reject_alignment_record() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.version_id=2026101601 THEN RAISE EXCEPTION 'isolated late failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER reject_alignment_record BEFORE INSERT ON goose_store_center_version FOR EACH ROW EXECUTE FUNCTION reject_alignment_record()`).Error)
		before := coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
		require.NoError(t, f.owner.Exec(`DROP TRIGGER reject_alignment_record ON goose_store_center_version; DROP FUNCTION reject_alignment_record()`).Error)
		require.NoError(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
	})

	t.Run("cancelled lock wait does not change retained database", func(t *testing.T) {
		f := newFixture(t)
		pool, err := f.owner.DB()
		require.NoError(t, err)
		lock, err := pool.BeginTx(ctx, nil)
		require.NoError(t, err)
		require.NoError(t, func() error {
			_, err := lock.ExecContext(ctx, `LOCK TABLE workbench_stores IN ACCESS EXCLUSIVE MODE`)
			return err
		}())
		short, stop := context.WithTimeout(ctx, 200*time.Millisecond)
		require.Error(t, storeschema.AlignCoreContract(short, f.owner, f.name, storecenter.RuntimeCapabilities{}))
		stop()
		require.NoError(t, lock.Rollback())
		require.NoError(t, alignCoreContract(f.config, f.name, storecenter.RuntimeCapabilities{}))
	})
}

const coreAlignmentTargetChecks = `ALTER TABLE workbench_stores DROP CONSTRAINT store_platform, DROP CONSTRAINT IF EXISTS store_tiktok_identity, DROP CONSTRAINT IF EXISTS store_walmart_market,
ADD CONSTRAINT store_platform CHECK(platform IN ('shein','temu','tiktokshop','walmart')),
ADD CONSTRAINT store_tiktok_identity CHECK(platform<>'tiktokshop' OR (region='US' AND external_store_id<>'')),
ADD CONSTRAINT store_walmart_market CHECK(platform<>'walmart' OR region='US');
ALTER TABLE workbench_store_merchant_bindings DROP CONSTRAINT store_merchant_binding_shape,
ADD CONSTRAINT store_merchant_binding_shape CHECK(octet_length(identity_hash)=64 AND app_id<>'' AND ((platform='shein' AND open_key_id<>'') OR (platform IN ('temu','walmart') AND open_key_id='')))`

func openCoreAlignmentDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(2)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func seedCoreAlignmentFacts(t *testing.T, db *gorm.DB) {
	t.Helper()
	for i, platform := range []string{"shein", "temu"} {
		id := fmt.Sprintf("00000000-0000-4000-8000-%012d", i+1)
		openKey := ""
		if platform == "shein" {
			openKey = "private-key-reference"
		}
		partition, method, digest, verification := "", "callback", "", ""
		if platform == "temu" {
			partition, method, digest, verification = "GLOBAL", "manual_token", strings.Repeat("d", 64), "{}"
		}
		require.NoError(t, db.Exec(fmt.Sprintf(`
INSERT INTO workbench_stores VALUES('%[1]s','org','%[2]s','%[2]s','US','external','active','pending_activation',NULL,NULL,'retained-ref',1,'actor','actor',now(),now(),NULL,'%[1]s','','%[3]s','%[4]s');
INSERT INTO workbench_store_audit_logs VALUES('%[1]s','org','%[1]s','%[1]s','create','success','actor','name','','','active','',1,now(),now());
INSERT INTO workbench_store_member_grants VALUES('org','%[1]s','member',true,1,'actor',now());
INSERT INTO workbench_store_member_grant_operations VALUES('org','%[1]s','%[1]s','member',true,0,1,'actor','retained-fingerprint',now());
INSERT INTO workbench_store_service_operations VALUES('org','%[1]s','%[1]s','{}','','admitted','','','',now(),now());
INSERT INTO workbench_store_connections VALUES('org','%[1]s','%[1]s',1,'disconnected',NULL);
INSERT INTO workbench_store_connection_attempts VALUES('org','%[1]s','%[1]s','actor','member','app','v1','%[2]s','%[6]s','%[7]s','%[8]s','%[9]s','%[4]s',1,1,'verified','key','retained-ciphertext',now()+interval '4 minutes',now(),now());
INSERT INTO workbench_store_merchant_bindings VALUES('%[3]s','%[2]s','app','%[5]s','org','%[1]s');
INSERT INTO workbench_store_groups VALUES('org','%[1]s','retained-group',1,'actor',now(),NULL);
INSERT INTO workbench_store_profiles VALUES('org','%[1]s','%[1]s','member',7,1,'actor',now());
INSERT INTO workbench_store_metadata_operations VALUES('org','%[1]s','actor','member','set_profile','%[4]s','{}','{}',now());`,
			id, platform, strings.Repeat(fmt.Sprint(i), 64), strings.Repeat("f", 64), openKey, partition, method, digest, verification)).Error)
	}
}

// The allowed four CHECK definitions and new version row are excluded; every
// original business row, history row, other schema object and ACL is compared.
func coreAlignmentSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var tables []struct{ Schema, Name string }
	require.NoError(t, db.Raw(`SELECT schemaname AS schema,tablename AS name FROM pg_tables WHERE schemaname !~ '^pg_' AND schemaname<>'information_schema' ORDER BY schemaname,tablename`).Scan(&tables).Error)
	parts := []string{}
	for _, table := range tables {
		where := ""
		if table.Schema == "public" && table.Name == "goose_store_center_version" {
			where = " WHERE version_id<>2026101601"
		}
		var data string
		qualified := `"` + strings.ReplaceAll(table.Schema, `"`, `""`) + `"."` + strings.ReplaceAll(table.Name, `"`, `""`) + `"`
		require.NoError(t, db.Raw(`SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM `+qualified+` t`+where).Scan(&data).Error)
		parts = append(parts, qualified+":"+data)
	}
	for _, query := range []string{
		`SELECT coalesce(jsonb_agg(jsonb_build_array(n.nspname,n.nspowner,n.nspacl) ORDER BY n.nspname),'[]')::text FROM pg_namespace n WHERE n.nspname !~ '^pg_' AND n.nspname<>'information_schema'`,
		`SELECT coalesce(jsonb_agg(jsonb_build_array(n.nspname,c.relname,c.relkind,c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity) ORDER BY n.nspname,c.relname),'[]')::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('store_observations','operations_cockpit')`,
		`SELECT coalesce(jsonb_agg(jsonb_build_array(c.oid::regclass::text,a.attname,a.attacl,a.atttypid,a.atttypmod,a.attnotnull,pg_get_expr(d.adbin,d.adrelid)) ORDER BY c.oid,a.attnum),'[]')::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE n.nspname IN ('store_observations','operations_cockpit') AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped`,
		`SELECT coalesce(jsonb_agg(jsonb_build_array(conrelid::regclass::text,conname,pg_get_constraintdef(k.oid),convalidated) ORDER BY conrelid::regclass::text,conname),'[]')::text FROM pg_constraint k JOIN pg_namespace n ON n.oid=k.connamespace WHERE n.nspname IN ('store_observations','operations_cockpit')`,
		`SELECT coalesce(jsonb_agg(jsonb_build_array(indexrelid::regclass::text,pg_get_indexdef(indexrelid),indisvalid,indisready) ORDER BY indexrelid::regclass::text),'[]')::text FROM pg_index WHERE indrelid IN(SELECT c.oid FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname IN ('store_observations','operations_cockpit'))`,
		`SELECT coalesce(jsonb_agg(jsonb_build_array(c.relname,c.relkind,c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity) ORDER BY c.relname),'[]')::text FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public' AND c.relkind IN ('r','p')`,
		`SELECT jsonb_agg(jsonb_build_array(c.relname,a.attname,a.attacl,a.atttypid,a.atttypmod,a.attnotnull,pg_get_expr(d.adbin,d.adrelid)) ORDER BY c.relname,a.attnum)::text FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE c.relnamespace='public'::regnamespace AND c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped`,
		`SELECT jsonb_agg(jsonb_build_array(conrelid::regclass::text,conname,pg_get_constraintdef(oid),convalidated) ORDER BY conrelid::regclass::text,conname)::text FROM pg_constraint WHERE connamespace='public'::regnamespace AND conname NOT IN ('store_platform','store_tiktok_identity','store_walmart_market','store_merchant_binding_shape')`,
		`SELECT jsonb_agg(jsonb_build_array(indexrelid::regclass::text,pg_get_indexdef(indexrelid),indisvalid,indisready) ORDER BY indexrelid::regclass::text)::text FROM pg_index WHERE indrelid IN(SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace)`,
		`SELECT jsonb_build_array((SELECT datacl FROM pg_database WHERE datname=current_database()),(SELECT nspacl FROM pg_namespace WHERE nspname='public'),(SELECT coalesce(jsonb_agg(to_jsonb(d)),'[]') FROM pg_default_acl d))::text`,
	} {
		var value string
		require.NoError(t, db.Raw(query).Scan(&value).Error)
		parts = append(parts, value)
	}
	return strings.Join(parts, "\n")
}

func coreAlignmentFullSnapshot(t *testing.T, db *gorm.DB) string {
	t.Helper()
	var checks, history string
	require.NoError(t, db.Raw(`SELECT jsonb_agg(jsonb_build_array(conname,pg_get_constraintdef(oid),convalidated) ORDER BY conname)::text FROM pg_constraint WHERE connamespace='public'::regnamespace AND conname IN ('store_platform','store_tiktok_identity','store_walmart_market','store_merchant_binding_shape')`).Scan(&checks).Error)
	require.NoError(t, db.Raw(`SELECT jsonb_agg(to_jsonb(t) ORDER BY id)::text FROM goose_store_center_version t`).Scan(&history).Error)
	return coreAlignmentSnapshot(t, db) + "\n" + checks + "\n" + history
}
