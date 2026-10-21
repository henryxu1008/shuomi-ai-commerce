package storecenter_test

import (
	"context"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/storecenter"
)

// Reuses the existing task-owned PostgreSQL fixture, not an environment DSN.
func TestSheinLoginPostgresPermissionsSavepointAndConcurrentFence(t *testing.T) {
	db := newTikTokStoreDatabase(t)
	ctx := context.Background()
	require.NoError(t, db.Exec(`CREATE ROLE shein_login_worker LOGIN PASSWORD 'synthetic-worker-only'`).Error)
	require.NoError(t, db.Exec(`REVOKE ALL ON DATABASE tiktok_schema FROM PUBLIC`).Error)
	require.NoError(t, db.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, storecenter.InstallSheinLoginCapability(ctx, db))
	// The ordinary Store schema continues to work with and without this optional
	// module; it does not require worker grants or consume its facts.
	require.NoError(t, storecenter.VerifyCurrentSchema(ctx, db))
	dsn := db.Dialector.(*postgres.Dialector).Config.DSN
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.User = url.UserPassword("shein_login_worker", "synthetic-worker-only")
	worker, e := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, e := worker.DB()
	require.NoError(t, e)
	t.Cleanup(func() { pool.Close() })
	require.NoError(t, storecenter.VerifySheinLoginRuntime(ctx, worker))
	repo, _ := storecenter.NewGormStoreRepository(db)
	makeStore := func(external string) *storecenter.Store {
		s, e := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "operator", Name: "Store", Platform: "shein", Region: "SG", ExternalStoreID: external, CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
		require.NoError(t, e)
		s, _, e = repo.CreateOrReplay(ctx, "org-a", s)
		require.NoError(t, e)
		return s
	}
	_ = makeStore("occupied")
	target := makeStore("")
	owner, e := storecenter.NewSheinLoginRepository(worker, sheinMember{"org-a"})
	require.NoError(t, e)
	scope := storecenter.SheinLoginScope{OrganizationID: "org-a", StoreID: target.ID(), InstallationID: uuid.NewString(), BindingRevision: uuid.NewString(), ConfigFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	command := storecenter.SheinBindingCommand{Scope: scope, ActorID: "operator", MemberID: "membership", RequestKey: uuid.NewString()}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	generations := make(chan int64, 8)
	for range 8 {
		wg.Go(func() { r, e := owner.Activate(ctx, command); errs <- e; generations <- r.Generation })
	}
	wg.Wait()
	close(errs)
	close(generations)
	for e := range errs {
		require.NoError(t, e)
	}
	for g := range generations {
		require.EqualValues(t, 1, g)
	}
	scope.Generation = 1
	attempt, nonce := uuid.NewString(), uuid.NewString()
	require.NoError(t, owner.Claim(ctx, scope, "operator", "membership", attempt, nonce))
	require.NoError(t, owner.Claim(ctx, scope, "operator", "membership", attempt, nonce), "same executor can reconcile a lost claim response")
	require.Error(t, owner.Claim(ctx, scope, "operator", "membership", attempt, uuid.NewString()), "different executor remains fenced")
	require.Error(t, owner.Claim(ctx, scope, "operator", "membership", uuid.NewString(), uuid.NewString()))
	sync := storecenter.SheinIdentityCommand{Scope: scope, ActorID: "operator", MemberID: "membership", AttemptID: attempt, OwnerNonce: nonce, ExpectedStoreVersion: 1, MerchantID: "occupied"}
	// A real unique violation aborts PostgreSQL until ROLLBACK TO SAVEPOINT.
	// The blocked receipt must commit in that same outer transaction.
	receipt, e := owner.RecordSheinLoginIdentity(ctx, sync)
	require.NoError(t, e)
	require.Equal(t, "blocked", receipt.Status)
	again, e := owner.RecordSheinLoginIdentity(ctx, sync)
	require.NoError(t, e)
	require.Equal(t, receipt, again)
	quiesce := command
	quiesce.Scope = scope
	quiesce.RequestKey = uuid.NewString()
	original, e := owner.Quiesce(ctx, quiesce)
	require.NoError(t, e)
	require.Equal(t, "quiescing", original.State)
	require.False(t, original.ClosedConfirmed)
	replayed, e := owner.Quiesce(ctx, quiesce)
	require.NoError(t, e)
	require.Equal(t, original.State, replayed.State)
	require.False(t, replayed.ClosedConfirmed)
	require.NoError(t, owner.ConfirmClosed(ctx, scope, attempt, nonce))
	replayed, e = owner.Quiesce(ctx, quiesce)
	require.NoError(t, e)
	require.Equal(t, original.State, replayed.State)
	require.False(t, replayed.ClosedConfirmed)
	quiesce.RequestKey = uuid.NewString()
	idle, e := owner.Quiesce(ctx, quiesce)
	require.NoError(t, e)
	require.Equal(t, "quiesced", idle.State)
	replayed, e = owner.Quiesce(ctx, quiesce)
	require.NoError(t, e)
	require.Equal(t, idle.State, replayed.State)
	require.True(t, replayed.ClosedConfirmed)
	var count int64
	require.NoError(t, db.Table("workbench_store_audit_logs").Where("request_key=? AND action=?", attempt, storecenter.AuditActionSheinIdentitySync).Count(&count).Error)
	require.EqualValues(t, 1, count)
	for _, q := range []string{`UPDATE workbench_stores SET connection_ref='forbidden'`, `UPDATE workbench_stores SET service_status='active'`, `DELETE FROM workbench_stores`, `UPDATE workbench_store_member_grants SET active=false`, `CREATE TABLE forbidden(id int)`} {
		require.Error(t, worker.Exec(q).Error)
	}
	// Unknown privilege additions remain rejected, including column-only grants.
	require.NoError(t, db.Exec(`GRANT UPDATE(connection_ref) ON workbench_stores TO shein_login_worker`).Error)
	require.Error(t, storecenter.VerifySheinLoginRuntime(ctx, worker))
	require.NoError(t, db.Exec(`REVOKE UPDATE(connection_ref) ON workbench_stores FROM shein_login_worker`).Error)
	require.NoError(t, storecenter.VerifySheinLoginRuntime(ctx, worker))
	require.NoError(t, db.Exec(`ALTER TABLE workbench_store_shein_login_bindings DROP CONSTRAINT shein_login_state,ADD CONSTRAINT shein_login_state CHECK(true)`).Error)
	require.Error(t, storecenter.VerifySheinLoginRuntime(ctx, worker), "named but weakened fence must fail startup")
}
