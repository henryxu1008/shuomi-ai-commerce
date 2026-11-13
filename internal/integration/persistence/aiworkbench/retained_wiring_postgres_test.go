//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"database/sql"
	"errors"
	"os"
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

func retainedFixture(t *testing.T) (*gorm.DB, RetainedTarget) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
	c, err := tcpostgres.Run(ctx, "postgres:17-alpine", tcpostgres.WithDatabase("retained_fixture"), tcpostgres.WithUsername("fixture_admin"), tcpostgres.WithPassword("fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, c.Terminate(context.Background())) })
	dsn, err := c.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	raw, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	require.NoError(t, db.Exec(`CREATE ROLE product_agent_owner LOGIN; CREATE ROLE ai_workbench_runtime LOGIN; ALTER DATABASE retained_fixture OWNER TO product_agent_owner; REVOKE ALL ON DATABASE retained_fixture FROM PUBLIC; GRANT CONNECT ON DATABASE retained_fixture TO ai_workbench_runtime; SET ROLE product_agent_owner`).Error)
	old, err := os.ReadFile("testdata/chat_retained_old.sql")
	require.NoError(t, err)
	require.NoError(t, db.Exec(string(old)).Error)
	require.NoError(t, db.Exec(`GRANT USAGE ON SCHEMA ai_workbench TO ai_workbench_runtime;
GRANT SELECT, INSERT ON ALL TABLES IN SCHEMA ai_workbench TO ai_workbench_runtime;
GRANT UPDATE (title,favorite,lifecycle,metadata_revision,next_sequence,updated_at) ON ai_workbench.conversations TO ai_workbench_runtime;
GRANT UPDATE (state,assistant_message_id,proposal_id,terminal_digest,mode,committed_at) ON ai_workbench.commands TO ai_workbench_runtime;
GRANT UPDATE (state,error_code,finished_at) ON ai_workbench.task_action_receipts TO ai_workbench_runtime;
GRANT UPDATE (state,revision,consumed_occurrences,active_occurrence_id,updated_at,registration_state,rule_start_at,frequency,time_zone,once_at_utc,local_hour,local_minute,local_weekday,until_utc) ON ai_workbench.scheduled_plans TO ai_workbench_runtime;
GRANT UPDATE (state,business_task_id,updated_at,reason,reconciliation_digest) ON ai_workbench.scheduled_occurrences TO ai_workbench_runtime;
RESET ROLE`).Error)
	require.NoError(t, db.Exec(`INSERT INTO ai_workbench.conversations(id,organization_id,owner_user_id,title,lifecycle,metadata_revision,next_sequence,created_at,updated_at) VALUES ('00000000-0000-0000-0000-000000000001','fixture-org','fixture-user','preserve me','ACTIVE',1,1,now(),now())`).Error)
	var target RetainedTarget
	require.NoError(t, db.Raw(`SELECT current_database() AS database, coalesce(inet_server_addr()::text,'') AS address, coalesce(inet_server_port(),0) AS port, session_user AS session_user, (SELECT system_identifier::text FROM pg_control_system()) AS system_identifier, (SELECT oid::text FROM pg_database WHERE datname=current_database()) AS database_oid`).Scan(&target).Error)
	target.MaintenanceWindow = "isolated-fixture-window"
	target.BackupEvidence = "isolated-fixture-backup"
	target.NotAfter = time.Now().Add(time.Minute)
	target.FactsDigest, target.PrivilegesDigest, err = retainedDigests(db)
	require.NoError(t, err)
	return db, target
}

func TestRetainedRejectsDatabasePrivilegeDrift(t *testing.T) {
	db, target := retainedFixture(t)
	require.NoError(t, db.Exec(`CREATE ROLE unrelated_fixture LOGIN`).Error)
	for _, grant := range []string{
		"GRANT CREATE ON DATABASE retained_fixture TO ai_workbench_runtime",
		"GRANT TEMP ON DATABASE retained_fixture TO PUBLIC",
		"GRANT CONNECT ON DATABASE retained_fixture TO ai_workbench_runtime WITH GRANT OPTION",
		"GRANT CONNECT ON DATABASE retained_fixture TO unrelated_fixture",
	} {
		require.NoError(t, db.Exec(grant).Error)
		_, err := InspectRetainedWorkbench(context.Background(), db, target)
		require.Error(t, err, grant)
		require.NoError(t, db.Exec(`REVOKE ALL ON DATABASE retained_fixture FROM PUBLIC, ai_workbench_runtime, unrelated_fixture; GRANT CONNECT ON DATABASE retained_fixture TO ai_workbench_runtime`).Error)
	}
	require.NoError(t, db.Exec(`ALTER DATABASE retained_fixture OWNER TO fixture_admin`).Error)
	_, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err)
	require.NoError(t, db.Exec(`ALTER DATABASE retained_fixture OWNER TO product_agent_owner`).Error)
	require.NoError(t, db.Exec(`ALTER ROLE ai_workbench_runtime IN DATABASE retained_fixture SET default_transaction_read_only = on`).Error)
	_, err = InspectRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err)
	require.NoError(t, db.Exec(`ALTER ROLE ai_workbench_runtime IN DATABASE retained_fixture RESET default_transaction_read_only`).Error)
}

func TestRetainedWiringRejectsExpiredWindow(t *testing.T) {
	db, target := retainedFixture(t)
	target.NotAfter = time.Now().Add(-time.Second)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	require.ErrorContains(t, err, "maintenance window")
}

func TestCaptureRetainedBaselineReadsOnlyRecognizedOldTarget(t *testing.T) {
	db, target := retainedFixture(t)
	wantFacts, wantACL := target.FactsDigest, target.PrivilegesDigest
	target.FactsDigest = ""
	target.PrivilegesDigest = ""
	facts, acl, err := CaptureRetainedBaseline(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, wantFacts, facts)
	require.Equal(t, wantACL, acl)
	target.DatabaseOID = "0"
	_, _, err = CaptureRetainedBaseline(context.Background(), db, target)
	require.Error(t, err)
}

func TestRetainedWiringDoesNotWriteAfterWindowExpiresWaitingForLock(t *testing.T) {
	db, target := retainedFixture(t)
	locked := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(`LOCK TABLE ai_workbench.conversations IN ACCESS EXCLUSIVE MODE`).Error; err != nil {
				return err
			}
			close(locked)
			<-release
			return nil
		})
	}()
	<-locked
	target.NotAfter = time.Now().Add(300 * time.Millisecond)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	close(release)
	require.NoError(t, <-done)
	require.Error(t, err)
	state, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedOld, state)
}

func TestRetainedWiringPreservesFactsAndSerializes(t *testing.T) {
	db, target := retainedFixture(t)
	ctx := context.Background()
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := prepareRetainedWorkbench(ctx, db, target); errs <- err }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var title string
	require.NoError(t, db.Raw(`SELECT title FROM ai_workbench.conversations WHERE id='00000000-0000-0000-0000-000000000001'`).Scan(&title).Error)
	require.Equal(t, "preserve me", title)
	state, err := InspectRetainedWorkbench(ctx, db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedReady, state)
}

func TestRetainedWiringRejectsWrongTargetAndPrivilegeDrift(t *testing.T) {
	db, target := retainedFixture(t)
	bad := target
	bad.DatabaseOID = "0"
	_, err := prepareRetainedWorkbench(context.Background(), db, bad)
	require.Error(t, err)
	require.NoError(t, db.Exec(`GRANT UPDATE ON ai_workbench.messages TO ai_workbench_runtime`).Error)
	_, err = prepareRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err)
	var exists bool
	require.NoError(t, db.Raw(`SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='ai_workbench' AND table_name='messages' AND column_name='message_kind')`).Scan(&exists).Error)
	require.False(t, exists)
}

func TestRetainedWiringRejectsConstraintDrift(t *testing.T) {
	db, target := retainedFixture(t)
	require.NoError(t, db.Exec(`ALTER TABLE ai_workbench.commands DROP CONSTRAINT commands_state_check`).Error)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err)
}

func TestRetainedWiringRejectsColumnReferencePrivilege(t *testing.T) {
	db, target := retainedFixture(t)
	require.NoError(t, db.Exec(`GRANT REFERENCES (id) ON ai_workbench.messages TO ai_workbench_runtime`).Error)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err, "an added column privilege is not part of the approved grant")
}

func TestRetainedWiringRejectsInFlightCommand(t *testing.T) {
	db, target := retainedFixture(t)
	require.NoError(t, db.Exec(`INSERT INTO ai_workbench.messages(id,organization_id,owner_user_id,conversation_id,sequence,author_kind,content,created_at) VALUES ('00000000-0000-0000-0000-000000000002','fixture-org','fixture-user','00000000-0000-0000-0000-000000000001',1,'USER','fixture',now());
INSERT INTO ai_workbench.commands(organization_id,actor_id,idempotency_key,operation,request_fingerprint,conversation_id,state,user_message_id,source_sequence,planner_invocation_id,member_id,planner_input_hash,work_scope,planner_model_profile,planner_started_at,planner_deadline,created_at)
VALUES ('fixture-org','fixture-user','00000000-0000-0000-0000-000000000003','chat_message_plan',repeat('a',64),'00000000-0000-0000-0000-000000000001','READY_TO_DISPATCH','00000000-0000-0000-0000-000000000002',1,repeat('b',64),'member',repeat('c',64),'{}','{}',now(),now(),now())`).Error)
	var digestErr error
	target.FactsDigest, target.PrivilegesDigest, digestErr = retainedDigests(db)
	require.NoError(t, digestErr)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	require.ErrorContains(t, err, "in-flight")
	state, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedOld, state)
}

func TestRetainedInspectionRejectsChangedFactsAfterCommit(t *testing.T) {
	db, target := retainedFixture(t)
	_, err := prepareRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE ai_workbench.conversations SET title='drift'`).Error)
	_, err = InspectRetainedWorkbench(context.Background(), db, target)
	require.ErrorContains(t, err, "facts or privilege snapshot drift")
}

func TestRetainedInspectionRejectsChangedNewFieldsOnOldRows(t *testing.T) {
	db, target := retainedFixture(t)
	require.NoError(t, db.Exec(`INSERT INTO ai_workbench.messages(id,organization_id,owner_user_id,conversation_id,sequence,author_kind,content,created_at) VALUES ('00000000-0000-0000-0000-000000000002','fixture-org','fixture-user','00000000-0000-0000-0000-000000000001',1,'ASSISTANT','店铺概况',now())`).Error)
	var err error
	target.FactsDigest, target.PrivilegesDigest, err = retainedDigests(db)
	require.NoError(t, err)
	_, err = prepareRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`UPDATE ai_workbench.messages SET message_kind='STORE_OVERVIEW_REF',store_query_status='ACTIVE'`).Error)
	_, err = InspectRetainedWorkbench(context.Background(), db, target)
	require.Error(t, err, "old messages cannot silently acquire a different meaning")
}

type retainedFaultPool struct{ lostCommitPool }

func (p retainedFaultPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &retainedFaultTx{Tx: tx}, nil
}

type retainedFaultTx struct{ *sql.Tx }

func (tx *retainedFaultTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := tx.Tx.ExecContext(ctx, query, args...)
	if err == nil && strings.HasPrefix(query, "ALTER TABLE ai_workbench.messages ADD COLUMN") {
		return result, errors.New("synthetic failure after DDL")
	}
	return result, err
}

type retainedSlowPool struct{ lostCommitPool }

func (p retainedSlowPool) BeginTx(ctx context.Context, opts *sql.TxOptions) (gorm.ConnPool, error) {
	tx, err := p.conn.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &retainedSlowTx{Tx: tx}, nil
}

type retainedSlowTx struct {
	*sql.Tx
	snapshots int
}

func (tx *retainedSlowTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if strings.Contains(query, "WITH p AS") {
		tx.snapshots++
		if tx.snapshots == 2 {
			time.Sleep(400 * time.Millisecond)
		}
	}
	return tx.Tx.QueryContext(ctx, query, args...)
}

func TestRetainedWiringDoesNotCommitIfFinalSnapshotOutlivesWindow(t *testing.T) {
	db, target := retainedFixture(t)
	target.NotAfter = time.Now().Add(300 * time.Millisecond)
	err := db.Connection(func(conn *gorm.DB) error {
		physical, ok := conn.Statement.ConnPool.(*sql.Conn)
		require.True(t, ok)
		conn.Statement.ConnPool = retainedSlowPool{lostCommitPool{conn: physical}}
		_, err := prepareRetainedOnConnection(conn, target)
		return err
	})
	require.Error(t, err)
	state, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedOld, state)
}

func TestRetainedWiringRollsBackAfterDDLFailure(t *testing.T) {
	db, target := retainedFixture(t)
	err := db.Connection(func(conn *gorm.DB) error {
		physical, ok := conn.Statement.ConnPool.(*sql.Conn)
		require.True(t, ok)
		conn.Statement.ConnPool = retainedFaultPool{lostCommitPool{conn: physical}}
		_, err := prepareRetainedOnConnection(conn, target)
		return err
	})
	require.ErrorContains(t, err, "synthetic failure")
	state, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedOld, state)
}

func TestRetainedWiringLostCommitAcknowledgmentIsInspected(t *testing.T) {
	db, target := retainedFixture(t)
	var commits atomic.Int32
	err := db.Connection(func(conn *gorm.DB) error {
		physical, ok := conn.Statement.ConnPool.(*sql.Conn)
		require.True(t, ok)
		conn.Statement.ConnPool = lostCommitPool{conn: physical, commits: &commits}
		_, err := prepareRetainedOnConnection(conn, target)
		return err
	})
	require.ErrorContains(t, err, "lost commit")
	require.EqualValues(t, 1, commits.Load())
	state, err := InspectRetainedWorkbench(context.Background(), db, target)
	require.NoError(t, err)
	require.Equal(t, RetainedReady, state)
}
