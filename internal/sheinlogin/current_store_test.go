package sheinlogin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"task-processor/internal/storecenter"
)

func fixtureRedis(t *testing.T) *RedisStore {
	t.Helper()
	m := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: m.Addr()})
	t.Cleanup(func() { client.Close() })
	s, e := NewRedisStoreFromClient(client, uuid.NewString())
	if e != nil {
		t.Fatal(e)
	}
	return s
}
func fixtureAttempt() LoginAttempt {
	return LoginAttempt{ID: uuid.NewString(), Scope: storecenter.SheinLoginScope{OrganizationID: "org-a", StoreID: uuid.NewString(), InstallationID: "00000000-0000-4000-8000-000000000001", BindingRevision: uuid.NewString(), ConfigFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Generation: 1}, ActorID: "actor", MemberID: "member", Status: LoginAttemptQueued, SyncStatus: "not_requested", CreatedAt: time.Now().UTC()}
}
func TestCurrentRedisAttemptIsolationAndCancelWins(t *testing.T) {
	s := fixtureRedis(t)
	a := fixtureAttempt()
	a.Scope.InstallationID = s.installation
	got, e := s.Enqueue(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	a = *got
	replay, e := s.Enqueue(context.Background(), a)
	if e != nil || replay.ID != a.ID {
		t.Fatalf("replay=%+v %v", replay, e)
	}
	altered := a
	altered.ForceLogin = true
	if _, e = s.Enqueue(context.Background(), altered); !errors.Is(e, storecenter.ErrIdempotencyConflict) {
		t.Fatalf("payload=%v", e)
	}
	other := a
	other.ID = uuid.NewString()
	other.Scope.OrganizationID = "org-b"
	if _, e = s.Enqueue(context.Background(), other); e != nil {
		t.Fatal(e)
	}
	if e = s.Cancel(context.Background(), a); e != nil {
		t.Fatal(e)
	}
	success := a
	success.Status = LoginAttemptSucceeded
	success.SyncStatus = "pending"
	if e = s.CompareAndSwap(context.Background(), a, success, map[string]any{"cookies": []any{map[string]any{"name": "s", "value": "private"}}}); e == nil {
		t.Fatal("late success overwrote cancel")
	}
	if _, ok, e := s.Cookie(context.Background(), a.Scope); e != nil || ok {
		t.Fatal("late cookie persisted")
	}
	final, e := s.Load(context.Background(), a.ID)
	if e != nil || final.Status != LoginAttemptCancelled {
		t.Fatalf("final=%+v %v", final, e)
	}
}
func TestCurrentRedisSuccessAtomicAndFixedSyncPayload(t *testing.T) {
	s := fixtureRedis(t)
	a := fixtureAttempt()
	a.Scope.InstallationID = s.installation
	got, e := s.Enqueue(context.Background(), a)
	if e != nil {
		t.Fatal(e)
	}
	a = *got
	launching := a
	launching.Status = LoginAttemptLaunching
	if e = s.CompareAndSwap(context.Background(), a, launching, nil); e != nil {
		t.Fatal(e)
	}
	a = launching
	next := a
	next.Status = LoginAttemptSucceeded
	next.SyncStatus = "pending"
	if e = s.CompareAndSwap(context.Background(), a, next, map[string]any{"cookies": []any{map[string]any{"name": "s", "value": "private"}}, "origins": []any{"must disappear"}}); e != nil {
		t.Fatal(e)
	}
	state, ok, e := s.Cookie(context.Background(), a.Scope)
	if e != nil || !ok || state["origins"] != nil {
		t.Fatalf("cookie=%+v %v", state, e)
	}
	forged := a.Scope
	forged.Generation++
	if _, ok, e = s.Cookie(context.Background(), forged); e != nil || ok {
		t.Fatalf("generation read=%v %v", ok, e)
	}
	fixed := next
	fixed.MerchantID = "m"
	fixed.ExpectedStoreVersion = 2
	if e = s.CompareAndSwap(context.Background(), next, fixed, nil); e != nil {
		t.Fatal(e)
	}
	rewritten := fixed
	rewritten.MerchantID = "different"
	if e = s.CompareAndSwap(context.Background(), fixed, rewritten, nil); e == nil {
		t.Fatal("sync payload rewritten")
	}
}

func TestAttemptStateCannotSkipOrRegressBrowserExecution(t *testing.T) {
	a := fixtureAttempt()
	next := a
	next.Status = LoginAttemptSucceeded
	next.SyncStatus = "pending"
	if validTransition(a, next, syntheticState()) {
		t.Fatal("queued attempt skipped executor claim/launch")
	}
	a.Status = LoginAttemptWaitingVerifyCode
	next = a
	next.Status = LoginAttemptLaunching
	if validTransition(a, next, nil) {
		t.Fatal("verification regressed to launch")
	}
}
