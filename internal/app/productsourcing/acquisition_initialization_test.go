package productsourcing

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestEnablePlansPreservesExistingFactsAndDoesNotOpenExecution(t *testing.T) {
	db := acquisitionDatabase(t)
	ctx := context.Background()
	require.NoError(t, InstallAcquisitionSchema(db))
	require.NoError(t, collectionstore.InstallSchema(db))
	require.NoError(t, db.Exec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='source_acquisition_runtime') THEN CREATE ROLE source_acquisition_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE retained_fixture(id integer PRIMARY KEY); INSERT INTO retained_fixture VALUES(7); GRANT SELECT ON retained_fixture TO source_acquisition_runtime`).Error)
	parsed, e := pgx.ParseConfig(os.Getenv("ISSUE398_TEST_DSN"))
	require.NoError(t, e)
	var name string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&name).Error)
	body, e := json.Marshal(map[string]any{"schemaVersion": 1, "collections": true, "acquisitionPlans": true, "database": map[string]any{"host": "127.0.0.1", "port": parsed.Port, "user": parsed.User, "password": parsed.Password, "database": name}})
	require.NoError(t, e)
	path := filepath.Join(t.TempDir(), "feature.json")
	require.NoError(t, os.WriteFile(path, body, 0600))
	require.Error(t, EnableAcquisitionPlans(ctx, path, "wrong"))
	require.NoError(t, EnableAcquisitionPlans(ctx, path, name))
	require.NoError(t, EnableAcquisitionPlans(ctx, path, name))
	var count int
	require.NoError(t, db.Raw("SELECT count(*) FROM retained_fixture WHERE id=7").Scan(&count).Error)
	require.Equal(t, 1, count)
	var granted bool
	require.NoError(t, db.Raw("SELECT has_table_privilege('source_acquisition_runtime','retained_fixture','SELECT')").Scan(&granted).Error)
	require.True(t, granted)
	var executionMissing bool
	require.NoError(t, db.Raw("SELECT to_regclass('public.listing_preparation_operations') IS NULL AND to_regclass('public.listing_target_records') IS NULL").Scan(&executionMissing).Error)
	require.True(t, executionMissing)
	require.NoError(t, db.Raw("SELECT has_table_privilege('source_acquisition_runtime','product_acquisition_plans','SELECT,INSERT,UPDATE')").Scan(&granted).Error)
	require.True(t, granted)
	require.NoError(t, db.Raw("SELECT has_table_privilege('source_acquisition_runtime','product_acquisition_plans','DELETE')").Scan(&granted).Error)
	require.False(t, granted)
}

func TestAcquisitionExplicitInitializerRequiresEmptyConfirmedDatabase(t *testing.T) {
	db := acquisitionDatabase(t)
	parsed, err := pgx.ParseConfig(os.Getenv("ISSUE398_TEST_DSN"))
	require.NoError(t, err)
	var name string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&name).Error)
	require.NoError(t, db.Exec(`DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='source_acquisition_runtime') THEN CREATE ROLE source_acquisition_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$`).Error)
	manifest := filepath.Join(t.TempDir(), "init.json")
	raw, err := json.Marshal(map[string]any{"schemaVersion": 1, "database": map[string]any{"host": "127.0.0.1", "port": parsed.Port, "user": parsed.User, "password": parsed.Password, "database": name}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(manifest, raw, 0600))
	require.Error(t, InitializeAcquisitionDatabase(context.Background(), manifest, "another-database"))
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'").Scan(&count).Error)
	require.Zero(t, count)
	require.NoError(t, InitializeAcquisitionDatabase(context.Background(), manifest, name))
	require.NoError(t, db.Raw("SELECT count(*) FROM information_schema.tables WHERE table_schema='public'").Scan(&count).Error)
	require.EqualValues(t, 6, count)
	require.Error(t, InitializeAcquisitionDatabase(context.Background(), manifest, name), "never migrate or repair a nonempty database")
}

func TestAcquisitionInitializerRejectsUnsafeManifestWithoutConnecting(t *testing.T) {
	for _, body := range []string{`{}`, `{"schemaVersion":1,"schemaVersion":1}`, `{"schemaVersion":1,"database":{"host":"remote.example"}}`} {
		path := filepath.Join(t.TempDir(), "init.json")
		require.NoError(t, os.WriteFile(path, []byte(body), 0600))
		require.Error(t, InitializeAcquisitionDatabase(context.Background(), path, "product"))
	}
	require.Error(t, InitializeAcquisitionDatabase(context.Background(), "relative.json", "product"))
}
