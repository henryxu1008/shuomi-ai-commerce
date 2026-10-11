package storecenter_test

import (
	"context"
	"github.com/google/uuid"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type unknownStatusSignals struct{}

func (unknownStatusSignals) ReadCatalogSignals(_ context.Context, _ string, ids []string, _ map[string]*int64) (map[string]storecenter.CatalogSignals, error) {
	out := map[string]storecenter.CatalogSignals{}
	for _, id := range ids {
		out[id] = storecenter.CatalogSignals{Connection: storecenter.ConnectionStatusConnected, SyncHealthy: true, Inventory: storecenter.CatalogMetric{Reason: "threshold_not_set"}}
	}
	return out, nil
}

func TestCatalogNeverCountsMissingProductOrOrderStatusAsNormal(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "member-a"}
	records, _ := storecenter.NewMemberScopedStoreRepository(db, access)
	candidate := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "A", "US", "A", time.Now().UTC())
	_, _, err := records.CreateOrReplay(context.Background(), "org-a", candidate)
	if err != nil {
		t.Fatal(err)
	}
	catalog, _ := storecenter.NewCatalog(db, access, unknownStatusSignals{})
	result, err := catalog.List(context.Background(), storecenter.CatalogRequest{OrganizationID: "org-a", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if result.Items[0].Attention != "unknown" || result.Metrics.Normal != nil || result.Metrics.Unknown != 1 {
		t.Fatalf("false normal: %+v", result)
	}
}

func TestCatalogFiltersFullAuthorizedCollectionBeforePagination(t *testing.T) {
	db := openStoreDB(t)
	access := &storeMemberAuthorizer{member: "member-a"}
	records, _ := storecenter.NewMemberScopedStoreRepository(db, access)
	for _, name := range []string{"Other", "STRASSE 100%_", "Third"} {
		candidate := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), name, "US", name, time.Now().UTC())
		if _, _, err := records.CreateOrReplay(context.Background(), "org-a", candidate); err != nil {
			t.Fatal(err)
		}
	}
	access.member = "member-b"
	candidate := newPersistenceStore(t, "org-a", uuid.NewString(), uuid.NewString(), uuid.NewString(), "Private 100%_", "CA", "private", time.Now().UTC())
	if _, _, err := records.CreateOrReplay(context.Background(), "org-a", candidate); err != nil {
		t.Fatal(err)
	}
	access.member = "member-a"
	catalog, _ := storecenter.NewCatalog(db, access, nil)
	result, err := catalog.List(context.Background(), storecenter.CatalogRequest{OrganizationID: "org-a", Page: 1, PageSize: 1, Search: "straße 100%_"})
	if err != nil || result.Total != 1 || len(result.Items) != 1 || result.Items[0].Store.Name() != "STRASSE 100%_" {
		t.Fatalf("literal folded search: %+v %v", result, err)
	}
	result, err = catalog.List(context.Background(), storecenter.CatalogRequest{OrganizationID: "org-a", Page: 1, PageSize: 1})
	if err != nil || result.Total != 3 || result.Metrics.Unknown != 3 || result.Metrics.Normal != nil || len(result.Facets.Regions) != 1 {
		t.Fatalf("scope or false normal: %+v %v", result, err)
	}
	_, err = catalog.List(context.Background(), storecenter.CatalogRequest{OrganizationID: "org-a", Page: 1, PageSize: 1, Attention: "normal"})
	if err == nil {
		t.Fatal("summary predicate silently ignored without signals")
	}
}
