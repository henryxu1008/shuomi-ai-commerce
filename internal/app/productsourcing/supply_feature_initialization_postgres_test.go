package productsourcing

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
)

type supplyFeatureFixture struct {
	owner, runtime *gorm.DB
	manifest       string
	database       string
	baseTables     []string
}

func TestSupplyFeaturePreservesNonemptyProductAndRejectsDrift(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:17.2-alpine", tcpostgres.WithDatabase("installer_tests"), tcpostgres.WithUsername("test_admin"), tcpostgres.WithPassword("isolated-admin"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	admin := openSupplyFeatureTestDB(t, dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE acquisition_owner LOGIN PASSWORD 'isolated-owner' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE source_acquisition_runtime LOGIN PASSWORD 'isolated-runtime' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE retained_reader NOLOGIN`).Error)
	newFixture := func(t *testing.T) supplyFeatureFixture {
		t.Helper()
		name := "supply_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, admin.Exec("CREATE DATABASE "+name+" OWNER acquisition_owner").Error)
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		parsed.Path, parsed.User = "/"+name, url.UserPassword("acquisition_owner", "isolated-owner")
		owner := openSupplyFeatureTestDB(t, parsed.String())
		port, err := strconv.Atoi(parsed.Port())
		require.NoError(t, err)
		manifest := acquisitionInitManifest{SchemaVersion: 1, Collections: true, CompletionDrafts: true}
		manifest.Database.Host, manifest.Database.Port = "127.0.0.1", port
		manifest.Database.User, manifest.Database.Password, manifest.Database.Database = "acquisition_owner", "isolated-owner", name
		path := filepath.Join(t.TempDir(), "owner.json")
		writeSupplyFeatureManifest(t, path, manifest)
		require.NoError(t, InitializeAcquisitionDatabase(ctx, path, name))
		seedSupplyFeatureRows(t, owner)
		require.NoError(t, owner.Exec("GRANT SELECT ON product_collection_items TO retained_reader").Error)
		var tables []string
		require.NoError(t, owner.Raw("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").Scan(&tables).Error)
		require.Len(t, tables, 14)
		manifest.SupplyChain = true
		writeSupplyFeatureManifest(t, path, manifest)
		parsed.User = url.UserPassword(acquisitionstore.RuntimeRole, "isolated-runtime")
		return supplyFeatureFixture{owner: owner, runtime: openSupplyFeatureTestDB(t, parsed.String()), manifest: path, database: name, baseTables: tables}
	}

	t.Run("atomic concurrent install and receipt verification preserve all existing facts and grants", func(t *testing.T) {
		f := newFixture(t)
		before := supplyFeatureSnapshot(t, f.owner, f.baseTables)
		require.NoError(t, acquisitionstore.VerifyOwnerRuntimePermissions(ctx, f.owner, acquisitionstore.RuntimeCapabilities{Collections: true, CompletionDrafts: true}))
		require.Error(t, EnableSupplyChain(ctx, f.manifest, "wrong_database"))
		var wg sync.WaitGroup
		errs := make(chan error, 2)
		for range 2 {
			wg.Add(1)
			go func() { defer wg.Done(); errs <- EnableSupplyChain(ctx, f.manifest, f.database) }()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}
		require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, f.baseTables))
		capabilities := acquisitionstore.RuntimeCapabilities{Collections: true, CompletionDrafts: true, SupplyChain: true}
		require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, f.runtime, capabilities))
		require.Error(t, acquisitionstore.VerifyRuntimePermissions(ctx, f.owner, capabilities), "maintenance verification must not admit serving as owner")
		var tables []string
		require.NoError(t, f.owner.Raw("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").Scan(&tables).Error)
		require.Len(t, tables, 25)
		require.NoError(t, f.owner.Exec(`INSERT INTO product_title_operations VALUES ('org','actor','retained-command','fingerprint','retained-response')`).Error)
		installed := supplyFeatureSnapshot(t, f.owner, tables)
		require.NoError(t, EnableSupplyChain(ctx, f.manifest, f.database), "lost success response can verify without re-installation")
		require.Equal(t, installed, supplyFeatureSnapshot(t, f.owner, tables))
		var denied bool
		require.NoError(t, f.runtime.Raw(`SELECT NOT has_table_privilege(current_user,'product_title_operations','SELECT,INSERT,UPDATE,DELETE') AND NOT has_table_privilege(current_user,'listing_target_records','UPDATE,DELETE')`).Scan(&denied).Error)
		require.True(t, denied)
	})

	for _, tc := range []struct {
		name, change string
		installed    bool
	}{
		{"partial installation", "CREATE TABLE product_title_operations(id integer)", false},
		{"missing old grant", "REVOKE INSERT ON listing_completion_draft_records FROM source_acquisition_runtime", false},
		{"old column drift", "ALTER TABLE listing_preparations RENAME COLUMN revision TO drifted", false},
		{"old index missing", "DROP INDEX preparation_source_page", false},
		{"old grant option", "GRANT SELECT ON product_collection_items TO source_acquisition_runtime WITH GRANT OPTION", false},
		{"old column grant option", "GRANT SELECT(publication_id) ON product_collection_items TO source_acquisition_runtime WITH GRANT OPTION", false},
		{"unexpected sequence privilege", "CREATE SEQUENCE retained_sequence; GRANT USAGE ON retained_sequence TO source_acquisition_runtime", false},
		{"Review column drift", "ALTER TABLE product_title_operations ALTER COLUMN fingerprint TYPE varchar(8)", true},
		{"Review constraint drift", "ALTER TABLE product_title_proposals DROP CONSTRAINT ck_product_title_proposals_state_payload", true},
		{"Review index missing", "DROP INDEX ix_product_title_proposals_actionable_owner", true},
		{"new column grant", "GRANT UPDATE(body_json) ON listing_target_records TO source_acquisition_runtime", true},
		{"new grant option", "GRANT SELECT ON listing_target_records TO source_acquisition_runtime WITH GRANT OPTION", true},
		{"new column grant option", "GRANT SELECT(body_json) ON listing_target_records TO source_acquisition_runtime WITH GRANT OPTION", true},
		{"missing new grant", "REVOKE INSERT ON listing_target_records FROM source_acquisition_runtime", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.installed {
				require.NoError(t, EnableSupplyChain(ctx, f.manifest, f.database))
			}
			require.NoError(t, f.owner.Exec(tc.change).Error)
			var tables []string
			require.NoError(t, f.owner.Raw("SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename").Scan(&tables).Error)
			before := supplyFeatureSnapshot(t, f.owner, tables)
			require.Error(t, EnableSupplyChain(ctx, f.manifest, f.database))
			require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, tables), "refusal must not repair a drift or an ACL")
			var count int64
			require.NoError(t, f.owner.Raw("SELECT count(*) FROM pg_tables WHERE schemaname='public'").Scan(&count).Error)
			require.Len(t, tables, int(count))
		})
	}

	t.Run("feature manifest cannot expand or omit its exact baseline scope", func(t *testing.T) {
		f := newFixture(t)
		before := supplyFeatureSnapshot(t, f.owner, f.baseTables)
		raw, err := os.ReadFile(f.manifest)
		require.NoError(t, err)
		for _, change := range []func(*acquisitionInitManifest){
			func(m *acquisitionInitManifest) { m.CompletionDrafts = false },
			func(m *acquisitionInitManifest) { m.SupplyChain = false },
			func(m *acquisitionInitManifest) { m.DataServices = true },
			func(m *acquisitionInitManifest) { m.ImageSets = true },
			func(m *acquisitionInitManifest) { m.AcquisitionPlans = true },
			func(m *acquisitionInitManifest) { m.Database.User = acquisitionstore.RuntimeRole },
		} {
			var manifest acquisitionInitManifest
			require.NoError(t, json.Unmarshal(raw, &manifest))
			change(&manifest)
			writeSupplyFeatureManifest(t, f.manifest, manifest)
			require.Error(t, EnableSupplyChain(ctx, f.manifest, f.database))
		}
		require.Error(t, EnableSupplyChain(nil, f.manifest, f.database))
		require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, f.baseTables))
	})

	t.Run("a different maintenance identity cannot adopt the existing owner", func(t *testing.T) {
		f := newFixture(t)
		before := supplyFeatureSnapshot(t, f.owner, f.baseTables)
		var manifest acquisitionInitManifest
		raw, err := os.ReadFile(f.manifest)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &manifest))
		manifest.Database.User, manifest.Database.Password = "test_admin", "isolated-admin"
		writeSupplyFeatureManifest(t, f.manifest, manifest)
		require.Error(t, EnableSupplyChain(ctx, f.manifest, f.database), "even a cluster administrator is not the current table owner")
		require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, f.baseTables))
	})

	t.Run("late permission refusal rolls back every new table and grant", func(t *testing.T) {
		f := newFixture(t)
		require.NoError(t, f.owner.Exec("ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT UPDATE ON TABLES TO source_acquisition_runtime").Error)
		before := supplyFeatureSnapshot(t, f.owner, f.baseTables)
		require.Error(t, EnableSupplyChain(ctx, f.manifest, f.database))
		require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, f.baseTables))
		var count int64
		require.NoError(t, f.owner.Raw("SELECT count(*) FROM pg_tables WHERE schemaname='public'").Scan(&count).Error)
		require.EqualValues(t, 14, count)
		require.NoError(t, f.owner.Exec("ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE UPDATE ON TABLES FROM source_acquisition_runtime").Error)
		require.NoError(t, EnableSupplyChain(ctx, f.manifest, f.database), "a rolled-back attempt can be safely retried")
	})

	t.Run("cancelled lock wait has no persisted effect", func(t *testing.T) {
		f := newFixture(t)
		before := supplyFeatureSnapshot(t, f.owner, f.baseTables)
		lock := f.owner.Begin()
		require.NoError(t, lock.Error)
		defer lock.Rollback()
		require.NoError(t, lock.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':source-acquisition-init',0))").Error)
		short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		require.Error(t, EnableSupplyChain(short, f.manifest, f.database))
		require.Equal(t, before, supplyFeatureSnapshot(t, f.owner, f.baseTables))
	})
}

func openSupplyFeatureTestDB(t *testing.T, dsn string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	pool.SetMaxOpenConns(4)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func writeSupplyFeatureManifest(t *testing.T, path string, manifest acquisitionInitManifest) {
	t.Helper()
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0600))
}

func seedSupplyFeatureRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	// Valid FK-linked retained facts; image URLs and structured zero values are
	// opaque saved content to this installer, never fetched or rewritten.
	require.NoError(t, db.Exec(`
INSERT INTO product_snapshot_versions(tenant_id,product_key,version,publication_id,payload_hash,snapshot_json) VALUES ('org','own-product',1,'publication',repeat('1',64),'{"images":["https://images.example.org/retained.png"]}');
INSERT INTO product_collection_batches VALUES ('org','actor','member','11111111-1111-4111-8111-111111111111','retained','own',1,now(),NULL);
INSERT INTO product_collection_items VALUES ('org','actor','member','22222222-2222-4222-8222-222222222222','11111111-1111-4111-8111-111111111111','own-product','publication',1,'own','own-operation',1,now(),NULL);
INSERT INTO listing_preparations VALUES ('org','actor','member','33333333-3333-4333-8333-333333333333','44444444-4444-4444-8444-444444444444',repeat('2',64),'{}','11111111-1111-4111-8111-111111111111',1,'retained preparation',1,1,now());
INSERT INTO listing_preparation_sources VALUES ('org','actor','member','55555555-5555-4555-8555-555555555555','33333333-3333-4333-8333-333333333333','22222222-2222-4222-8222-222222222222',1,'own-product','publication',1,'own','own-operation');
INSERT INTO listing_completion_draft_records VALUES ('org','actor','member','66666666-6666-4666-8666-666666666666','55555555-5555-4555-8555-555555555555','77777777-7777-4777-8777-777777777777',1,'{"version":1}','{"title":"retained","structured":{"skus":[{"price":0,"stock":0}]}}',repeat('3',64),now());
INSERT INTO listing_completion_draft_heads VALUES ('org','actor','member','55555555-5555-4555-8555-555555555555','66666666-6666-4666-8666-666666666666','77777777-7777-4777-8777-777777777777',1);
INSERT INTO listing_completion_draft_commands VALUES ('org','actor','member','88888888-8888-4888-8888-888888888888',repeat('3',64),'{}','66666666-6666-4666-8666-666666666666','55555555-5555-4555-8555-555555555555','77777777-7777-4777-8777-777777777777',1);`).Error)
}

func supplyFeatureSnapshot(t *testing.T, db *gorm.DB, tables []string) map[string]string {
	t.Helper()
	result := map[string]string{}
	for _, table := range tables {
		var snapshot string
		query := `SELECT jsonb_build_object('rows',(SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)), '[]') FROM ` + table + ` t),
 'owner',pg_get_userbyid(c.relowner),'acl',c.relacl::text,
 'columns',(SELECT jsonb_agg(jsonb_build_array(a.attname,format_type(a.atttypid,a.atttypmod),a.attnotnull,a.attacl::text) ORDER BY a.attnum) FROM pg_attribute a WHERE a.attrelid=c.oid AND a.attnum>0 AND NOT a.attisdropped),
 'constraints',(SELECT jsonb_agg(pg_get_constraintdef(k.oid) ORDER BY k.conname) FROM pg_constraint k WHERE k.conrelid=c.oid),
 'indexes',(SELECT jsonb_agg(pg_get_indexdef(i.indexrelid) ORDER BY i.indexrelid::regclass::text) FROM pg_index i WHERE i.indrelid=c.oid))::text
 FROM pg_class c WHERE c.oid=to_regclass(?)`
		require.NoError(t, db.Raw(query, "public."+table).Scan(&snapshot).Error)
		result[table] = snapshot
	}
	return result
}
