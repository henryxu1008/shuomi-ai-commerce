package toolmarketpersistence

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
	tm "task-processor/internal/toolmarket"
	"testing"
)

func fixture(t *testing.T) (*gorm.DB, *Store) {
	t.Helper()
	dsn := os.Getenv("TOOLMARKET_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated TOOLMARKET_TEST_DSN")
	}
	root, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	name := "toolmarket_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	require.NoError(t, InstallSchema(db))
	s, e := New(db)
	require.NoError(t, e)
	return db, s
}

var allow = func(context.Context) error { return nil }

func create() tm.Command {
	return tm.Command{Scope: tm.Scope{OrganizationID: "org-a", ActorID: "admin-a"}, Key: uuid.NewString(), Operation: "create", Absent: true, Demand: tm.Demand{Kind: "DATA", Title: "商品需求", Description: "保存商品资料", Scenario: "商品管理", ContactName: "测试联系人", ContactMethod: "test-wechat", Consent: true}}
}

func TestCompleteDemandFilesAreAtomicPrivateAndUsableByRuntimeRole(t *testing.T) {
	db, owner := fixture(t)
	require.NoError(t, db.Exec("DO $$ BEGIN IF NOT EXISTS(SELECT FROM pg_roles WHERE rolname='tool_market_runtime') THEN CREATE ROLE tool_market_runtime LOGIN PASSWORD 'isolated-test-only'; END IF; END $$").Error)
	// Exercise the actual fresh-install serving grants, so this test cannot hide
	// a missing bootstrap permission behind a richer hand-written test role.
	bootstrap, err := os.ReadFile("../../../../deployments/docker/account-compose/native-modules-init.sh")
	require.NoError(t, err)
	_, block, ok := strings.Cut(strings.ReplaceAll(string(bootstrap), "\r\n", "\n"), "psql -X -v ON_ERROR_STOP=1 -d tool_market <<'SQL'\n")
	require.True(t, ok)
	block, _, ok = strings.Cut(block, "\nSQL")
	require.True(t, ok)
	grants := 0
	for _, line := range strings.Split(block, "\n") {
		if strings.HasPrefix(line, "GRANT ") {
			require.NoError(t, db.Exec(line).Error)
			grants++
		}
	}
	require.Positive(t, grants)
	var database string
	require.NoError(t, db.Raw("SELECT current_database()").Scan(&database).Error)
	u, err := url.Parse(os.Getenv("TOOLMARKET_TEST_DSN"))
	require.NoError(t, err)
	u.Path = "/" + database
	u.User = url.UserPassword("tool_market_runtime", "isolated-test-only")
	runtimeDB, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	sqlDB, err := runtimeDB.DB()
	require.NoError(t, err)
	defer sqlDB.Close()
	runtime, err := New(runtimeDB)
	require.NoError(t, err)
	ctx := context.Background()
	require.NoError(t, VerifySchema(ctx, runtimeDB))
	c := create()
	c.Demand.Files = []tm.Upload{{Name: "流程说明.txt", Data: []byte("授权采集后保存商品资料")}}
	receipt, err := runtime.Execute(ctx, c, allow)
	require.NoError(t, err)
	detail, err := owner.Detail(ctx, c.Scope, false, receipt.ID, "")
	require.NoError(t, err)
	require.Equal(t, c.Demand.Scenario, detail.Request.Scenario)
	require.Equal(t, c.Demand.ContactName, detail.Request.ContactName)
	require.True(t, detail.Request.Consent)
	require.Equal(t, tm.ConsentVersion, detail.Request.ConsentVersion)
	require.Equal(t, c.Scope.ActorID, detail.Request.CreatedBy)
	require.Empty(t, detail.Request.Files)
	require.Len(t, detail.Request.Attachments, 1)
	file := detail.Request.Attachments[0]
	meta, bytes, err := runtime.Download(ctx, c.Scope, false, receipt.ID, file.ID)
	require.NoError(t, err)
	require.Equal(t, c.Demand.Files[0].Data, bytes)
	require.Equal(t, file, meta)
	_, _, err = runtime.Download(ctx, tm.Scope{ActorID: "other", OrganizationID: "org-b"}, false, receipt.ID, file.ID)
	require.ErrorIs(t, err, tm.ErrNotFound)
	other, err := runtime.Execute(ctx, create(), allow)
	require.NoError(t, err)
	_, _, err = runtime.Download(ctx, c.Scope, false, other.ID, file.ID)
	require.ErrorIs(t, err, tm.ErrNotFound)
	_, _, err = runtime.Download(ctx, tm.Scope{ActorID: "specialist"}, true, receipt.ID, file.ID)
	require.NoError(t, err)
	replay, err := runtime.Execute(ctx, c, allow)
	require.NoError(t, err)
	require.Equal(t, receipt, replay)
	changed := c
	changed.Demand.ContactMethod = "another"
	_, err = runtime.Execute(ctx, changed, allow)
	require.ErrorIs(t, err, tm.ErrConflict)
	changed = c
	changed.Demand.Files = []tm.Upload{{Name: "流程说明.txt", Data: []byte("changed")}}
	_, err = runtime.Execute(ctx, changed, allow)
	require.ErrorIs(t, err, tm.ErrConflict)
	rollback := create()
	rollback.Demand.Files = c.Demand.Files
	n := 0
	_, err = runtime.Execute(ctx, rollback, func(context.Context) error {
		n++
		if n == 2 {
			return tm.ErrForbidden
		}
		return nil
	})
	require.ErrorIs(t, err, tm.ErrForbidden)
	for _, table := range []string{"requests", "commands", "events"} {
		var count int64
		require.NoError(t, db.Table("tool_market."+table).Count(&count).Error)
		require.EqualValues(t, 2, count)
	}
	var count int64
	require.NoError(t, db.Table("tool_market.attachments").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.Error(t, runtimeDB.Exec("DELETE FROM tool_market.attachments").Error, "runtime must not delete private reference facts")
}
func TestAtomicConcurrentReplayAndTenantIsolation(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	c := create()
	var wg sync.WaitGroup
	receipts := make([]tm.Receipt, 8)
	errs := make([]error, 8)
	for i := range receipts {
		wg.Add(1)
		go func(i int) { defer wg.Done(); receipts[i], errs[i] = s.Execute(ctx, c, allow) }(i)
	}
	wg.Wait()
	for i := range receipts {
		require.NoError(t, errs[i])
		require.Equal(t, receipts[0], receipts[i])
	}
	var n int64
	require.NoError(t, db.Table("tool_market.requests").Count(&n).Error)
	require.EqualValues(t, 1, n)
	_, e := s.Detail(ctx, tm.Scope{ActorID: "other", OrganizationID: "org-b"}, false, receipts[0].ID, "")
	require.ErrorIs(t, e, tm.ErrNotFound)
	c.Demand.Title = "different"
	_, e = s.Execute(ctx, c, allow)
	require.ErrorIs(t, e, tm.ErrConflict)
	c.Demand.Title = "商品需求"
	_, e = s.Execute(ctx, c, func(context.Context) error { return tm.ErrForbidden })
	require.ErrorIs(t, e, tm.ErrForbidden)
	// Independent connection/store after a lost response must recover the same fact.
	raw, _ := db.DB()
	second, e := New(db.Session(&gorm.Session{NewDB: true}))
	require.NoError(t, e)
	require.NotNil(t, raw)
	replay, e := second.Execute(ctx, c, allow)
	require.NoError(t, e)
	require.Equal(t, receipts[0], replay)
}
func TestConcurrentProgressAndRollback(t *testing.T) {
	db, s := fixture(t)
	ctx := context.Background()
	r, e := s.Execute(ctx, create(), allow)
	require.NoError(t, e)
	p := tm.Command{Scope: tm.Scope{ActorID: "specialist"}, Platform: true, Key: uuid.NewString(), Operation: "progress", ID: r.ID, Expected: 1, Progress: tm.Progress{Stage: "EVALUATING", Note: "正在人工评估"}}
	var wg sync.WaitGroup
	errs := make([]error, 2)
	for i := range errs {
		wg.Add(1)
		go func(i int) { defer wg.Done(); c := p; c.Key = uuid.NewString(); _, errs[i] = s.Execute(ctx, c, allow) }(i)
	}
	wg.Wait()
	if errs[0] == nil {
		require.ErrorIs(t, errs[1], tm.ErrRevision)
	} else {
		require.ErrorIs(t, errs[0], tm.ErrRevision)
		require.NoError(t, errs[1])
	}
	p.Expected = 2
	p.Progress = tm.Progress{Stage: "PLAN_CONFIRMED", Note: "确认方案"}
	calls := 0
	_, e = s.Execute(ctx, p, func(context.Context) error {
		calls++
		if calls == 2 {
			return tm.ErrForbidden
		}
		return nil
	})
	require.ErrorIs(t, e, tm.ErrForbidden)
	detail, e := s.Detail(ctx, tm.Scope{ActorID: "member", OrganizationID: "org-a"}, false, r.ID, "")
	require.NoError(t, e)
	require.Equal(t, "2", detail.Request.Revision)
	require.Len(t, detail.Events, 2)
	// Inject a database failure after the state write: neither state nor receipt survives.
	require.NoError(t, db.Exec(`CREATE FUNCTION tool_market.reject_event() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test fault'; END $$; CREATE TRIGGER reject_event BEFORE INSERT ON tool_market.events FOR EACH ROW EXECUTE FUNCTION tool_market.reject_event();`).Error)
	_, e = s.Execute(ctx, p, allow)
	require.Error(t, e)
	detail, e = s.Detail(ctx, p.Scope, true, r.ID, "")
	require.NoError(t, e)
	require.Equal(t, "2", detail.Request.Revision)
	require.NoError(t, db.Exec("DROP TRIGGER reject_event ON tool_market.events").Error)
	rr, e := s.Execute(ctx, p, allow)
	require.NoError(t, e)
	require.Equal(t, "3", rr.Revision)
	p.Progress = tm.Progress{Stage: "DELIVERED", Note: "skip"}
	p.Expected = 3
	p.Key = uuid.NewString()
	_, e = s.Execute(ctx, p, allow)
	require.ErrorIs(t, e, tm.ErrInvalid)
}
func TestFirstActivationCannotOverwriteAndNeedsGuard(t *testing.T) {
	_, s := fixture(t)
	ctx := context.Background()
	c := tm.Command{Scope: tm.Scope{ActorID: "a", OrganizationID: "org-a"}, Key: uuid.NewString(), Operation: "activation", ID: tm.AcquisitionID, Absent: true, Enabled: true}
	_, e := s.Execute(ctx, c, nil)
	require.ErrorIs(t, e, tm.ErrForbidden)
	r, e := s.Execute(ctx, c, allow)
	require.NoError(t, e)
	require.Equal(t, "1", r.Revision)
	c.Key = uuid.NewString()
	_, e = s.Execute(ctx, c, allow)
	require.ErrorIs(t, e, tm.ErrRevision)
	c.Absent = false
	c.Expected = 1
	c.Enabled = false
	r, e = s.Execute(ctx, c, allow)
	require.NoError(t, e)
	require.Equal(t, "2", r.Revision)
	rows, e := s.Activations(ctx, c.Scope)
	require.NoError(t, e)
	require.Len(t, rows, 1)
	require.False(t, rows[0].Enabled)
}
