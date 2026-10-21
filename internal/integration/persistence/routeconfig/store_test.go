package routeconfigstore

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	d "task-processor/internal/aicapability/routeconfig"
	qualified "task-processor/internal/integration/aicapability/routeconfig"
	"task-processor/internal/integration/openai"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestOptionalPublicationDoesNotMoveEnterpriseDefault(t *testing.T) {
	_, s := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	initial, err := s.Publish(ctx, command(scope, "bailian-default", d.NoRevision), build)
	require.NoError(t, err)
	cmd := command(scope, "vip-option", d.NoRevision)
	require.NoError(t, json.Unmarshal([]byte(`{"optional":true,"bindingId":"vip","expectedRevision":"NONE"}`), &cmd.Input))
	option, err := s.Publish(ctx, cmd, func(reader d.CredentialReader, id string) (d.Payload, error) {
		p, e := build(reader, id)
		require.NoError(t, json.Unmarshal([]byte(`{"optional":true,"bindingId":"vip"}`), &p))
		return p, e
	})
	require.NoError(t, err, "optional publication must use its own CAS head")
	current, err := s.Current(ctx, scope)
	require.NoError(t, err)
	require.Equal(t, initial.ID, current.ID, "optional publication must not change Bailian default")
	replay, err := s.Publish(ctx, cmd, func(d.CredentialReader, string) (d.Payload, error) {
		t.Fatal("receipt must not rebuild")
		return d.Payload{}, nil
	})
	require.NoError(t, err)
	require.Equal(t, option.ID, replay.ID)
}

func TestOptionalServiceQualifiedVersionTenantRollbackAndCredentialGuards(t *testing.T) {
	db, repo := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	credentials := openai.NewGormCredentialResolver(db)
	require.NoError(t, credentials.SaveCredential(ctx, openai.AIClientCredential{TenantID: scope.OrganizationID, ClientName: "image-vip", APIKey: "fixture", BaseURL: "https://provider.example.test/v1", Model: "gpt-image-2-vip", APIStyle: "grsai", TimeoutSecond: 30, Enabled: true}))
	row, e := credentials.GetCredential(ctx, scope.OrganizationID, "", "image-vip")
	require.NoError(t, e)
	catalog, e := qualified.NewCatalog([]qualified.Definition{{Scope: scope, Image: &qualified.ImageAdmission{ClientName: row.ClientName, Provider: "grsai", Model: row.Model, Endpoint: row.BaseURL, CredentialVersion: openai.CredentialVersion(*row), MaximumPointsPerImage: 10}}})
	require.NoError(t, e)
	s := &d.Service{Repository: repo, Catalog: catalog}
	options, e := s.Options(ctx, scope)
	require.NoError(t, e)
	require.Empty(t, options, "qualified candidate alone is not published")
	in := d.Input{Optional: true, ExpectedRevision: d.NoRevision, BindingID: row.ClientName, CredentialVersion: openai.CredentialVersion(*row), Rates: d.Rates{PointsPerImage: "2"}}
	first, e := s.Publish(ctx, scope, "maintainer", "option-first", in)
	require.NoError(t, e)
	_, e = s.Current(ctx, scope)
	require.ErrorIs(t, e, d.ErrNotFound, "optional publication cannot initialize default")
	options, e = s.Options(ctx, scope)
	require.NoError(t, e)
	require.Len(t, options, 1)
	require.Equal(t, first.ID, options[0].ID)
	_, e = s.Optional(ctx, d.Scope{OrganizationID: "org-b", Capability: d.ImageCapability}, row.ClientName)
	require.Error(t, e)
	wrongMode := in
	wrongMode.Optional = false
	wrongMode.RollbackFrom = first.ID
	_, e = s.Publish(ctx, scope, "maintainer", "bad-mode", wrongMode)
	require.ErrorIs(t, e, d.ErrConflict)
	_, e = s.Publish(ctx, scope, "maintainer", "option-first", wrongMode)
	require.ErrorIs(t, e, d.ErrConflict, "same key cannot change publication mode")
	in.ExpectedRevision = first.ID
	second, e := s.Publish(ctx, scope, "maintainer", "option-second", in)
	require.NoError(t, e)
	latest, e := s.Optional(ctx, scope, row.ClientName)
	require.NoError(t, e)
	require.Equal(t, second.ID, latest.ID)
	_, e = s.Publish(ctx, scope, "maintainer", "stale-head", in)
	require.ErrorIs(t, e, d.ErrConflict)
	row.Enabled = false
	require.NoError(t, credentials.SaveCredential(ctx, *row))
	_, e = s.Optional(ctx, scope, row.ClientName)
	require.ErrorIs(t, e, d.ErrNeedsConfiguration)
	options, e = s.Options(ctx, scope)
	require.NoError(t, e)
	require.Empty(t, options)
	replayed, e := s.Publish(ctx, scope, "maintainer", "option-first", d.Input{Optional: true, ExpectedRevision: d.NoRevision, BindingID: row.ClientName, CredentialVersion: in.CredentialVersion, Rates: in.Rates})
	require.NoError(t, e)
	require.Equal(t, first.ID, replayed.ID, "durable receipt remains readable after revocation")
}

func TestConcurrentOptionalInitialPublicationUsesOneHeadAndKeepsDefault(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	initial, e := s.Publish(ctx, command(scope, "default", d.NoRevision), build)
	require.NoError(t, e)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for _, key := range []string{"option-a", "option-b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			cmd := command(scope, key, d.NoRevision)
			cmd.Input.Optional = true
			cmd.Input.BindingID = "vip"
			_, e := s.Publish(ctx, cmd, func(r d.CredentialReader, id string) (d.Payload, error) {
				p, e := build(r, id)
				p.Optional = true
				p.BindingID = "vip"
				return p, e
			})
			errs <- e
		}(key)
	}
	wg.Wait()
	close(errs)
	wins := 0
	for e := range errs {
		if e == nil {
			wins++
		} else {
			require.ErrorIs(t, e, d.ErrConflict)
		}
	}
	require.Equal(t, 1, wins)
	current, e := s.Current(ctx, scope)
	require.NoError(t, e)
	require.Equal(t, initial.ID, current.ID)
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM ai_route_config_revisions").Scan(&count).Error)
	require.EqualValues(t, 2, count, "loser revision is rolled back")
}

func TestCommitAcknowledgementLossRecoversOriginalReceipt(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	scope := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	s.commitTransaction = func(tx *gorm.DB) error {
		require.NoError(t, tx.Commit().Error)
		return errors.New("fixture acknowledgement lost after durable commit")
	}
	_, e := s.Publish(ctx, command(scope, "unknown", d.NoRevision), build)
	require.ErrorIs(t, e, d.ErrOutcomeUnknown)
	restarted, e := New(db)
	require.NoError(t, e)
	receipt, e := restarted.Receipt(ctx, scope, "unknown")
	require.NoError(t, e)
	replay, e := restarted.Publish(ctx, command(scope, "unknown", d.NoRevision), func(d.CredentialReader, string) (d.Payload, error) {
		t.Fatal("must not rebuild/re-publish")
		return d.Payload{}, nil
	})
	require.NoError(t, e)
	require.Equal(t, receipt.ID, replay.ID)
	history, e := restarted.History(ctx, scope, "", 50)
	require.NoError(t, e)
	require.Len(t, history, 1)
}

type rotatingPublication struct {
	*Store
	rotate func()
}

func (s rotatingPublication) Publish(ctx context.Context, cmd d.Command, build func(d.CredentialReader, string) (d.Payload, error)) (d.Revision, error) {
	return s.Store.Publish(ctx, cmd, func(reader d.CredentialReader, id string) (d.Payload, error) {
		p, e := build(reader, id)
		if e == nil {
			s.rotate()
		}
		return p, e
	})
}
func TestCredentialDisabledAfterProjectionBeforeCommitFailsCurrentClosed(t *testing.T) {
	db, repo := fixture(t)
	ctx := context.Background()
	sc := d.Scope{OrganizationID: "org-a", Capability: d.ImageCapability}
	owner := openai.NewGormCredentialResolver(db)
	require.NoError(t, owner.SaveCredential(ctx, openai.AIClientCredential{TenantID: sc.OrganizationID, ClientName: "image-a", APIKey: "fixture", BaseURL: "https://provider.example.test/v1", Model: "gpt-image-2.5", APIStyle: "grsai", TimeoutSecond: 30, Enabled: true}))
	row, e := owner.GetCredential(ctx, sc.OrganizationID, "", "image-a")
	require.NoError(t, e)
	catalog, e := qualified.NewCatalog([]qualified.Definition{{Scope: sc, Image: &qualified.ImageAdmission{ClientName: row.ClientName, Provider: "grsai", Model: row.Model, Endpoint: row.BaseURL, CredentialVersion: openai.CredentialVersion(*row), MaximumPointsPerImage: 10}}})
	require.NoError(t, e)
	interleaved := rotatingPublication{Store: repo, rotate: func() { row.Enabled = false; require.NoError(t, owner.SaveCredential(ctx, *row)) }}
	service := &d.Service{Repository: interleaved, Catalog: catalog}
	in := d.Input{ExpectedRevision: d.NoRevision, BindingID: row.ClientName, CredentialVersion: openai.CredentialVersion(*row), Rates: d.Rates{PointsPerImage: "5"}}
	r, e := service.Publish(ctx, sc, "maintainer", "race", in)
	require.NoError(t, e)
	current, e := service.Current(ctx, sc)
	require.ErrorIs(t, e, d.ErrNeedsConfiguration)
	require.Equal(t, r.ID, current.ID)
	receipt, e := repo.Receipt(ctx, sc, "race")
	require.NoError(t, e)
	require.Equal(t, r.ID, receipt.ID)
	_, e = service.Validate(ctx, sc, d.Input{ExpectedRevision: r.ID, BindingID: in.BindingID, CredentialVersion: in.CredentialVersion, Rates: in.Rates})
	require.ErrorIs(t, e, d.ErrNeedsConfiguration)
}

func fixture(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("ISSUE830_TEST_DSN")
	if dsn == "" {
		if os.Getenv("CI") == "" {
			t.Skip("requires task-isolated PostgreSQL ISSUE830_TEST_DSN; CI uses an owned testcontainer")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		container, e := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("postgres"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("fixture830"), tcpostgres.BasicWaitStrategies())
		require.NoError(t, e)
		t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
		dsn, e = container.ConnectionString(ctx, "sslmode=disable")
		require.NoError(t, e)
	}
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	if u.Hostname() == "localhost" {
		u.Host = "127.0.0.1:" + u.Port()
		dsn = u.String()
	}
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
func TestSchemaRejectsMissingCurrentRevisionConstraint(t *testing.T) {
	db, _ := fixture(t)
	require.NoError(t, VerifySchema(context.Background(), db))
	require.NoError(t, db.Exec("ALTER TABLE public.ai_route_config_current DROP CONSTRAINT ai_route_config_current_revision").Error)
	require.Error(t, VerifySchema(context.Background(), db))
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
	originalDSN := db.Dialector.(*postgres.Dialector).Config.DSN
	u, e := url.Parse(originalDSN)
	require.NoError(t, e)
	u.Path = "/" + name
	u.User = url.UserPassword(role, "fixture830")
	writer, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	t.Cleanup(func() { raw, _ := writer.DB(); _ = raw.Close() })
	require.NoError(t, VerifyWriter(ctx, writer))
	require.Error(t, VerifyReader(ctx, writer), "configuration writer must never be injected as serving reader")
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
