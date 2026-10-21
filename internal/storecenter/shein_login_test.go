package storecenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/storecenter"
)

type sheinMember struct{ org string }

func (a sheinMember) AuthorizeStoreMember(context.Context, string) (storecenter.StoreMemberAccess, error) {
	return storecenter.StoreMemberAccess{OrganizationID: a.org, ActorID: "operator", MemberID: "membership", Administrator: true, CanWrite: true}, nil
}

func TestSheinLoginFenceAndIdentityReceipts(t *testing.T) {
	db := openStoreDB(t)
	if err := storecenter.AutoMigrateSheinLoginForTests(db); err != nil {
		t.Fatal(err)
	}
	repo, _ := storecenter.NewGormStoreRepository(db)
	s := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), "", "Login", "SG", "", time.Now().UTC())
	if _, _, err := repo.CreateOrReplay(context.Background(), "org-a", s); err != nil {
		t.Fatal(err)
	}
	owner, _ := storecenter.NewSheinLoginRepository(db, sheinMember{"org-a"})
	scope := storecenter.SheinLoginScope{OrganizationID: "org-a", StoreID: s.ID(), InstallationID: uuid.NewString(), BindingRevision: uuid.NewString(), ConfigFingerprint: string(make([]byte, 0))}
	scope.ConfigFingerprint = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cmd := storecenter.SheinBindingCommand{Scope: scope, ActorID: "operator", MemberID: "membership", RequestKey: uuid.NewString()}
	binding, err := owner.Activate(context.Background(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	scope.Generation = binding.Generation
	attempt, nonce := uuid.NewString(), uuid.NewString()
	if err := owner.Claim(context.Background(), scope, "operator", "membership", attempt, nonce); err != nil {
		t.Fatal(err)
	}
	other := cmd
	other.Scope.InstallationID = uuid.NewString()
	other.RequestKey = uuid.NewString()
	if _, err := owner.Activate(context.Background(), other); !errors.Is(err, storecenter.ErrSheinLoginFenced) {
		t.Fatalf("overlap = %v", err)
	}
	sync := storecenter.SheinIdentityCommand{Scope: scope, ActorID: "operator", MemberID: "membership", AttemptID: attempt, OwnerNonce: nonce, ExpectedStoreVersion: 1, MerchantID: "merchant-a"}
	receipt, err := owner.RecordSheinLoginIdentity(context.Background(), sync)
	if err != nil || receipt.Status != "applied" || receipt.StoreVersion != 2 {
		t.Fatalf("sync=%+v %v", receipt, err)
	}
	again, err := owner.RecordSheinLoginIdentity(context.Background(), sync)
	if err != nil || again != receipt {
		t.Fatalf("replay=%+v %v", again, err)
	}
	changed := sync
	changed.MerchantID = "other"
	if _, err := owner.RecordSheinLoginIdentity(context.Background(), changed); !errors.Is(err, storecenter.ErrIdempotencyConflict) {
		t.Fatalf("payload reuse=%v", err)
	}
	q := cmd
	q.Scope = scope
	q.RequestKey = uuid.NewString()
	quiescing, err := owner.Quiesce(context.Background(), q)
	if err != nil || quiescing.State != "quiescing" {
		t.Fatalf("quiesce=%+v %v", quiescing, err)
	}
	if _, err := owner.Activate(context.Background(), other); !errors.Is(err, storecenter.ErrSheinLoginFenced) {
		t.Fatalf("UNKNOWN takeover=%v", err)
	}
	if err := owner.ConfirmClosed(context.Background(), scope, attempt, "wrong"); err == nil {
		t.Fatal("foreign owner closed")
	}
	if err := owner.ConfirmClosed(context.Background(), scope, attempt, nonce); err != nil {
		t.Fatal(err)
	}
	other.ExpectedGeneration = scope.Generation
	newer, err := owner.Activate(context.Background(), other)
	if err != nil || newer.Generation != 2 {
		t.Fatalf("handoff=%+v %v", newer, err)
	}
	if err := owner.Claim(context.Background(), scope, "operator", "membership", uuid.NewString(), uuid.NewString()); err == nil {
		t.Fatal("old generation reclaimed")
	}
	// An old immutable receipt is readable but cannot rewind the Store or generation.
	again, err = owner.RecordSheinLoginIdentity(context.Background(), sync)
	if err != nil || again != receipt {
		t.Fatalf("old receipt=%+v %v", again, err)
	}
	got, err := repo.Get(context.Background(), "org-a", s.ID())
	if err != nil || got.ExternalStoreID() != "merchant-a" || got.Version() != 2 || got.ConnectionRef() != "" {
		t.Fatalf("Store=%+v %v", got, err)
	}
	foreign, _ := storecenter.NewSheinLoginRepository(db, sheinMember{"org-b"})
	if _, err := foreign.Activate(context.Background(), cmd); err == nil {
		t.Fatal("cross organization allowed")
	}
}
