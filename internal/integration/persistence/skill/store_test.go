package skillpersistence

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	configstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/agentconfig"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/skill"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fixture(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("ISSUE688_TEST_DSN")
	if dsn == "" {
		t.Skip("requires task-isolated PostgreSQL ISSUE688_TEST_DSN")
	}
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "skill688_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.Path = "/" + name
	db, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() {
		raw, _ := db.DB()
		_ = raw.Close()
		_ = root.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error
		raw, _ = root.DB()
		_ = raw.Close()
	})
	require.NoError(t, configstore.InstallSchema(db))
	require.NoError(t, InstallSchema(context.Background(), db))
	s, e := New(db)
	require.NoError(t, e)
	return db, s
}
func packageInput(t *testing.T, body string) d.Package {
	t.Helper()
	p, e := d.ValidatePackage([]d.File{{Path: "SKILL.md", Text: "---\nname: product-title-normalization\ndescription: Normalize titles using saved facts.\n---\n" + body + "\n"}})
	require.NoError(t, e)
	return p
}
func TestVersionPersistenceAndExactCommandReplay(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	p := d.Principal{UserID: "alice"}
	resolve := func(context.Context) (d.Principal, error) { return p, nil }
	create := d.Command{Principal: p, Owner: d.Owner{Kind: d.UserOwner, ID: p.UserID}, Key: uuid.NewString(), Operation: "create", Package: packageInput(t, "Do not invent facts."), AllowCopy: true, AllowExport: true}
	receipt, e := s.Execute(ctx, create, resolve)
	require.NoError(t, e)
	require.Equal(t, uint64(1), receipt.Version)
	replay, e := s.Execute(ctx, create, resolve)
	require.NoError(t, e)
	require.Equal(t, receipt, replay)
	changed := create
	changed.Package = packageInput(t, "Different method.")
	_, e = s.Execute(ctx, changed, resolve)
	require.ErrorIs(t, e, d.ErrConflict)
	read, e := s.Read(ctx, p, create.Owner, receipt.ID, 1)
	require.NoError(t, e)
	require.Equal(t, create.Package.Digest, read.Package.Digest)
	update := create
	update.Key = uuid.NewString()
	update.ID = receipt.ID
	update.Operation = "save-version"
	update.Expected = 1
	update.Package = packageInput(t, "Updated method.")
	second, e := s.Execute(ctx, update, resolve)
	require.NoError(t, e)
	require.Equal(t, uint64(2), second.Version)
	old, e := s.Read(ctx, p, create.Owner, receipt.ID, 1)
	require.NoError(t, e)
	require.Equal(t, read.Package, old.Package)
	_, e = s.Read(ctx, d.Principal{UserID: "organization-admin", OrganizationID: "org", ConfigureOrganization: true, PlatformMaintainer: true}, create.Owner, receipt.ID, 1)
	require.Error(t, e)
	stale := update
	stale.Key = uuid.NewString()
	_, e = s.Execute(ctx, stale, resolve)
	require.ErrorIs(t, e, d.ErrRevision)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM skill_capability.versions WHERE skill_id=?", receipt.ID).Scan(&count).Error)
	require.EqualValues(t, 2, count)
}

func TestPrivateChatTextSaveUseAndRevocation(t *testing.T) {
	_, store := fixture(t)
	ctx := context.Background()
	principal := d.Principal{UserID: "alice"}
	resolve := func(context.Context) (d.Principal, error) { return principal, nil }
	pkg, err := d.ValidatePackage([]d.File{{Path: "SKILL.md", Text: "---\nname: handoff-notes\ndescription: Organize notes.\nmetadata:\n  listingkit-method-v1: chat-text\n---\nReturn actions.\n"}})
	require.NoError(t, err)
	owner := d.Owner{Kind: d.UserOwner, ID: principal.UserID}
	create := d.Command{Principal: principal, Owner: owner, Operation: d.CreateChatText, Key: uuid.NewString(), Package: pkg}
	receipt, err := store.Execute(ctx, create, resolve)
	require.NoError(t, err)
	replay, err := store.Execute(ctx, create, resolve)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	version, release, err := store.AcquireChatText(ctx, principal.UserID, receipt.ID, receipt.Version, pkg.Digest)
	require.NoError(t, err)
	release()
	require.Equal(t, d.Enabled, version.State)
	require.Equal(t, d.Private, version.Visibility)
	require.False(t, version.Package.Supports(d.TitleTarget))
	_, _, err = store.AcquireChatText(ctx, "bob", receipt.ID, receipt.Version, pkg.Digest)
	require.ErrorIs(t, err, d.ErrNotFound)
	_, err = store.Execute(ctx, d.Command{Principal: principal, Owner: owner, Operation: "revoke", Key: uuid.NewString(), ID: receipt.ID, Version: receipt.Version, Expected: receipt.Revision}, resolve)
	require.NoError(t, err)
	_, _, err = store.AcquireChatText(ctx, principal.UserID, receipt.ID, receipt.Version, pkg.Digest)
	require.ErrorIs(t, err, d.ErrNotEnabled)
}

func TestVerifySchemaRejectsConstraintDrift(t *testing.T) {
	db, _ := fixture(t)
	require.NoError(t, db.Exec("ALTER TABLE skill_capability.versions DROP CONSTRAINT versions_files_check").Error)
	require.NoError(t, db.Exec("ALTER TABLE skill_capability.versions ADD CONSTRAINT versions_files_check CHECK (TRUE)").Error)
	require.ErrorIs(t, VerifySchema(context.Background(), db), d.ErrUnavailable)
}
