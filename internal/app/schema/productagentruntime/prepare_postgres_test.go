package productagentruntime

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/aicapability"
	aistore "task-processor/internal/aicapability/store"
	"task-processor/internal/app/httpapi"
	"task-processor/internal/integration/openai"
	agentstore "task-processor/internal/integration/persistence/agent"
)

type prepareFixture struct {
	db        *gorm.DB
	dsn, name string
}

func newPrepareFixture(t *testing.T, target string) prepareFixture {
	t.Helper()
	dsn := os.Getenv("ISSUE688_RUNTIME_TEST_DSN")
	if dsn == "" {
		t.Skip("requires task-owned ISSUE688_RUNTIME_TEST_DSN")
	}
	require.Contains(t, dsn, "host=127.0.0.1")
	require.Contains(t, dsn, "user=issue398_owner")
	require.Contains(t, dsn, "dbname=issue688_tests")
	root, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	rootPool, err := root.DB()
	require.NoError(t, err)
	for _, role := range []string{"product_agent_owner", "acquisition_owner", "product_agent_runtime", "title_credential_writer", "product_review_runtime"} {
		login := "NOLOGIN"
		if strings.HasSuffix(role, "runtime") || role == "title_credential_writer" {
			login = "LOGIN PASSWORD 'synthetic-native-role'"
		}
		require.NoError(t, root.Exec("DO $$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='"+role+"') THEN CREATE ROLE "+role+" "+login+" NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; END IF; END $$").Error)
	}
	owner := "product_agent_owner"
	if target == "review" {
		owner = "acquisition_owner"
	}
	name := "issue688_native_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	quoted := pgx.Identifier{name}.Sanitize()
	require.NoError(t, root.Exec("CREATE DATABASE "+quoted+" OWNER "+owner).Error)
	require.NoError(t, root.Exec("REVOKE ALL ON DATABASE "+quoted+" FROM PUBLIC").Error)
	db, err := gorm.Open(postgres.Open(dsn+" dbname="+name), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pool.Close())
		require.NoError(t, root.Exec("DROP DATABASE "+quoted+" WITH (FORCE)").Error)
		require.NoError(t, rootPool.Close())
	})
	require.NoError(t, db.Exec("REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL ROLE " + owner).Error; err != nil {
			return err
		}
		if target == "run" {
			if err := aistore.AutoMigrateInvocationLedger(tx); err != nil {
				return err
			}
			if err := tx.Exec("GRANT SELECT ON public.ai_invocations TO product_agent_runtime").Error; err != nil {
				return err
			}
		} else if err := httpapi.InstallProductReviewSchema(tx); err != nil {
			return err
		}
		return tx.Exec("CREATE TABLE public.preserved_fact(id int PRIMARY KEY,value text NOT NULL); INSERT INTO public.preserved_fact VALUES(1,'original')").Error
	}))
	return prepareFixture{db: db, dsn: dsn, name: name}
}

func (f prepareFixture) role(t *testing.T, role string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.Open(f.dsn+" dbname="+f.name+" user="+role+" password=synthetic-native-role"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	return db
}

func TestPrepareRunUsesOriginalOwnersAndBoundedRuntimeRoles(t *testing.T) {
	f := newPrepareFixture(t, "run")
	result, err := PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.Equal(t, "CREATED", result.Outcome)
	runtime, writer := f.role(t, "product_agent_runtime"), f.role(t, "title_credential_writer")
	require.NoError(t, runtime.Exec("INSERT INTO public.product_agent_runs VALUES('run','org','actor','acquisition','context','request',? ,1,'running',?)", strings.Repeat("a", 64), []byte(`{}`)).Error)
	require.NoError(t, runtime.Exec("UPDATE public.product_agent_runs SET revision=2,phase='interrupted',payload=? WHERE run_id='run'", []byte(`{}`)).Error)
	require.Error(t, runtime.Exec("UPDATE public.product_agent_runs SET actor='another' WHERE run_id='run'").Error)
	require.NoError(t, runtime.Exec("INSERT INTO public.product_agent_tool_calls VALUES('run','call',?)", []byte(`{}`)).Error)
	require.Error(t, runtime.Exec("UPDATE public.product_agent_tool_calls SET payload=?", []byte(`{}`)).Error)
	store := openai.NewGormCredentialResolver(writer)
	row := openai.AIClientCredential{TenantID: "org", ClientName: "content", APIKey: "synthetic-key", BaseURL: "https://example.test/v1", Model: "synthetic-model", APIStyle: "openai-compatible", Enabled: true}
	require.NoError(t, store.SaveCredential(context.Background(), row))
	row.Enabled = false
	require.NoError(t, store.SaveCredential(context.Background(), row))
	var count int64
	require.NoError(t, runtime.Table("public.ai_client_credentials").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.Error(t, runtime.Exec("UPDATE public.ai_client_credentials SET enabled=true").Error)
	require.Error(t, writer.Exec("DELETE FROM public.ai_client_credentials").Error)
	require.Error(t, writer.Exec("UPDATE public.ai_client_credentials SET tenant_id='other'").Error)
	ledger := aistore.NewGormInvocationRecorder(runtime)
	fact := aicapability.InvocationRecord{InvocationID: "run:step:1", TenantID: "org", UserID: "actor", MemberID: "member", AgentRunID: "run", InputHash: "input", StartedAt: time.Now().UTC(), Outcome: aicapability.InvocationDispatched, Operation: aicapability.OperationProductContentDecision}
	won, err := ledger.ClaimInvocation(context.Background(), fact)
	require.NoError(t, err)
	require.True(t, won)
	fact.Outcome, fact.FinishedAt, fact.UsageKnown = aicapability.InvocationSucceeded, time.Now().UTC(), true
	fact.PromptTokens, fact.CompletionTokens, fact.TotalTokens = 3, 2, 5
	require.NoError(t, ledger.RecordInvocation(context.Background(), fact))
	require.Error(t, runtime.Exec("DELETE FROM public.ai_invocations").Error)
	result, err = PrepareRun(context.Background(), f.db)
	require.NoError(t, err)
	require.Equal(t, "ALREADY_READY", result.Outcome)
	var preserved string
	require.NoError(t, f.db.Raw("SELECT value FROM public.preserved_fact WHERE id=1").Scan(&preserved).Error)
	require.Equal(t, "original", preserved)
}

func TestPrepareRunRejectsPartialAndDriftWithoutRepair(t *testing.T) {
	for _, test := range []string{"partial", "drift"} {
		t.Run(test, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			if test == "partial" {
				require.NoError(t, f.db.Transaction(func(tx *gorm.DB) error {
					if err := tx.Exec("SET LOCAL ROLE product_agent_owner").Error; err != nil {
						return err
					}
					return agentstore.InstallSchema(tx)
				}))
			} else {
				_, err := PrepareRun(context.Background(), f.db)
				require.NoError(t, err)
				require.NoError(t, f.db.Exec("ALTER TABLE public.product_agent_runs ADD COLUMN foreign_fact text").Error)
			}
			_, err := PrepareRun(context.Background(), f.db)
			require.Error(t, err)
			if test == "partial" {
				require.False(t, f.db.Migrator().HasTable("ai_client_credentials"))
			} else {
				require.True(t, f.db.Migrator().HasColumn("product_agent_runs", "foreign_fact"))
			}
		})
	}
}

func TestPrepareReviewKeepsExistingFactsAndUsesExactMutableColumns(t *testing.T) {
	f := newPrepareFixture(t, "review")
	result, err := PrepareReview(context.Background(), f.db)
	require.NoError(t, err)
	require.Equal(t, "GRANTED", result.Outcome)
	runtime := f.role(t, "product_review_runtime")
	require.NoError(t, runtime.Exec("INSERT INTO public.product_title_proposals VALUES('org','proposal','actor','pending',?)", []byte(`{"State":"pending"}`)).Error)
	require.NoError(t, runtime.Exec("UPDATE public.product_title_proposals SET state='accepted',payload=? WHERE org='org' AND id='proposal'", []byte(`{"State":"accepted"}`)).Error)
	require.Error(t, runtime.Exec("UPDATE public.product_title_proposals SET owner='other'").Error)
	require.NoError(t, runtime.Exec("INSERT INTO public.product_title_operations VALUES('org','actor','operation','fingerprint',?)", []byte(`{}`)).Error)
	require.NoError(t, runtime.Exec("INSERT INTO public.product_snapshot_versions(tenant_id,product_key,version,publication_id,payload_hash,snapshot_json) VALUES('org','product',1,'publication',?,?)", strings.Repeat("a", 64), `{}`).Error)
	require.NoError(t, runtime.Exec("INSERT INTO public.product_snapshot_heads VALUES('org','product',1,'publication')").Error)
	require.NoError(t, runtime.Exec("UPDATE public.product_snapshot_heads SET current_version=2,publication_id='next'").Error)
	require.Error(t, runtime.Exec("UPDATE public.product_snapshot_heads SET tenant_id='other'").Error)
	for _, table := range []string{"product_source_publications", "product_source_publication_receipts"} {
		require.NoError(t, runtime.Table("public."+table).Count(new(int64)).Error)
		require.Error(t, runtime.Exec("DELETE FROM public."+table).Error)
	}
	result, err = PrepareReview(context.Background(), f.db)
	require.NoError(t, err)
	require.Equal(t, "ALREADY_READY", result.Outcome)
	var preserved string
	require.NoError(t, f.db.Raw("SELECT value FROM public.preserved_fact WHERE id=1").Scan(&preserved).Error)
	require.Equal(t, "original", preserved)
}
