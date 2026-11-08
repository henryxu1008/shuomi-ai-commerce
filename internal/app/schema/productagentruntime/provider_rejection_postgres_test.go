package productagentruntime

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	aistore "task-processor/internal/aicapability/store"
	routeconfigstore "task-processor/internal/integration/persistence/routeconfig"
)

func installOriginalRouteReader(t *testing.T, f prepareFixture) {
	t.Helper()
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		if err := tx.Exec(routeconfigstore.SchemaSQL).Error; err != nil {
			return err
		}
		return tx.Exec(routeconfigstore.GrantSQL("product_agent_runtime", false)).Error
	}))
}

func installOriginalInvocationAuditReader(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='account_audit_product_reader') THEN CREATE ROLE account_audit_product_reader NOLOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$").Error)
	require.NoError(t, db.Exec("GRANT SELECT ON public.ai_invocations TO account_audit_product_reader").Error)
}

func TestPrepareProviderRejectionProofAddsOnlyEmptyColumnAndPreciseGrant(t *testing.T) {
	f := newPrepareFixture(t, "run")
	ctx := context.Background()
	_, err := PrepareRun(ctx, f.db)
	require.NoError(t, err)
	require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
			return err
		}
		// Construct the current 52-column baseline in this empty fixture. Dropping
		// a column from a 53-column table would leave an artificial attnum gap.
		model := reflect.TypeOf(aistore.InvocationSchemaModel()).Elem()
		var fields []reflect.StructField
		for n := 0; n < model.NumField(); n++ {
			field := model.Field(n)
			if field.Name != "ProviderRejectionProofJSON" {
				fields = append(fields, field)
			}
		}
		if err := tx.Migrator().DropTable("ai_invocations"); err != nil {
			return err
		}
		if err := tx.Table("ai_invocations").AutoMigrate(reflect.New(reflect.StructOf(fields)).Interface()); err != nil {
			return err
		}
		base, err := modelContract(tx, reflect.New(reflect.StructOf(fields)).Interface(), "product_agent_owner")
		if err != nil {
			return err
		}
		base.Name = "ai_invocations"
		return applyGrants(tx, runGrants(base)[:1])
	}))
	require.NoError(t, f.db.Exec("INSERT INTO ai_invocations(invocation_id,tenant_id,member_id,input_hash,operation,outcome,review_score,review_needs_human,review_reasons) VALUES('kept','org','member','input','product.own.intake.generate','dispatched',0,false,'null')").Error)
	installOriginalRouteReader(t, f)
	installOriginalInvocationAuditReader(t, f.db)
	routeTables := []string{"public.ai_route_config_revisions", "public.ai_route_config_current", "public.ai_route_config_options"}
	routeBefore := retainedSnapshot(t, f.db, routeTables)
	var before string
	require.NoError(t, f.db.Raw("SELECT to_jsonb(v)::text FROM ai_invocations v WHERE invocation_id='kept'").Scan(&before).Error)
	// The same owner command can qualify the actual 52-column baseline in a
	// database-enforced read-only rehearsal without applying its final DDL.
	err = f.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET TRANSACTION READ ONLY").Error; err != nil {
			return err
		}
		_, err := PrepareProviderRejectionProof(ctx, tx)
		return err
	})
	require.ErrorContains(t, err, "cannot execute ALTER TABLE in a read-only transaction")
	result, err := PrepareProviderRejectionProof(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	var after string
	require.NoError(t, f.db.Raw("SELECT (to_jsonb(v)-'provider_rejection_proof_json')::text FROM ai_invocations v WHERE invocation_id='kept'").Scan(&after).Error)
	require.Equal(t, before, after)
	runtime := f.role(t, "product_agent_runtime")
	require.NoError(t, VerifyInvocationRecoveryRuntime(ctx, runtime))
	require.NoError(t, runtime.Exec("UPDATE ai_invocations SET provider_rejection_proof_json=NULL WHERE invocation_id='kept'").Error)
	require.Error(t, runtime.Exec("DELETE FROM ai_invocations WHERE invocation_id='kept'").Error)
	require.NoError(t, VerifyInvocationRelation(ctx, f.db, "product_agent_owner"))
	result, err = PrepareProviderRejectionProof(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "ALREADY_READY", result.Outcome)
	// PostgreSQL keeps a physical attnum gap after the approved empty-column
	// rollback. Reinstall must qualify the same active columns without row edits.
	require.NoError(t, f.db.Exec("ALTER TABLE ai_invocations DROP COLUMN provider_rejection_proof_json").Error)
	result, err = PrepareProviderRejectionProof(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	require.Equal(t, routeBefore, retainedSnapshot(t, f.db, routeTables), "capability preparation must preserve the original route reader facts and ACL")
}

func TestInvocationRecoverySharedLedgerRetainsAuditAndRejectsRuntimeExcess(t *testing.T) {
	for _, privilege := range []string{"DELETE", "UPDATE", "SELECT WITH GRANT OPTION"} {
		t.Run(privilege, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			_, err := PrepareRun(context.Background(), f.db)
			require.NoError(t, err)
			installOriginalInvocationAuditReader(t, f.db)
			runtime := f.role(t, "product_agent_runtime")
			require.NoError(t, VerifyInvocationRecoveryRuntime(context.Background(), runtime))
			statement := "GRANT " + privilege + " ON ai_invocations TO product_agent_runtime"
			if privilege == "SELECT WITH GRANT OPTION" {
				statement = "GRANT SELECT ON ai_invocations TO product_agent_runtime WITH GRANT OPTION"
			}
			require.NoError(t, f.db.Exec(statement).Error)
			before := retainedSnapshot(t, f.db, []string{"public.ai_invocations"})
			_, err = PrepareProviderRejectionProof(context.Background(), f.db)
			require.Error(t, err)
			require.Error(t, VerifyInvocationRecoveryRuntime(context.Background(), runtime))
			require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.ai_invocations"}))
		})
	}
}

func TestProviderRejectionRouteReaderRejectsWritesAndGrantOptions(t *testing.T) {
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	installOriginalRouteReader(t, f)
	runtime := f.role(t, "product_agent_runtime")
	for _, test := range []struct{ table, privilege string }{
		{"ai_route_config_revisions", "INSERT"},
		{"ai_route_config_current", "UPDATE"},
		{"ai_route_config_options", "DELETE"},
		{"ai_route_config_options", "SELECT WITH GRANT OPTION"},
	} {
		t.Run(test.table+"/"+test.privilege, func(t *testing.T) {
			require.NoError(t, VerifyInvocationRecoveryRuntime(context.Background(), runtime))
			statement := "GRANT " + test.privilege + " ON public." + test.table + " TO product_agent_runtime"
			if test.privilege == "SELECT WITH GRANT OPTION" {
				statement = "GRANT SELECT ON public." + test.table + " TO product_agent_runtime WITH GRANT OPTION"
			}
			require.NoError(t, f.db.Exec(statement).Error)
			before := retainedSnapshot(t, f.db, []string{"public." + test.table, "public.ai_invocations"})
			_, err := PrepareProviderRejectionProof(context.Background(), f.db)
			require.Error(t, err)
			require.Error(t, VerifyInvocationRecoveryRuntime(context.Background(), runtime))
			require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public." + test.table, "public.ai_invocations"}))
			revoke := "REVOKE " + test.privilege + " ON public." + test.table + " FROM product_agent_runtime"
			if test.privilege == "SELECT WITH GRANT OPTION" {
				revoke = "REVOKE GRANT OPTION FOR SELECT ON public." + test.table + " FROM product_agent_runtime"
			}
			require.NoError(t, f.db.Exec(revoke).Error)
		})
	}
}
