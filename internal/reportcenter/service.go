package reportcenter

import "context"

func (s *Service) authorize(ctx context.Context, scope Scope, manage bool) (context.Context, error) {
	if !scope.Valid() || s == nil || s.Repository == nil || s.Authorize == nil {
		return nil, ErrForbidden
	}
	return s.Authorize(ctx, scope, manage)
}
func (s *Service) Source(ctx context.Context, scope Scope, kind, id string) (SourceInfo, error) {
	ctx, e := s.authorize(ctx, scope, false)
	if e != nil {
		return SourceInfo{}, e
	}
	if !KindValid(kind) || kind == StoreDailyReportKind || !UUID(id) {
		return SourceInfo{}, ErrInvalid
	}
	if s.Sources == nil {
		return SourceInfo{}, ErrUnavailable
	}
	snap, e := s.Sources.Read(ctx, scope, kind, id)
	if e != nil {
		return SourceInfo{}, e
	}
	if snap.Ref.Kind != kind || snap.Ref.ID != id {
		return SourceInfo{}, ErrUnavailable
	}
	if _, _, e = snap.Bytes(); e != nil {
		return SourceInfo{}, e
	}
	return snap.SourceInfo, nil
}
func (s *Service) Save(ctx context.Context, scope Scope, key string, ref SourceRef) (Result, error) {
	ctx, e := s.authorize(ctx, scope, true)
	if e != nil {
		return Result{}, e
	}
	if !UUID(key) || !ref.Valid() || ref.Kind == StoreDailyReportKind {
		return Result{}, ErrInvalid
	}
	fp := Fingerprint("save", ref)
	id, e := s.Repository.Lookup(ctx, scope, key, fp)
	if e != nil {
		return Result{}, e
	}
	if id != "" {
		r, e := s.Repository.Read(ctx, scope, id)
		return Result{CommandID: key, Report: r, Replayed: true}, e
	}
	if s.Sources == nil {
		return Result{}, ErrUnavailable
	}
	snap, e := s.Sources.Read(ctx, scope, ref.Kind, ref.ID)
	if e != nil {
		return Result{}, e
	}
	if snap.Ref != ref {
		return Result{}, ErrConflict
	}
	if _, _, e = snap.Bytes(); e != nil {
		return Result{}, e
	}
	result, e := s.Repository.Save(ctx, scope, key, fp, snap, func(c context.Context) error { _, e := s.authorize(c, scope, true); return e })
	result.CommandID = key
	return result, e
}
func (s *Service) Favorite(ctx context.Context, scope Scope, key, id string, target bool) (Result, error) {
	ctx, e := s.authorize(ctx, scope, true)
	if e != nil {
		return Result{}, e
	}
	if !UUID(key) || !UUID(id) {
		return Result{}, ErrInvalid
	}
	fp := Fingerprint("favorite", struct {
		ID     string
		Target bool
	}{id, target})
	report, e := s.Read(ctx, scope, id)
	if e != nil {
		return Result{}, e
	}
	result, e := s.Repository.Favorite(ctx, scope, key, fp, id, target, func(c context.Context) error {
		if _, e := s.authorize(c, scope, true); e != nil {
			return e
		}
		// Reports are immutable. Recheck current visibility against the report
		// already read, without borrowing another Report DB connection from
		// inside the repository transaction (including single-connection pools).
		if report.Ref.Kind == StoreDailyReportKind {
			fresh, e := s.dailyContext(c, scope)
			if e != nil {
				return e
			}
			if !dailyVisible(fresh, report) {
				return ErrNotFound
			}
		}
		return nil
	})
	result.CommandID = key
	return result, e
}
func (s *Service) Read(ctx context.Context, scope Scope, id string) (Report, error) {
	ctx, e := s.authorize(ctx, scope, false)
	if e != nil {
		return Report{}, e
	}
	if !UUID(id) {
		return Report{}, ErrInvalid
	}
	r, e := s.Repository.Read(ctx, scope, id)
	if e != nil || r.Ref.Kind != StoreDailyReportKind {
		return r, e
	}
	ctx, e = s.dailyContext(ctx, scope)
	if e != nil {
		return Report{}, e
	}
	if !dailyVisible(ctx, r) {
		return Report{}, ErrNotFound
	}
	return r, nil
}
func (s *Service) List(ctx context.Context, scope Scope, f Filter) (Page, error) {
	ctx, e := s.authorize(ctx, scope, false)
	if e != nil {
		return Page{}, e
	}
	if !f.Valid() {
		return Page{}, ErrInvalid
	}
	ctx, e = s.dailyContext(ctx, scope)
	if e != nil {
		return Page{}, e
	}
	page, e := s.Repository.List(ctx, scope, f)
	if e != nil {
		return Page{}, e
	}
	for _, item := range page.Items {
		if !dailyVisible(ctx, Report{ReportSummary: item}) {
			return Page{}, ErrUnavailable
		}
	}
	if e := s.recheckDailyVisibility(ctx, scope); e != nil {
		return Page{}, e
	}
	return page, nil
}
func (s *Service) Summary(ctx context.Context, scope Scope) (Summary, error) {
	ctx, e := s.authorize(ctx, scope, false)
	if e != nil {
		return Summary{}, e
	}
	ctx, e = s.dailyContext(ctx, scope)
	if e != nil {
		return Summary{}, e
	}
	summary, e := s.Repository.Summary(ctx, scope)
	if e != nil {
		return Summary{}, e
	}
	if e = s.recheckDailyVisibility(ctx, scope); e != nil {
		return Summary{}, e
	}
	return summary, nil
}
