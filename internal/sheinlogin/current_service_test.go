package sheinlogin

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/core/config"
	"task-processor/internal/storecenter"
)

func privateFixture(t *testing.T, path string, data []byte) {
	t.Helper()
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	if goruntime.GOOS == "windows" {
		who, e := user.Current()
		if e != nil {
			t.Fatal(e)
		}
		if e = exec.Command("icacls", path, "/inheritance:r", "/grant:r", "*"+who.Uid+":F").Run(); e != nil {
			t.Fatal(e)
		}
	}
}

type liveMember struct{ revoked atomic.Bool }

func (a *liveMember) AuthorizeStoreMember(context.Context, string) (storecenter.StoreMemberAccess, error) {
	if a.revoked.Load() {
		return storecenter.StoreMemberAccess{}, ErrInvalidInput
	}
	return storecenter.StoreMemberAccess{OrganizationID: "123", ActorID: "actor", MemberID: "member", Administrator: true, CanWrite: true}, nil
}

type syntheticAutomation struct {
	calls   atomic.Int32
	waiting bool
	session VerifySession
}

func (a *syntheticAutomation) StartLogin(ctx context.Context, account Account, cfg AutomationConfig) (*AutomationResult, VerifySession, error) {
	a.calls.Add(1)
	if account.Password != "private-password" {
		return nil, nil, ErrInvalidInput
	}
	if a.waiting {
		return &AutomationResult{WaitingForVerifyCode: true}, a.session, nil
	}
	return &AutomationResult{BrowserState: syntheticState()}, a.session, nil
}
func syntheticState() map[string]any {
	return map[string]any{"cookies": []any{map[string]any{"name": "session", "value": "private-cookie"}}}
}

type syntheticReader struct {
	calls       atomic.Int32
	unavailable atomic.Bool
}

func (r *syntheticReader) Merchant(context.Context, Account, map[string]any) (string, error) {
	r.calls.Add(1)
	if r.unavailable.Load() {
		return "", ErrUnavailable
	}
	return "merchant", nil
}
func (r *syntheticReader) Warehouses(context.Context, Account, map[string]any) ([]WarehouseOption, error) {
	return []WarehouseOption{{WarehouseCode: "warehouse"}}, nil
}

type lostReceiptOwner struct {
	StoreOwner
	lost atomic.Int32
}

func (o *lostReceiptOwner) RecordSheinLoginIdentity(ctx context.Context, c storecenter.SheinIdentityCommand) (storecenter.SheinIdentityReceipt, error) {
	r, e := o.StoreOwner.RecordSheinLoginIdentity(ctx, c)
	if e == nil && o.lost.Add(-1) >= 0 {
		return storecenter.SheinIdentityReceipt{}, ErrUnavailable
	}
	return r, e
}
func currentService(t *testing.T) (*Service, *storecenter.SheinLoginRepository, *liveMember, *syntheticAutomation, *syntheticReader) {
	t.Helper()
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "store.sqlite")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		t.Fatal(e)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	if e = storecenter.AutoMigrateStoreRepository(db); e != nil {
		t.Fatal(e)
	}
	if e = storecenter.AutoMigrateSheinLoginForTests(db); e != nil {
		t.Fatal(e)
	}
	store, e := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "123", ActorSubject: "actor", Name: "Current", Platform: "shein", Region: "SG", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
	if e != nil {
		t.Fatal(e)
	}
	repo, _ := storecenter.NewGormStoreRepository(db)
	if _, _, e = repo.CreateOrReplay(context.Background(), "123", store); e != nil {
		t.Fatal(e)
	}
	live := &liveMember{}
	owner, _ := storecenter.NewSheinLoginRepository(db, live)
	redis := fixtureRedis(t)
	root := t.TempDir()
	credential := filepath.Join(root, "fresh-credential.json")
	privateFixture(t, credential, []byte(`{"username":"private-user","password":"private-password"}`))
	cfg := Config{InstallationID: redis.installation, StoreDatabase: config.DatabaseConfig{Host: "synthetic", Port: 5432, Database: "synthetic", User: "shein_login_worker"}, Redis: config.RedisConfig{Host: "synthetic", Port: 6379}, ProfileRoot: filepath.Join(root, "profiles"), ArtifactRoot: filepath.Join(root, "artifacts"), MaxConcurrentLogins: 2, BrowserPath: filepath.Join(root, "browser.exe"), DriverDirectory: filepath.Join(root, "driver"), Auth: AuthConfig{IssuerURL: "https://identity.example", AuthorizationAPIURL: "https://identity.example", ClientID: "client", ProjectID: "project", ClientSecretFile: filepath.Join(root, "client-secret"), ServiceTokenFile: filepath.Join(root, "service-token")}, Bindings: []Binding{{OrganizationID: "123", StoreID: store.ID(), Revision: uuid.NewString(), LoginURL: "https://sellerhub.shein.com", CredentialFile: credential, Headless: true}}}
	service, e := NewService(cfg, redis, owner)
	if e != nil {
		t.Fatal(e)
	}
	automation := &syntheticAutomation{}
	reader := &syntheticReader{}
	service.automation = automation
	service.identityReader = reader
	if _, e = service.Execute(context.Background(), Command{Action: "activate", OrganizationID: "123", StoreID: store.ID(), ActorID: "actor", MemberID: "member", RequestKey: uuid.NewString()}); e != nil {
		t.Fatal(e)
	}
	return service, owner, live, automation, reader
}
func enqueueCurrent(t *testing.T, s *Service) Command {
	t.Helper()
	c := Command{Action: "login", OrganizationID: "123", StoreID: s.config.Bindings[0].StoreID, ActorID: "actor", MemberID: "member", RequestKey: uuid.NewString()}
	if _, e := s.Execute(context.Background(), c); e != nil {
		t.Fatal(e)
	}
	return c
}
func TestCurrentWorkerResponseLossResumesOnlyFixedSync(t *testing.T) {
	s, owner, _, automation, reader := currentService(t)
	lost := &lostReceiptOwner{StoreOwner: owner}
	lost.lost.Store(2)
	s.owner = lost
	c := enqueueCurrent(t, s)
	if e := s.processAttempt(context.Background(), c.RequestKey); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
	attempt, e := s.store.Load(context.Background(), c.RequestKey)
	if e != nil || attempt.Status != LoginAttemptSucceeded || attempt.SyncStatus != "unavailable" || attempt.MerchantID != "merchant" || attempt.ExpectedStoreVersion != 1 {
		t.Fatalf("pending=%+v %v", attempt, e)
	}
	if e = os.Remove(s.config.Bindings[0].CredentialFile); e != nil {
		t.Fatal(e)
	}
	if e = s.store.client.Del(context.Background(), s.store.scopeKey(attempt.Scope)+":cookie").Err(); e != nil {
		t.Fatal(e)
	}
	if e = s.resumePending(context.Background(), *attempt); e != nil {
		t.Fatal(e)
	}
	done, e := s.store.Load(context.Background(), c.RequestKey)
	if e != nil || done.SyncStatus != "applied" || done.ErrorCode != "" || automation.calls.Load() != 1 || reader.calls.Load() != 1 {
		t.Fatalf("replayed login/read=%+v %v", done, e)
	}
	replay, e := s.Execute(context.Background(), c)
	if e != nil || replay.AttemptID != c.RequestKey || replay.Status != "succeeded" {
		t.Fatal(replay, e)
	}
	raw, _ := json.Marshal(done)
	for _, secret := range []string{"private-password", "private-cookie", "private-user"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("private input queued")
		}
	}
}

func TestPendingIdentityReadAndCachedCookieDoNotRequireLoginSecret(t *testing.T) {
	for _, pending := range []bool{true, false} {
		t.Run(map[bool]string{true: "pending-identity", false: "cached-cookie"}[pending], func(t *testing.T) {
			s, _, _, automation, reader := currentService(t)
			reader.unavailable.Store(pending)
			c := enqueueCurrent(t, s)
			err := s.processAttempt(context.Background(), c.RequestKey)
			if pending && !errors.Is(err, ErrUnavailable) || !pending && err != nil {
				t.Fatal(err)
			}
			if err = os.Remove(s.config.Bindings[0].CredentialFile); err != nil {
				t.Fatal(err)
			}
			reader.unavailable.Store(false)
			if pending {
				a, loadErr := s.store.Load(context.Background(), c.RequestKey)
				if loadErr != nil {
					t.Fatal(loadErr)
				}
				err = s.resumePending(context.Background(), *a)
			} else {
				c = enqueueCurrent(t, s)
				err = s.processAttempt(context.Background(), c.RequestKey)
			}
			if err != nil {
				t.Fatal("readonly Cookie path reopened login secret", err)
			}
			a, err := s.store.Load(context.Background(), c.RequestKey)
			if err != nil || a.SyncStatus != "applied" || automation.calls.Load() != 1 || reader.calls.Load() != 2 {
				t.Fatalf("readonly path=%+v %v", a, err)
			}
		})
	}
}

type countedOwner struct {
	StoreOwner
	checks atomic.Int32
}

func (o *countedOwner) CheckOwner(ctx context.Context, s storecenter.SheinLoginScope, actor, member, attempt, nonce string) error {
	o.checks.Add(1)
	return o.StoreOwner.CheckOwner(ctx, s, actor, member, attempt, nonce)
}

func TestVerificationCancellationDoesNotContinuouslyPollLiveAuthorization(t *testing.T) {
	s, owner, _, automation, _ := currentService(t)
	counted := &countedOwner{StoreOwner: owner}
	s.owner = counted
	allow := make(chan struct{})
	close(allow)
	automation.waiting = true
	automation.session = &closingSession{closeStarted: make(chan struct{}), allowClose: allow}
	c := enqueueCurrent(t, s)
	done := make(chan error, 1)
	go func() { done <- s.processAttempt(context.Background(), c.RequestKey) }()
	waitStatus(t, s, c.RequestKey, LoginAttemptWaitingVerifyCode)
	checks := counted.checks.Load()
	time.Sleep(1100 * time.Millisecond)
	if counted.checks.Load() != checks {
		t.Error("local cancellation polling called live authorization repeatedly")
	}
	c.Action, c.AttemptID, c.RequestKey = "cancel", c.RequestKey, ""
	if _, err := s.Execute(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("local cancellation waited for remote revalidation interval")
	}
}

type closingSession struct {
	closeStarted chan struct{}
	allowClose   chan struct{}
	submitted    chan string
	closeErr     error
}

func (s *closingSession) SubmitCode(_ context.Context, code string) (*AutomationResult, error) {
	if s.submitted != nil {
		s.submitted <- code
	}
	return &AutomationResult{BrowserState: syntheticState()}, nil
}
func (s *closingSession) Close() error {
	close(s.closeStarted)
	<-s.allowClose
	return s.closeErr
}
func waitStatus(t *testing.T, s *Service, id string, status LoginAttemptStatus) {
	t.Helper()
	until := time.Now().Add(5 * time.Second)
	for time.Now().Before(until) {
		a, e := s.store.Load(context.Background(), id)
		if e == nil && a.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("attempt never reached", status)
}
func TestQuiesceRequiresActualSessionCloseAndUnknownCannotTakeOver(t *testing.T) {
	s, owner, _, automation, _ := currentService(t)
	session := &closingSession{closeStarted: make(chan struct{}), allowClose: make(chan struct{})}
	automation.waiting = true
	automation.session = session
	c := enqueueCurrent(t, s)
	done := make(chan error, 1)
	go func() { done <- s.processAttempt(context.Background(), c.RequestKey) }()
	waitStatus(t, s, c.RequestKey, LoginAttemptWaitingVerifyCode)
	q := c
	q.Action = "quiesce"
	q.RequestKey = uuid.NewString()
	if _, e := s.Execute(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	select {
	case <-session.closeStarted:
	case <-time.After(20 * time.Second):
		t.Fatal("owner did not drain")
	}
	scope := s.config.Scope(s.config.Bindings[0])
	other := scope
	other.InstallationID = uuid.NewString()
	command := storecenter.SheinBindingCommand{Scope: other, ActorID: "actor", MemberID: "member", RequestKey: uuid.NewString(), ExpectedGeneration: 1}
	if _, e := owner.Activate(context.Background(), command); !errors.Is(e, storecenter.ErrSheinLoginFenced) {
		t.Fatal("took over before close", e)
	}
	close(session.allowClose)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("owned close stuck")
	}
	next, e := owner.Activate(context.Background(), command)
	if e != nil || next.Generation != 2 {
		t.Fatal(next, e)
	}
	// Even an expired/interrupted Redis attempt cannot clear an unconfirmed owner.
	other.Generation = 2
	if e = owner.Claim(context.Background(), other, "actor", "member", uuid.NewString(), uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	back := command
	back.Scope = scope
	back.ExpectedGeneration = 2
	back.RequestKey = uuid.NewString()
	if _, e = owner.Activate(context.Background(), back); !errors.Is(e, storecenter.ErrSheinLoginFenced) {
		t.Fatal("UNKNOWN takeover", e)
	}
}
func TestFailedBrowserClosureKeepsSuccessfulAttemptOwnerFenced(t *testing.T) {
	s, owner, _, automation, _ := currentService(t)
	allowClose := make(chan struct{})
	close(allowClose)
	automation.session = &closingSession{closeStarted: make(chan struct{}), allowClose: allowClose, closeErr: errors.New("synthetic close failure")}
	c := enqueueCurrent(t, s)
	if e := s.processAttempt(context.Background(), c.RequestKey); e != nil {
		t.Fatal(e)
	}
	scope := s.config.Scope(s.config.Bindings[0])
	b, e := owner.ReadBinding(context.Background(), scope, "actor", "member")
	if e != nil || b.ActiveAttemptID != c.RequestKey || b.ClosedConfirmed {
		t.Fatalf("failed close released owner: %+v %v", b, e)
	}
	q := c
	q.Action, q.RequestKey = "quiesce", uuid.NewString()
	if _, e = s.Execute(context.Background(), q); e != nil {
		t.Fatal(e)
	}
	other := scope
	other.InstallationID = uuid.NewString()
	if _, e = owner.Activate(context.Background(), storecenter.SheinBindingCommand{Scope: other, ActorID: "actor", MemberID: "member", RequestKey: uuid.NewString(), ExpectedGeneration: b.Generation}); !errors.Is(e, storecenter.ErrSheinLoginFenced) {
		t.Fatal("successful login was treated as browser closure proof", e)
	}
}

func TestProducerNeverReadsLoginCredentialAndStatusHasNoExecutionSideEffect(t *testing.T) {
	s, _, live, automation, reader := currentService(t)
	if e := os.Remove(s.config.Bindings[0].CredentialFile); e != nil {
		t.Fatal(e)
	}
	c := enqueueCurrent(t, s)
	status := c
	status.Action = "status"
	status.AttemptID = c.RequestKey
	status.RequestKey = ""
	if _, e := s.Execute(context.Background(), status); e != nil {
		t.Fatal(e)
	}
	if automation.calls.Load() != 0 || reader.calls.Load() != 0 {
		t.Fatal("producer performed login/sync")
	}
	status.MemberID = "replacement"
	if _, e := s.Execute(context.Background(), status); e == nil {
		t.Fatal("foreign member read attempt")
	}
	live.revoked.Store(true)
	status.MemberID = "member"
	if _, e := s.Execute(context.Background(), status); e == nil {
		t.Fatal("revoked scope readable")
	}
}
