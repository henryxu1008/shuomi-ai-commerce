package observations

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"task-processor/internal/marketplace/walmart"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type walmartAmountAccess struct {
	*serviceAccess
	binding Binding
	after   func()
}

func TestWalmartAmountsKeepUnknownDatesSourceDriftAndMaximumPrecision(t *testing.T) {
	s, r, _, q := walmartAmountFixture(t)
	order := walmart.Order{ID: "po", Site: "walmart-us", Lines: []walmart.OrderLine{{Number: "1", Charges: []walmart.OrderCharge{{Currency: "USD", Amount: "0.10"}}}}}
	r.rows = []WalmartOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: order}}
	scope := Scope{"org-a", "actor", "member"}
	got, e := s.SavedWalmartOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "missing_fields", got.Status)
	require.Equal(t, 1, got.Missing.UnknownTimeCount)
	require.Zero(t, *got.MatchedOrderCount)
	require.Nil(t, got.Amounts)
	r.rows = nil
	for i := 0; i < 250; i++ {
		v := walmart.Order{ID: strconv.Itoa(i), Site: "walmart-us", Date: q.Start.Format(time.RFC3339Nano)}
		for line := 0; line < 100; line++ {
			v.Lines = append(v.Lines, walmart.OrderLine{Number: strconv.Itoa(line), Charges: []walmart.OrderCharge{{Currency: "USD", Amount: "99999999999999999999.99999999"}}})
		}
		r.rows = append(r.rows, WalmartOrderAmountRecord{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: v})
	}
	r.rows[0].Order.Lines[0].Charges[0].Amount = "99999999999999999999.99999998"
	got, e = s.SavedWalmartOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "2499999999999999999999999.99974999", (*got.Amounts)[0].Amount)
	r.after = func() { r.heads[0].Progress.WalmartOrders.Complete = false }
	got, e = s.SavedWalmartOrderAmounts(context.Background(), scope, q)
	require.Error(t, e)
	require.Nil(t, got.Amounts)
}

func (a *walmartAmountAccess) ReadSavedWalmartOrderBinding(context.Context, Scope, string) (Binding, error) {
	if a.err != nil {
		return Binding{}, a.err
	}
	return a.binding, nil
}

type walmartAmountRepo struct {
	*amountRepo
	rows []WalmartOrderAmountRecord
}

func (r *walmartAmountRepo) WalmartOrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]WalmartOrderAmountRecord, error) {
	if r.after != nil {
		r.after()
	}
	return r.rows, nil
}
func walmartAmountFixture(t *testing.T) (*Service, *walmartAmountRepo, *walmartAmountAccess, OrderAmountQuery) {
	s, base, a, _, q := amountFixture(t)
	r := &walmartAmountRepo{amountRepo: base}
	source := &r.heads[0]
	b := Binding{OrganizationID: "org-a", StoreID: q.StoreID, Platform: storecenter.PlatformWalmart, Site: "walmart-us", ApplicationID: "app", ApplicationRevision: "revision", ScopesHash: strings.Repeat("a", 64), SupplierIdentityHash: strings.Repeat("b", 64), CredentialExpiresAt: q.End.Add(24 * time.Hour), Walmart: storecenter.WalmartMerchantBinding{Environment: "sandbox", Market: "US", GrantID: uuid.NewString(), Generation: 1, GrantVersion: 1}}
	source.Binding = b
	source.Range.End = q.End
	source.Progress.WalmartOrders = &WalmartOrderProgress{Complete: true}
	r.latest = r.heads
	access := &walmartAmountAccess{serviceAccess: a, binding: b}
	s.Access = access
	s.Repository = r
	return s, r, access, q
}
func TestWalmartSavedAmountsRetainMillisAndHideMissingCoverage(t *testing.T) {
	for _, kind := range []string{"complete", "last second", "missing", "coverage", "partition", "changed", "no sync"} {
		t.Run(kind, func(t *testing.T) {
			s, r, a, q := walmartAmountFixture(t)
			o := walmart.Order{ID: "po", Site: "walmart-us", Date: q.End.Add(-time.Millisecond).Format(time.RFC3339Nano), Lines: []walmart.OrderLine{{Number: "1", Quantity: "9", Charges: []walmart.OrderCharge{{Currency: "USD", Amount: "19.90"}}}}}
			r.rows = []WalmartOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: o}}
			switch kind {
			case "last second":
				r.rows[0].Order.Date = q.End.Format(time.RFC3339Nano)
			case "missing":
				r.rows[0].Order.Lines[0].Charges[0].Amount = ""
			case "coverage":
				r.heads[0].Range.End = q.End.Add(-time.Second)
			case "partition":
				r.heads[0].Progress.WalmartOrders.Complete = false
			case "changed":
				r.after = func() { a.err = ErrForbidden }
			case "no sync":
				r.heads = []Sync{}
				r.latest = []Sync{}
			}
			got, e := s.SavedWalmartOrderAmounts(context.Background(), Scope{"org-a", "actor", "member"}, q)
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
