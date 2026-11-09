package productsourcing

import (
	"context"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/gorm"
	researchstore "task-processor/internal/integration/persistence/product/research"
)

// Only the task-owned temporary PostgreSQL below receives writes.
func TestAssessmentFeaturePreservesCurrentProduct(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg, err := tcpostgres.Run(ctx, "postgres:17.2-alpine", tcpostgres.WithDatabase("assessment_installer"), tcpostgres.WithUsername("test_admin"), tcpostgres.WithPassword("isolated-admin"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pg.Terminate(context.Background())) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	admin := openSupplyFeatureTestDB(t, dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE acquisition_owner LOGIN PASSWORD 'isolated-owner' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
CREATE ROLE product_research_runtime LOGIN PASSWORD 'isolated-runtime' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;`).Error)
	fixture := func(t *testing.T) (*gorm.DB, string, string) {
		t.Helper()
		name := "assessment_" + strings.ReplaceAll(uuid.NewString(), "-", "")
		require.NoError(t, admin.Exec("CREATE DATABASE "+name+" OWNER acquisition_owner").Error)
		u, e := url.Parse(dsn)
		require.NoError(t, e)
		u.Path, u.User = "/"+name, url.UserPassword("acquisition_owner", "isolated-owner")
		owner := openSupplyFeatureTestDB(t, u.String())
		require.NoError(t, owner.Exec("REVOKE CREATE ON SCHEMA public FROM PUBLIC").Error)
		require.NoError(t, researchstore.InstallSchema(owner))
		require.NoError(t, researchstore.GrantRuntime(owner))
		require.NoError(t, researchstore.InstallSupplierReviewSchema(owner))
		require.NoError(t, owner.Exec(`INSERT INTO research_opportunities VALUES ('org','actor','11111111-1111-4111-8111-111111111111','22222222-2222-4222-8222-222222222222','amazon','us','original',NULL,'TO_RESEARCH','unchanged',1);
INSERT INTO research_commands VALUES ('org','actor','11111111-1111-4111-8111-111111111111','33333333-3333-4333-8333-333333333333',repeat('a',64),NULL,'{}','{}',NULL,now());`).Error)
		port, e := strconv.Atoi(u.Port())
		require.NoError(t, e)
		m := acquisitionInitManifest{SchemaVersion: 1, Collections: true, SupplierReview: true, DataServices: true}
		m.Database.Host, m.Database.Port, m.Database.Database = "127.0.0.1", port, name
		m.Database.User, m.Database.Password = "acquisition_owner", "isolated-owner"
		path := filepath.Join(t.TempDir(), "owner.json")
		writeSupplyFeatureManifest(t, path, m)
		return owner, path, name
	}
	t.Run("concurrent installation and replay preserve facts and base grants", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		var before string
		var grantsBefore string
		const baseGrants = `SELECT string_agg(relname||':'||COALESCE(relacl::text,''),',' ORDER BY relname) FROM pg_class WHERE relnamespace='public'::regnamespace AND relname IN ('research_commands','research_opportunities','research_evidence_links','research_supplier_reviews')`
		require.NoError(t, owner.Raw(baseGrants).Scan(&grantsBefore).Error)
		require.NoError(t, owner.Raw("SELECT md5(string_agg(row_to_json(t)::text,',' ORDER BY tab)) FROM (SELECT 'commands' tab, jsonb_agg(to_jsonb(x)) rows FROM research_commands x UNION ALL SELECT 'candidates',jsonb_agg(to_jsonb(x)) FROM research_opportunities x) t").Scan(&before).Error)
		var wg sync.WaitGroup
		errors := make(chan error, 2)
		for range 2 {
			wg.Add(1)
			go func() { defer wg.Done(); errors <- EnableResearchAssessments(ctx, manifest, name) }()
		}
		wg.Wait()
		close(errors)
		for e := range errors {
			require.NoError(t, e)
		}
		require.NoError(t, EnableResearchAssessments(ctx, manifest, name))
		var after string
		require.NoError(t, owner.Raw("SELECT md5(string_agg(row_to_json(t)::text,',' ORDER BY tab)) FROM (SELECT 'commands' tab, jsonb_agg(to_jsonb(x)) rows FROM research_commands x UNION ALL SELECT 'candidates',jsonb_agg(to_jsonb(x)) FROM research_opportunities x) t").Scan(&after).Error)
		require.Equal(t, before, after)
		var grantsAfter string
		require.NoError(t, owner.Raw(baseGrants).Scan(&grantsAfter).Error)
		require.Equal(t, grantsBefore, grantsAfter)
		var bounded bool
		require.NoError(t, owner.Raw(`SELECT has_table_privilege('product_research_runtime','research_assessments','SELECT') AND has_table_privilege('product_research_runtime','research_assessments','INSERT') AND NOT has_table_privilege('product_research_runtime','research_assessments','UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')`).Scan(&bounded).Error)
		require.True(t, bounded)
	})
	t.Run("wrong confirmation and missing context do not create table", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		require.Error(t, EnableResearchAssessments(nil, manifest, name))
		require.Error(t, EnableResearchAssessments(ctx, manifest, "another_database"))
		var absent bool
		require.NoError(t, owner.Raw("SELECT to_regclass('research_assessments') IS NULL").Scan(&absent).Error)
		require.True(t, absent)
	})
	t.Run("base drift refuses before installation", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		require.NoError(t, owner.Exec("ALTER TABLE research_opportunities DROP CONSTRAINT research_opportunities_revision_check").Error)
		require.Error(t, EnableResearchAssessments(ctx, manifest, name))
		var absent bool
		require.NoError(t, owner.Raw("SELECT to_regclass('research_assessments') IS NULL").Scan(&absent).Error)
		require.True(t, absent)
	})
	t.Run("target and ACL drift are not repaired", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		require.NoError(t, EnableResearchAssessments(ctx, manifest, name))
		require.NoError(t, owner.Exec("GRANT SELECT ON research_assessments TO PUBLIC").Error)
		require.Error(t, EnableResearchAssessments(ctx, manifest, name))
		var publicGrant bool
		require.NoError(t, owner.Raw("SELECT EXISTS(SELECT 1 FROM pg_class c CROSS JOIN LATERAL aclexplode(c.relacl) a WHERE c.oid='research_assessments'::regclass AND a.grantee=0)").Scan(&publicGrant).Error)
		require.True(t, publicGrant)
		require.NoError(t, owner.Exec("REVOKE SELECT ON research_assessments FROM PUBLIC; ALTER TABLE research_assessments DROP CONSTRAINT assessment_result").Error)
		require.Error(t, EnableResearchAssessments(ctx, manifest, name))
	})
	t.Run("missing insert default is not an installed feature", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		require.NoError(t, EnableResearchAssessments(ctx, manifest, name))
		require.NoError(t, owner.Exec("ALTER TABLE research_assessments ALTER COLUMN assessment_revision DROP DEFAULT").Error)
		require.Error(t, EnableResearchAssessments(ctx, manifest, name))
	})
	t.Run("failed post-install check rolls back the new table", func(t *testing.T) {
		owner, manifest, name := fixture(t)
		require.NoError(t, owner.Exec("REVOKE USAGE ON SCHEMA public FROM PUBLIC,product_research_runtime").Error)
		require.Error(t, EnableResearchAssessments(ctx, manifest, name))
		var absent bool
		require.NoError(t, owner.Raw("SELECT to_regclass('research_assessments') IS NULL").Scan(&absent).Error)
		require.True(t, absent)
		require.NoError(t, owner.Exec("GRANT USAGE ON SCHEMA public TO product_research_runtime").Error)
		require.NoError(t, EnableResearchAssessments(ctx, manifest, name))
	})
}
