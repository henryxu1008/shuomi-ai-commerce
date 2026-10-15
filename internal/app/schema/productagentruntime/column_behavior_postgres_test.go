package productagentruntime

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/openai"
)

func TestPrepareRejectsIdentityRunColumnWithoutRepair(t *testing.T) {
	f := newPrepareFixture(t, "run")
	_, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.NoError(t, f.db.Exec("ALTER TABLE public.product_agent_runs ALTER COLUMN revision ADD GENERATED ALWAYS AS IDENTITY").Error)
	require.Error(t, f.role(t, "product_agent_runtime").Exec("INSERT INTO public.product_agent_runs VALUES('run','org','actor','acquisition','context','request',?,1,'running',?)", strings.Repeat("a", 64), []byte(`{}`)).Error, "the original explicit revision insert is unusable")
	tables := []string{"public.product_agent_runs", "public.preserved_fact"}
	before := retainedSnapshot(t, f.db, tables)
	_, err = PrepareRun(context.Background(), f.db)
	require.Error(t, err, "identity is not the original ordinary column definition")
	require.Equal(t, before, retainedSnapshot(t, f.db, tables))
	var identity string
	require.NoError(t, f.db.Raw("SELECT attidentity::text FROM pg_catalog.pg_attribute WHERE attrelid='public.product_agent_runs'::regclass AND attname='revision'").Scan(&identity).Error)
	require.Equal(t, "a", identity)
}

func TestRelationRejectsGeneratedColumnMatchingOriginalDefaultWithoutRepair(t *testing.T) {
	f := newPrepareFixture(t, "run")
	credential, err := modelContract(f.db, &openai.AIClientCredential{}, "product_agent_owner")
	require.NoError(t, err)
	var enabled columnContract
	for _, column := range credential.Columns {
		if column.Name == "enabled" {
			enabled = column
		}
	}
	require.Equal(t, "true", enabled.Default)
	// PostgreSQL exposes the same expression for a default and a generated
	// constant. The original credential writer explicitly supplies enabled.
	require.NoError(t, f.db.Exec("CREATE TABLE public.generated_probe(enabled boolean GENERATED ALWAYS AS (true) STORED NOT NULL); ALTER TABLE public.generated_probe OWNER TO product_agent_owner").Error)
	require.Error(t, f.db.Exec("INSERT INTO public.generated_probe(enabled) VALUES(true)").Error)
	before := retainedSnapshot(t, f.db, []string{"public.generated_probe", "public.preserved_fact"})
	err = verifyRelation(f.db, relationContract{Name: "generated_probe", Owner: credential.Owner, Columns: []columnContract{enabled}})
	require.Error(t, err, "a generated expression matching a default must not qualify as an ordinary writable column")
	require.Equal(t, before, retainedSnapshot(t, f.db, []string{"public.generated_probe", "public.preserved_fact"}))
	var generated string
	require.NoError(t, f.db.Raw("SELECT attgenerated::text FROM pg_catalog.pg_attribute WHERE attrelid='public.generated_probe'::regclass AND attname='enabled'").Scan(&generated).Error)
	require.Equal(t, "s", generated)
}
