// This historical CI path remains test-only. The retained race target exercises
// current Store persistence and audit contracts, without a login runtime owner.
package listingadmin_test

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-processor/internal/storecenter"
)

func TestCurrentStoreConcurrentReceiptsAndOrganizationIsolation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "current.sqlite")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := storecenter.AutoMigrateStoreRepository(db); err != nil {
		t.Fatal(err)
	}
	repo, err := storecenter.NewGormStoreRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	audit, err := storecenter.NewGormAuditRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	now := time.Now().UTC()
	fixture := func(org, id, key, name string) *storecenter.Store {
		t.Helper()
		s, err := storecenter.NewStore(storecenter.CreateStoreInput{
			ID: id, OrganizationID: org, ActorSubject: "operator", Name: name,
			Platform: "shein", Region: "SG", ExternalStoreID: "synthetic-merchant",
			CreateIdempotencyKey: key, OccurredAt: now,
		})
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	key := uuid.NewString()
	first := fixture("org-a", uuid.NewString(), key, "Store")
	stored, replayed, err := repo.CreateOrReplay(ctx, "org-a", first)
	if err != nil || replayed {
		t.Fatal("initial create", replayed, err)
	}
	before := stored.Snapshot()
	duplicate := fixture("org-a", uuid.NewString(), uuid.NewString(), "Other")
	if _, _, err := repo.CreateOrReplay(ctx, "org-a", duplicate); !errors.Is(err, storecenter.ErrAlreadyExists) {
		t.Fatalf("same-org identity conflict = %v", err)
	}
	if _, err := repo.Get(ctx, "org-a", duplicate.ID()); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("identity conflict created a Store: %v", err)
	}
	if got, err := repo.Get(ctx, "org-a", first.ID()); err != nil || !reflect.DeepEqual(got.Snapshot(), before) {
		t.Fatal("identity conflict mutated original Store", err)
	}
	if _, err := repo.Get(ctx, "org-b", first.ID()); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("foreign Store read = %v", err)
	}
	if _, _, err := repo.CreateOrReplay(ctx, "org-b", first); err == nil {
		t.Fatal("foreign create scope admitted")
	}
	if err := repo.Save(ctx, "org-b", stored, 0); err == nil {
		t.Fatal("foreign save scope admitted")
	}

	// The same merchant and request key are independent in another Organization.
	candidate := fixture("org-b", uuid.NewString(), key, "Store")
	type result struct {
		store    *storecenter.Store
		replayed bool
		err      error
	}
	results := make(chan result, 16)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			s, replayed, err := repo.CreateOrReplay(ctx, "org-b", candidate)
			results <- result{s, replayed, err}
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	inserts := 0
	for r := range results {
		if r.err != nil {
			t.Fatal(r.err)
		}
		if r.store.ID() != candidate.ID() || r.store.OrganizationID() != "org-b" || r.store.Version() != 1 || r.store.ExternalStoreID() != "synthetic-merchant" || r.store.ConnectionRef() != "" {
			t.Fatalf("convergence = %+v", r.store.Snapshot())
		}
		if !r.replayed {
			inserts++
		}
	}
	if inserts != 1 {
		t.Fatalf("concurrent inserts = %d, want 1", inserts)
	}
	for _, org := range []string{"org-a", "org-b"} {
		page, err := repo.List(ctx, org, storecenter.StoreListQuery{})
		if err != nil || page.Total != 1 || len(page.Stores) != 1 || page.Stores[0].OrganizationID() != org {
			t.Fatalf("scoped Store rows for %s = %+v, %v", org, page, err)
		}
		var rows int64
		if err := db.Table("workbench_store_audit_logs").Where("organization_id=? AND request_key=? AND action=?", org, key, storecenter.AuditActionStoreCreated).Count(&rows).Error; err != nil || rows != 1 {
			t.Fatalf("scoped creation audit count = %d, %v", rows, err)
		}
	}
	receipt, err := audit.Get(ctx, "org-b", key, storecenter.AuditActionStoreCreated)
	if err != nil || receipt == nil || receipt.StoreID != candidate.ID() || receipt.StoreVersion != 1 {
		t.Fatal("creation receipt", receipt, err)
	}
	foreignReceipt, err := audit.Get(ctx, "org-a", key, storecenter.AuditActionStoreCreated)
	if err != nil || foreignReceipt == nil || foreignReceipt.StoreID != first.ID() || foreignReceipt.EventID == receipt.EventID {
		t.Fatal("audit organization boundary", foreignReceipt, err)
	}
	changed := fixture("org-b", uuid.NewString(), key, "Different")
	if _, _, err := repo.CreateOrReplay(ctx, "org-b", changed); !errors.Is(err, storecenter.ErrAlreadyExists) {
		t.Fatalf("same-key payload conflict = %v", err)
	}

	// A pristine creation retry returns the later durable version, never rewinds it.
	latest, err := repo.Get(ctx, "org-b", candidate.ID())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := latest.EditBasic("Later", "SG", "editor", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, "org-b", latest, 1); err != nil {
		t.Fatal(err)
	}
	got, replayed, err := repo.CreateOrReplay(ctx, "org-b", candidate)
	if err != nil || !replayed || !reflect.DeepEqual(got.Snapshot(), latest.Snapshot()) || got.ConnectionRef() != "" {
		t.Fatal("creation replay rewound current Store", replayed, err)
	}
	stale := fixture("org-b", candidate.ID(), key, "Store")
	if _, err := stale.EditBasic("Stale", "SG", "editor", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, "org-b", stale, 1); !errors.Is(err, storecenter.ErrVersionConflict) {
		t.Fatalf("stale save = %v", err)
	}
	after, err := repo.Get(ctx, "org-b", candidate.ID())
	if err != nil || !reflect.DeepEqual(after.Snapshot(), latest.Snapshot()) {
		t.Fatal("stale save mutated current Store", err)
	}
	afterReceipt, err := audit.Get(ctx, "org-b", key, storecenter.AuditActionStoreCreated)
	if err != nil || !reflect.DeepEqual(afterReceipt, receipt) {
		t.Fatal("replay changed durable creation receipt", err)
	}
}
