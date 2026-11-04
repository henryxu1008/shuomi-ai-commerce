package productagentruntime

import (
	"context"
	credential "task-processor/internal/aicapability/channelcredential"
	channelstore "task-processor/internal/integration/persistence/channelcredential"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareRejectsCredentialScopeCollationDriftWithoutRepair(t *testing.T) {
	for _, test := range []struct{ column, tenant, lookupTenant, client, lookupClient string }{
		{"organization_id", "TenantA", "tenanta", "content", "content"},
		{"connection_id", "123456789", "123456789", "content-A", "content-a"},
	} {
		t.Run(test.column, func(t *testing.T) {
			f := newPrepareFixture(t, "run")
			_, err := PrepareRun(context.Background(), f.db)
			require.NoError(t, err)
			seedNativeChannel(t, f.role(t, "title_credential_writer"), test.tenant, test.client)
			reader, e := channelstore.New(f.db, nil)
			require.NoError(t, e)
			ctx := context.Background()
			_, err = reader.Metadata(ctx, test.lookupTenant, test.lookupClient)
			require.ErrorIs(t, err, credential.ErrNotFound)

			require.NoError(t, f.db.Exec("CREATE COLLATION public.case_insensitive (provider=icu,locale='und-u-ks-level2',deterministic=false); ALTER TABLE public.ai_channel_connections ALTER COLUMN "+quote(test.column)+" TYPE varchar(128) COLLATE public.case_insensitive").Error)
			_, err = reader.Metadata(ctx, test.lookupTenant, test.lookupClient)
			require.Error(t, err, "even a drifted database cannot change exact tenant/connection identity")
			tables := []string{"public.ai_channel_connections", "public.preserved_fact"}
			before := retainedSnapshot(t, f.db, tables)
			_, err = PrepareRun(context.Background(), f.db)
			require.Error(t, err, "only the original owner's default column collation is qualified")
			require.Equal(t, before, retainedSnapshot(t, f.db, tables))
			var collation string
			require.NoError(t, f.db.Raw("SELECT c.collname FROM pg_catalog.pg_attribute a JOIN pg_catalog.pg_collation c ON c.oid=a.attcollation WHERE a.attrelid='public.ai_channel_connections'::regclass AND a.attname=?", test.column).Scan(&collation).Error)
			require.Equal(t, "case_insensitive", collation)
		})
	}
}
