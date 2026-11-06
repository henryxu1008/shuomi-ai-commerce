//go:build integration

package productsourcing

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"path/filepath"
	acquisitionstore "task-processor/internal/integration/persistence/product/acquisition"
	researchstore "task-processor/internal/integration/persistence/product/research"
	"testing"
	"time"
)

func TestDataServicesFreshInstallerKeepsProductRolesNarrow(t *testing.T) {
	t.Run("data services", func(t *testing.T) { verifyDataServicesFreshInstaller(t, false) })
	t.Run("data services with supply market", func(t *testing.T) { verifyDataServicesFreshInstaller(t, true) })
}

func verifyDataServicesFreshInstaller(t *testing.T, supplyMarket bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("product_acquisition"), tcpostgres.WithUsername("test_owner"), tcpostgres.WithPassword("isolated-test-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	address, err := url.Parse(dsn)
	require.NoError(t, err)
	open := func(address string, max int) *gorm.DB {
		db, err := gorm.Open(postgres.Open(address), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		pool.SetMaxOpenConns(max)
		t.Cleanup(func() { require.NoError(t, pool.Close()) })
		return db
	}
	owner := open(dsn, 2)
	require.NoError(t, owner.Exec(`CREATE ROLE source_acquisition_runtime LOGIN PASSWORD 'runtime-fixture'; CREATE ROLE data_services_runtime LOGIN PASSWORD 'runtime-fixture'; CREATE ROLE product_research_runtime LOGIN PASSWORD 'runtime-fixture'`).Error)
	port, err := container.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "install.json")
	manifest := map[string]any{"schemaVersion": 1, "collections": true, "dataServices": true, "supplyMarket": supplyMarket, "researchSupplierReview": supplyMarket, "database": map[string]any{"host": "127.0.0.1", "port": int(port.Num()), "database": "product_acquisition", "user": "test_owner", "password": "isolated-test-only"}}
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0600))
	require.NoError(t, InitializeAcquisitionDatabase(ctx, path, "product_acquisition"))
	require.Error(t, InitializeAcquisitionDatabase(ctx, path, "product_acquisition"), "serving/retained installs cannot repair or migrate Product")
	sourceURL := *address
	sourceURL.User = url.UserPassword(acquisitionstore.RuntimeRole, "runtime-fixture")
	dataURL := *address
	dataURL.User = url.UserPassword(acquisitionstore.DataServicesRuntimeRole, "runtime-fixture")
	source := open(sourceURL.String(), 4)
	data := open(dataURL.String(), 4)
	require.NoError(t, acquisitionstore.VerifyRuntimePermissions(ctx, source, acquisitionstore.RuntimeCapabilities{Collections: true, SupplyMarket: supplyMarket}))
	require.NoError(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data))
	if !supplyMarket {
		verifySHEINLedgerInstallation(t, ctx, path, owner, data)
		var installed bool
		require.NoError(t, owner.Raw("SELECT to_regclass('public.research_supplier_reviews') IS NOT NULL").Scan(&installed).Error)
		require.False(t, installed)
		manifest["researchSupplierReview"] = true
		raw, err := json.Marshal(manifest)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, raw, 0600))
		require.Error(t, InitializeAcquisitionDatabase(ctx, path, "product_acquisition"))
		require.NoError(t, owner.Raw("SELECT to_regclass('public.research_supplier_reviews') IS NOT NULL").Scan(&installed).Error)
		require.False(t, installed, "existing Product database cannot be upgraded")
	} else {
		researchURL := *address
		researchURL.User = url.UserPassword(researchstore.RuntimeRole, "runtime-fixture")
		researchDB := open(researchURL.String(), 4)
		require.NoError(t, researchstore.VerifyRuntimePermissions(ctx, researchDB))
		require.NoError(t, researchstore.VerifySupplierReviewSchema(ctx, researchDB))
		require.Error(t, researchDB.Exec("DELETE FROM public.research_supplier_reviews").Error)
		require.Error(t, researchDB.Exec("UPDATE public.research_supplier_reviews SET record_json=record_json").Error)
		require.Error(t, source.Exec("SELECT record_json FROM public.research_supplier_reviews").Error)
		require.Error(t, data.Exec("SELECT record_json FROM public.research_supplier_reviews").Error)
		require.NoError(t, owner.Exec("GRANT SELECT ON public.research_supplier_reviews TO PUBLIC").Error)
		require.Error(t, researchstore.VerifyRuntimePermissions(ctx, researchDB), "unrelated roles cannot receive supplier evidence")
		require.NoError(t, owner.Exec("REVOKE SELECT ON public.research_supplier_reviews FROM PUBLIC").Error)
		require.NoError(t, researchstore.VerifyRuntimePermissions(ctx, researchDB))
	}
	require.Error(t, source.Exec("SELECT id FROM data_service_credentials LIMIT 0").Error)
	require.Error(t, data.Exec("SELECT operation_id FROM product_acquisition_operations LIMIT 0").Error)
	require.Error(t, data.Exec("CREATE TABLE forbidden_runtime_ddl(id int)").Error)
	require.Error(t, data.Exec("DELETE FROM data_service_commands WHERE false").Error)
	require.NoError(t, owner.Exec("GRANT SELECT (operation_id) ON product_acquisition_operations TO data_services_runtime").Error)
	require.Error(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data), "extra column privileges fail closed")
	require.NoError(t, owner.Exec("REVOKE SELECT (operation_id) ON product_acquisition_operations FROM data_services_runtime").Error)
	require.NoError(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data))
	require.NoError(t, owner.Exec("GRANT test_owner TO data_services_runtime").Error)
	require.Error(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data), "owner membership fails closed")
}

func verifySHEINLedgerInstallation(t *testing.T, ctx context.Context, manifest string, owner, data *gorm.DB) {
	t.Helper()
	assertAbsent := func() {
		var missing bool
		require.NoError(t, owner.Raw("SELECT to_regclass('public.data_acquisition_shein_us_trials') IS NULL").Scan(&missing).Error)
		require.True(t, missing, "refused installation must leave the ledger absent")
	}
	require.Error(t, EnableSHEINUSTrialLedger(ctx, manifest, "wrong_product"))
	assertAbsent()
	require.NoError(t, owner.Exec("ALTER TABLE data_acquisition_jobs RENAME COLUMN trial_binding TO fixture_old_binding").Error)
	require.Error(t, EnableSHEINUSTrialLedger(ctx, manifest, "product_acquisition"))
	assertAbsent()
	require.NoError(t, owner.Exec("ALTER TABLE data_acquisition_jobs RENAME COLUMN fixture_old_binding TO trial_binding").Error)

	require.NoError(t, owner.Exec("ALTER DEFAULT PRIVILEGES GRANT SELECT ON TABLES TO PUBLIC").Error)
	require.Error(t, EnableSHEINUSTrialLedger(ctx, manifest, "product_acquisition"))
	assertAbsent()
	require.NoError(t, owner.Exec("ALTER DEFAULT PRIVILEGES REVOKE SELECT ON TABLES FROM PUBLIC").Error)

	require.NoError(t, EnableSHEINUSTrialLedger(ctx, manifest, "product_acquisition"))
	require.Error(t, EnableSHEINUSTrialLedger(ctx, manifest, "product_acquisition"), "existing ledger cannot be repaired or overwritten")
	require.NoError(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data), "default closed startup admits the exact installed ledger grants")
	const table = "public.data_acquisition_shein_us_trials"
	require.NoError(t, data.Exec("INSERT INTO "+table+" (approval_id,binding,claimed_at) VALUES ('10000000-0000-4000-8000-000000000001',repeat('a',64),now())").Error)
	require.NoError(t, data.Exec("UPDATE "+table+" SET request_id='fixture-request',credits=5").Error)
	require.Error(t, data.Exec("UPDATE "+table+" SET binding=repeat('b',64)").Error)
	require.Error(t, data.Exec("INSERT INTO "+table+" (credits) VALUES (5)").Error)
	require.Error(t, data.Exec("DELETE FROM "+table).Error)

	for _, drift := range []struct{ add, remove string }{
		{"GRANT UPDATE (binding) ON " + table + " TO data_services_runtime", "REVOKE UPDATE (binding) ON " + table + " FROM data_services_runtime"},
		{"GRANT SELECT ON " + table + " TO source_acquisition_runtime", "REVOKE SELECT ON " + table + " FROM source_acquisition_runtime"},
		{"GRANT SELECT ON " + table + " TO data_services_runtime WITH GRANT OPTION", "REVOKE GRANT OPTION FOR SELECT ON " + table + " FROM data_services_runtime"},
		{"REVOKE UPDATE (credits) ON " + table + " FROM data_services_runtime", "GRANT UPDATE (credits) ON " + table + " TO data_services_runtime"},
	} {
		require.NoError(t, owner.Exec(drift.add).Error)
		require.Error(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data))
		require.NoError(t, owner.Exec(drift.remove).Error)
	}
	require.NoError(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data))
	require.NoError(t, owner.Exec("REVOKE SELECT,INSERT (approval_id,binding,claimed_at),UPDATE (request_id,credits) ON "+table+" FROM data_services_runtime").Error)
	require.NoError(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data), "default closed startup accepts the retained ledger with no runtime access")
	require.Error(t, data.Exec("SELECT request_id FROM "+table+" LIMIT 0").Error)
	require.NoError(t, owner.Exec("GRANT SELECT ON "+table+" TO source_acquisition_runtime").Error)
	require.Error(t, acquisitionstore.VerifyDataServicesRuntimePermissions(ctx, data), "an extra grantee is rejected even without runtime grants")
	require.NoError(t, owner.Exec("REVOKE SELECT ON "+table+" FROM source_acquisition_runtime").Error)
	var retained int
	require.NoError(t, owner.Raw("SELECT credits FROM "+table+" WHERE request_id='fixture-request'").Scan(&retained).Error)
	require.Equal(t, 5, retained, "removing privileges must not delete the trial record")
}
