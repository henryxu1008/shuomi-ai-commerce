//go:build integration

package productsourcing

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	jobstore "task-processor/internal/integration/persistence/product/dataacquisition"
	"task-processor/internal/ledger/orgresource"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
)

func privateAlignmentFile(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(path, 0600))
		return
	}
	// Only constructs an isolated test fixture. Production reuses VerifyPrivateFiles.
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", `$p=[Console]::In.ReadToEnd();$acl=New-Object System.Security.AccessControl.FileSecurity;$sid=[System.Security.Principal.WindowsIdentity]::GetCurrent().User;$rule=New-Object System.Security.AccessControl.FileSystemAccessRule($sid,'FullControl','Allow');$acl.AddAccessRule($rule);$acl.SetAccessRuleProtection($true,$false);[System.IO.File]::SetAccessControl($p,$acl)`)
	cmd.Stdin = strings.NewReader(path)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
}

func TestPriceAlignmentNativeOwnerPreservesAndReenters(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("product_acquisition"), tcpostgres.WithUsername("fixture_admin"), tcpostgres.WithPassword("fixture-only"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	defer func() { p, _ := admin.DB(); p.Close() }()
	require.NoError(t, admin.Exec(`CREATE ROLE acquisition_owner LOGIN PASSWORD 'owner-fixture'; CREATE ROLE source_acquisition_runtime LOGIN PASSWORD 'runtime-fixture'; CREATE ROLE data_services_runtime LOGIN PASSWORD 'runtime-fixture'; CREATE ROLE product_research_runtime LOGIN PASSWORD 'runtime-fixture'; ALTER DATABASE product_acquisition OWNER TO acquisition_owner`).Error)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword("acquisition_owner", "owner-fixture")
	owner, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	p, err := owner.DB()
	require.NoError(t, err)
	p.SetMaxOpenConns(4)
	defer p.Close()
	port, err := c.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	manifest := acquisitionInitManifest{SchemaVersion: 1, Collections: true, DataServices: true}
	manifest.Database.Host, manifest.Database.Port, manifest.Database.User, manifest.Database.Password, manifest.Database.Database = "127.0.0.1", int(port.Num()), "acquisition_owner", "owner-fixture", "product_acquisition"
	path := filepath.Join(t.TempDir(), "owner.json")
	raw, err := json.Marshal(manifest)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, raw, 0600))
	privateAlignmentFile(t, path)
	require.NoError(t, InitializeAcquisitionDatabase(ctx, path, "product_acquisition"))
	require.NoError(t, owner.Exec(`ALTER TABLE data_acquisition_jobs DROP COLUMN price_snapshot CASCADE`).Error)
	current, err := jobstore.PriceContractState(ctx, owner)
	require.NoError(t, err)
	require.False(t, current)
	// This existing wrapper must still refuse A. Only the explicit new entry
	// separates complete effective permissions from B-only structure admission.
	require.Error(t, acquisitionstore.VerifyDataServicesOwnerPermissions(ctx, owner, false))
	scope := collection.Scope{OrganizationID: "original-org", ActorID: "original-actor", MemberID: "original-member"}
	jobID, itemID, command := uuid.NewString(), uuid.NewString(), uuid.NewString()
	q, err := dataacquisition.NormalizeQuery(dataacquisition.Query{Site: "us", Mode: "asin", ASINs: []string{"B000123456"}, Limit: 1})
	require.NoError(t, err)
	qraw, err := json.Marshal(q)
	require.NoError(t, err)
	intent := orgresource.ConsumerChargeIntent{Identity: orgresource.ConsumerChargeIdentity{OrganizationID: scope.OrganizationID, Consumer: orgresource.ConsumerAmazonData, OperationID: itemID}, ActorID: scope.ActorID, MemberID: scope.MemberID, Funding: orgresource.FundingEnterprise, ResourceType: orgresource.ResourceDataRow, Quantity: 1, Fingerprint: collection.Digest("original-before-price"), BusinessScope: "amazon-data:" + jobID + ":" + itemID}
	iraw, err := json.Marshal(intent)
	require.NoError(t, err)
	require.NoError(t, owner.Exec(`INSERT INTO data_acquisition_jobs(organization_id,actor_id,member_id,id,command_key,input_hash,query_json,funding,state,discovered,quota_reserved,day_window,month_window,created_at,deadline) VALUES(?,?,?,?,?,?,?,'enterprise_unallocated','FAILED',true,0,CURRENT_DATE,CURRENT_DATE,now(),now()+interval '30 minutes')`, scope.OrganizationID, scope.ActorID, scope.MemberID, jobID, command, collection.Digest("original-hash"), string(qraw)).Error)
	require.NoError(t, owner.Exec(`INSERT INTO data_acquisition_items(organization_id,actor_id,job_id,id,asin,state,intent_json,terminal_evidence,reason,created_at) VALUES(?,?,?,?,?,'FAILED',?,?,'resource_unavailable',now())`, scope.OrganizationID, scope.ActorID, jobID, itemID, q.ASINs[0], string(iraw), uuid.NewString()).Error)
	before, err := pricePreservation(owner)
	require.NoError(t, err)
	t.Run("DDL failure after column add rolls everything back", func(t *testing.T) {
		require.NoError(t, admin.Exec(`CREATE FUNCTION public.reject_price_add() RETURNS event_trigger LANGUAGE plpgsql AS $$ BEGIN IF EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid='public.data_acquisition_jobs'::regclass AND attname='price_snapshot' AND NOT attisdropped) AND NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conrelid='public.data_acquisition_jobs'::regclass AND conname='data_acquisition_price_snapshot') THEN RAISE EXCEPTION 'injected_after_price_column'; END IF; END $$; CREATE EVENT TRIGGER reject_price_add ON ddl_command_end WHEN TAG IN ('ALTER TABLE') EXECUTE FUNCTION public.reject_price_add()`).Error)
		err := AlignAcquisitionPriceSnapshot(ctx, path, "product_acquisition")
		require.ErrorContains(t, err, "injected_after_price_column")
		require.NoError(t, admin.Exec(`DROP EVENT TRIGGER reject_price_add; DROP FUNCTION public.reject_price_add()`).Error)
		current, err := jobstore.PriceContractState(ctx, owner)
		require.NoError(t, err)
		require.False(t, current)
		got, err := pricePreservation(owner)
		require.NoError(t, err)
		require.Equal(t, before, got)
	})
	errs := make(chan error, 6)
	var wg sync.WaitGroup
	for range 6 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- AlignAcquisitionPriceSnapshot(ctx, path, "product_acquisition") }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	after, err := pricePreservation(owner)
	require.NoError(t, err)
	require.Equal(t, before, after)
	var oid uint32
	require.NoError(t, owner.Raw(`SELECT oid FROM pg_constraint WHERE conname='data_acquisition_price_snapshot' AND conrelid='data_acquisition_jobs'::regclass`).Scan(&oid).Error)
	require.NotZero(t, oid)
	require.NoError(t, AlignAcquisitionPriceSnapshot(ctx, path, "product_acquisition"))
	var replay uint32
	require.NoError(t, owner.Raw(`SELECT oid FROM pg_constraint WHERE conname='data_acquisition_price_snapshot' AND conrelid='data_acquisition_jobs'::regclass`).Scan(&replay).Error)
	require.Equal(t, oid, replay)
	require.NoError(t, acquisitionstore.VerifyDataServicesOwnerPermissions(ctx, owner, false))
	require.NoError(t, jobstore.VerifySchema(ctx, owner))
	t.Run("committed writer wins before maintenance", func(t *testing.T) {
		writer := owner.Begin()
		require.NoError(t, writer.Error)
		require.NoError(t, writer.Exec(`UPDATE data_acquisition_items SET reservation_id='cd1ae944-ffbd-4dc8-a408-75ce77a8b080'`).Error)
		done := make(chan error, 1)
		go func() { done <- AlignAcquisitionPriceSnapshot(ctx, path, "product_acquisition") }()
		require.NoError(t, writer.Commit().Error)
		require.Error(t, <-done)
		require.NoError(t, owner.Exec(`UPDATE data_acquisition_items SET reservation_id=NULL`).Error)
	})
	t.Run("cancellation while native table lock held", func(t *testing.T) {
		writer := owner.Begin()
		require.NoError(t, writer.Error)
		defer writer.Rollback()
		require.NoError(t, writer.Exec(`LOCK TABLE data_acquisition_jobs,data_acquisition_items IN ACCESS EXCLUSIVE MODE`).Error)
		short, stop := context.WithTimeout(ctx, 500*time.Millisecond)
		defer stop()
		// Direct pinned owner helper avoids spending the short deadline in ACL-file parsing.
		require.Error(t, alignAcquisitionPriceDB(short, owner, "product_acquisition", manifest))
		require.NoError(t, writer.Rollback().Error)
		got, err := pricePreservation(owner)
		require.NoError(t, err)
		require.Equal(t, before, got)
	})
	for name, drift := range map[string]string{
		"database grant option": `GRANT CONNECT ON DATABASE product_acquisition TO data_services_runtime WITH GRANT OPTION`,
		"runtime grant option":  `GRANT SELECT ON data_acquisition_jobs TO data_services_runtime WITH GRANT OPTION`,
		"schema grant option":   `GRANT USAGE ON SCHEMA public TO data_services_runtime WITH GRANT OPTION`,
		"unknown reader":        `CREATE ROLE extra_price_reader; GRANT SELECT ON data_acquisition_jobs TO extra_price_reader`,
		"weak price constraint": `ALTER TABLE data_acquisition_jobs DROP CONSTRAINT data_acquisition_price_snapshot; ALTER TABLE data_acquisition_jobs ADD CONSTRAINT data_acquisition_price_snapshot CHECK(true)`,
		"writer first":          `UPDATE data_acquisition_items SET reservation_id='cd1ae944-ffbd-4dc8-a408-75ce77a8b080'`,
	} {
		t.Run(name, func(t *testing.T) {
			// Savepoint rollback preserves the single fixture and proves no repair of drift.
			tx := owner.Begin()
			require.NoError(t, tx.Error)
			defer tx.Rollback()
			if name == "unknown reader" {
				require.NoError(t, admin.Exec(`CREATE ROLE extra_price_reader`).Error)
				drift = `GRANT SELECT ON data_acquisition_jobs TO extra_price_reader`
			}
			require.NoError(t, tx.Exec(drift).Error)
			err := verifyPriceAlignmentBoundary(ctx, tx, "product_acquisition", manifest)
			if err == nil {
				err = jobstore.AlignPriceContractTx(ctx, tx)
			}
			require.Error(t, err)
		})
	}
}

func TestPriceAlignmentRejectsBroadPrivateFileBeforeDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broad-owner.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"schemaVersion":1}`), 0644))
	if runtime.GOOS == "windows" {
		privateAlignmentFile(t, path)
		out, err := exec.Command("icacls.exe", path, "/grant", "*S-1-1-0:(R)").CombinedOutput()
		require.NoError(t, err, "%s", out)
	}
	err := AlignAcquisitionPriceSnapshot(context.Background(), path, "product_acquisition")
	require.ErrorContains(t, err, "private file permissions invalid")
}
