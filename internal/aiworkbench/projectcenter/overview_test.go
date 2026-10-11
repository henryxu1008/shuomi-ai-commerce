package projectcenter

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type overviewRepo struct {
	repositorySpy
	snapshot   OverviewSnapshot
	start, end time.Time
	calls      int
}

func (r *overviewRepo) Overview(_ context.Context, scope Scope, mode string, start, end time.Time) (OverviewSnapshot, error) {
	r.calls++
	r.start = start
	r.end = end
	return r.snapshot, nil
}

type overviewReader struct {
	calls  int
	denied string
	state  string
}

func (r *overviewReader) Resolve(_ context.Context, _ Scope, ref Reference) (ReferenceView, error) {
	r.calls++
	if ref.TargetID == r.denied {
		return ReferenceView{}, ErrForbidden
	}
	return ReferenceView{Kind: ref.Kind, TargetID: ref.TargetID, Available: true, Title: "original source", Href: "/workbench/stores/a", TaskState: r.state}, nil
}
func TestOverviewUsesCompleteAuthorizedDistinctTaskFactsAndLocalDay(t *testing.T) {
	scope := Scope{"org-a", "actor-a"}
	id := "550e8400-e29b-41d4-a716-446655440000"
	repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: scope, ActiveProjects: 35, TodayUpdatedProjects: 25, StoreProjects: 20, GeneralProjects: 15, TaskIDs: []string{id, id}, StoreIDs: []string{"store-a", "denied"}}}
	reader := &overviewReader{state: "WAITING_CONFIRMATION", denied: "denied"}
	checks := 0
	s := Service{Store: repo, Reader: reader, Authorize: func(context.Context, Scope, bool) error { checks++; return nil }}
	out, e := s.overviewAt(context.Background(), scope, "active", "Asia/Singapore", time.Date(2026, 10, 11, 0, 30, 0, 0, time.UTC))
	require.NoError(t, e)
	require.EqualValues(t, 35, out.ActiveProjects)
	require.Equal(t, 1, *out.WaitingTasks)
	require.Equal(t, "2026-10-11", out.Today)
	require.Equal(t, time.Date(2026, 10, 10, 16, 0, 0, 0, time.UTC), repo.start.UTC())
	require.Equal(t, 24*time.Hour, repo.end.Sub(repo.start))
	require.Equal(t, 2, checks)
	require.False(t, out.StoresAvailable)
	require.Equal(t, []StoreOption{{ID: "store-a", Name: "original source"}}, out.Stores)
	require.Equal(t, 3, reader.calls)
	out, e = s.overviewAt(context.Background(), scope, "active", "America/New_York", time.Date(2026, 3, 8, 12, 0, 0, 0, time.UTC))
	require.NoError(t, e)
	require.Equal(t, 23*time.Hour, repo.end.Sub(repo.start))
	require.Equal(t, "2026-03-08", out.Today)
}
func TestOverviewNeverTurnsDeniedMissingOrBudgetedTasksIntoZero(t *testing.T) {
	scope := Scope{"org-a", "actor-a"}
	id := "550e8400-e29b-41d4-a716-446655440000"
	for _, scenario := range []string{"denied", "missing", "budget"} {
		t.Run(scenario, func(t *testing.T) {
			repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: scope, TaskIDs: []string{id}}}
			reader := &overviewReader{state: "COMPLETED"}
			var source ReferenceReader = reader
			if scenario == "denied" {
				reader.denied = id
			}
			if scenario == "missing" {
				source = nil
			}
			if scenario == "budget" {
				for i := 0; i < 200; i++ {
					repo.snapshot.TaskIDs = append(repo.snapshot.TaskIDs, id)
				}
			}
			s := Service{Store: repo, Reader: source, Authorize: func(context.Context, Scope, bool) error { return nil }}
			out, e := s.Overview(context.Background(), scope, "active", "UTC")
			require.NoError(t, e)
			require.Nil(t, out.WaitingTasks)
		})
	}
	repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: scope}}
	s := Service{Store: repo, Authorize: func(context.Context, Scope, bool) error { return nil }}
	out, e := s.Overview(context.Background(), scope, "active", "UTC")
	require.NoError(t, e)
	require.NotNil(t, out.WaitingTasks)
	require.Zero(t, *out.WaitingTasks)
}
func TestOverviewFailsClosedOnScopeInvalidZoneOrResponseAuthorization(t *testing.T) {
	scope := Scope{"org-a", "actor-a"}
	repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: Scope{"org-b", "actor-a"}}}
	s := Service{Store: repo, Authorize: func(context.Context, Scope, bool) error { return nil }}
	_, e := s.Overview(context.Background(), scope, "active", "../UTC")
	require.ErrorIs(t, e, ErrInvalid)
	require.Zero(t, repo.calls)
	_, e = s.Overview(context.Background(), scope, "active", "UTC")
	require.ErrorIs(t, e, ErrUnavailable)
	repo.snapshot.Scope = scope
	checks := 0
	s.Authorize = func(context.Context, Scope, bool) error {
		checks++
		if checks > 1 {
			return ErrForbidden
		}
		return nil
	}
	_, e = s.Overview(context.Background(), scope, "active", "UTC")
	require.ErrorIs(t, e, ErrForbidden)
}
func TestOverviewSourceDeadlineKeepsLocalFiguresAndUnknownWaiting(t *testing.T) {
	scope := Scope{"org-a", "actor-a"}
	repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: scope, ActiveProjects: 3, GeneralProjects: 3, TaskIDs: []string{"550e8400-e29b-41d4-a716-446655440000"}}}
	source := &blockedCardReader{}
	s := Service{Store: repo, Reader: source, Authorize: func(context.Context, Scope, bool) error { return nil }}
	start := time.Now()
	out, e := s.Overview(context.Background(), scope, "active", "UTC")
	require.NoError(t, e)
	require.Less(t, time.Since(start), 3*time.Second)
	require.EqualValues(t, 3, out.ActiveProjects)
	require.Nil(t, out.WaitingTasks)
	require.Equal(t, 1, source.calls)
	repo.snapshot.TaskIDs = nil
	repo.snapshot.StoreIDs = make([]string, 101)
	out, e = s.Overview(context.Background(), scope, "active", "UTC")
	require.NoError(t, e)
	require.False(t, out.StoresAvailable)
	require.Empty(t, out.Stores)
}

type lateOverviewReader struct{}

func (*lateOverviewReader) Resolve(ctx context.Context, _ Scope, ref Reference) (ReferenceView, error) {
	<-ctx.Done()
	return ReferenceView{Kind: ref.Kind, TargetID: ref.TargetID, Available: true, Title: "late source", Href: "/workbench/stores/a", TaskState: "WAITING_CONFIRMATION"}, nil
}
func TestOverviewRejectsLastSourceResultReturnedAfterItsDeadline(t *testing.T) {
	scope := Scope{"org-a", "actor-a"}
	repo := &overviewRepo{snapshot: OverviewSnapshot{Scope: scope, TaskIDs: []string{"550e8400-e29b-41d4-a716-446655440000"}, StoreIDs: []string{"store-a"}}}
	s := Service{Store: repo, Reader: &lateOverviewReader{}, Authorize: func(context.Context, Scope, bool) error { return nil }}
	out, e := s.Overview(context.Background(), scope, "active", "UTC")
	require.NoError(t, e)
	require.Nil(t, out.WaitingTasks)
	require.False(t, out.StoresAvailable)
	require.Empty(t, out.Stores)
}
