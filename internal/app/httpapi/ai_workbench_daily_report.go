package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"net/http"
	"strconv"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/app/scheduledreport"
	storeapp "task-processor/internal/app/storecenter"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/integration/dataserviceauth"
	rc "task-processor/internal/reportcenter"
	"task-processor/internal/storecenter/observations"
	"task-processor/internal/workbenchcontext"
	"time"
)

// Saved-report visibility is composed with Report Center, independently of Chat
// and model execution. Disabling the new schedule kind preserves history reads.
func buildCurrentDailyReports(ctx context.Context, cfg *config.Config, resolver organizationIdentityResolver, policy *authz.ListingKitAuthorizer, stores, reports *gorm.DB, cockpit bool, reportService *rc.Service, saved *observations.Service, apps *storeapp.OfficialApplicationRegistry) *scheduledreport.Application {
	if cfg == nil || stores == nil || reports == nil || !cockpit || reportService == nil {
		return nil
	}
	current, ok := resolver.(*workbenchcontext.Resolver)
	if !ok || current == nil {
		return nil
	}
	z := cfg.ListingKit.Zitadel
	users, e := dataserviceauth.NewActiveUserClient(z.AuthorizationAPIURL, nil)
	if e != nil {
		return nil
	}
	app, e := scheduledreport.New(ctx, scheduledreport.Dependencies{Observations: saved, Applications: apps, StoreDB: stores, ReportDB: reports, IAM: zitadel.NewAuthorizationClient(z.AuthorizationAPIURL, &http.Client{Timeout: 5 * time.Second}), Users: users, ServiceToken: func(context.Context) (string, error) { return z.TenantDirectoryToken, nil }, ProjectID: z.ProjectID, Policy: policy, OrganizationStatus: current.BusinessStatusChecker()})
	if e != nil {
		return nil
	}
	reportService.DailyVisibility = app.Visibility
	return app
}
func (a *aiWorkbenchApplication) attachDailyReports(ctx context.Context, app *scheduledreport.Application) {
	a.dailyReports = app
	if app == nil || a.store.VerifyDailyReports(ctx) != nil {
		a.dailyReportsEnabled = false
	}
}
func (a *aiWorkbenchApplication) contentSchedulesEnabled() bool {
	return a != nil && a.agent != nil && !a.conversationOnly && a.agent.workbenchKind(aiworkbench.ContentTaskKind) && !a.agent.config.ContentTitleOnly
}

func (a *aiWorkbenchApplication) dailyReportReady() bool {
	return a != nil && a.dailyReports != nil && a.dailyReportsEnabled && a.schedules != nil && a.schedules.Ready()
}
func reportOriginForPlan(p aiworkbench.ScheduledPlan) rc.ScheduledOrigin {
	return rc.ScheduledOrigin{OrganizationID: p.Scope.OrganizationID, ActorID: p.Scope.ActorID, MemberID: p.MemberID, StoreID: p.Report.StoreID}
}
func dailyInputSpec(input scheduledInput) bool {
	return input.Report != nil && input.Report.Valid() && input.ProposalID == "" && input.Selection == nil && input.GoalSummary == "" && (input.Kind == "" || input.Kind == aiworkbench.DailyReportKind)
}
func (a *aiWorkbenchApplication) quoteDailyReport(ctx context.Context, owner aiworkbench.Scope, input scheduledInput) (scheduledQuote, error) {
	if !a.dailyReportReady() {
		return scheduledQuote{}, aiworkbench.ErrUnavailable
	}
	if !dailyInputSpec(input) {
		return scheduledQuote{}, aiworkbench.ErrInvalid
	}
	count, e := strconv.ParseUint(input.MaxOccurrences, 10, 63)
	if e != nil || count == 0 || strconv.FormatUint(count, 10) != input.MaxOccurrences {
		return scheduledQuote{}, aiworkbench.ErrInvalid
	}
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || id.TenantID != owner.OrganizationID || id.UserID != owner.ActorID {
		return scheduledQuote{}, rc.ErrForbidden
	}
	if input.ScheduleID != "" {
		p, e := a.store.ReadScheduledPlan(ctx, owner, input.ScheduleID)
		if e != nil {
			return scheduledQuote{}, e
		}
		if p.Report == nil || *p.Report != *input.Report || p.Budget.MaxOccurrences != count || p.MemberID != id.EffectiveMemberID {
			return scheduledQuote{}, aiworkbench.ErrInvalid
		}
	}
	now := time.Now().UTC()
	draft := aiworkbench.ScheduledPlanDraft{ID: "d32716d6-4423-4b41-ae5f-46e336c9dc11", Scope: owner, MemberID: id.EffectiveMemberID, Report: input.Report, GoalSummary: "店铺经营日报", CreatedAt: now, Rule: input.Rule, Budget: aiworkbench.ScheduleBudget{Unit: "none", MaxOccurrences: count}}
	if !draft.Valid() || input.Rule.Frequency == aiworkbench.ScheduleOnce && input.Rule.AtUTC.Nanosecond() != 0 {
		return scheduledQuote{}, aiworkbench.ErrInvalid
	}
	store, e := a.dailyReports.Store(ctx, rc.ScheduledOrigin{OrganizationID: owner.OrganizationID, ActorID: owner.ActorID, MemberID: id.EffectiveMemberID, StoreID: input.Report.StoreID}, true)
	if e != nil {
		return scheduledQuote{}, e
	}
	next, _ := input.Rule.NextAfter(now)
	return scheduledQuote{StoreName: store.Name(), Report: input.Report, Kind: aiworkbench.DailyReportKind, Confirmation: dailyConfirmation(*input.Report, input.Rule, count), GoalSummary: "店铺经营日报", Rule: input.Rule, MaxOccurrences: input.MaxOccurrences, Unit: "none", PerRunUpperBoundMicros: "0", TotalUpperBoundMicros: "0", NextAt: next}, nil
}
func dailyConfirmation(spec aiworkbench.DailyReportSpec, rule aiworkbench.ScheduleRule, count uint64) string {
	return scheduledHash(struct {
		Spec        aiworkbench.DailyReportSpec
		Rule        aiworkbench.ScheduleRule
		Count       uint64
		SaveConsent string
	}{spec, rule, count, "automatically-save-personal-report-v1"})
}

func (a *aiWorkbenchApplication) dispatchDailyReport(ctx context.Context, plan aiworkbench.ScheduledPlan, slot time.Time) error {
	occurrence, e := a.store.ReadScheduledOccurrence(ctx, plan.Scope, plan.ID, slot)
	if errors.Is(e, aiworkbench.ErrNotFound) {
		if !a.admittedOrganization(plan.Scope.OrganizationID) {
			return rc.ErrForbidden
		}
		if !a.dailyReportReady() {
			return aiworkbench.ErrUnavailable
		}
		occurrence, _, e = a.store.ClaimScheduledReport(ctx, plan.Scope, plan.ID, plan.Revision, slot, time.Now().UTC())
	}
	if e != nil {
		return e
	}
	if occurrence.State == "SKIPPED" || occurrence.State == "BLOCKED" || occurrence.State == "RECONCILED" {
		return nil
	}
	execution, e := a.store.ReadReportExecution(ctx, plan.Scope, occurrence.ID)
	if e != nil {
		return e
	}
	if a.dailyReports == nil {
		return aiworkbench.ErrUnavailable
	}
	receipt, found, e := a.dailyReports.Capture.ReadScheduledCaptureReceipt(ctx, execution.Origin)
	if e != nil {
		return e
	}
	if found {
		return a.store.FinishReportCapture(ctx, receipt)
	}
	if !a.dailyReportReady() {
		return aiworkbench.ErrUnavailable
	}
	if !a.admittedOrganization(plan.Scope.OrganizationID) {
		return rc.ErrForbidden
	}
	if e = a.dailyReports.Authorize(ctx, execution.Origin, true); e != nil {
		if errors.Is(e, rc.ErrForbidden) && !execution.CaptureStarted {
			return a.store.BlockUnstartedReport(ctx, execution.Origin)
		}
		return e
	}
	if e = a.store.BeginReportCapture(ctx, execution.Origin); e != nil {
		return e
	}
	receipt, e = a.dailyReports.Capture.CaptureScheduledDaily(ctx, execution.Origin)
	if e != nil {
		_ = a.store.NoteScheduledOccurrence(ctx, plan.Scope, occurrence.ID, "UNKNOWN", "REPORT_CAPTURE_UNRESOLVED")
		return e
	}
	return a.store.FinishReportCapture(ctx, receipt)
}
