package productagentruntime

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/integration/openai"
)

func TestPrepareRejectsCredentialScopeCollationDriftWithoutRepair(t *testing.T) {
	for _, test := range []struct{ column, tenant, lookupTenant, client, lookupClient string }{
		{"tenant_id", "TenantA", "tenanta", "content", "content"},
		{"client_name", "123456789", "123456789", "content-A", "content-a"},
	} {
		t.Run(test.column, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			_, err := PrepareRun(context.Background(), f.db)
			require.NoError(t, err)
			store := openai.NewOrganizationOnlyCredentialResolver(f.role(t, "title_credential_writer"))
			row := openai.AIClientCredential{TenantID: test.tenant, ClientName: test.client, APIKey: "synthetic-only", BaseURL: "https://example.test/v1", Model: "synthetic", Enabled: true}
			require.NoError(t, store.SaveCredential(context.Background(), row))
			reader := openai.NewOrganizationOnlyCredentialResolver(f.role(t, "product_agent_runtime"))
			ctx := openai.WithIdentity(context.Background(), openai.Identity{TenantID: test.lookupTenant})
			_, err = reader.ResolveClientConfig(ctx, test.lookupClient, nil)
			require.ErrorIs(t, err, openai.ErrClientConfigurationUnavailable)
			require.NoError(t, f.db.Exec("CREATE COLLATION public.case_insensitive (provider=icu,locale='und-u-ks-level2',deterministic=false); ALTER TABLE public.ai_client_credentials ALTER COLUMN "+quote(test.column)+" TYPE varchar(64) COLLATE public.case_insensitive").Error)
			reader = openai.NewOrganizationOnlyCredentialResolver(f.role(t, "product_agent_runtime"))
			resolved, err := reader.ResolveClientConfig(ctx, test.lookupClient, nil)
			require.NoError(t, err)
			require.NotNil(t, resolved)
			require.Equal(t, row.APIKey, resolved.Config.APIKey, "collation drift confuses the original exact credential scope")
			tables := []string{"public.ai_client_credentials", "public.preserved_fact"}
			before := retainedSnapshot(t, f.db, tables)
			_, err = PrepareRun(context.Background(), f.db)
			require.Error(t, err, "only the original owner's default column collation is qualified")
			require.Equal(t, before, retainedSnapshot(t, f.db, tables))
			var collation string
			require.NoError(t, f.db.Raw("SELECT c.collname FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_collation c ON c.oid=a.attcollation WHERE a.attrelid='public.ai_client_credentials'::regclass AND a.attname=?", test.column).Scan(&collation).Error)
			require.Equal(t, "case_insensitive", collation)
		})
	}
}
