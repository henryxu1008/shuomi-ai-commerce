package observations

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type walmartPages struct {
	page  walmart.OrderPage
	query WalmartOrderQuery
	after func()
}

func (r *walmartPages) ReadWalmartOrders(_ context.Context, _ Scope, _ Binding, q WalmartOrderQuery) (walmart.OrderPage, error) {
	r.query = q
	if r.after != nil {
		r.after()
	}
	return r.page, nil
}
func TestWalmartOrdersUseThreePartitionsAndMillisDayBoundaries(t *testing.T) {
	s, r, _, m := serviceFixture()
	r.sync.Kind = Orders
	r.sync.Binding.Platform = storecenter.PlatformWalmart
	r.sync.Binding.Site = "walmart-us"
	m.binding = r.sync.Binding
	start := r.sync.CreatedAt.Add(-24 * time.Hour)
	end := r.sync.CreatedAt
	r.sync.Range = &Window{Start: start, End: end}
	r.sync.Progress.Windows = []Window{*r.sync.Range}
	reader := &walmartPages{page: walmart.OrderPage{Total: 1, Items: []walmart.Order{{ID: "po", Site: "walmart-us", Date: end.Add(-time.Millisecond).Format(time.RFC3339Nano), Lines: []walmart.OrderLine{{Number: "1", Charges: []walmart.OrderCharge{{Currency: "USD", Amount: "1.00"}}}}}}}}
	s.WalmartOrders = reader
	for partition := 0; partition < 3; partition++ {
		got, err := s.Step(context.Background(), "org-a", r.sync.ID)
		require.NoError(t, err)
		require.Equal(t, partition, reader.query.Partition)
		require.Equal(t, start.Add(-time.Second), reader.query.Start)
		require.Equal(t, end.Add(time.Second), reader.query.End)
		require.Len(t, r.rows, 1)
		require.NotNil(t, r.rows[0].WalmartOrder)
		require.Nil(t, r.rows[0].Order)
		if partition < 2 {
			require.Equal(t, "running", got.Status)
		} else {
			require.Equal(t, "completed", got.Status)
			require.True(t, got.Progress.WalmartOrders.Complete)
		}
		r.sync = got
	}
}
func TestWalmartCursorRequiresExhaustionAndNeverLeavesPublicSync(t *testing.T) {
	c := Checkpoint{Page: 1, Windows: []Window{{Start: time.Now().Add(-time.Hour), End: time.Now()}}}
	total := 2
	next, status, err := advanceWalmartOrders(c, 0, total, "private-cursor")
	require.NoError(t, err)
	require.Equal(t, "running", status)
	next, status, err = advanceWalmartOrders(next, 0, total, "private-cursor")
	require.NoError(t, err)
	require.Equal(t, "partial", status)
	require.True(t, next.Incomplete)
	raw, _ := json.Marshal(Sync{Progress: next})
	require.NotContains(t, string(raw), "private-cursor")
	require.NotContains(t, string(raw), "cursorHashes")
	copy := copyCheckpoint(next)
	copy.WalmartOrders.CursorHashes[0] = "changed"
	require.NotEqual(t, copy.WalmartOrders.CursorHashes, next.WalmartOrders.CursorHashes)
}

func TestWalmartOrdersRevocationAfterReadUsesOriginalSuspension(t *testing.T) {
	s, r, _, m := serviceFixture()
	r.sync.Kind = Orders
	r.sync.Binding.Platform = storecenter.PlatformWalmart
	m.binding = r.sync.Binding
	r.sync.Range = &Window{Start: r.sync.CreatedAt.Add(-time.Hour), End: r.sync.CreatedAt}
	r.sync.Progress.Windows = []Window{*r.sync.Range}
	s.WalmartOrders = &walmartPages{page: walmart.OrderPage{Total: 0}, after: func() { m.err = ErrForbidden }}
	got, e := s.Step(context.Background(), "org-a", r.sync.ID)
	require.NoError(t, e)
	require.Equal(t, "suspended", got.Status)
	require.Zero(t, r.commits)
	require.Equal(t, 1, r.stops)
}

func TestWalmartOrderBudgetsAndUnconfiguredReaderCannotClaimComplete(t *testing.T) {
	c := Checkpoint{Page: 1, Windows: []Window{{Start: time.Now().Add(-time.Hour), End: time.Now()}}}
	for _, limit := range []string{"count", "pages", "checkpoint"} {
		t.Run(limit, func(t *testing.T) {
			input := copyCheckpoint(c)
			total := 1
			switch limit {
			case "count":
				total = 10001
			case "pages":
				input.Pages = 29999
			case "checkpoint":
				input.Page = 512
			}
			got, status, e := advanceWalmartOrders(input, 1, total, "next")
			require.NoError(t, e)
			require.Equal(t, "partial", status)
			require.True(t, got.Incomplete)
			require.False(t, got.WalmartOrders.Complete)
		})
	}
	s, r, _, m := serviceFixture()
	r.sync.Kind = Orders
	r.sync.Binding.Platform = storecenter.PlatformWalmart
	m.binding = r.sync.Binding
	_, e := s.Step(context.Background(), "org-a", r.sync.ID)
	require.ErrorIs(t, e, ErrUnavailable)
	require.Zero(t, r.commits)
}
