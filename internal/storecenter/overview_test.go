package storecenter_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"task-processor/internal/storecenter"
)

type changingLiveOverview struct {
	calls int
	deny  bool
}

func (a *changingLiveOverview) AuthorizeLiveOverview(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	a.calls++
	if a.calls == 2 && a.deny {
		return storecenter.StoreMemberAccess{}, errors.New("read revoked")
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: "subject-create", MemberID: "member-a", Administrator: a.calls == 1, CanWrite: true}, nil
}

func TestVisibleOverviewFailsClosedOnLiveRoleOrPermissionChanges(t *testing.T) {
	for _, deny := range []bool{false, true} {
		t.Run(fmt.Sprint(deny), func(t *testing.T) {
			db := openStoreDB(t)
			catalog, err := storecenter.NewCatalog(db, &storeMemberAuthorizer{member: "member-a"}, nil)
			if err != nil {
				t.Fatal(err)
			}
			live := &changingLiveOverview{deny: deny}
			got, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewAll, live)
			if err == nil || got.Platforms != nil || live.calls != 2 {
				t.Fatalf("live permission change accepted: %+v %v", got, err)
			}
		})
	}
}

type revokingOverviewAccess struct {
	db    *gorm.DB
	calls int
}

func (a *storeMemberAuthorizer) AuthorizeLiveOverview(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	return a.AuthorizeStoreMember(ctx, org)
}
func (a *revokingOverviewAccess) AuthorizeLiveOverview(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	return a.AuthorizeStoreMember(ctx, org)
}

func (a *revokingOverviewAccess) AuthorizeStoreMember(_ context.Context, org string) (storecenter.StoreMemberAccess, error) {
	a.calls++
	if a.calls == 2 {
		if err := a.db.Exec("UPDATE workbench_store_member_grants SET active = ? WHERE organization_id = ? AND member_id = ?", false, org, "member-a").Error; err != nil {
			return storecenter.StoreMemberAccess{}, err
		}
	}
	return storecenter.StoreMemberAccess{OrganizationID: org, ActorID: "subject-create", MemberID: "member-a", CanWrite: true}, nil
}

func TestVisibleOverviewRejectsRevocationDuringRead(t *testing.T) {
	db := openStoreDB(t)
	initial := &storeMemberAuthorizer{member: "member-a"}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, initial)
	if err != nil {
		t.Fatal(err)
	}
	candidate := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "visible", "US", "visible", time.Now().UTC())
	if _, _, err = repo.CreateOrReplay(context.Background(), "org-a", candidate); err != nil {
		t.Fatal(err)
	}
	live := &revokingOverviewAccess{db: db}
	catalog, err := storecenter.NewCatalog(db, live, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewAll, live)
	if err == nil || got.Total != 0 || got.Platforms != nil {
		t.Fatalf("revoked facts leaked: %+v %v", got, err)
	}
}

func TestVisibleOverviewEmptyAndCrossOrganization(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "member-a"}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, access)
	if err != nil {
		t.Fatal(err)
	}
	item := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "one", "US", "one", time.Now().UTC())
	if _, _, err = repo.CreateOrReplay(context.Background(), "org-a", item); err != nil {
		t.Fatal(err)
	}
	catalog, err := storecenter.NewCatalog(db, access, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-b", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewAll, access)
	if err != nil || got.Total != 0 || len(got.Platforms) != 0 {
		t.Fatalf("cross-org or invalid empty: %+v %v", got, err)
	}
	if _, err = catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewScope("UNKNOWN"), access); err == nil {
		t.Fatal("invalid scope accepted")
	}
	for _, identity := range []storecenter.OverviewIdentity{
		{OrganizationID: "org-a", ActorID: "other", MemberID: "member-a"},
		{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "other"},
	} {
		if result, err := catalog.VisibleOverview(context.Background(), identity, storecenter.OverviewAll, access); err == nil || result.Platforms != nil {
			t.Fatalf("identity mismatch accepted: %+v %v", result, err)
		}
	}
}

func TestVisibleOverviewUsesAuthorizedCompleteSnapshotAndStatuses(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "member-a"}
	repo, err := storecenter.NewMemberScopedStoreRepository(db, access)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 105; i++ {
		platform := "shein"
		if i%2 == 1 {
			platform = "temu"
		}
		candidate, e := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: "org-a", ActorSubject: "subject-create", Name: fmt.Sprintf("店铺%d", i), Platform: platform, Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: time.Now().UTC()})
		if e != nil {
			t.Fatal(e)
		}
		if _, _, e = repo.CreateOrReplay(context.Background(), "org-a", candidate); e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			if e = db.Exec("UPDATE workbench_stores SET record_status = ? WHERE id = ?", "disabled", candidate.ID()).Error; e != nil {
				t.Fatal(e)
			}
		}
		if i == 1 {
			if e = db.Exec("UPDATE workbench_stores SET record_status = ? WHERE id = ?", "deleting", candidate.ID()).Error; e != nil {
				t.Fatal(e)
			}
		}
	}
	access.member = "member-b"
	hidden := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "hidden", "US", "hidden", time.Now().UTC())
	if _, _, err = repo.CreateOrReplay(context.Background(), "org-a", hidden); err != nil {
		t.Fatal(err)
	}
	access.member = "member-a"
	catalog, err := storecenter.NewCatalog(db, access, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewAll, access)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 105 || got.Platforms["shein"] != 53 || got.Platforms["temu"] != 52 || got.Statuses["disabled"] != 1 || got.Statuses["deleting"] != 1 {
		t.Fatalf("wrong overview: %+v", got)
	}
	active, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewActive, access)
	if err != nil {
		t.Fatal(err)
	}
	if active.Total != 103 {
		t.Fatalf("active total: %+v", active)
	}
	access.admin = true
	all, err := catalog.VisibleOverview(context.Background(), storecenter.OverviewIdentity{OrganizationID: "org-a", ActorID: "subject-create", MemberID: "member-a"}, storecenter.OverviewAll, access)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 106 {
		t.Fatalf("admin total: %+v", all)
	}
}
