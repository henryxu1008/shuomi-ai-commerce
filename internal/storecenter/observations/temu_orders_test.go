package observations

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type temuPages struct {
	page  TemuOrderPage
	query TemuOrderQuery
	after func()
}

func (r *temuPages) ReadTemuOrders(_ context.Context, _ Scope, _ Binding, q TemuOrderQuery) (TemuOrderPage, error) {
	r.query = q
	if r.after != nil {
		r.after()
	}
	return r.page, nil
}
func TestTemuOrdersUseHalfOpenWindowsAndRecheckRevocation(t *testing.T) {
	for _, test := range []string{"last-second", "next-window", "revoked", "unconfigured"} {
		t.Run(test, func(t *testing.T) {
			s, r, _, m := serviceFixture()
			r.sync.Kind = Orders
			r.sync.Binding = testTemuOrderBinding(r.sync.Binding)
			r.sync.Binding.Site = "temu-order-us"
			m.binding = r.sync.Binding
			end := r.sync.CreatedAt
			start := end.Add(-time.Hour)
			r.sync.Range = &Window{Start: start, End: end}
			r.sync.Progress.Windows = []Window{*r.sync.Range}
			reader := &temuPages{page: TemuOrderPage{Total: 1, Items: []TemuOrder{{ID: "one", Site: "temu-order-us", RegionID: 211, Date: end.Add(-time.Second).Format(time.RFC3339)}}}}
			s.TemuOrders = reader
			if test == "next-window" {
				reader.page.Items[0].Date = end.Format(time.RFC3339)
			}
			if test == "revoked" {
				reader.after = func() { m.err = ErrForbidden }
			}
			if test == "unconfigured" {
				s.TemuOrders = nil
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
				require.True(t, got.Progress.TemuOrders.Complete)
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

func testTemuOrderBinding(b Binding) Binding {
	b.Platform = storecenter.PlatformTemu
	b.Site = "temu-order-us"
	b.IdentityPartition = storecenter.PartitionGlobal
	b.MerchantMode = storecenter.MerchantSemiManaged
	b.RegionID = 211
	b.ApplicationID = "app"
	b.ApplicationRevision = "v:temu:GLOBAL:manual_token:3:r2"
	b.VerificationRevision = "temu-regional-order-identity-v2"
	b.ScopesHash = strings.Repeat("a", 64)
	b.SupplierIdentityHash = strings.Repeat("b", 64)
	b.CredentialExpiresAt = time.Now().Add(time.Hour)
	return b
}
func TestTemuCoverageRequiresStableCountsAndFullPages(t *testing.T) {
	w := Window{time.Now().Add(-time.Hour), time.Now()}
	c := Checkpoint{Page: 1, Windows: []Window{w}}
	n, status, e := advanceTemuOrders(c, 100, 101)
	require.NoError(t, e)
	require.Equal(t, "running", status)
	require.Equal(t, 2, n.Page)
	clone := copyCheckpoint(n)
	*clone.TemuOrders.Total = 99
	require.Equal(t, 101, *n.TemuOrders.Total)
	done, status, e := advanceTemuOrders(n, 1, 101)
	require.NoError(t, e)
	require.Equal(t, "completed", status)
	require.True(t, done.TemuOrders.Complete)
	for _, pair := range [][2]int{{99, 101}, {0, 1}, {100, 10001}} {
		bad, status, e := advanceTemuOrders(c, pair[0], pair[1])
		require.NoError(t, e)
		require.Equal(t, "partial", status)
		require.False(t, bad.TemuOrders.Complete)
	}
	bad, status, e := advanceTemuOrders(n, 1, 102)
	require.NoError(t, e)
	require.Equal(t, "partial", status)
	require.False(t, bad.TemuOrders.Complete)
	q := TemuOrderQuery{Start: time.Unix(1000, 0), End: time.Unix(2000, 0), Page: 1}
	lo, hi, e := q.ClosedCreationRange()
	require.NoError(t, e)
	require.EqualValues(t, 1000, lo)
	require.EqualValues(t, 1999, hi)
}
