// This historical CI path is test-only. The retired numeric Store owner has no
// production package. These tests execute the current Store/login contracts;
// `go test -race ./internal/listingadmin -count=1` remains unchanged.
package listingadmin_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/storecenter"
)

type member struct{ org string }

func (a member) AuthorizeStoreMember(context.Context, string) (storecenter.StoreMemberAccess, error) {
	return storecenter.StoreMemberAccess{OrganizationID: a.org, ActorID: "operator", MemberID: "member", Administrator: true, CanWrite: true}, nil
}
func TestCurrentStoreLoginConcurrentReceiptsAndOrganizationIsolation(t *testing.T) {
	db, e := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "current.sqlite")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
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
	repo, _ := storecenter.NewGormStoreRepository(db)
	ctx := context.Background()
	create := func(org, external string) *storecenter.Store {
		t.Helper()
		s, e := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: org, ActorSubject: "operator", Name: "Store", Platform: "shein", Region: "SG", ExternalStoreID: external, CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
		if e != nil {
			t.Fatal(e)
		}
		s, _, e = repo.CreateOrReplay(ctx, org, s)
		if e != nil {
			t.Fatal(e)
		}
		return s
	}
	_ = create("org-a", "occupied")
	fixture := func(org string) (*storecenter.SheinLoginRepository, storecenter.SheinIdentityCommand) {
		t.Helper()
		s := create(org, "")
		owner, _ := storecenter.NewSheinLoginRepository(db, member{org})
		scope := storecenter.SheinLoginScope{OrganizationID: org, StoreID: s.ID(), InstallationID: uuid.NewString(), BindingRevision: uuid.NewString(), ConfigFingerprint: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
		active, e := owner.Activate(ctx, storecenter.SheinBindingCommand{Scope: scope, ActorID: "operator", MemberID: "member", RequestKey: uuid.NewString()})
		if e != nil {
			t.Fatal(e)
		}
		scope.Generation = active.Generation
		c := storecenter.SheinIdentityCommand{Scope: scope, ActorID: "operator", MemberID: "member", AttemptID: uuid.NewString(), OwnerNonce: uuid.NewString(), ExpectedStoreVersion: 1, MerchantID: "occupied"}
		if e = owner.Claim(ctx, scope, c.ActorID, c.MemberID, c.AttemptID, c.OwnerNonce); e != nil {
			t.Fatal(e)
		}
		return owner, c
	}
	owner, c := fixture("org-a")
	blocked, e := owner.RecordSheinLoginIdentity(ctx, c)
	if e != nil || blocked.Status != "blocked" {
		t.Fatalf("same-org identity conflict=%+v %v", blocked, e)
	}
	before, e := repo.Get(ctx, "org-a", c.Scope.StoreID)
	if e != nil {
		t.Fatal(e)
	}
	if before.ExternalStoreID() != "" || before.Version() != 1 || before.ConnectionRef() != "" {
		t.Fatal("conflict mutated Store")
	}
	foreign, _ := storecenter.NewSheinLoginRepository(db, member{"org-b"})
	if _, e = foreign.RecordSheinLoginIdentity(ctx, c); e == nil {
		t.Fatal("foreign scope admitted")
	}
	owner, c = fixture("org-b") // same merchant is valid in a different organization
	var wg sync.WaitGroup
	results := make(chan storecenter.SheinIdentityReceipt, 16)
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := owner.RecordSheinLoginIdentity(ctx, c); results <- r; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for r := range results {
		if r.Status != "applied" || r.StoreVersion != 2 {
			t.Fatalf("convergence=%+v", r)
		}
	}
	changed := c
	changed.MerchantID = "different"
	if _, e = owner.RecordSheinLoginIdentity(ctx, changed); !errors.Is(e, storecenter.ErrIdempotencyConflict) {
		t.Fatalf("payload conflict=%v", e)
	}
	// A later current-owner update cannot be rewound by the old login receipt.
	if e = db.Table("workbench_stores").Where("organization_id=? AND id=?", "org-b", c.Scope.StoreID).Updates(map[string]any{"name": "Later", "version": 3}).Error; e != nil {
		t.Fatal(e)
	}
	receipt, e := owner.RecordSheinLoginIdentity(ctx, c)
	if e != nil || receipt.StoreVersion != 2 {
		t.Fatal(receipt, e)
	}
	got, e := repo.Get(ctx, "org-b", c.Scope.StoreID)
	if e != nil || got.Version() != 3 || got.Name() != "Later" || got.ExternalStoreID() != "occupied" {
		t.Fatal("old receipt rewound current state", e)
	}
}
