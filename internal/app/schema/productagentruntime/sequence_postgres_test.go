package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/openai"
)

func TestPrepareRunQualifiesCredentialSequenceWithoutAdvancingOrRepair(t *testing.T) {
	for _, kind := range []string{"reset", "increment", "cache", "cycle", "exhausted", "valid-gap"} {
		t.Run(kind, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			_, err := PrepareRun(context.Background(), f.db)
			require.NoError(t, err)
			store := openai.NewGormCredentialResolver(f.db)
			for _, client := range []string{"first", "second"} {
				require.NoError(t, store.SaveCredential(context.Background(), openai.AIClientCredential{TenantID: "org", ClientName: client, APIKey: "synthetic-only", BaseURL: "https://example.test/v1", Model: "synthetic", Enabled: true}))
			}
			sql := map[string]string{
				"reset":     "SELECT setval('public.ai_client_credentials_id_seq',1,false)",
				"increment": "ALTER SEQUENCE public.ai_client_credentials_id_seq INCREMENT BY 2",
				"cache":     "ALTER SEQUENCE public.ai_client_credentials_id_seq CACHE 2",
				"cycle":     "ALTER SEQUENCE public.ai_client_credentials_id_seq CYCLE",
				"exhausted": "SELECT setval('public.ai_client_credentials_id_seq',9223372036854775807,true)",
				"valid-gap": "SELECT setval('public.ai_client_credentials_id_seq',200,true)",
			}[kind]
			require.NoError(t, f.db.Exec(sql).Error) // dedicated synthetic fixture setup only.
			snapshot := func() string {
				var value string
				require.NoError(t, f.db.Raw(`SELECT jsonb_build_object('value',s.last_value,'called',s.is_called,'acl',c.relacl,'parameters',to_jsonb(p))::text FROM public.ai_client_credentials_id_seq s,pg_catalog.pg_class c JOIN pg_catalog.pg_sequence p ON p.seqrelid=c.oid WHERE c.oid='public.ai_client_credentials_id_seq'::regclass`).Scan(&value).Error)
				return value
			}
			beforeSequence := snapshot()
			before := retainedSnapshot(t, f.db, []string{"public.ai_client_credentials", "public.product_agent_runs", "public.product_agent_tool_calls", "public.ai_invocations"})
			result, err := PrepareRun(context.Background(), f.db)
			if kind == "valid-gap" {
				require.NoError(t, err)
				require.Equal(t, "ALREADY_READY", result.Outcome)
			} else {
				require.Error(t, err, "drift/exhaustion must not be qualified as a usable sequence")
			}
			require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.ai_client_credentials", "public.product_agent_runs", "public.product_agent_tool_calls", "public.ai_invocations"}))
			require.Equal(t, beforeSequence, snapshot(), "qualification does not nextval, setval or repair")
		})
	}
}
