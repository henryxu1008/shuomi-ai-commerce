//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type lostCommitPool struct {
	conn    *sql.Conn
	commits *atomic.Int32
}

func (p lostCommitPool) PrepareContext(c context.Context, q string) (*sql.Stmt, error) {
	return p.conn.PrepareContext(c, q)
}
func (p lostCommitPool) ExecContext(c context.Context, q string, a ...any) (sql.Result, error) {
	return p.conn.ExecContext(c, q, a...)
}
func (p lostCommitPool) QueryContext(c context.Context, q string, a ...any) (*sql.Rows, error) {
	return p.conn.QueryContext(c, q, a...)
}
func (p lostCommitPool) QueryRowContext(c context.Context, q string, a ...any) *sql.Row {
	return p.conn.QueryRowContext(c, q, a...)
}
func (p lostCommitPool) BeginTx(c context.Context, o *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.conn.BeginTx(c, o)
	if err != nil {
		return nil, err
	}
	return &lostCommitTx{Tx: tx, commits: p.commits}, nil
}

type lostCommitTx struct {
	*sql.Tx
	commits *atomic.Int32
}

func (t lostCommitTx) Commit() error {
	t.commits.Add(1)
	if err := t.Tx.Commit(); err != nil {
		return err
	}
	return errors.New("synthetic lost commit acknowledgment")
}

func emptyScopeFixture(t *testing.T) (*gorm.DB, *gorm.DB, EmptyScopeTarget) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("chat_fixture"), tcpostgres.WithUsername("test_admin"), tcpostgres.WithPassword("fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	open := func(dsn string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		raw, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = raw.Close() })
		return db
	}
	admin := open(dsn)
	require.NoError(t, admin.Exec(`CREATE ROLE product_agent_owner LOGIN PASSWORD 'fixture'; CREATE ROLE ai_workbench_runtime LOGIN PASSWORD 'fixture'; GRANT CREATE ON DATABASE chat_fixture TO product_agent_owner`).Error)
	urlDSN, err := url.Parse(dsn)
	require.NoError(t, err)
	urlDSN.User = url.UserPassword("product_agent_owner", "fixture")
	owner := open(urlDSN.String())
	require.NoError(t, InstallSchema(owner))
	require.NoError(t, GrantRuntime(owner, "ai_workbench_runtime"))
	require.NoError(t, owner.Exec(`ALTER TABLE ai_workbench.conversations DROP COLUMN business_scope; ALTER TABLE ai_workbench.commands DROP COLUMN frozen_scope, DROP COLUMN scope_digest; ALTER TABLE ai_workbench.execution_proposals DROP COLUMN workbench_context_ref`).Error)
	urlDSN.User = url.UserPassword("ai_workbench_runtime", "fixture")
	runtime := open(urlDSN.String())
	var target EmptyScopeTarget
	require.NoError(t, owner.Raw(`SELECT current_database() AS database, inet_server_addr()::text AS address, inet_server_port() AS port`).Scan(&target).Error)
	return owner, runtime, target
}

func TestEmptyScopeInstallerIsAtomicAndRepeatable(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	require.Error(t, VerifySchema(context.Background(), runtime))
	require.NoError(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.NoError(t, VerifySchema(context.Background(), runtime))
	require.NoError(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.NoError(t, owner.Exec(`INSERT INTO ai_workbench.conversations(id,organization_id,owner_user_id,lifecycle,metadata_revision,next_sequence,created_at,updated_at) VALUES ('00000000-0000-0000-0000-000000000003','org','owner','ACTIVE',1,1,now(),now())`).Error)
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target), "even a current schema must not pass the empty preparation mode with existing Chat facts")
}

func TestEmptyScopeInstallerRejectsNonemptyOrWrongTarget(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	require.NoError(t, owner.Exec(`INSERT INTO ai_workbench.conversations(id,organization_id,owner_user_id,lifecycle,metadata_revision,next_sequence,created_at,updated_at) VALUES ('00000000-0000-0000-0000-000000000001','org','owner','ACTIVE',1,1,now(),now())`).Error)
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	var count int64
	require.NoError(t, owner.Table("ai_workbench.conversations").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, owner.Exec(`DELETE FROM ai_workbench.conversations`).Error)
	target.Database = "wrong"
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.Error(t, VerifySchema(context.Background(), runtime))
}

func TestEmptyScopeInstallerRejectsSchemaOrPrivilegeDrift(t *testing.T) {
	for _, tc := range []struct{ name, drift string }{
		{"extra_column", `ALTER TABLE ai_workbench.commands ADD COLUMN unexpected text`},
		{"public_grant", `GRANT SELECT ON ai_workbench.messages TO PUBLIC`},
		{"runtime_broad_update", `GRANT UPDATE ON ai_workbench.messages TO ai_workbench_runtime`},
		{"unexpected_object", `CREATE TABLE ai_workbench.unexpected(id integer)`},
		{"rewrite_rule", `CREATE RULE unexpected AS ON INSERT TO ai_workbench.messages DO INSTEAD NOTHING`},
		{"changed_collation", `ALTER TABLE ai_workbench.commands ALTER COLUMN organization_id TYPE text COLLATE "C"`},
		{"external_function", `CREATE SCHEMA other; CREATE FUNCTION other.dependent(ai_workbench.commands) RETURNS integer LANGUAGE SQL AS 'SELECT 1'`},
		{"external_column", `CREATE SCHEMA other; CREATE TABLE other.dependent(row ai_workbench.commands)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner, runtime, target := emptyScopeFixture(t)
			require.NoError(t, owner.Exec(tc.drift).Error)
			require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
			require.Error(t, VerifySchema(context.Background(), runtime))
		})
	}
}

func TestEmptyScopeInstallerRejectsDisabledForeignKeyTrigger(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	u, err := url.Parse(owner.Dialector.(*postgres.Dialector).DSN)
	require.NoError(t, err)
	u.User = url.UserPassword("test_admin", "fixture")
	admin, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, err := admin.DB()
	require.NoError(t, err)
	defer raw.Close()
	require.NoError(t, admin.Exec(`ALTER TABLE ai_workbench.messages DISABLE TRIGGER ALL`).Error)
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.Error(t, VerifySchema(context.Background(), runtime))
}

func TestEmptyScopeInstallerSerializesConcurrentAttempts(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- InstallEmptyScope(context.Background(), owner, runtime, target) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.NoError(t, VerifySchema(context.Background(), runtime))
}

func TestEmptyScopeInstallerLockTimeoutDoesNotInstall(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	holder := owner.Begin()
	require.NoError(t, holder.Error)
	require.NoError(t, holder.Exec(`LOCK TABLE ai_workbench.commands IN ACCESS EXCLUSIVE MODE`).Error)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	require.Error(t, InstallEmptyScope(ctx, owner, runtime, target))
	require.NoError(t, holder.Rollback().Error)
	require.Error(t, VerifySchema(context.Background(), runtime))
}

func TestEmptyScopeInstallerReportsPostCommitRuntimeFailure(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	require.NoError(t, owner.Exec(`CREATE SCHEMA unrelated; CREATE TABLE unrelated.sentinel(value text); INSERT INTO unrelated.sentinel VALUES ('preserve')`).Error)
	require.NoError(t, runtime.Callback().Row().Before("gorm:row").Register("fixture:reject-runtime-verification", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "SELECT current_user") {
			tx.AddError(errors.New("fixture runtime verification failed"))
		}
	}))
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.NoError(t, runtime.Callback().Row().Remove("fixture:reject-runtime-verification"))
	require.NoError(t, VerifySchema(context.Background(), runtime), "committed columns stay in place after failed post-commit verification")
	var sentinel string
	require.NoError(t, owner.Raw(`SELECT value FROM unrelated.sentinel`).Scan(&sentinel).Error)
	require.Equal(t, "preserve", sentinel)
}

func TestEmptyScopeInstallerRollsBackWhenFinalCheckFails(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	var checks atomic.Int32
	require.NoError(t, owner.Callback().Row().Before("gorm:row").Register("fixture:fail-after-ddl", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "WITH facts AS") && checks.Add(1) == 2 {
			tx.AddError(errors.New("fixture final verification failed"))
		}
	}))
	require.Error(t, InstallEmptyScope(context.Background(), owner, runtime, target))
	require.NoError(t, owner.Callback().Row().Remove("fixture:fail-after-ddl"))
	require.Error(t, VerifySchema(context.Background(), runtime))
	fingerprint, err := emptyScopeFingerprint(owner)
	require.NoError(t, err)
	require.Equal(t, emptyScopeOldFingerprint, fingerprint)
}

func TestEmptyScopeInstallerWaitsForWriterThenRefusesNonempty(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	writer := owner.Begin()
	require.NoError(t, writer.Error)
	require.NoError(t, writer.Exec(`INSERT INTO ai_workbench.conversations(id,organization_id,owner_user_id,lifecycle,metadata_revision,next_sequence,created_at,updated_at) VALUES ('00000000-0000-0000-0000-000000000002','org','owner','ACTIVE',1,1,now(),now())`).Error)
	result := make(chan error, 1)
	go func() { result <- InstallEmptyScope(context.Background(), owner, runtime, target) }()
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, writer.Commit().Error)
	require.Error(t, <-result)
	require.Error(t, VerifySchema(context.Background(), runtime))
	var count int64
	require.NoError(t, owner.Table("ai_workbench.conversations").Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestEmptyScopeInstallerCommitAcknowledgmentUnknownDoesNotRetry(t *testing.T) {
	owner, runtime, target := emptyScopeFixture(t)
	var commits atomic.Int32
	err := owner.Connection(func(o *gorm.DB) error {
		conn, ok := o.Statement.ConnPool.(*sql.Conn)
		require.True(t, ok)
		o.Statement.ConnPool = lostCommitPool{conn: conn, commits: &commits}
		return runtime.Connection(func(r *gorm.DB) error { return installEmptyScopeOnConnections(context.Background(), o, r, target) })
	})
	require.Error(t, err)
	require.EqualValues(t, 1, commits.Load(), "an uncertain acknowledgment never triggers a second commit")
	require.NoError(t, VerifySchema(context.Background(), runtime), "a lost acknowledgment can follow a successful commit")
}
