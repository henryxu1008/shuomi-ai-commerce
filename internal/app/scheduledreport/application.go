// Package scheduledreport composes current Store/Cockpit/Report owners for a
// deterministic background report. It has no model, ledger or platform port.
package scheduledreport

import (
	"context"
	"errors"
	"gorm.io/gorm"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/integration/dataserviceauth"
	cockpitstore "task-processor/internal/integration/persistence/operationscockpit"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	c "task-processor/internal/operationscockpit"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter"
	"task-processor/internal/storecenter/observations"
	"task-processor/internal/workbenchcontext"
	"time"
)

type Dependencies struct {
	Observations *observations.Service
	Applications *storeapp.OfficialApplicationRegistry
	StoreDB      *gorm.DB
	ReportDB     *gorm.DB
	IAM          dataserviceauth.ExactReader
	Users        dataserviceauth.ActiveUserReader
	ServiceToken func(context.Context) (string, error)
	ProjectID    string
	Policy       *authz.ListingKitAuthorizer
	// Optional deny-only overlay, shared with the current Workbench resolver.
	// Nil means no additional local suspension policy is configured.
	OrganizationStatus workbenchcontext.OrganizationBusinessStatusChecker
}
type Application struct {
	d       Dependencies
	cockpit *cockpitstore.Store
	Capture *rc.ScheduledCapture
}
type originKey struct{}

func New(ctx context.Context, d Dependencies) (*Application, error) {
	if d.StoreDB == nil || d.ReportDB == nil || d.IAM == nil || d.Users == nil || d.ServiceToken == nil || d.ProjectID == "" || d.Policy == nil {
		return nil, rc.ErrUnavailable
	}
	if e := reportstore.VerifyDailyReportSchema(ctx, d.ReportDB); e != nil {
		return nil, e
	}
	a := &Application{d: d}
	cockpit, e := cockpitstore.New(ctx, d.StoreDB, dailyBoundary{a}, time.Now)
	if e != nil {
		return nil, e
	}
	a.cockpit = cockpit
	reports, e := reportstore.New(d.ReportDB)
	if e != nil {
		return nil, e
	}
	a.Capture = &rc.ScheduledCapture{Repository: reports, Authorize: func(ctx context.Context, o rc.ScheduledOrigin) error { return a.Authorize(ctx, o, true) }, Read: a.read}
	return a, nil
}
func (a *Application) current(ctx context.Context, o rc.ScheduledOrigin, generate bool) (storecenter.StoreMemberAccess, error) {
	if a == nil || ctx == nil || !o.Scope().Valid() || !authidentity.IsBoundedIdentifier(o.MemberID) {
		return storecenter.StoreMemberAccess{}, rc.ErrForbidden
	}
	if _, ok := ctx.Deadline(); !ok {
		return storecenter.StoreMemberAccess{}, rc.ErrForbidden
	}
	if browser, ok := authidentity.AuthenticatedIdentityFromContext(ctx); ok && (browser.UserID != o.ActorID || browser.TenantID != o.OrganizationID || browser.EffectiveOrganizationID != o.OrganizationID || browser.EffectiveMemberID != o.MemberID || !browser.TokenExpiresAt.After(time.Now())) {
		return storecenter.StoreMemberAccess{}, rc.ErrForbidden
	}
	token, e := a.d.ServiceToken(ctx)
	if e != nil {
		return storecenter.StoreMemberAccess{}, rc.ErrUnavailable
	}
	grant, e := a.d.IAM.ReadExactServiceProjectAuthorization(ctx, token, o.ActorID, a.d.ProjectID, o.OrganizationID)
	if e != nil {
		return storecenter.StoreMemberAccess{}, rc.ErrUnavailable
	}
	if !grant.Found || grant.State != "STATE_ACTIVE" || grant.AuthorizationID != o.MemberID {
		return storecenter.StoreMemberAccess{}, rc.ErrForbidden
	}
	active, e := a.d.Users.IsUserActive(ctx, token, o.ActorID)
	if e != nil {
		return storecenter.StoreMemberAccess{}, rc.ErrUnavailable
	}
	if !active {
		return storecenter.StoreMemberAccess{}, rc.ErrForbidden
	}
	if a.d.OrganizationStatus != nil {
		suspended, e := a.d.OrganizationStatus.IsOrganizationSuspended(ctx, o.OrganizationID)
		if e != nil {
			return storecenter.StoreMemberAccess{}, rc.ErrUnavailable
		}
		if suspended {
			return storecenter.StoreMemberAccess{}, rc.ErrForbidden
		}
	}
	permissions := []string{rc.ReadPermission, authz.PermissionCockpitStoresRead, authz.PermissionWorkbenchStoreRead}
	if generate {
		permissions = append(permissions, rc.ManagePermission, authz.PermissionWorkbenchTaskRead, authz.PermissionWorkbenchAgentUse)
	}
	for _, p := range permissions {
		allowed, e := authz.AuthorizeOrganization(ctx, a.d.Policy, o.ActorID, o.OrganizationID, grant.Roles, p)
		if e != nil {
			return storecenter.StoreMemberAccess{}, rc.ErrUnavailable
		}
		if !allowed {
			return storecenter.StoreMemberAccess{}, rc.ErrForbidden
		}
	}
	return storecenter.StoreMemberAccess{OrganizationID: o.OrganizationID, ActorID: o.ActorID, MemberID: o.MemberID, Administrator: a.d.Policy.IsTenantAdmin(o.ActorID, grant.Roles)}, nil
}

type memberAccess struct {
	app      *Application
	origin   rc.ScheduledOrigin
	generate bool
}

func (m memberAccess) AuthorizeStoreMember(ctx context.Context, org string) (storecenter.StoreMemberAccess, error) {
	if org != m.origin.OrganizationID {
		return storecenter.StoreMemberAccess{}, storecenter.ErrNotFound
	}
	return m.app.current(ctx, m.origin, m.generate)
}
func (a *Application) Store(ctx context.Context, o rc.ScheduledOrigin, generate bool) (*storecenter.Store, error) {
	repo, e := storecenter.NewMemberScopedStoreRepository(a.d.StoreDB, memberAccess{a, o, generate})
	if e != nil {
		return nil, rc.ErrUnavailable
	}
	store, e := repo.Get(ctx, o.OrganizationID, o.StoreID)
	if errors.Is(e, storecenter.ErrNotFound) || errors.Is(e, rc.ErrForbidden) {
		return nil, rc.ErrForbidden
	}
	if e != nil {
		return nil, rc.ErrUnavailable
	}
	if store.RecordStatus() != storecenter.RecordStatusActive {
		return nil, rc.ErrForbidden
	}
	return store, nil
}
func (a *Application) Authorize(ctx context.Context, o rc.ScheduledOrigin, generate bool) error {
	_, e := a.Store(ctx, o, generate)
	return e
}

// Visibility is computed from live IAM and current member-scoped Store rows
// before pagination/counting. Old report types keep their original contract.
func (a *Application) Visibility(ctx context.Context, s rc.Scope) (rc.DailyVisibility, error) {
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || id.TenantID != s.OrganizationID || id.EffectiveOrganizationID != s.OrganizationID || id.UserID != s.ActorID {
		return rc.DailyVisibility{}, nil
	}
	o := rc.ScheduledOrigin{OrganizationID: s.OrganizationID, ActorID: s.ActorID, MemberID: id.EffectiveMemberID}
	if _, e := a.current(ctx, o, false); e != nil {
		return rc.DailyVisibility{}, nil
	}
	repo, e := storecenter.NewMemberScopedStoreRepository(a.d.StoreDB, memberAccess{a, o, false})
	if e != nil {
		return rc.DailyVisibility{}, rc.ErrUnavailable
	}
	v := rc.DailyVisibility{MemberID: id.EffectiveMemberID, StoreIDs: []string{}}
	for page := 1; page <= 5; page++ {
		result, e := repo.List(ctx, s.OrganizationID, storecenter.StoreListQuery{Page: page, PageSize: 100, Status: storecenter.RecordStatusActive})
		if e != nil {
			return rc.DailyVisibility{}, nil
		}
		for _, store := range result.Stores {
			v.StoreIDs = append(v.StoreIDs, store.ID())
		}
		if int64(page*100) >= result.Total {
			return v, nil
		}
	}
	return rc.DailyVisibility{}, nil
}

type dailyBoundary struct{ app *Application }

func (b dailyBoundary) Current(ctx context.Context, s c.Scope) (c.Access, error) {
	o, ok := ctx.Value(originKey{}).(rc.ScheduledOrigin)
	if !ok || s != (c.Scope{OrganizationID: o.OrganizationID, ActorID: o.ActorID}) {
		return c.Access{}, c.ErrForbidden
	}
	if e := b.app.Authorize(ctx, o, true); e != nil {
		return c.Access{}, e
	}
	return c.Access{StoresRead: true}, nil
}
func (b dailyBoundary) ReadStores(ctx context.Context, s c.Scope, ids []string) error {
	o, ok := ctx.Value(originKey{}).(rc.ScheduledOrigin)
	if !ok || s != (c.Scope{OrganizationID: o.OrganizationID, ActorID: o.ActorID}) || len(ids) != 1 || ids[0] != o.StoreID {
		return c.ErrForbidden
	}
	return b.app.Authorize(ctx, o, true)
}
func (dailyBoundary) LockStores(context.Context, *gorm.DB, c.Scope, []string) error {
	return c.ErrForbidden
}
func (a *Application) read(ctx context.Context, o rc.ScheduledOrigin) (rc.ScheduledSnapshot, error) {
	store, e := a.Store(ctx, o, true)
	if e != nil {
		return rc.ScheduledSnapshot{}, e
	}
	ctx = context.WithValue(ctx, originKey{}, o)
	snapshot, e := a.cockpit.Snapshot(ctx, c.Scope{OrganizationID: o.OrganizationID, ActorID: o.ActorID}, c.Query{Module: "stores", Period: c.Period{Start: o.Day, End: o.Day}, StoreIDs: []string{o.StoreID}})
	if e != nil {
		return rc.ScheduledSnapshot{}, e
	}
	result := rc.ScheduledSnapshot{Snapshot: Render(o, store.Name(), string(store.Platform()), snapshot)}
	section, revalidate := a.savedOrders(ctx, o, store.Platform())
	index := len(result.Content.Sections)
	result.Content.Sections = append(result.Content.Sections, section)
	result.Revalidate = func(ctx context.Context, snap *rc.Snapshot) error {
		if revalidate != nil && revalidate(ctx) != nil {
			snap.Content.Sections[index] = ordersUnavailable("当前无法读取该店铺的已保存订单")
		}
		return nil
	}
	return result, nil
}
