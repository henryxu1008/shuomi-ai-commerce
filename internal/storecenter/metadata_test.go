package storecenter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"task-processor/internal/storecenter"
)

type metadataMembers struct{ active bool }

func (m *metadataMembers) ValidateResponsible(context.Context, string, string) error {
	if !m.active {
		return storecenter.ErrNotFound
	}
	return nil
}

func TestMetadataAtomicBatchReceiptAndOwnership(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "member-a", admin: true}
	records, _ := storecenter.NewMemberScopedStoreRepository(db, access)
	members := &metadataMembers{active: true}
	metadata, err := storecenter.NewMetadataRepository(db, access, members)
	if err != nil {
		t.Fatal(err)
	}
	makeStore := func(org, name string) string {
		candidate := newPersistenceStore(t, org, uuid.NewString(), uuid.NewString(), uuid.NewString(), name, "US", name, time.Now().UTC())
		saved, _, err := records.CreateOrReplay(context.Background(), org, candidate)
		if err != nil {
			t.Fatal(err)
		}
		return saved.ID()
	}
	a, b, foreign := makeStore("org-a", "A"), makeStore("org-a", "B"), makeStore("org-b", "foreign")
	run := func(command storecenter.MetadataCommand) storecenter.MetadataResult {
		t.Helper()
		result, err := metadata.Execute(context.Background(), "org-a", command)
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	group := run(storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "create_group", Name: "旺季"}).Group
	threshold := int64(5)
	original := storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "set_profile", Stores: []storecenter.ProfileTarget{{StoreID: a}}, GroupID: &group.ID, ResponsibleMemberID: ptr("member-b"), LowStockThreshold: &threshold}
	result := run(original)
	if result.Profiles[0].Version != 1 {
		t.Fatal(result)
	}
	// A later write, member departure and group deletion cannot reinterpret the receipt.
	run(storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "set_profile", Stores: []storecenter.ProfileTarget{{StoreID: a, Version: 1}}})
	run(storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "delete_group", GroupID: &group.ID, GroupVersion: 1})
	members.active = false
	replay := run(original)
	if !replay.Replayed || replay.Profiles[0].Version != 1 || replay.Profiles[0].GroupID == nil {
		t.Fatal(replay)
	}
	changed := original
	changed.LowStockThreshold = nil
	if _, err := metadata.Execute(context.Background(), "org-a", changed); !errors.Is(err, storecenter.ErrMetadataKeyConflict) {
		t.Fatalf("changed key: %v", err)
	}
	// Batch versions and target access are checked before any update commits.
	_, err = metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "assign_group", Stores: []storecenter.ProfileTarget{{StoreID: a, Version: 2}, {StoreID: b, Version: 99}}})
	if !errors.Is(err, storecenter.ErrVersionConflict) {
		t.Fatal(err)
	}
	profile, err := metadata.Profile(context.Background(), "org-a", b)
	if err != nil || profile.Version != 0 {
		t.Fatalf("partial batch: %+v %v", profile, err)
	}
	if _, err = metadata.Profile(context.Background(), "org-a", foreign); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("cross org: %v", err)
	}
	// Responsible ownership never creates a Store grant.
	grant, err := records.ReadMemberStoreGrant(context.Background(), "org-a", a, "member-b")
	if err != nil || grant.Active {
		t.Fatalf("responsible granted access: %+v %v", grant, err)
	}
	access.admin = false
	access.member = "member-b"
	if _, err = metadata.Profile(context.Background(), "org-a", a); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("unassigned profile readable: %v", err)
	}
	if _, err = metadata.Execute(context.Background(), "org-a", original); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("replay bypassed grant: %v", err)
	}
	if _, err = metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "create_group", Name: "no"}); !errors.Is(err, storecenter.ErrNotFound) {
		t.Fatalf("member mutated shared group: %v", err)
	}
}

func TestMetadataDeleteGroupClearsProfilesAndReceiptFailureRollsBack(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "m", admin: true}
	repo, _ := storecenter.NewMemberScopedStoreRepository(db, access)
	metadata, _ := storecenter.NewMetadataRepository(db, access, &metadataMembers{active: true})
	candidate := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "name", "US", "ext", time.Now().UTC())
	saved, _, err := repo.CreateOrReplay(context.Background(), "org-a", candidate)
	if err != nil {
		t.Fatal(err)
	}
	group, err := metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "create_group", Name: "one"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "assign_group", GroupID: &group.Group.ID, Stores: []storecenter.ProfileTarget{{StoreID: saved.ID()}}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "delete_group", GroupID: &group.Group.ID, GroupVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := metadata.Profile(context.Background(), "org-a", saved.ID())
	if err != nil || profile.GroupID != nil || profile.Version != 2 {
		t.Fatalf("delete: %+v %v", profile, err)
	}
	if err = db.Exec(`CREATE TRIGGER reject_metadata BEFORE INSERT ON workbench_store_metadata_operations BEGIN SELECT RAISE(ABORT,'receipt failure'); END`).Error; err != nil {
		t.Fatal(err)
	}
	threshold := int64(2)
	_, err = metadata.Execute(context.Background(), "org-a", storecenter.MetadataCommand{OperationID: uuid.NewString(), Action: "set_profile", Stores: []storecenter.ProfileTarget{{StoreID: saved.ID(), Version: 2}}, LowStockThreshold: &threshold})
	if err == nil {
		t.Fatal("mutation committed without receipt")
	}
	profile, _ = metadata.Profile(context.Background(), "org-a", saved.ID())
	if profile.Version != 2 || profile.LowStockThreshold != nil {
		t.Fatal(profile)
	}
}

func ptr(s string) *string { return &s }
