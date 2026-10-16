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
	platformmigration "task-processor/internal/platform/database/migration"
	"task-processor/internal/storecenter"
)

type coreAlignmentFixture struct {
	owner, serving *gorm.DB
	name, config   string
}

func TestCoreAlignmentPreservesRetainedStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
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

	t.Run("two concurrent entries, preserved facts and ACLs, verified replay", func(t *testing.T) {
		f := newFixture(t)
		before := coreAlignmentSnapshot(t, f.owner)
		require.Error(t, alignCoreContract(f.config, "wrong_database"))
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for range 2 {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- alignCoreContract(f.config, f.name) }()
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
		require.NoError(t, alignCoreContract(f.config, f.name), "a lost success response is resolved by verified replay")
		require.Equal(t, complete, coreAlignmentFullSnapshot(t, f.owner))
	})

	t.Run("already current without version only appends Goose record", func(t *testing.T) {
		f := newFixture(t)
		require.NoError(t, f.owner.Exec(coreAlignmentTargetChecks).Error)
		before := coreAlignmentSnapshot(t, f.owner)
		var definitions string
		require.NoError(t, f.owner.Raw(`SELECT string_agg(oid::text||':'||pg_get_constraintdef(oid),',' ORDER BY conname) FROM pg_constraint WHERE connamespace='public'::regnamespace`).Scan(&definitions).Error)
		require.NoError(t, alignCoreContract(f.config, f.name))
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
			require.Error(t, alignCoreContract(f.config, f.name))
			require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner), "refusal cannot repair schema, facts, history or privileges")
		})
	}

	t.Run("only original owner may maintain and runtime cannot inherit a role", func(t *testing.T) {
		f := newFixture(t)
		before := coreAlignmentFullSnapshot(t, f.owner)
		require.Error(t, storeschema.AlignCoreContract(ctx, f.serving, f.name))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
		require.NoError(t, admin.Exec(`GRANT retained_reader TO store_center_runtime`).Error)
		t.Cleanup(func() { require.NoError(t, admin.Exec(`REVOKE retained_reader FROM store_center_runtime`).Error) })
		require.Error(t, alignCoreContract(f.config, f.name))
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
		require.Error(t, alignCoreContract(f.config, f.name), "core maintenance cannot broaden into optional platform tables")
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
		require.Error(t, alignCoreContract(f.config, f.name))
		require.Equal(t, before, coreAlignmentFullSnapshot(t, f.owner))
		require.NoError(t, f.owner.Exec(`DROP TRIGGER reject_alignment_record ON goose_store_center_version; DROP FUNCTION reject_alignment_record()`).Error)
		require.NoError(t, alignCoreContract(f.config, f.name))
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
		require.Error(t, storeschema.AlignCoreContract(short, f.owner, f.name))
		stop()
		require.NoError(t, lock.Rollback())
		require.NoError(t, alignCoreContract(f.config, f.name))
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
	var tables []string
	require.NoError(t, db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename`).Scan(&tables).Error)
	parts := []string{}
	for _, table := range tables {
		where := ""
		if table == "goose_store_center_version" {
			where = " WHERE version_id<>2026101601"
		}
		var data string
		require.NoError(t, db.Raw(`SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM public.`+table+` t`+where).Scan(&data).Error)
		parts = append(parts, table+":"+data)
	}
	for _, query := range []string{
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
