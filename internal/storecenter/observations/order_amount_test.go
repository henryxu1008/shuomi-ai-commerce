package observations

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type amountRepo struct {
	*serviceRepo
	amounts []OrderAmountRecord
	after   func()
	query   OrderAmountQuery
}

func (r *amountRepo) OrderAmountRecords(_ context.Context, _ string, syncID string, q OrderAmountQuery, limit int) ([]OrderAmountRecord, error) {
	r.query = q
	if r.after != nil {
		r.after()
	}
	return r.amounts, nil
}

type amountDirectory struct {
	ids []string
	err error
}

func (d *amountDirectory) ListStores(context.Context, Scope, Kind, string) ([]string, error) {
	return d.ids, d.err
}
func amountFixture(t *testing.T) (*Service, *amountRepo, *serviceAccess, *serviceMerchant, OrderAmountQuery) {
	t.Helper()
	now := time.Date(2026, 11, 8, 5, 0, 0, 0, time.UTC)
	start := time.Date(2026, 11, 6, 16, 0, 0, 0, time.UTC)
	store, syncID := uuid.NewString(), uuid.NewString()
	binding := Binding{OrganizationID: "org-a", StoreID: store, Platform: storecenter.PlatformShein, Site: "shein-us"}
	source := Sync{ID: syncID, StoreID: store, Kind: Orders, Owner: Scope{"org-a", "sync-actor", "sync-member"}, Binding: binding, Status: "completed", Revision: 2, Generation: 1, ObservedAt: &now, Range: &Window{Start: start.Add(-29 * 24 * time.Hour), End: start.Add(24*time.Hour - time.Second)}}
	r := &amountRepo{serviceRepo: &serviceRepo{heads: []Sync{source}, latest: []Sync{source}}}
	m := &serviceMerchant{binding: binding}
	a := &serviceAccess{merchant: m}
	s := &Service{Repository: r, Access: a, Directory: &amountDirectory{ids: []string{store}}, Now: func() time.Time { return now }}
	return s, r, a, m, OrderAmountQuery{StoreID: store, Start: start, End: start.Add(24 * time.Hour)}
}
func amountRow(r *amountRepo, q OrderAmountQuery, id, value, currency string, at time.Time) OrderAmountRecord {
	return OrderAmountRecord{StoreID: q.StoreID, SyncID: r.heads[0].ID, ID: id, Site: "shein-us", IssuedAt: at.Format(time.RFC3339), Amount: &Price{Value: value, Currency: currency}}
}
func TestSavedOrderAmountUsesListDateExactDecimalsAndOriginalOrderIdentity(t *testing.T) {
	s, r, _, _, q := amountFixture(t)
	one := amountRow(r, q, "order-1", "0.10", "USD", q.Start)
	r.amounts = []OrderAmountRecord{one, one, amountRow(r, q, "order-2", "0.20", "USD", q.End.Add(-time.Second)), amountRow(r, q, "order-3", "99999999999999999999.00000001", "CNY", q.Start), amountRow(r, q, "end-excluded", "123", "USD", q.End)}
	got, e := s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
	require.NoError(t, e)
	require.Equal(t, "complete", got.Status)
	require.Equal(t, 3, *got.MatchedOrderCount)
	require.Equal(t, []CurrencyOrderAmount{{Currency: "CNY", Amount: "99999999999999999999.00000001", OrderCount: 1}, {Currency: "USD", Amount: "0.30", OrderCount: 2}}, *got.Amounts)
	require.Equal(t, q, r.query)
}
func TestSavedOrderAmountHidesValuesForMissingFieldsAndCoverage(t *testing.T) {
	for _, kind := range []string{"amount", "currency", "time", "coverage", "newer", "limited"} {
		t.Run(kind, func(t *testing.T) {
			s, r, _, _, q := amountFixture(t)
			row := amountRow(r, q, "order-1", "1.25", "USD", q.Start)
			want := "missing_fields"
			switch kind {
			case "amount":
				row.Amount.Value = ""
			case "currency":
				row.Amount.Currency = ""
			case "time":
				row.IssuedAt = ""
			case "coverage":
				r.heads[0].Range.End = q.End.Add(-2 * time.Second)
				r.latest = r.heads
				want = "coverage_incomplete"
			case "newer":
				next := r.heads[0]
				next.ID = uuid.NewString()
				next.Generation++
				next.Status = "running"
				r.latest = []Sync{next}
				want = "coverage_incomplete"
			case "limited":
				r.amounts = make([]OrderAmountRecord, 251)
				want = "limited"
			}
			if kind != "limited" {
				r.amounts = []OrderAmountRecord{row}
			}
			got, e := s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
			require.NoError(t, e)
			require.Equal(t, want, got.Status)
			require.Nil(t, got.Amounts)
			if kind == "time" {
				require.Zero(t, *got.MatchedOrderCount)
				require.Equal(t, 1, got.Missing.UnknownTimeCount)
			}
		})
	}
}
func TestSavedOrderAmountDiscardsChangedScopeAndConflictingDuplicates(t *testing.T) {
	for _, kind := range []string{"scope", "binding", "head", "duplicate"} {
		t.Run(kind, func(t *testing.T) {
			s, r, a, m, q := amountFixture(t)
			row := amountRow(r, q, "order-1", "2.00", "USD", q.Start)
			r.amounts = []OrderAmountRecord{row}
			r.after = func() {
				switch kind {
				case "scope":
					a.err = ErrForbidden
				case "binding":
					m.err = ErrConflict
				case "head":
					r.heads[0].Revision++
				}
			}
			if kind == "duplicate" {
				bad := row
				bad.Amount = &Price{Value: "3.00", Currency: "USD"}
				r.amounts = append(r.amounts, bad)
			}
			got, e := s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
			require.Error(t, e)
			require.Nil(t, got.Amounts)
		})
	}
}

func TestSavedOrderAmountEmptyAndBadQueryDoNotInventCurrency(t *testing.T) {
	s, r, _, _, q := amountFixture(t)
	r.heads[0].Progress.Windows = []Window{}
	r.latest[0].Progress.Windows = []Window{}
	got, e := s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
	require.NoError(t, e)
	require.Equal(t, "complete", got.Status)
	require.Zero(t, *got.MatchedOrderCount)
	require.Empty(t, *got.Amounts)
	q.Start = q.Start.Add(time.Hour)
	_, e = s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
	require.ErrorIs(t, e, ErrInvalid)
}

func TestSavedOrderAmountWithoutSyncExplainsCoverage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		heads, latest []Sync
	}{
		{name: "nil"},
		{name: "empty", heads: []Sync{}, latest: []Sync{}},
		{name: "empty heads", heads: []Sync{}},
		{name: "empty latest", latest: []Sync{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, r, _, _, q := amountFixture(t)
			r.heads, r.latest = tc.heads, tc.latest
			got, err := s.SavedOrderAmounts(context.Background(), Scope{"org-a", "actor-a", "member-a"}, q)
			require.NoError(t, err)
			require.Equal(t, "coverage_incomplete", got.Status)
			require.Equal(t, "no_saved_sync", got.Reason)
			require.Nil(t, got.Source)
			require.Nil(t, got.MatchedOrderCount)
			require.Nil(t, got.Amounts)
		})
	}
}
