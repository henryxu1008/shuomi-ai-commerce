//go:build integration

package storeobservations

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

func TestSavedOrderAmountPostgresFiltersBeforeLimitAndRetainsUnknownDates(t *testing.T) {
	ctx, db, repo := observationDatabase(t)
	scope := o.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	store := uuid.NewString()
	in, children := syncInput(scope, store)
	in.Kind = o.Orders
	children[0].Kind = o.Orders
	cmd, e := repo.Begin(ctx, scope, uuid.NewString(), "order-amount", in, children)
	require.NoError(t, e)
	source := cmd.Syncs[0]
	at := time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	insert := func(id, date string) {
		body, _ := json.Marshal(o.Record{ID: id, StoreID: store, SyncID: source.ID, Order: &o.Order{ID: id, Site: "shein-us", IssuedAt: date, CreatedAt: at.Add(-10 * 24 * time.Hour).Format(time.RFC3339), Amount: &o.Price{Value: "0.10", Currency: "USD"}}})
		require.NoError(t, db.Exec("INSERT INTO store_observations.records(organization_id,store_id,sync_id,kind,platform_id,payload,observed_at,page_index) VALUES(?,?,?,'orders',?,?::jsonb,?,1)", scope.OrganizationID, store, source.ID, id, string(body), at).Error)
	}
	for j := 0; j < 251; j++ {
		insert(uuid.NewString(), at.Add(-10*24*time.Hour).Format(time.RFC3339))
	}
	insert("first", at.Format(time.RFC3339))
	insert("last", at.Add(24*time.Hour-time.Second).Format(time.RFC3339))
	insert("excluded-end", at.Add(24*time.Hour).Format(time.RFC3339))
	insert("unknown", "")
	q := o.OrderAmountQuery{StoreID: store, Start: at, End: at.Add(24 * time.Hour)}
	rows, e := repo.OrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.NoError(t, e)
	require.Len(t, rows, 3)
	ids := []string{}
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	require.ElementsMatch(t, []string{"first", "last", "unknown"}, ids)
	rows, e = repo.OrderAmountRecords(ctx, "other-org", source.ID, q, 251)
	require.NoError(t, e)
	require.Empty(t, rows)
	q.StoreID = uuid.NewString()
	rows, e = repo.OrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.NoError(t, e)
	require.Empty(t, rows)
	q.StoreID = store
	insert("invalid", "not-a-date")
	_, e = repo.OrderAmountRecords(ctx, scope.OrganizationID, source.ID, q, 251)
	require.ErrorIs(t, e, o.ErrUnavailable)
}
