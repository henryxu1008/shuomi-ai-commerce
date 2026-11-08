package observations

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"strconv"
	"strings"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type tiktokAmountAccess struct {
	*serviceAccess
	binding Binding
	after   func()
}

func (a *tiktokAmountAccess) ReadSavedTikTokOrderBinding(context.Context, Scope, string) (Binding, error) {
	if a.err != nil {
		return Binding{}, a.err
	}
	return a.binding, nil
}

type tiktokAmountRepo struct {
	*amountRepo
	rows []TikTokOrderAmountRecord
}

func (r *tiktokAmountRepo) TikTokOrderAmountRecords(context.Context, string, string, OrderAmountQuery, int) ([]TikTokOrderAmountRecord, error) {
	if r.after != nil {
		r.after()
	}
	return r.rows, nil
}
func tiktokAmountFixture(t *testing.T) (*Service, *tiktokAmountRepo, *tiktokAmountAccess, OrderAmountQuery) {
	s, base, a, _, q := amountFixture(t)
	r := &tiktokAmountRepo{amountRepo: base}
	source := &r.heads[0]
	b := Binding{OrganizationID: "org-a", StoreID: q.StoreID, Platform: storecenter.PlatformTikTokShop, Site: "tiktokshop-us", ApplicationID: "app", ApplicationRevision: "revision", ScopesHash: strings.Repeat("a", 64), SupplierIdentityHash: strings.Repeat("b", 64), CredentialExpiresAt: q.End.Add(24 * time.Hour), TikTok: storecenter.TikTokMerchantBinding{GrantID: uuid.NewString(), ShopID: "shop", GrantGeneration: 1, GrantRevision: 1, BindingRevision: 1}}
	source.Binding = b
	source.Range.End = q.End
	source.Progress.TikTokOrders = &TikTokOrderProgress{Complete: true}
	r.latest = r.heads
	local := b
	local.ScopesHash = ""
	access := &tiktokAmountAccess{serviceAccess: a, binding: local}
	s.Access = access
	s.Repository = r
	return s, r, access, q
}
func TestTikTokSavedAmountsRetainSecondsAndHideMissingCoverage(t *testing.T) {
	for _, kind := range []string{"complete", "last second", "missing", "coverage", "partition", "changed", "no sync"} {
		t.Run(kind, func(t *testing.T) {
			s, r, a, q := tiktokAmountFixture(t)
			o := TikTokOrder{ID: "po", Site: "tiktokshop-us", Date: q.End.Add(-time.Second).Format(time.RFC3339Nano), Amount: &Price{Value: "19.90", Currency: "USD"}}
			r.rows = []TikTokOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: o}}
			switch kind {
			case "last second":
				r.rows[0].Order.Date = q.End.Format(time.RFC3339Nano)
			case "missing":
				r.rows[0].Order.Amount.Value = ""
			case "coverage":
				r.heads[0].Range.End = q.End.Add(-time.Second)
			case "partition":
				r.heads[0].Progress.TikTokOrders.Complete = false
			case "changed":
				r.after = func() { a.err = ErrForbidden }
			case "no sync":
				r.heads = []Sync{}
				r.latest = []Sync{}
			}
			got, e := s.SavedTikTokOrderAmounts(context.Background(), Scope{"org-a", "actor", "member"}, q)
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

func TestTikTokAmountsKeepUnknownTimesExactSubtotalsAndSourceFences(t *testing.T) {
	s, r, a, q := tiktokAmountFixture(t)
	scope := Scope{"org-a", "actor", "member"}
	r.rows = []TikTokOrderAmountRecord{{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: TikTokOrder{ID: "unknown", Site: "tiktokshop-us", Amount: &Price{Value: "0", Currency: "USD"}}}}
	got, e := s.SavedTikTokOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "missing_fields", got.Status)
	require.Equal(t, 1, got.Missing.UnknownTimeCount)
	require.Zero(t, *got.MatchedOrderCount)
	r.rows = nil
	for i := 0; i < 250; i++ {
		r.rows = append(r.rows, TikTokOrderAmountRecord{StoreID: q.StoreID, SyncID: r.heads[0].ID, Order: TikTokOrder{ID: strconv.Itoa(i), Site: "tiktokshop-us", Date: q.Start.Format(time.RFC3339), Amount: &Price{Value: "99999999999999999999.99999999", Currency: "USD"}}})
	}
	r.rows[0].Order.Amount.Value = "99999999999999999999.99999998"
	got, e = s.SavedTikTokOrderAmounts(context.Background(), scope, q)
	require.NoError(t, e)
	require.Equal(t, "24999999999999999999999.99999749", (*got.Amounts)[0].Amount)
	r.after = func() { r.heads[0].Progress.TikTokOrders.Complete = false }
	got, e = s.SavedTikTokOrderAmounts(context.Background(), scope, q)
	require.Error(t, e)
	require.Nil(t, got.Amounts)
	r.heads[0].Progress.TikTokOrders.Complete = true
	r.after = func() { a.binding.TikTok.GrantRevision++ }
	got, e = s.SavedTikTokOrderAmounts(context.Background(), scope, q)
	require.Error(t, e)
	require.Nil(t, got.Amounts)
}
