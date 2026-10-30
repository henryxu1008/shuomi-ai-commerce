//go:build integration

package kernelpersistence

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"strings"
	"sync"
	k "task-processor/internal/browserkernel"
	"testing"
	"time"
)

func TestKernelCatalogPostgres(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	container, e := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("browser_kernel"), tcpostgres.WithUsername("fixture"), tcpostgres.WithPassword("fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { _ = container.Terminate(context.Background()) })
	dsn, e := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	open := func(d string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(d), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		raw, e := db.DB()
		require.NoError(t, e)
		raw.SetMaxOpenConns(4)
		t.Cleanup(func() { _ = raw.Close() })
		return db
	}
	admin := open(dsn)
	require.NoError(t, InstallSchema(admin))
	require.NoError(t, admin.Exec("CREATE ROLE browser_kernel_runtime LOGIN PASSWORD 'fixture-runtime'").Error)
	require.NoError(t, GrantRuntime(admin, "browser_kernel_runtime"))
	runtimeDSN := strings.Replace(dsn, "fixture:fixture@", "browser_kernel_runtime:fixture-runtime@", 1)
	require.NotEqual(t, dsn, runtimeDSN)
	runtime := open(runtimeDSN)
	require.NoError(t, VerifySchema(ctx, runtime))
	require.ErrorIs(t, VerifySchema(ctx, admin), k.ErrUnavailable)
	for _, c := range []struct{ setup, undo string }{
		{"GRANT UPDATE(download_url) ON browser_kernel.releases TO browser_kernel_runtime", "REVOKE UPDATE(download_url) ON browser_kernel.releases FROM browser_kernel_runtime"},
		{"GRANT SELECT ON browser_kernel.releases TO browser_kernel_runtime WITH GRANT OPTION", "REVOKE GRANT OPTION FOR SELECT ON browser_kernel.releases FROM browser_kernel_runtime"},
		{"REVOKE UPDATE(recommended) ON browser_kernel.releases FROM browser_kernel_runtime", "GRANT UPDATE(recommended) ON browser_kernel.releases TO browser_kernel_runtime"},
		{"CREATE TABLE public.other_owner(id text); GRANT SELECT ON public.other_owner TO browser_kernel_runtime", "REVOKE SELECT ON public.other_owner FROM browser_kernel_runtime"},
		{"ALTER TABLE browser_kernel.releases ENABLE ROW LEVEL SECURITY", "ALTER TABLE browser_kernel.releases DISABLE ROW LEVEL SECURITY"},
		{"ALTER TABLE browser_kernel.releases DROP CONSTRAINT releases_pkey", "ALTER TABLE browser_kernel.releases ADD PRIMARY KEY(release_id)"},
	} {
		require.NoError(t, admin.Exec(c.setup).Error)
		require.ErrorIs(t, VerifySchema(ctx, runtime), k.ErrUnavailable, c.setup)
		require.NoError(t, admin.Exec(c.undo).Error)
	}
	require.NoError(t, VerifySchema(ctx, runtime))
	repo, e := New(runtime)
	require.NoError(t, e)
	svc, e := k.New(repo, []string{"https://assets.example.test"})
	require.NoError(t, e)
	p := k.Publish{Version: "132.0.0.1", OS: "windows", Arch: "amd64", MinClientVersion: "1.0.0", ArchiveFormat: "zip", Size: 25, SHA256: strings.Repeat("a", 64), DownloadURL: "https://assets.example.test/132.zip"}
	id := uuid.NewString()
	var wg sync.WaitGroup
	errs := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, e := svc.Publish(ctx, id, p, fmt.Sprintf("publisher-%d", i))
			errs <- e
		}(i)
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		require.NoError(t, e)
	}
	first, e := repo.Find(ctx, id)
	require.NoError(t, e)
	require.False(t, first.PublishedAt.IsZero())
	var count int64
	require.NoError(t, runtime.Table("browser_kernel.releases").Count(&count).Error)
	require.EqualValues(t, 1, count)
	for _, mutate := range []func(*k.Publish){func(v *k.Publish) { v.Version = "133.0.0.1" }, func(v *k.Publish) { v.DownloadURL = "https://assets.example.test/other.zip" }, func(v *k.Publish) { v.SHA256 = strings.Repeat("b", 64) }, func(v *k.Publish) { v.MinClientVersion = "1.1.0" }, func(v *k.Publish) { v.ArchiveFormat = "tar.gz" }} {
		v := p
		mutate(&v)
		_, e := svc.Publish(ctx, id, v, "publisher")
		require.ErrorIs(t, e, k.ErrConflict)
	}
	_, e = svc.Recommend(ctx, id, true, "recommender")
	require.NoError(t, e)
	recommended, e := repo.Find(ctx, id)
	require.NoError(t, e)
	_, e = svc.Recommend(ctx, id, true, "another")
	require.NoError(t, e)
	replayed, e := repo.Find(ctx, id)
	require.NoError(t, e)
	require.Equal(t, recommended, replayed)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				_, e := svc.Withdraw(ctx, id, "withdrawer")
				if e != nil {
					t.Error(e)
				}
			} else {
				_, e := svc.Recommend(ctx, id, true, "recommender")
				if e != nil && e != k.ErrWithdrawn {
					t.Error(e)
				}
			}
		}(i)
	}
	wg.Wait()
	saved, e := repo.Find(ctx, id)
	require.NoError(t, e)
	require.NotNil(t, saved.WithdrawnAt)
	require.False(t, saved.Recommended)
	require.Equal(t, first.PublishedAt, saved.PublishedAt)
	require.Equal(t, first.PublishedBy, saved.PublishedBy)
	v, e := svc.Publish(ctx, id, p, "retry-publisher")
	require.NoError(t, e)
	require.Equal(t, "withdrawn", v.State)
	_, e = svc.Withdraw(ctx, id, "retry-withdrawer")
	require.NoError(t, e)
	replayed, e = repo.Find(ctx, id)
	require.NoError(t, e)
	require.Equal(t, saved, replayed)
	q := k.Query{OS: "windows", Arch: "amd64", ClientVersion: "1.0.0"}
	_, e = svc.Artifact(ctx, id, q)
	require.ErrorIs(t, e, k.ErrNotFound)
	page, e := svc.List(ctx, q)
	require.NoError(t, e)
	require.Empty(t, page.Releases)
	reopened, e := New(open(runtimeDSN))
	require.NoError(t, e)
	again, e := reopened.Find(ctx, id)
	require.NoError(t, e)
	require.Equal(t, saved, again)
	for _, sql := range []string{"UPDATE browser_kernel.releases SET download_url='https://evil.test/a.zip'", "DELETE FROM browser_kernel.releases", "CREATE TABLE browser_kernel.forbidden(id text)", "TRUNCATE browser_kernel.releases"} {
		require.Error(t, runtime.Exec(sql).Error, sql)
	}
	expired, stop := context.WithCancel(ctx)
	stop()
	_, e = svc.Publish(expired, uuid.NewString(), p, "publisher")
	require.ErrorIs(t, e, k.ErrUnavailable)
	raw, _ := runtime.DB()
	require.NoError(t, raw.Close())
	_, e = svc.List(ctx, q)
	require.ErrorIs(t, e, k.ErrUnavailable)
}
