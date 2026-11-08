package orgresourceadapter

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"task-processor/internal/aicapability"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
	routeconfigstore "task-processor/internal/integration/persistence/routeconfig"
)

func prepareRecoveryCommandRoles(t *testing.T, resource, native *gorm.DB) (map[string]any, map[string]any) {
	t.Helper()
	for _, role := range []string{"product_agent_owner", "product_agent_runtime", "commercial_owner_runtime"} {
		require.NoError(t, native.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='"+role+"') THEN CREATE ROLE "+role+" LOGIN PASSWORD 'synthetic-native-role' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$").Error)
	}
	var connections []map[string]any
	for n, db := range []*gorm.DB{native, resource} {
		cfg, err := pgx.ParseConfig(db.Dialector.(*postgres.Dialector).DSN)
		require.NoError(t, err)
		role := "product_agent_runtime"
		if n == 1 {
			role = "commercial_owner_runtime"
		}
		require.NoError(t, db.Exec("REVOKE ALL ON DATABASE "+pgx.Identifier{cfg.Database}.Sanitize()+" FROM PUBLIC").Error)
		require.NoError(t, db.Exec("GRANT CONNECT ON DATABASE "+pgx.Identifier{cfg.Database}.Sanitize()+" TO "+role).Error)
		require.NoError(t, db.Exec("REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)
		require.NoError(t, db.Exec("GRANT USAGE ON SCHEMA public TO "+role).Error)
		connections = append(connections, map[string]any{"host": "127.0.0.1", "port": cfg.Port, "user": role, "password": "synthetic-native-role", "database": cfg.Database})
	}
	require.NoError(t, native.Exec("ALTER TABLE ai_invocations OWNER TO product_agent_owner").Error)
	require.NoError(t, native.Exec("GRANT SELECT,INSERT ON ai_invocations TO product_agent_runtime").Error)
	var columns []string
	require.NoError(t, native.Raw("SELECT attname FROM pg_attribute WHERE attrelid='public.ai_invocations'::regclass AND attnum>0 AND NOT attisdropped ORDER BY attnum").Scan(&columns).Error)
	var quoted []string
	for _, column := range columns {
		quoted = append(quoted, pgx.Identifier{column}.Sanitize())
	}
	require.NoError(t, native.Exec("GRANT UPDATE("+strings.Join(quoted, ",")+") ON ai_invocations TO product_agent_runtime").Error)
	require.NoError(t, native.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='account_audit_product_reader') THEN CREATE ROLE account_audit_product_reader NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$; GRANT SELECT ON ai_invocations TO account_audit_product_reader").Error)
	require.NoError(t, native.Exec("GRANT USAGE,CREATE ON SCHEMA public TO product_agent_owner").Error)
	require.NoError(t, native.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		if err := channelstore.EnsureSchema(context.Background(), tx); err != nil {
			return err
		}
		if err := tx.Exec(routeconfigstore.SchemaSQL).Error; err != nil {
			return err
		}
		return tx.Exec(routeconfigstore.GrantSQL("product_agent_runtime", false)).Error
	}))
	for _, table := range []string{"saas_organization_resource_buckets", "saas_organization_resource_operations", "saas_organization_resource_reservations", "saas_organization_resource_debts", "saas_member_ai_point_limits", "saas_member_ai_point_months", "saas_member_resource_positions"} {
		require.NoError(t, resource.Exec("GRANT SELECT,INSERT,UPDATE ON "+table+" TO commercial_owner_runtime").Error)
	}
	for _, table := range []string{"saas_organization_resource_source_claims", "saas_organization_resource_events", "saas_organization_resource_audit_logs"} {
		require.NoError(t, resource.Exec("GRANT SELECT,INSERT ON "+table+" TO commercial_owner_runtime").Error)
	}
	require.NoError(t, resource.Exec("GRANT USAGE,SELECT ON SEQUENCE saas_organization_resource_audit_logs_id_seq TO commercial_owner_runtime").Error)
	return connections[0], connections[1]
}

func TestProviderRejectionCommandInspectAndResolve(t *testing.T) {
	binary := os.Getenv("OWN_INTAKE_RECOVERY_BINARY")
	if binary == "" {
		t.Skip("requires this candidate's built native recovery command")
	}
	resource, native := rejectionPostgresDatabases(t)
	recorder, _, fact := rejectedInvocationFixture(t, resource, native)
	invocationDB, resourceDB := prepareRecoveryCommandRoles(t, resource, native)
	p := fact.ProviderRejectionProof
	cfg := map[string]any{
		"organizationId": fact.TenantID, "memberId": fact.MemberID, "invocationId": fact.InvocationID, "reservationId": p.ReservationID,
		"expectedDispatchFingerprint": p.DispatchFingerprint, "expectedProofDigest": "", "operatorId": p.OperatorID, "approvalEvidenceId": p.ApprovalEvidenceID,
		"observedAt": p.ObservedAt, "evidence": p.Evidence, "resolveEnabled": false, "invocationDatabase": invocationDB, "resourceDatabase": resourceDB,
	}
	path := filepath.Join(t.TempDir(), "recovery.json")
	invoke := func(action string, wantSuccess bool) map[string]any {
		t.Helper()
		raw, err := json.Marshal(cfg)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(path, raw, 0600))
		if runtime.GOOS == "windows" {
			current, err := user.Current()
			require.NoError(t, err)
			output, err := exec.Command("icacls", path, "/inheritance:r", "/grant:r", "*"+current.Uid+":F", "*S-1-5-18:F", "*S-1-5-32-544:F").CombinedOutput()
			require.NoError(t, err, "%s", output)
		}
		output, err := exec.Command(binary, "-config", path, "-action", action).CombinedOutput()
		if !wantSuccess {
			require.Error(t, err)
			return nil
		}
		require.NoError(t, err, "%s", output)
		var result map[string]any
		require.NoError(t, json.Unmarshal(output, &result))
		return result
	}
	inspected := invoke("inspect", true)
	require.Equal(t, true, inspected["proofReady"])
	missingReceipt := p.Evidence
	missingReceipt.NoChargeReceiptHash = ""
	cfg["evidence"] = missingReceipt
	require.Equal(t, false, invoke("inspect", true)["proofReady"])
	cfg["resolveEnabled"] = true
	cfg["expectedProofDigest"] = inspected["proofDigest"]
	invoke("resolve", false)
	cfg["evidence"] = p.Evidence
	cfg["resolveEnabled"] = false
	invoke("resolve", false)
	var count int64
	require.NoError(t, resource.Model(&organizationResourceEventRow{}).Count(&count).Error)
	require.EqualValues(t, 1, count, "inspect and a disabled resolve must not change the ledger")
	cfg["resolveEnabled"] = true
	cfg["expectedProofDigest"] = inspected["proofDigest"]
	for range 3 {
		result := invoke("resolve", true)
		require.Equal(t, true, result["settled"])
		require.EqualValues(t, 0, result["settledMicropoints"])
		require.Equal(t, "failed", result["outcome"])
	}
	assertRejectedReservationSettledOnce(t, resource)
	stored, found, err := recorder.FindInvocation(context.Background(), "org", "member", fact.InvocationID, "input")
	require.NoError(t, err)
	require.True(t, found)
	require.False(t, stored.UsageKnown)
	require.Equal(t, aicapability.ProviderRejectedUnbilledCode, stored.ErrorCode)
	badEvidence := p.Evidence
	badEvidence.NoChargeReceiptHash = ""
	cfg["evidence"] = badEvidence
	invoke("resolve", false)
	assertRejectedReservationSettledOnce(t, resource)
}
