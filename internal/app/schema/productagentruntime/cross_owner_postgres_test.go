package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareRejectsCrossOwnerEffectiveGrantsBeforeChanges(t *testing.T) {
	for _, role := range []string{"product_agent_runtime", "title_credential_writer", "product_review_runtime"} {
		for _, kind := range []string{"table", "column", "sequence", "maintenance", "schema"} {
			t.Run(role+"/"+kind, func(t *testing.T) {
				target := "run"
				if role == "product_review_runtime" {
					target = "review"
				}
				f := newPrepareFixture(t, target)
				require.NoError(t, f.db.Exec("CREATE SEQUENCE public.unrelated_private_seq").Error)
				require.NoError(t, f.db.Exec("CREATE SCHEMA unrelated_private").Error)
				statement := map[string]string{
					"table":       "GRANT SELECT ON public.preserved_fact TO ",
					"column":      "GRANT SELECT(value) ON public.preserved_fact TO ",
					"sequence":    "GRANT USAGE ON SEQUENCE public.unrelated_private_seq TO ",
					"maintenance": "GRANT MAINTAIN ON public.preserved_fact TO ",
					"schema":      "GRANT USAGE ON SCHEMA unrelated_private TO ",
				}[kind]
				require.NoError(t, f.db.Exec(statement+quote(role)).Error)
				tables := []string{"public.preserved_fact", "public.ai_invocations"}
				if target == "review" {
					tables = []string{"public.preserved_fact", "public.product_title_proposals", "public.product_title_operations", "public.product_snapshot_versions", "public.product_snapshot_heads", "public.product_source_publications", "public.product_source_publication_receipts"}
				}
				before := retainedSnapshot(t, f.db, tables)
				sequenceSnapshot := func() string {
					var value string
					require.NoError(t, f.db.Raw(`SELECT jsonb_build_object('value',s.last_value,'called',s.is_called,'acl',c.relacl)::text FROM public.unrelated_private_seq s,pg_catalog.pg_class c WHERE c.oid='public.unrelated_private_seq'::regclass`).Scan(&value).Error)
					return value
				}
				beforeSequence := sequenceSnapshot()
				schemaSnapshot := func() string {
					var value string
					require.NoError(t, f.db.Raw("SELECT COALESCE(nspacl::text,'') FROM pg_catalog.pg_namespace WHERE nspname='unrelated_private'").Scan(&value).Error)
					return value
				}
				beforeSchema := schemaSnapshot()
				var err error
				if target == "run" {
					_, err = PrepareRun(context.Background(), f.db)
				} else {
					_, err = PrepareReview(context.Background(), f.db)
				}
				require.Error(t, err, "unrelated direct or column/sequence authority must be rejected before target grants")
				require.Equal(t, before, retainedSnapshot(t, f.db, tables))
				require.Equal(t, beforeSequence, sequenceSnapshot())
				require.Equal(t, beforeSchema, schemaSnapshot())
				if target == "run" {
					require.False(t, f.db.Migrator().HasTable("product_agent_runs"))
				}
			})
		}
	}
}
