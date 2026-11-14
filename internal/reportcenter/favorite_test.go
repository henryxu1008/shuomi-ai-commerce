package reportcenter

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

type favoriteVisibilityRepository struct {
	Repository
	report   Report
	atCommit func()
	reads    int
}

func (r *favoriteVisibilityRepository) Read(context.Context, Scope, string) (Report, error) {
	r.reads++
	return r.report, nil
}
func (r *favoriteVisibilityRepository) Favorite(ctx context.Context, _ Scope, _, _, _ string, _ bool, check func(context.Context) error) (Result, error) {
	r.atCommit()
	return Result{}, check(ctx)
}
func TestFavoriteRechecksDailyVisibilityWithoutNestedReportRead(t *testing.T) {
	for _, revoke := range []bool{false, true} {
		t.Run(map[bool]string{false: "allowed", true: "revoked before write"}[revoke], func(t *testing.T) {
			id, store := uuid.NewString(), uuid.NewString()
			visible := true
			repository := &favoriteVisibilityRepository{report: Report{ReportSummary: ReportSummary{ID: id, SourceInfo: SourceInfo{Ref: SourceRef{Kind: StoreDailyReportKind}, StoreID: store, ScheduledOrigin: &ScheduledOrigin{MemberID: "member"}}}}, atCommit: func() { visible = !revoke }}
			service := Service{Repository: repository, Authorize: func(c context.Context, _ Scope, _ bool) (context.Context, error) { return c, nil }, DailyVisibility: func(context.Context, Scope) (DailyVisibility, error) {
				v := DailyVisibility{MemberID: "member"}
				if visible {
					v.StoreIDs = []string{store}
				}
				return v, nil
			}}
			_, err := service.Favorite(context.Background(), Scope{OrganizationID: "org", ActorID: "actor"}, uuid.NewString(), id, true)
			if revoke {
				require.ErrorIs(t, err, ErrNotFound)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, repository.reads, "the immutable report is read before the repository transaction only")
		})
	}
}
