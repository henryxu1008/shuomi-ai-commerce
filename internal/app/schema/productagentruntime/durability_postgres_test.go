package productagentruntime

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/integration/openai"
)

func TestPrepareRejectsUnloggedFactsWithoutRepair(t *testing.T) {
	for _, test := range []struct{ target, table string }{
		{"run", "ai_invocations"},
		{"run", "ai_client_credentials"},
		{"review", "product_snapshot_heads"},
	} {
		t.Run(test.table, func(t *testing.T) {
			f := newPrepareFixture(t, test.target)
			prepareTarget := PrepareRun
			if test.target == "review" {
				prepareTarget = PrepareReview
			}
			_, err := prepareTarget(context.Background(), f.db)
			require.NoError(t, err)
			switch test.table {
			case "ai_invocations":
				_, err = aistore.NewGormInvocationRecorder(f.db).ClaimInvocation(context.Background(), aicapability.InvocationRecord{InvocationID: "retained", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", Operation: aicapability.OperationProductContentDecision, StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched})
			case "ai_client_credentials":
				err = openai.NewGormCredentialResolver(f.db).SaveCredential(context.Background(), openai.AIClientCredential{TenantID: "org", ClientName: "content", APIKey: "synthetic-only", BaseURL: "https://example.test/v1", Model: "synthetic", Enabled: true})
			case "product_snapshot_heads":
				err = f.db.Exec("INSERT INTO public.product_snapshot_heads VALUES('org','product',1,'publication')").Error
			}
			require.NoError(t, err)
			require.NoError(t, f.db.Exec("ALTER TABLE public."+quote(test.table)+" SET UNLOGGED").Error)
			tables := []string{"public." + test.table, "public.preserved_fact"}
			before := retainedSnapshot(t, f.db, tables)
			_, err = prepareTarget(context.Background(), f.db)
			require.Error(t, err, "permanent facts are required even when all other metadata/grants match")
			require.Equal(t, before, retainedSnapshot(t, f.db, tables), "refusal must not repair persistence or change facts/schema/ACL")
		})
	}
}
