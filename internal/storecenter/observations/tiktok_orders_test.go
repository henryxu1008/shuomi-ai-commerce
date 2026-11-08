package observations

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type tikTokPages struct {
	page  TikTokOrderPage
	query TikTokOrderQuery
	after func()
}

func (r *tikTokPages) ReadTikTokOrders(_ context.Context, _ Scope, _ Binding, q TikTokOrderQuery) (TikTokOrderPage, error) {
	r.query = q
	if r.after != nil {
		r.after()
	}
	return r.page, nil
}
func TestTikTokOrdersUseHalfOpenWindowsAndRecheckRevocation(t *testing.T) {
	for _, test := range []string{"last-second", "next-window", "revoked", "unconfigured"} {
		t.Run(test, func(t *testing.T) {
			s, r, _, m := serviceFixture()
			r.sync.Kind = Orders
			r.sync.Binding.Platform = storecenter.PlatformTikTokShop
			r.sync.Binding.Site = "tiktokshop-us"
			m.binding = r.sync.Binding
			end := r.sync.CreatedAt
			start := end.Add(-time.Hour)
			r.sync.Range = &Window{Start: start, End: end}
			r.sync.Progress.Windows = []Window{*r.sync.Range}
			reader := &tikTokPages{page: TikTokOrderPage{Total: 1, Items: []TikTokOrder{{ID: "one", Site: "tiktokshop-us", Date: end.Add(-time.Second).Format(time.RFC3339)}}}}
			s.TikTokOrders = reader
			if test == "next-window" {
				reader.page.Items[0].Date = end.Format(time.RFC3339)
			}
			if test == "revoked" {
				reader.after = func() { m.err = ErrForbidden }
			}
			if test == "unconfigured" {
				s.TikTokOrders = nil
			}
			got, e := s.Step(context.Background(), "org-a", r.sync.ID)
			if test == "next-window" || test == "unconfigured" {
				require.ErrorIs(t, e, ErrUnavailable)
				require.Zero(t, r.commits)
				return
			}
			require.NoError(t, e)
			switch test {
			case "last-second":
				require.Equal(t, "completed", got.Status)
				require.True(t, got.Progress.TikTokOrders.Complete)
				require.Equal(t, start, reader.query.Start)
				require.Equal(t, end, reader.query.End)
				require.Len(t, r.rows, 1)
				require.Nil(t, r.rows[0].Order)
			case "revoked":
				require.Equal(t, "suspended", got.Status)
				require.Zero(t, r.commits)
			default:
				require.NotEqual(t, "completed", got.Status)
				require.Zero(t, r.commits)
			}
		})
	}
}

func TestTikTokCursorRequiresExhaustionAndConsistentCoverage(t *testing.T) {
	w := Window{time.Now().Add(-time.Hour), time.Now()}
	c := Checkpoint{Page: 1, Windows: []Window{w}}
	n, status, e := advanceTikTokOrders(c, 1, 1, "next")
	require.NoError(t, e)
	require.Equal(t, "running", status)
	require.Equal(t, 2, n.Page)
	// Empty page with a new token still continues; cycles cannot certify coverage.
	n, status, e = advanceTikTokOrders(n, 0, 1, "second")
	require.NoError(t, e)
	require.Equal(t, "running", status)
	cyclic, status, e := advanceTikTokOrders(n, 0, 1, "next")
	require.NoError(t, e)
	require.Equal(t, "partial", status)
	require.True(t, cyclic.Incomplete)
	final, status, e := advanceTikTokOrders(n, 0, 1, "")
	require.NoError(t, e)
	require.Equal(t, "completed", status)
	require.True(t, final.TikTokOrders.Complete)
	bad, status, e := advanceTikTokOrders(c, 2, 1, "")
	require.NoError(t, e)
	require.Equal(t, "partial", status)
	require.False(t, bad.TikTokOrders.Complete)
	require.Nil(t, c.TikTokOrders)
	clone := copyCheckpoint(n)
	*clone.TikTokOrders.Total = 99
	clone.TikTokOrders.TokenHashes[0] = "changed"
	require.Equal(t, 1, *n.TikTokOrders.Total)
	require.NotEqual(t, "changed", n.TikTokOrders.TokenHashes[0])
	n.TikTokOrders.Token = "PRIVATE_TOKEN"
	bytes, e := json.Marshal(Sync{Progress: n})
	require.NoError(t, e)
	require.NotContains(t, string(bytes), "PRIVATE_TOKEN")
	require.NotContains(t, string(bytes), n.TikTokOrders.TokenHashes[0])
	require.True(t, (TikTokOrder{ID: "o", Site: "tiktokshop-us", Amount: &Price{Value: "0", Currency: "USD"}}).Valid())
	require.False(t, (TikTokOrder{ID: "o", Site: "tiktokshop-us", Amount: &Price{Value: strings.Repeat("9", 21), Currency: "USD"}}).Valid())
}
