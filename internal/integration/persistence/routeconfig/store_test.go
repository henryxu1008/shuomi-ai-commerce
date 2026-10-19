package routeconfigstore

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"strings"
	"sync"
	d "task-processor/internal/aicapability/routeconfig"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/openai"
	"testing"
)

func fixture(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("ISSUE830_TEST_DSN")
	if dsn == "" {
		t.Skip("requires task-isolated PostgreSQL ISSUE830_TEST_DSN")
	}
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	require.Equal(t, "127.0.0.1", u.Hostname())
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "route830_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, root.Exec("CREATE DATABASE "+name).Error)
	u.Path = "/" + name
	db, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() {
		r, _ := db.DB()
		_ = r.Close()
		_ = root.Exec("DROP DATABASE " + name + " WITH (FORCE)").Error
		r, _ = root.DB()
		_ = r.Close()
	})
	require.NoError(t, db.Migrator().CreateTable(&openai.AIClientCredential{}))
	require.NoError(t, InstallSchema(context.Background(), db))
	s, e := New(db)
	require.NoError(t, e)
	return db, s
}
func command(scope d.Scope, key, expected string) d.Command {
	return d.Command{Scope: scope, ActorID: "maintainer", Key: key, Fingerprint: key, Input: d.Input{ExpectedRevision: expected}}
}
func build(_ d.CredentialReader, id string) (d.Payload, error) {
	return d.Payload{BindingID: "candidate-a", CredentialVersion: "credential-a", Rates: d.Rates{PointsPerImage: "10"}, Image: &d.ImageProfile{ConfigurationVersion: id, PriceVersion: id, Points: 10}}, nil
}
func TestAtomicCASReplayHistoryAndRestart(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	first, e := s.Publish(ctx, command(scope, "one", d.NoRevision), build)
	require.NoError(t, e)
	replay, e := s.Publish(ctx, command(scope, "one", d.NoRevision), func(d.CredentialReader, string) (d.Payload, error) {
		t.Fatal("replay must precede credential revalidation")
		return d.Payload{}, nil
	})
	require.NoError(t, e)
	require.Equal(t, first.ID, replay.ID)
	bad := command(scope, "one", d.NoRevision)
	bad.Fingerprint = "different"
	_, e = s.Publish(ctx, bad, build)
	require.ErrorIs(t, e, d.ErrConflict)
	second, e := s.Publish(ctx, command(scope, "two", first.ID), build)
	require.NoError(t, e)
	restarted, e := New(db)
	require.NoError(t, e)
	current, e := restarted.Current(ctx, scope)
	require.NoError(t, e)
	require.Equal(t, second.ID, current.ID)
	old, e := restarted.Receipt(ctx, scope, "one")
	require.NoError(t, e)
	require.Equal(t, first.ID, old.ID)
	_, e = s.Receipt(ctx, d.Scope{OrganizationID: "org-b", Capability: d.ImageCapability}, "one")
	require.ErrorIs(t, e, d.ErrNotFound)
	history, e := s.History(ctx, scope, "", 50)
	require.NoError(t, e)
	require.Len(t, history, 2)
}
func TestConcurrentFirstPublicationAndRollbackAreAtomic(t *testing.T) {
	db, s := fixture(t)
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, key := range []string{"one", "two"} {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			_, e := s.Publish(ctx, command(scope, k, d.NoRevision), build)
			errs <- e
		}(key)
	}
	wg.Wait()
	close(errs)
	wins, conflicts := 0, 0
	for e := range errs {
		if e == nil {
			wins++
		} else {
			require.ErrorIs(t, e, d.ErrConflict)
			conflicts++
		}
	}
	require.Equal(t, 1, wins)
	require.Equal(t, 1, conflicts)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM public.ai_route_config_revisions").Scan(&count).Error)
	require.EqualValues(t, 1, count)
	current, e := s.Current(ctx, scope)
	require.NoError(t, e)
	_, e = s.Publish(ctx, command(scope, "fail", current.ID), func(d.CredentialReader, string) (d.Payload, error) { return d.Payload{}, d.ErrNeedsConfiguration })
	require.ErrorIs(t, e, d.ErrNeedsConfiguration)
	after, e := s.Current(ctx, scope)
	require.NoError(t, e)
	require.Equal(t, current.ID, after.ID)
}

func TestConcurrentSameKeyRecoversOneReceipt(t *testing.T) {
	_, s := fixture(t)
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	ctx := context.Background()
	first, e := s.Publish(ctx, command(scope, "seed", d.NoRevision), build)
	require.NoError(t, e)
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup
	ids := make(chan string, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := s.Publish(ctx, command(scope, "same", first.ID), func(reader d.CredentialReader, id string) (d.Payload, error) {
				entered <- struct{}{}
				<-release
				return build(reader, id)
			})
			if e != nil {
				ids <- e.Error()
			} else {
				ids <- r.ID
			}
		}()
	}
	<-entered
	<-entered
	close(release)
	wg.Wait()
	close(ids)
	var got []string
	for id := range ids {
		got = append(got, id)
	}
	require.Equal(t, got[0], got[1], "same-key concurrency must recover the original receipt")
}

func TestRestrictedWriterPublicationRotationRollbackAndLostResponse(t *testing.T) {
	db, _ := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	owner := openai.NewGormCredentialResolver(db)
	for _, id := range []string{"image-a", "image-b"} {
		require.NoError(t, owner.SaveCredential(ctx, openai.AIClientCredential{TenantID: scope.OrganizationID, ClientName: id, APIKey: "fixture-only-secret", BaseURL: "https://provider.example.test/v1", Model: "gpt-image-2.5", APIStyle: "grsai", TimeoutSecond: 30, Enabled: true}))
	}
	var defs []qualified.Definition
	for _, id := range []string{"image-a", "image-b"} {
		row, e := owner.GetCredential(ctx, scope.OrganizationID, "", id)
		require.NoError(t, e)
		defs = append(defs, qualified.Definition{Scope: scope, Image: &qualified.ImageAdmission{ClientName: id, Provider: "grsai", Model: row.Model, Endpoint: row.BaseURL, CredentialVersion: openai.CredentialVersion(*row), MaximumPointsPerImage: 100}})
	}
	catalog, e := qualified.NewCatalog(defs)
	require.NoError(t, e)
	role := "route830_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, db.Exec("CREATE ROLE "+role+" LOGIN PASSWORD 'fixture830'").Error)
	t.Cleanup(func() { _ = db.Exec("DROP OWNED BY " + role).Error; _ = db.Exec("DROP ROLE " + role).Error })
	require.NoError(t, db.Exec(GrantSQL(role, true)).Error)
	var name string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&name).Error)
	u, e := url.Parse(os.Getenv("ISSUE830_TEST_DSN"))
	require.NoError(t, e)
	u.Path = "/" + name
	u.User = url.UserPassword(role, "fixture830")
	writer, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() { raw, _ := writer.DB(); _ = raw.Close() })
	require.NoError(t, VerifyWriter(ctx, writer))
	require.Error(t, writer.Exec("UPDATE public.ai_client_credentials SET enabled=false").Error)
	var secret string
	require.Error(t, writer.Raw("SELECT api_key FROM public.ai_client_credentials LIMIT 1").Scan(&secret).Error)
	repo, e := New(writer)
	require.NoError(t, e)
	service := &d.Service{Repository: repo, Catalog: catalog}
	bindings, e := service.Bindings(ctx, scope)
	require.NoError(t, e)
	require.Len(t, bindings, 2)
	in := d.Input{ExpectedRevision: d.NoRevision, BindingID: bindings[0].ID, CredentialVersion: bindings[0].CredentialVersion, Rates: d.Rates{PointsPerImage: "10"}}
	_, e = service.Validate(ctx, scope, in)
	require.NoError(t, e)
	first, e := service.Publish(ctx, scope, "maintainer", "one", in)
	require.NoError(t, e)
	// Discarding the response does not lose its receipt or justify a new key.
	receipt, e := repo.Receipt(ctx, scope, "one")
	require.NoError(t, e)
	require.Equal(t, first.ID, receipt.ID)
	next := in
	next.ExpectedRevision = first.ID
	next.BindingID = bindings[1].ID
	next.CredentialVersion = bindings[1].CredentialVersion
	second, e := service.Publish(ctx, scope, "maintainer", "two", next)
	require.NoError(t, e)
	rollback := in
	rollback.ExpectedRevision = second.ID
	rollback.RollbackFrom = first.ID
	restored, e := service.Publish(ctx, scope, "maintainer", "restore", rollback)
	require.NoError(t, e)
	require.NotEqual(t, first.Payload.Image.ConfigurationVersion, restored.Payload.Image.ConfigurationVersion)
	row, e := owner.GetCredential(ctx, scope.OrganizationID, "", in.BindingID)
	require.NoError(t, e)
	row.Enabled = false
	require.NoError(t, owner.SaveCredential(ctx, *row))
	current, e := service.Current(ctx, scope)
	require.ErrorIs(t, e, d.ErrNeedsConfiguration)
	require.Equal(t, restored.ID, current.ID)
	replay, e := service.Publish(ctx, scope, "maintainer", "one", in)
	require.NoError(t, e)
	require.Equal(t, first.ID, replay.ID, "rotation does not replace historical publication receipt")
	require.Error(t, writer.Exec("UPDATE public.ai_route_config_revisions SET payload='{}'").Error)
	require.Error(t, writer.Exec("DELETE FROM public.ai_route_config_current").Error)
}
