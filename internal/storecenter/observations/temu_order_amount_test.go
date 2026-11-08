package observations

import (
	"context"

	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type temuAmountAccess struct {
	*serviceAccess
	binding Binding
	after   func()
}

func (a *temuAmountAccess) ReadSavedTemuOrderBinding(context.Context, Scope, string) (Binding, error) {
	if a.err != nil {
		return Binding{}, a.err
	}
	return a.binding, nil
}

type temuAmountRepo struct {
	*amountRepo
	rows []TemuOrderAmountRecord
}

func (r *temuAmountRepo) TemuOrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]TemuOrderAmountRecord, error) {
	if r.after != nil {
		r.after()
	}
	return r.rows, nil
}
func temuAmountFixture(t *testing.T) (*Service, *temuAmountRepo, *temuAmountAccess, OrderAmountQuery) {
	s, base, a, _, q := amountFixture(t)
	r := &temuAmountRepo{amountRepo: base}
	source := &r.heads[0]
	b := Binding{OrganizationID: "org-a", StoreID: q.StoreID, Platform: storecenter.PlatformTemu, Site: "temu-order-us", RegionID: 211, ApplicationID: "app", ApplicationRevision: "revision", ScopesHash: strings.Repeat("a", 64), SupplierIdentityHash: strings.Repeat("b", 64), CredentialExpiresAt: q.End.Add(24 * time.Hour)}
	b = testTemuOrderBinding(b)
	source.Binding = b
	source.Range.End = q.End
	source.Progress.TemuOrders = &TemuOrderProgress{Complete: true}
	r.latest = r.heads
	local := b
	local.ScopesHash = b.ScopesHash
	access := &temuAmountAccess{serviceAccess: a, binding: local}
	s.Access = access
	s.Repository = r
	return s, r, access, q
}
func TestTemuSavedAmountsRetainSecondsAndHideMissingCoverage(t *testing.T) {
	for _, kind := range []string{"complete", "last second", "missing", "coverage", "partition", "changed", "no sync"} {
		t.Run(kind, func(t *testing.T) {
			s, r, a, q := temuAmountFixture(t)
			o := TemuOrder{ID: "po", Site: "temu-order-us", RegionID: 211, Date: q.End.Add(-time.Second).Format(time.RFC3339Nano), Amount: &Price{Value: "19.90", Currency: "USD"}}
			r.rows = []TemuOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: o}}
			switch kind {
			case "last second":
				r.rows[0].Order.Date = q.End.Format(time.RFC3339Nano)
			case "missing":
				r.rows[0].Order.Amount.Value = ""
			case "coverage":
				r.heads[0].Range.End = q.End.Add(-time.Second)
			case "partition":
				r.heads[0].Progress.TemuOrders.Complete = false
			case "changed":
				r.after = func() { a.err = ErrForbidden }
			case "no sync":
				r.heads = []Sync{}
				r.latest = []Sync{}
			}
			got, e := s.SavedTemuOrderAmounts(context.Background(), Scope{"org-a", "actor", "member"}, q)
			if kind == "changed" {
				require.Error(t, e)
				require.Nil(t, got.Amounts)
				return
			}
			require.NoError(t, e)
			switch kind {
			case "complete":
				require.Equal(t, "19.90", (*got.Amounts)[0].Amount)
			case "last second":
				require.Zero(t, *got.MatchedOrderCount)
				require.Empty(t, *got.Amounts)
			case "missing":
				require.Equal(t, "missing_fields", got.Status)
				require.Equal(t, 1, got.Missing.AmountCount)
				require.Nil(t, got.Amounts)
			default:
				require.Equal(t, "coverage_incomplete", got.Status)
				require.Nil(t, got.Amounts)
				require.Nil(t, got.MatchedOrderCount)
			}
		})
	}
}

func TestTemuAmountsKeepUnknownTimesExactSubtotalsAndSourceFences(t *testing.T) {
	s, r, a, q := temuAmountFixture(t)
	scope := Scope{"org-a", "actor", "member"}
	r.rows = []TemuOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: TemuOrder{ID: "unknown", Site: "temu-order-us", RegionID: 211, Amount: &Price{Value: "0.00", Currency: "USD"}}}}
	got, e := s.SavedTemuOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "missing_fields", got.Status)
	require.Equal(t, 1, got.Missing.UnknownTimeCount)
	require.Zero(t, *got.MatchedOrderCount)
	r.rows = nil
	for i := 0; i < 250; i++ {
		r.rows = append(r.rows, TemuOrderAmountRecord{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: TemuOrder{ID: strconv.Itoa(i), Site: "temu-order-us", RegionID: 211, Date: q.Start.Format(time.RFC3339), Amount: &Price{Value: "92233720368547758.07", Currency: "USD"}}})
	}
	r.rows[0].Order.Amount.Value = "92233720368547758.06"
	got, e = s.SavedTemuOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "23058430092136939517.49", (*got.Amounts)[0].Amount)
	r.after = func() { r.heads[0].Progress.TemuOrders.Complete = false }
	got, e = s.SavedTemuOrderAmounts(context.Background(), scope, q)
	require.Error(t, e)
	require.Nil(t, got.Amounts)
	r.heads[0].Progress.TemuOrders.Complete = true
	r.after = func() { a.binding.ConnectionRevision++ }
	got, e = s.SavedTemuOrderAmounts(context.Background(), scope, q)
	require.Error(t, e)
	require.Nil(t, got.Amounts)
}
