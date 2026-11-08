package productagentruntime

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	aistore "task-processor/internal/aicapability/store"
)

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
	var before string
	require.NoError(t, f.db.Raw("SELECT to_jsonb(v)::text FROM ai_invocations v WHERE invocation_id='kept'").Scan(&before).Error)
	result, err := PrepareProviderRejectionProof(ctx, f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	var after string
	require.NoError(t, f.db.Raw("SELECT (to_jsonb(v)-'provider_rejection_proof_json')::text FROM ai_invocations v WHERE invocation_id='kept'").Scan(&after).Error)
	require.Equal(t, before, after)
	runtime := f.role(t, "product_agent_runtime")
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
}
