package shopdiagnosis

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/agent"
	c "task-processor/internal/operationscockpit"
	o "task-processor/internal/storecenter/observations"
	"testing"
	"time"
)

const shopID = "22222222-2222-4222-8222-222222222222"
const contextID = "11111111-1111-4111-8111-111111111111"

var testPeriod = c.Period{Start: "2026-10-01", End: "2026-10-07"}
var testIdentity = agent.ExecutionIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member"}

type nativeFixture struct {
	access              c.Access
	snapshot            c.Snapshot
	platform            string
	reads, observations int
	view                *o.Result
}

func (f *nativeFixture) Current(context.Context, c.Scope) (c.Access, error)  { return f.access, nil }
func (f *nativeFixture) ReadStores(context.Context, c.Scope, []string) error { return nil }
func (f *nativeFixture) ListStores(context.Context, c.Scope) ([]c.StoreReference, error) {
	return []c.StoreReference{{ID: shopID, Name: "Saved shop", Platform: f.platform, Status: "active"}}, nil
}
func (f *nativeFixture) Snapshot(context.Context, c.Scope, c.Query) (c.Snapshot, error) {
	f.reads++
	return f.snapshot, nil
}
func (f *nativeFixture) List(_ context.Context, _ o.Scope, query o.Query) (o.Result, error) {
	f.observations++
	if f.view != nil {
		view := *f.view
		view.Syncs = append([]o.Sync{}, view.Syncs...)
		view.Latest = append([]o.Sync{}, view.Latest...)
		for n := range view.Syncs {
			view.Syncs[n].Kind = query.Kind
		}
		for n := range view.Latest {
			view.Latest[n].Kind = query.Kind
		}
		return view, nil
	}
	return o.Result{}, o.ErrUnavailable
}

func TestDiagnosisMissingSavedSourceIsNotZeroActivity(t *testing.T) {
	f, i := fixture()
	f.platform = "shein"
	f.view = &o.Result{}
	_, out, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.Contains(t, out.Missing, "observations.orders.counts")
	for _, fact := range out.Facts {
		require.NotEqual(t, "observations.orders.counts", fact.Path)
	}
}

func TestDiagnosisKeepsSourceRangeBeforeOptionalDisplayFieldsAndIgnoresToday(t *testing.T) {
	f, i := fixture()
	f.platform = "shein"
	at := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	// Each native query returns one kind's saved head with its own original range.
	f.view = &o.Result{Complete: true, Summary: o.Summary{Total: 12, Today: 4}, Syncs: []o.Sync{{ID: contextID, StoreID: shopID, Kind: o.Products, Status: "completed", ObservedAt: &at, Range: &o.Window{Start: at.Add(-24 * time.Hour), End: at}}}}
	b, _, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	f.view.Summary.Today = 9
	f.view.Summary.TodayUnknown = 7
	next, _, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.Equal(t, b, next, "read-clock counters cannot change saved evidence")
	// Project native facts directly to exercise a rich, already authorized view.
	n := NativeFacts{Store: c.StoreReference{ID: shopID, Name: "Saved", Platform: "shein"}, Current: f.snapshot.Stores[shopID]}
	n.Observations[0] = SavedObservation{State: "available", Complete: true, Sources: []c.ObservationEvidence{{StoreID: shopID, SyncID: contextID, Status: "completed", ObservedAt: &at, Start: &at, End: &at}}, Latest: []c.ObservationEvidence{{StoreID: shopID, SyncID: contextID, Status: "completed", ObservedAt: &at, Start: &at, End: &at}}}
	n.Observations[1] = n.Observations[0]
	out, e := project(n, testIdentity)
	require.NoError(t, e)
	for _, path := range []string{"observations.products.source", "observations.products.latest", "observations.orders.source", "observations.orders.latest"} {
		found := false
		for _, fact := range out.Facts {
			if fact.Path == path {
				found = true
				require.Contains(t, fact.Value, "2026-09-30")
				require.LessOrEqual(t, len(fact.Value), 512)
			}
		}
		require.True(t, found, path)
	}
	require.NotContains(t, out.Missing, "observations.orders.source")
}
func fixture() (*nativeFixture, Inspector) {
	agg, _ := c.Aggregate(testPeriod, []c.Record{{ID: contextID, StoreID: shopID, Revision: 1, Period: testPeriod, Amounts: c.Amounts{Revenue: 10000, Procurement: 5000}}})
	agg.StoreID = shopID
	f := &nativeFixture{access: c.Access{StoresRead: true, AdviceRead: true}, platform: "temu", snapshot: c.Snapshot{Period: testPeriod, Stores: map[string]c.StoreAggregate{shopID: agg}}}
	return f, Inspector{Reader: NativeReader{Repository: f, Boundary: f, Observations: f}, Identity: func(context.Context) (agent.ExecutionIdentity, error) { return testIdentity, nil }}
}
func TestDiagnosisRequiresBothFinancialPermissionsBeforeReading(t *testing.T) {
	for _, access := range []c.Access{{StoresRead: true}, {AdviceRead: true}, {}} {
		f, i := fixture()
		f.access = access
		_, out, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
		require.ErrorIs(t, e, c.ErrForbidden)
		require.Empty(t, out.Facts)
		require.Zero(t, f.reads)
		require.Zero(t, f.observations)
	}
}
func TestDiagnosisClockIndependentDigestAndTemuHasNoProviderPath(t *testing.T) {
	f, i := fixture()
	b, out, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.Zero(t, f.observations, "never enter TEMU saved-read branch which queries the provider")
	raw, _ := json.Marshal(out)
	require.Contains(t, string(raw), "offline_unavailable")
	require.Contains(t, string(raw), "CNY_fen")
	f.snapshot.CapturedAt = time.Now()
	f.snapshot.Today = "2030-01-01"
	next, _, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.Equal(t, b, next)
	a := f.snapshot.Stores[shopID]
	a.Records[0].Revision++
	f.snapshot.Stores[shopID] = a
	_, e = i.Inspect(context.Background(), b)
	require.ErrorIs(t, e, agent.ErrConflict)
}
func TestDiagnosisPreservesFullRevisionDigestWhenDisplayReferencesAreOmitted(t *testing.T) {
	f, i := fixture()
	a := f.snapshot.Stores[shopID]
	for n := 0; n < 100; n++ {
		a.Records = append(a.Records, c.RecordEvidence{ID: strings.Repeat("x", 128), Revision: int64(n + 1), Period: testPeriod})
	}
	f.snapshot.Stores[shopID] = a
	b, out, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.Contains(t, out.Missing, "financial.records")
	require.LessOrEqual(t, len(out.Facts), 32)
	raw, _ := json.Marshal(out)
	require.LessOrEqual(t, len(raw), 32<<10)
	a.Records[80].Revision++
	f.snapshot.Stores[shopID] = a
	next, _, e := i.Prepare(context.Background(), contextID, shopID, testPeriod)
	require.NoError(t, e)
	require.NotEqual(t, b.StoreDiagnosis.EvidenceDigest, next.StoreDiagnosis.EvidenceDigest)
	require.NotContains(t, string(raw), "CapturedAt")
}
