package channelcredentialstore

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	credential "task-processor/internal/aicapability/channelcredential"
)

func channelFixture(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	if os.Getenv("CI") == "" && os.Getenv("ISSUE1089_OWNED_POSTGRES") == "" {
		t.Skip("requires an explicitly enabled task-owned PostgreSQL testcontainer")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("channel1089"), tcpostgres.WithUsername("fixture1089"), tcpostgres.WithPassword("synthetic1089"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE ROLE product_agent_owner LOGIN PASSWORD 'synthetic-owner-1089' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	require.NoError(t, db.Exec("ALTER DATABASE channel1089 OWNER TO product_agent_owner").Error)
	require.NoError(t, db.Exec("CREATE ROLE ai_channel_secret_reader LOGIN CONNECTION LIMIT 2 PASSWORD 'synthetic-reader' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS; CREATE ROLE model_route_config_writer LOGIN CONNECTION LIMIT 2 PASSWORD 'synthetic-reader' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS").Error)
	require.NoError(t, db.Exec("REVOKE ALL ON DATABASE channel1089 FROM PUBLIC; REVOKE ALL ON SCHEMA public FROM PUBLIC").Error)

	root, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, root.Close())
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.User = url.UserPassword("product_agent_owner", "synthetic-owner-1089")
	db, err = gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() { raw, _ := db.DB(); require.NoError(t, raw.Close()) })
	require.NoError(t, InstallSchema(ctx, db))
	p, err := NewProtection("synthetic-protection-v1", []byte(strings.Repeat("a", 32)))
	require.NoError(t, err)
	s, err := New(db, p)
	require.NoError(t, err)
	return db, s
}

func channelCommand(key string, expected int64) credential.Command {
	return credential.Command{OrganizationID: "org-a", ActorID: "operator", Key: key, ConnectionID: "openai-account", ExpectedRevision: expected,
		Provider: "openai", Channel: "openai-direct", Account: "account-a", Region: "global", BaseURL: "https://api.openai.com/v1", Secret: "synthetic-key-a", Enabled: true}
}

func TestChannelManagementWriterAndReceiptUseOnlyOriginalIsolatedOwner(t *testing.T) {
	db, _ := channelFixture(t)
	// This DSN belongs exclusively to channelFixture's freshly created testcontainer.
	u, e := url.Parse(db.Dialector.(*postgres.Dialector).Config.DSN)
	require.NoError(t, e)
	u.User = url.UserPassword("fixture1089", "synthetic1089")
	root, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	raw, e := root.DB()
	require.NoError(t, e)
	raw.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, raw.Close()) })
	require.NoError(t, root.Exec(`CREATE ROLE title_credential_writer LOGIN CONNECTION LIMIT 2 PASSWORD 'synthetic-ui-writer' NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION NOBYPASSRLS;
 GRANT CONNECT ON DATABASE channel1089 TO title_credential_writer;
 GRANT USAGE ON SCHEMA public TO title_credential_writer;
 GRANT SELECT,INSERT ON public.ai_channel_connections TO title_credential_writer;
 GRANT UPDATE(revision,enabled,protection_key_id,secret_ciphertext) ON public.ai_channel_connections TO title_credential_writer;
 GRANT SELECT ON public.ai_channel_connection_receipts TO title_credential_writer`).Error)
	u.User = url.UserPassword("title_credential_writer", "synthetic-ui-writer")
	writer, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	w, e := writer.DB()
	require.NoError(t, e)
	w.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, w.Close()) })
	ctx := context.Background()
	require.ErrorIs(t, VerifyManagementWriter(ctx, writer), credential.ErrUnavailable, "SELECT alone cannot qualify the original receipt INSERT path")
	require.NoError(t, root.Exec(`GRANT INSERT ON public.ai_channel_connection_receipts TO title_credential_writer`).Error)
	require.NoError(t, VerifyManagementWriter(ctx, writer))
	require.NoError(t, root.Exec(`ALTER ROLE title_credential_writer CONNECTION LIMIT 4`).Error)
	require.NoError(t, VerifyManagementWriter(ctx, writer), "the original role contract is preserved; the application pool is independently bounded")
	protection, e := NewProtection("synthetic-ui", []byte(strings.Repeat("u", 32)))
	require.NoError(t, e)
	write, e := New(writer, protection)
	require.NoError(t, e)
	read, e := New(db, nil)
	require.NoError(t, e)
	manager, e := credential.NewManagement(read, write, []credential.ConnectionDefinition{{OrganizationID: "org-a", ConnectionID: "openai-account", Provider: "openai", Channel: "openai-direct", Region: "global", BaseURL: "https://api.openai.com/v1"}})
	require.NoError(t, e)
	input := credential.ManagementInput{Action: "set-key", Account: "account-a", APIKey: "synthetic-ui-key"}
	first, e := manager.Publish(ctx, "org-a", "operator", "openai-account", "original", input)
	require.NoError(t, e)
	input.ExpectedRevision = 1
	input.Account = ""
	input.APIKey = "synthetic-ui-next"
	commit := write.commitTransaction
	write.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("synthetic ACK loss")
	}
	_, e = manager.Publish(ctx, "org-a", "operator", "openai-account", "rotate", input)
	require.ErrorIs(t, e, credential.ErrOutcomeUnknown)
	write.commitTransaction = commit
	receipt, e := manager.Receipt(ctx, "org-a", "operator", "openai-account", "rotate")
	require.NoError(t, e)
	require.EqualValues(t, 2, receipt.Connection.Revision)
	_, e = manager.Receipt(ctx, "org-a", "operator", "another-connection", "rotate")
	require.ErrorIs(t, e, credential.ErrNotFound)
	_, e = manager.Publish(ctx, "org-a", "operator", "openai-account", "disable", credential.ManagementInput{Action: "disable", ExpectedRevision: 2})
	require.NoError(t, e)
	original, e := manager.Receipt(ctx, "org-a", "operator", "openai-account", "original")
	require.NoError(t, e)
	require.Equal(t, first, original, "current disable does not rewrite the original result")
	require.NoError(t, root.Exec(`GRANT DELETE ON public.ai_channel_connections TO title_credential_writer`).Error)
	require.ErrorIs(t, VerifyManagementWriter(ctx, writer), credential.ErrUnavailable, "the management port must reject excess authority")
}

func TestChannelSecretEncryptedAndTenantBound(t *testing.T) {
	db, store := channelFixture(t)
	ctx := context.Background()
	if e := VerifyMetadataProjection(ctx, db); e != nil {
		var def string
		_ = db.Raw("SELECT pg_get_viewdef('public.ai_channel_credential_bindings'::regclass,true)").Scan(&def).Error
		t.Fatalf("metadata projection qualification failed: %s", def)
	}
	meta, err := store.Publish(ctx, channelCommand("create", 0))
	require.NoError(t, err)
	require.EqualValues(t, 1, meta.Revision)
	var row ConnectionRow
	require.NoError(t, db.Take(&row).Error)
	require.NotContains(t, row.SecretCiphertext, "synthetic-key-a")
	encoded, err := json.Marshal(meta)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "synthetic-key")
	require.NotContains(t, string(encoded), row.SecretCiphertext)
	secret, err := store.Resolve(ctx, "org-a", meta.ConnectionID, meta.Reference)
	require.NoError(t, err)
	require.Equal(t, "synthetic-key-a", secret.APIKey)
	_, err = store.Resolve(ctx, "org-b", meta.ConnectionID, meta.Reference)
	require.ErrorIs(t, err, credential.ErrNotFound)
	// Copying ciphertext to a different tenant cannot make it decryptable.
	row.OrganizationID = "org-b"
	require.NoError(t, db.Create(&row).Error)
	other, err := store.Metadata(ctx, "org-b", row.ConnectionID)
	require.NoError(t, err)
	_, err = store.Resolve(ctx, "org-b", row.ConnectionID, other.Reference)
	require.ErrorIs(t, err, credential.ErrUnavailable)
}

func TestChannelCASReplayAndCommitAcknowledgementLoss(t *testing.T) {
	_, store := channelFixture(t)
	ctx := context.Background()
	first, err := store.Publish(ctx, channelCommand("create", 0))
	require.NoError(t, err)
	commit := store.commitTransaction
	store.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("synthetic lost commit acknowledgement")
	}
	rotate := channelCommand("rotate", first.Revision)
	rotate.Secret = "synthetic-key-b"
	_, err = store.Publish(ctx, rotate)
	require.ErrorIs(t, err, credential.ErrOutcomeUnknown)
	store.commitTransaction = commit
	replayed, err := store.Publish(ctx, rotate)
	require.NoError(t, err)
	require.EqualValues(t, 2, replayed.Revision)
	_, err = store.Publish(ctx, channelCommand("create", 0))
	require.NoError(t, err, "original command receipt remains replayable after later rotations")
	rotate.Secret = "different-secret"
	_, err = store.Publish(ctx, rotate)
	require.ErrorIs(t, err, credential.ErrConflict)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, key := range []string{"next-a", "next-b"} {
		wg.Add(1)
		go func(key string) { defer wg.Done(); _, e := store.Publish(ctx, channelCommand(key, 2)); results <- e }(key)
	}
	wg.Wait()
	close(results)
	success, conflicts := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else {
			require.ErrorIs(t, e, credential.ErrConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, success)
	require.Equal(t, 1, conflicts)
}

func TestChannelRotationABAAndMetadataWithoutExecutionSecrets(t *testing.T) {
	db, store := channelFixture(t)
	ctx := context.Background()
	first, err := store.Publish(ctx, channelCommand("a", 0))
	require.NoError(t, err)
	secondCmd := channelCommand("b", 1)
	secondCmd.Secret = "synthetic-key-b"
	second, err := store.Publish(ctx, secondCmd)
	require.NoError(t, err)
	third, err := store.Publish(ctx, channelCommand("back-to-a", 2))
	require.NoError(t, err)
	require.NotEqual(t, first.Reference, third.Reference, "ABA cannot revive an old prepare")
	_, err = store.Resolve(ctx, "org-a", first.ConnectionID, first.Reference)
	require.ErrorIs(t, err, credential.ErrChanged)
	_, err = store.Resolve(ctx, "org-a", second.ConnectionID, second.Reference)
	require.ErrorIs(t, err, credential.ErrChanged)
	disabled := channelCommand("disable", 3)
	disabled.Secret = ""
	disabled.Enabled = false
	fourth, err := store.Publish(ctx, disabled)
	require.NoError(t, err)
	reader, err := New(db, nil)
	require.NoError(t, err)
	metadata, err := reader.Metadata(ctx, "org-a", fourth.ConnectionID)
	require.NoError(t, err, "management metadata must not require decryption material")
	require.False(t, metadata.Enabled)
	receipt, err := reader.Receipt(ctx, "org-a", "operator", "a")
	require.NoError(t, err)
	require.Equal(t, first.Reference, receipt.Reference, "historical receipt does not consult current credential")
	_, err = reader.Resolve(ctx, "org-a", fourth.ConnectionID, fourth.Reference)
	require.ErrorIs(t, err, credential.ErrUnavailable)
}

func TestChannelProtectionBindsRevisionAndRequestSecret(t *testing.T) {
	p, err := NewProtection("synthetic", []byte(strings.Repeat("a", 32)))
	require.NoError(t, err)
	input := channelCommand("one", 0)
	first, err := p.Fingerprint(input)
	require.NoError(t, err)
	input.Secret = "another-key"
	second, err := p.Fingerprint(input)
	require.NoError(t, err)
	require.NotEqual(t, first, second, "same request key with a different secret conflicts")
	require.NotContains(t, first, "synthetic-key")
}

func fixtureReaderRole(t *testing.T, db *gorm.DB, role string) *gorm.DB {
	t.Helper()
	u, e := url.Parse(db.Dialector.(*postgres.Dialector).Config.DSN)
	require.NoError(t, e)
	u.User = url.UserPassword(role, "synthetic-reader")
	pool, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	raw, e := pool.DB()
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, raw.Close()) })
	return pool
}
func TestChannelMetadataAndSecretCapabilitiesAreSeparated(t *testing.T) {
	db, owner := channelFixture(t)
	ctx := context.Background()
	m, e := owner.Publish(ctx, channelCommand("create", 0))
	require.NoError(t, e)
	require.NoError(t, db.Exec("GRANT CONNECT ON DATABASE channel1089 TO ai_channel_secret_reader,model_route_config_writer; GRANT USAGE ON SCHEMA public TO ai_channel_secret_reader,model_route_config_writer; GRANT SELECT ON public.ai_channel_credential_bindings TO model_route_config_writer; GRANT SELECT ON public.ai_channel_connections TO ai_channel_secret_reader").Error)
	metadataDB := fixtureReaderRole(t, db, "model_route_config_writer")
	metadata, _ := New(metadataDB, nil)
	got, e := metadata.Metadata(ctx, "org-a", m.ConnectionID)
	require.NoError(t, e)
	require.Equal(t, m.Reference, got.Reference)
	var n int64
	require.Error(t, metadataDB.Table("public.ai_channel_connections").Count(&n).Error)
	require.Error(t, metadataDB.Table("public.ai_channel_connection_receipts").Count(&n).Error)
	secretsDB := fixtureReaderRole(t, db, "ai_channel_secret_reader")
	require.NoError(t, VerifySecretReader(ctx, secretsDB))
	secrets, _ := New(secretsDB, owner.protection)
	_, e = secrets.Resolve(ctx, "org-a", m.ConnectionID, m.Reference)
	require.NoError(t, e)
	require.Error(t, secretsDB.Exec("UPDATE public.ai_channel_connections SET enabled=false").Error)
	require.NoError(t, db.Exec("GRANT SELECT ON public.ai_channel_connection_receipts TO ai_channel_secret_reader").Error)
	require.Error(t, VerifySecretReader(ctx, secretsDB), "an overprivileged secret pool is not an admitted narrow reader")
}
func TestChannelSchemaRefusesRetainedAndDriftedObjectsWithoutRepair(t *testing.T) {
	db, _ := channelFixture(t)
	ctx := context.Background()
	require.NoError(t, VerifyMetadataProjection(ctx, db))
	require.NoError(t, db.Exec("CREATE TABLE public.ai_client_credentials(marker text); INSERT INTO public.ai_client_credentials VALUES('retained')").Error)
	require.Error(t, InstallSchema(ctx, db))
	require.Error(t, VerifyMetadataProjection(ctx, db))
	var marker string
	require.NoError(t, db.Raw("SELECT marker FROM public.ai_client_credentials").Scan(&marker).Error)
	require.Equal(t, "retained", marker)
	require.NoError(t, db.Exec("DROP TABLE public.ai_client_credentials; CREATE OR REPLACE VIEW public.ai_channel_credential_bindings AS SELECT organization_id,connection_id,provider,channel,account,region,base_url,revision,enabled,true AS key_present FROM public.ai_channel_connections").Error)
	require.Error(t, VerifyMetadataProjection(ctx, db))
	var def string
	require.NoError(t, db.Raw("SELECT pg_get_viewdef('public.ai_channel_credential_bindings'::regclass,true)").Scan(&def).Error)
	require.Contains(t, def, "true AS key_present", "qualification must not repair a drifted projection")
}
