//go:build integration

package storeobservations

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestWalmartOrderPostgresKeepsOriginalPOAcrossReplaysAndConflicts(t *testing.T) {
	ctx, db, r := observationDatabase(t)
	scope := o.Scope{"org-a", "actor", "member"}
	store := uuid.NewString()
	in, children := syncInput(scope, store)
	in.Kind = o.Orders
	children[0].Kind = o.Orders
	children[0].Binding.Platform = storecenter.PlatformWalmart
	children[0].Binding.Site = "walmart-us"
	command, e := r.Begin(ctx, scope, uuid.NewString(), "walmart", in, children)
	require.NoError(t, e)
	s := command.Syncs[0]
	po := o.WalmartOrder{ID: "po-1", Site: "walmart-us", Date: time.Now().UTC().Format(time.RFC3339Nano), Lines: []o.WalmartOrderLine{{Number: "1", Charges: []o.WalmartOrderCharge{{Currency: "USD", Amount: "10.00"}}}}}
	row := o.Record{StoreID: store, SyncID: s.ID, ID: po.ID, WalmartOrder: &po, WindowKey: "one"}
	next := s.Progress
	next.Page++
	s, e = r.CommitPage(ctx, s, next, []o.Record{row}, "running", time.Now().UTC())
	require.NoError(t, e)
	next.Page++
	row.WindowKey = "two"
	s, e = r.CommitPage(ctx, s, next, []o.Record{row}, "running", time.Now().UTC())
	require.NoError(t, e)
	require.False(t, s.Progress.Incomplete)
	po.Lines[0].Charges[0].Amount = "99.00"
	next.Page++
	s, e = r.CommitPage(ctx, s, next, []o.Record{row}, "completed", time.Now().UTC())
	require.NoError(t, e)
	require.Equal(t, "partial", s.Status)
	saved, e := r.Record(ctx, "org-a", store, o.Orders, s.ID, po.ID)
	require.NoError(t, e)
	require.Equal(t, "10.00", saved.WalmartOrder.Lines[0].Charges[0].Amount)
	heads, _, e := r.Heads(ctx, "org-a", o.Orders, []string{store})
	require.NoError(t, e)
	require.Equal(t, "partial", heads[0].Status)
	var published bool
	require.NoError(t, db.Raw("SELECT current_id IS NOT NULL FROM store_observations.heads WHERE organization_id=? AND store_id=? AND kind='orders'", scope.OrganizationID, store).Scan(&published).Error)
	require.False(t, published)
}

func TestWalmartOrderAmountPostgresFiltersBeforeCapAndPreservesMilliseconds(t *testing.T) {
	ctx, db, r := observationDatabase(t)
	scope := o.Scope{"org-a", "actor", "member"}
	store := uuid.NewString()
	in, children := syncInput(scope, store)
	in.Kind = o.Orders
	children[0].Kind = o.Orders
	command, e := r.Begin(ctx, scope, uuid.NewString(), "walmart-amount", in, children)
	require.NoError(t, e)
	source := command.Syncs[0]
	start := time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	insert := func(id, date string) {
		body, _ := json.Marshal(map[string]any{"walmartOrder": o.WalmartOrder{ID: id, Site: "walmart-us", Date: date, Lines: []o.WalmartOrderLine{{Number: "1", Charges: []o.WalmartOrderCharge{{Currency: "USD", Amount: "0.10"}}}}}})
		require.NoError(t, db.Exec("INSERT INTO store_observations.records(organization_id,store_id,sync_id,kind,platform_id,payload,observed_at,page_index) VALUES(?,?,?,'orders',?,?::jsonb,?,1)", scope.OrganizationID, store, source.ID, id, string(body), time.Now()).Error)
	}
	for i := 0; i < 251; i++ {
		insert(uuid.NewString(), start.Add(-time.Hour).Format(time.RFC3339Nano))
	}
	insert("first", start.Format(time.RFC3339Nano))
	insert("last-millisecond", end.Add(-time.Millisecond).Format(time.RFC3339Nano))
	insert("next-day", end.Format(time.RFC3339Nano))
	insert("unknown", "")
	q := o.OrderAmountQuery{StoreID: store, Start: start, End: end}
	rows, e := r.WalmartOrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.NoError(t, e)
	require.Len(t, rows, 3)
	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.Order.ID)
	}
	require.ElementsMatch(t, []string{"first", "last-millisecond", "unknown"}, ids)
	rows, e = r.WalmartOrderAmountRecords(ctx, "org-other", source.ID, q, 251)
	require.NoError(t, e)
	require.Empty(t, rows)
	insert("invalid", "not-a-date")
	_, e = r.WalmartOrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.Error(t, e)
}

func TestWalmartRepeatedPageCannotHideMissingPOAfterWindowOverlap(t *testing.T) {
	ctx, db, r := observationDatabase(t)
	scope := o.Scope{"org-a", "actor", "member"}
	store := uuid.NewString()
	in, children := syncInput(scope, store)
	in.Kind = o.Orders
	children[0].Kind = o.Orders
	children[0].Binding.Platform = storecenter.PlatformWalmart
	children[0].Binding.Site = "walmart-us"
	cmd, e := r.Begin(ctx, scope, uuid.NewString(), "duplicate-page", in, children)
	require.NoError(t, e)
	source := cmd.Syncs[0]
	po := o.WalmartOrder{ID: "po-a", Site: "walmart-us", Lines: []o.WalmartOrderLine{{Number: "1", Charges: []o.WalmartOrderCharge{{Currency: "USD", Amount: "10.00"}}}}}
	row := o.Record{StoreID: store, SyncID: source.ID, ID: po.ID, WalmartOrder: &po, WindowKey: "previous-window/0"}
	next := source.Progress
	next.Page = 1
	source, e = r.CommitPage(ctx, source, next, []o.Record{row}, "running", time.Now())
	require.NoError(t, e)
	row.WindowKey = "current-window/0"
	next.Page = 2
	source, e = r.CommitPage(ctx, source, next, []o.Record{row}, "running", time.Now())
	require.NoError(t, e)
	require.False(t, source.Progress.Incomplete)
	// total=2 with two raw page rows PO-A/PO-A must not certify missing PO-B.
	next.Page = 3
	next.Seen = 2
	total := 2
	next.WalmartOrders = &o.WalmartOrderProgress{Complete: true, Total: &total}
	source, e = r.CommitPage(ctx, source, next, []o.Record{row}, "completed", time.Now())
	require.NoError(t, e)
	require.Equal(t, "partial", source.Status)
	require.False(t, source.Progress.WalmartOrders.Complete)
	require.Contains(t, source.Progress.Notes, "walmart_duplicate_page_identity")
	var count int64
	require.NoError(t, db.Raw("SELECT count(*) FROM store_observations.records WHERE organization_id=? AND sync_id=?", scope.OrganizationID, source.ID).Scan(&count).Error)
	require.EqualValues(t, 1, count)
	var published bool
	require.NoError(t, db.Raw("SELECT current_id IS NOT NULL FROM store_observations.heads WHERE organization_id=? AND store_id=? AND kind='orders'", scope.OrganizationID, store).Scan(&published).Error)
	require.False(t, published)
}
