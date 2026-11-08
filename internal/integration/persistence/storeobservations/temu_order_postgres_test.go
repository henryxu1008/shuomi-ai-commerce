//go:build integration

package storeobservations

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestTemuOrderPostgresPageBoundsDedupAndConflict(t *testing.T) {
	ctx, db, r := observationDatabase(t)
	scope := o.Scope{"org-a", "actor", "member"}
	begin := func() o.Sync {
		in, children := syncInput(scope, uuid.NewString())
		in.Kind = o.Orders
		children[0].Kind = o.Orders
		children[0].Binding.Platform = storecenter.PlatformTemu
		children[0].Binding.Site = "temu-order-us"
		children[0].Binding.IdentityPartition = storecenter.PartitionGlobal
		children[0].Binding.MerchantMode = storecenter.MerchantSemiManaged
		children[0].Binding.RegionID = 211
		children[0].Binding.ApplicationID = "app"
		children[0].Binding.ApplicationRevision = "v:temu:GLOBAL:manual_token:3:r2"
		children[0].Binding.VerificationRevision = "temu-regional-order-identity-v2"
		children[0].Binding.ScopesHash = strings.Repeat("a", 64)
		children[0].Binding.SupplierIdentityHash = strings.Repeat("b", 64)
		children[0].Binding.CredentialExpiresAt = time.Now().Add(time.Hour)
		cmd, e := r.Begin(ctx, scope, uuid.NewString(), "temu", in, children)
		require.NoError(t, e)
		return cmd.Syncs[0]
	}
	for _, count := range []int{31, 100, 101} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s := begin()
			rows := []o.Record{}
			for i := 0; i < count; i++ {
				order := o.TemuOrder{ID: fmt.Sprint(i), Site: "temu-order-us", RegionID: 211, Amount: &o.Price{Value: "20.00", Currency: "USD"}}
				rows = append(rows, o.Record{StoreID: s.StoreID, SyncID: s.ID, ID: order.ID, TemuOrder: &order, WindowKey: "one"})
			}
			n := s.Progress
			n.Page++
			_, e := r.CommitPage(ctx, s, n, rows, "running", time.Now())
			if count > 100 {
				require.ErrorIs(t, e, o.ErrInvalid)
			} else {
				require.NoError(t, e)
			}
		})
	}
	s := begin()
	order := o.TemuOrder{ID: "order-1", Site: "temu-order-us", RegionID: 211, Amount: &o.Price{Value: "20.00", Currency: "USD"}}
	row := o.Record{StoreID: s.StoreID, SyncID: s.ID, ID: order.ID, TemuOrder: &order, WindowKey: "one"}
	next := s.Progress
	next.Page = 2
	s, e := r.CommitPage(ctx, s, next, []o.Record{row}, "running", time.Now())
	require.NoError(t, e)
	original, e := r.Record(ctx, scope.OrganizationID, s.StoreID, o.Orders, s.ID, row.ID)
	require.NoError(t, e)
	next.Page = 3
	next.Windows = nil
	next.TemuOrders = &o.TemuOrderProgress{Complete: true}
	s, e = r.CommitPage(ctx, s, next, []o.Record{row}, "completed", time.Now())
	require.NoError(t, e)
	require.Equal(t, "partial", s.Status)
	require.False(t, s.Progress.TemuOrders.Complete)
	require.Contains(t, s.Progress.Notes, "temu_duplicate_page_identity")
	var published bool
	require.NoError(t, db.Raw("SELECT current_id IS NOT NULL FROM store_observations.heads WHERE organization_id=? AND store_id=?", scope.OrganizationID, s.StoreID).Scan(&published).Error)
	require.False(t, published)
	conflict := begin()
	row.StoreID = conflict.StoreID
	row.SyncID = conflict.ID
	next = conflict.Progress
	next.Page = 2
	conflict, e = r.CommitPage(ctx, conflict, next, []o.Record{row}, "running", time.Now())
	require.NoError(t, e)
	row.TemuOrder.Amount.Value = "99.00"
	next.Page = 3
	next.Windows = nil
	next.TemuOrders = &o.TemuOrderProgress{Complete: true}
	conflict, e = r.CommitPage(ctx, conflict, next, []o.Record{row}, "completed", time.Now())
	require.NoError(t, e)
	require.Equal(t, "partial", conflict.Status)
	kept, e := r.Record(ctx, scope.OrganizationID, conflict.StoreID, o.Orders, conflict.ID, row.ID)
	require.NoError(t, e)
	require.Equal(t, "20.00", kept.TemuOrder.Amount.Value)
	saved, e := r.Record(ctx, scope.OrganizationID, s.StoreID, o.Orders, s.ID, "order-1")
	require.NoError(t, e)
	require.Equal(t, original.ObservedAt, saved.ObservedAt)
}
func TestTemuOrderAmountPostgresFiltersBeforeLimitAndKeepsSeconds(t *testing.T) {
	ctx, db, r := observationDatabase(t)
	scope := o.Scope{"org-a", "actor", "member"}
	store := uuid.NewString()
	in, children := syncInput(scope, store)
	in.Kind = o.Orders
	children[0].Kind = o.Orders
	cmd, e := r.Begin(ctx, scope, uuid.NewString(), "tk-filter", in, children)
	require.NoError(t, e)
	source := cmd.Syncs[0]
	start := time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	insert := func(id, date string) {
		body, _ := json.Marshal(map[string]any{"temuOrder": o.TemuOrder{ID: id, Site: "temu-order-us", RegionID: 211, Date: date, Amount: &o.Price{Value: "0.10", Currency: "USD"}}})
		require.NoError(t, db.Exec("INSERT INTO store_observations.records(organization_id,store_id,sync_id,kind,platform_id,payload,observed_at,page_index) VALUES(?,?,?,'orders',?,?::jsonb,?,1)", scope.OrganizationID, store, source.ID, id, string(body), time.Now()).Error)
	}
	for i := 0; i < 251; i++ {
		insert(uuid.NewString(), start.Add(-time.Second).Format(time.RFC3339))
	}
	insert("first", start.Format(time.RFC3339))
	insert("last", end.Add(-time.Second).Format(time.RFC3339))
	insert("next", end.Format(time.RFC3339))
	insert("unknown", "")
	q := o.OrderAmountQuery{StoreID: store, Start: start, End: end}
	rows, e := r.TemuOrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.NoError(t, e)
	require.Len(t, rows, 3)
	ids := []string{}
	for _, v := range rows {
		ids = append(ids, v.Order.ID)
	}
	require.ElementsMatch(t, []string{"first", "last", "unknown"}, ids)
	rows, e = r.TemuOrderAmountRecords(ctx, "org-other", source.ID, q, 251)
	require.NoError(t, e)
	require.Empty(t, rows)
	insert("invalid", "invalid-date")
	_, e = r.TemuOrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.Error(t, e)
}
