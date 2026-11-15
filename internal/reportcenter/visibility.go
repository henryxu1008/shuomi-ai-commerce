package reportcenter

import (
	"context"
	"slices"
)

type DailyVisibility struct {
	MemberID string
	StoreIDs []string
}
type dailyVisibilityKey struct{}

func DailyVisibilityFromContext(ctx context.Context) DailyVisibility {
	v, _ := ctx.Value(dailyVisibilityKey{}).(DailyVisibility)
	return v
}
func (s *Service) dailyContext(ctx context.Context, scope Scope) (context.Context, error) {
	var v DailyVisibility
	if s.DailyVisibility != nil {
		var e error
		v, e = s.DailyVisibility(ctx, scope)
		if e != nil {
			return nil, e
		}
	}
	v.StoreIDs = append([]string(nil), v.StoreIDs...)
	slices.Sort(v.StoreIDs)
	return context.WithValue(ctx, dailyVisibilityKey{}, v), nil
}
func dailyVisible(ctx context.Context, r Report) bool {
	if r.Ref.Kind != StoreDailyReportKind {
		return true
	}
	v := DailyVisibilityFromContext(ctx)
	return r.ScheduledOrigin != nil && r.ScheduledOrigin.MemberID == v.MemberID && slices.Contains(v.StoreIDs, r.StoreID)
}

func (s *Service) recheckDailyVisibility(ctx context.Context, scope Scope) error {
	fresh, e := s.dailyContext(ctx, scope)
	if e != nil {
		return e
	}
	before, after := DailyVisibilityFromContext(ctx), DailyVisibilityFromContext(fresh)
	if before.MemberID != after.MemberID || !slices.Equal(before.StoreIDs, after.StoreIDs) {
		return ErrForbidden
	}
	return nil
}
