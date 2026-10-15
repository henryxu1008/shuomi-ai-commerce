package productagentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareRejectsOrdinaryTableRewriteRulesWithoutRepair(t *testing.T) {
	for _, test := range []struct{ target, table string }{{"run", "product_agent_runs"}, {"review", "product_title_proposals"}} {
		t.Run(test.table, func(t *testing.T) {
			f := newPrepareFixture(t, test.target)
			prepareTarget := PrepareRun
			if test.target == "review" {
				prepareTarget = PrepareReview
			}
			_, err := prepareTarget(context.Background(), f.db)
			require.NoError(t, err)
			require.NoError(t, f.db.Exec("CREATE RULE suppress_insert AS ON INSERT TO public."+quote(test.table)+" DO INSTEAD NOTHING").Error)
			tables := []string{"public." + test.table, "public.preserved_fact"}
			before := retainedSnapshot(t, f.db, tables)
			rules := func() string {
				var definition string
				require.NoError(t, f.db.Raw("SELECT pg_catalog.pg_get_ruledef(oid) FROM pg_catalog.pg_rewrite WHERE ev_class=pg_catalog.to_regclass(?) AND rulename='suppress_insert'", "public."+test.table).Scan(&definition).Error)
				return definition
			}
			beforeRule := rules()
			require.NotEmpty(t, beforeRule)
			_, err = prepareTarget(context.Background(), f.db)
			require.Error(t, err, "write-suppression rules are not the original ordinary table definition")
			require.Equal(t, before, retainedSnapshot(t, f.db, tables))
			require.Equal(t, beforeRule, rules())
		})
	}
}

func TestPrepareRejectsInheritedRunWriteBoundaryWithoutRepair(t *testing.T) {
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.NoError(t, f.db.Exec("CREATE TABLE public.inherited_run () INHERITS (public.product_agent_runs)").Error)
	for _, table := range []string{"product_agent_runs", "inherited_run"} {
		require.NoError(t, f.db.Exec("INSERT INTO public."+quote(table)+" VALUES('same-run','org','actor','acquisition','context','request',?,1,'running',?)", strings.Repeat("a", 64), []byte(`{}`)).Error)
	}
	var count int64
	require.NoError(t, f.role(t, "product_agent_runtime").Raw("SELECT count(*) FROM public.product_agent_runs WHERE run_id='same-run'").Scan(&count).Error)
	require.EqualValues(t, 2, count, "ordinary owner reads include child rows outside the canonical primary key")
	tables := []string{"public.product_agent_runs", "public.inherited_run", "public.preserved_fact"}
	before := retainedSnapshot(t, f.db, tables)
	_, err = PrepareRun(context.Background(), f.db)
	require.Error(t, err, "the canonical Run table cannot include a second writable fact relation")
	require.Equal(t, before, retainedSnapshot(t, f.db, tables))
	var inheritance int64
	require.NoError(t, f.db.Raw("SELECT count(*) FROM pg_catalog.pg_inherits WHERE inhparent='public.product_agent_runs'::regclass AND inhrelid='public.inherited_run'::regclass").Scan(&inheritance).Error)
	require.EqualValues(t, 1, inheritance)
}
