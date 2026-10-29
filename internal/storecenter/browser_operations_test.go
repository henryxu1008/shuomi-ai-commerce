package storecenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/storecenter"
)

type browserAccessFixture struct {
	subject, member   string
	admin, write, use bool
}

func (a *browserAccessFixture) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: a.subject, MemberID: a.member, Administrator: a.admin, CanWrite: a.write}, nil
}
func (a *browserAccessFixture) AuthorizeStoreBrowser(context.Context, string) error {
	if !a.use {
		return storecenter.ErrBrowserDenied
	}
	return nil
}
func (a *browserAccessFixture) ReceiptSubject(context.Context) (string, error) {
	if a.subject == "" {
		return "", storecenter.ErrNotFound
	}
	return a.subject, nil
}
func browserFixture(t *testing.T) (*gorm.DB, *browserAccessFixture, *storecenter.MemberScopedStoreRepository, *storecenter.BrowserOperations, string) {
	t.Helper()
	db := openStoreDB(t)
	if err := storecenter.MigrateBrowserOperationsFixture(db); err != nil {
		t.Fatal(err)
	}
	a := &browserAccessFixture{subject: "subject-create", member: "membership-original", admin: true, write: true, use: true}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, a)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "first", "SG", "merchant", now)
	if _, _, err = repo.CreateOrReplay(context.Background(), "org-a", store); err != nil {
		t.Fatal(err)
	}
	if err = repo.SetMemberGrant(context.Background(), storecenter.MemberStoreGrantCommand{OrganizationID: "org-a", StoreID: store.ID(), MemberID: a.member, OperationID: uuid.NewString(), Active: true}); err != nil {
		t.Fatal(err)
	}
	a.admin = false
	a.write = false // browser use must not silently require store.update.
	service, err := storecenter.NewBrowserOperations(db, a, a, "https://identity.example.test", func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	return db, a, repo, service, store.ID()
}
func browserCommand() storecenter.BrowserCommand {
	return storecenter.BrowserCommand{SchemaVersion: "store-browser-v1", Action: "open_environment", BindingID: uuid.NewString(), ProfileID: uuid.NewString(), BindingVersion: "1"}
}
func TestBrowserAdmissionIsSingleUseAndReadPermissionSuffices(t *testing.T) {
	_, a, _, s, store := browserFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	command := browserCommand()
	first, err := s.Admit(ctx, "org-a", store, key, command)
	if err != nil || !first.ExecutionAuthorized {
		t.Fatalf("read+use+grant admission: %+v %v", first, err)
	}
	replay, err := s.Admit(ctx, "org-a", store, key, command)
	if err != nil || replay.ExecutionAuthorized {
		t.Fatalf("replay may not execute: %+v %v", replay, err)
	}
	read, err := s.Receipt(ctx, store, key)
	if err != nil || read.ExecutionAuthorized {
		t.Fatalf("GET may not execute: %+v %v", read, err)
	}
	command.ProfileID = uuid.NewString()
	if _, err = s.Admit(ctx, "org-a", store, key, command); !errors.Is(err, storecenter.ErrBrowserConflict) {
		t.Fatalf("payload replaced: %v", err)
	}
	a.subject = "other-subject"
	if _, err = s.Receipt(ctx, store, key); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("original owner changed: %v", err)
	}
}
func TestBrowserRevokedMemberCanReportOnlyOriginalOutcome(t *testing.T) {
	db, a, repo, s, store := browserFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	if _, err := s.Admit(ctx, "org-a", store, key, browserCommand()); err != nil {
		t.Fatal(err)
	}
	a.admin = true
	a.write = true
	if err := repo.SetMemberGrant(ctx, storecenter.MemberStoreGrantCommand{OrganizationID: "org-a", StoreID: store, MemberID: a.member, OperationID: uuid.NewString(), ExpectedVersion: 1, Active: false}); err != nil {
		t.Fatal(err)
	}
	a.admin = false
	a.write = false
	a.use = false
	if _, err := s.Admit(ctx, "org-a", store, uuid.NewString(), browserCommand()); err == nil {
		t.Fatal("revoked user admitted new operation")
	}
	report := storecenter.BrowserReport{SchemaVersion: "store-browser-v1", Kind: "open_result", Outcome: storecenter.AuditOutcomeUnknown, ReasonCode: "result_unknown", ObservedAt: time.Date(2026, 10, 25, 10, 0, 1, 0, time.UTC)}
	for range 2 {
		receipt, err := s.Report(ctx, store, key, report)
		if err != nil || receipt.ExecutionAuthorized || len(receipt.Reports) != 1 {
			t.Fatalf("original report after revocation: %+v %v", receipt, err)
		}
	}
	var count int64
	if err := db.Table("workbench_store_browser_reports").Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("report duplicated: %d %v", count, err)
	}
	report.Outcome = storecenter.AuditOutcomeSucceeded
	report.ReasonCode = "process_started"
	if _, err := s.Report(ctx, store, key, report); !errors.Is(err, storecenter.ErrBrowserConflict) {
		t.Fatalf("UNKNOWN overwritten: %v", err)
	}
}
func TestBrowserAdmissionAndAuditCommitTogether(t *testing.T) {
	db, _, _, s, store := browserFixture(t)
	injected := errors.New("fixture audit unavailable")
	if err := db.Callback().Create().Before("gorm:create").Register("fixture_browser_audit_failure", func(tx *gorm.DB) {
		if tx.Statement.Schema != nil && tx.Statement.Schema.Table == "workbench_store_audit_logs" {
			tx.AddError(injected)
		}
	}); err != nil {
		t.Fatal(err)
	}
	receipt, err := s.Admit(context.Background(), "org-a", store, uuid.NewString(), browserCommand())
	if err == nil || receipt.ExecutionAuthorized {
		t.Fatalf("audit failure granted execution: %+v %v", receipt, err)
	}
	var count int64
	if err = db.Table("workbench_store_browser_operations").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("operation escaped rollback: %d %v", count, err)
	}
}
func TestBrowserReportRejectsInvalidCombinationsAndDifferentPrincipal(t *testing.T) {
	_, a, _, s, store := browserFixture(t)
	ctx := context.Background()
	key := uuid.NewString()
	if _, err := s.Admit(ctx, "org-a", store, key, browserCommand()); err != nil {
		t.Fatal(err)
	}
	r := storecenter.BrowserReport{SchemaVersion: "store-browser-v1", Kind: "configure_result", Outcome: storecenter.AuditOutcomeSucceeded, ReasonCode: "configuration_saved", ObservedAt: time.Now().UTC()}
	if _, err := s.Report(ctx, store, key, r); !errors.Is(err, storecenter.ErrBrowserInvalid) {
		t.Fatalf("wrong operation kind accepted: %v", err)
	}
	r.Kind = "open_result"
	r.ReasonCode = "launch_failed"
	if _, err := s.Report(ctx, store, key, r); !errors.Is(err, storecenter.ErrBrowserInvalid) {
		t.Fatalf("success with failure reason: %v", err)
	}
	a.subject = "different-subject"
	r.ReasonCode = "process_started"
	if _, err := s.Report(ctx, store, key, r); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("report owner changed: %v", err)
	}
}
