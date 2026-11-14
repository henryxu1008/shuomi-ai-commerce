//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"task-processor/internal/aiworkbench"
	reportapp "task-processor/internal/app/reportcenter"
	"task-processor/internal/app/scheduledreport"
	"task-processor/internal/authidentity"
	"task-processor/internal/authruntime/zitadel"
	"task-processor/internal/authz"
	workstore "task-processor/internal/integration/persistence/aiworkbench"
	cockpitstore "task-processor/internal/integration/persistence/operationscockpit"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	rc "task-processor/internal/reportcenter"
	reporthttp "task-processor/internal/reportcenter/httpapi"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type dailyIAM struct {
	mu                sync.Mutex
	member            string
	active, suspended bool
	modules           []string
}

func (d *dailyIAM) ReadExactServiceProjectAuthorization(_ context.Context, token, actor, project, org string) (zitadel.ExactServiceProjectAuthorization, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if token != "synthetic" || actor != "daily-user" || org != "daily-org" || project != "project" {
		return zitadel.ExactServiceProjectAuthorization{}, rc.ErrForbidden
	}
	return zitadel.ExactServiceProjectAuthorization{Found: true, State: "STATE_ACTIVE", AuthorizationID: d.member, Roles: []string{authz.EnterpriseRoleKey(org, 1)}}, nil
}
func (d *dailyIAM) IsUserActive(context.Context, string, string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active, nil
}
func (d *dailyIAM) IsOrganizationSuspended(context.Context, string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.suspended, nil
}
func (d *dailyIAM) RoleModules(_ context.Context, org string, keys []string) (map[string][]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := map[string][]string{}
	if org == "daily-org" {
		for _, key := range keys {
			out[key] = append([]string(nil), d.modules...)
		}
	}
	return out, nil
}

type dailyMember struct{ access storecenter.StoreMemberAccess }

func (m dailyMember) AuthorizeStoreMember(context.Context, string) (storecenter.StoreMemberAccess, error) {
	return m.access, nil
}

// This reuses the existing PostgreSQL and HTTP acceptance fixtures. Only IAM,
// Auth.js session issuance and the nominal Temporal callback are controlled;
// quote/create, live grants, Cockpit reads, both databases and recovery are real.
func TestScheduledDailyReportHTTPAndRecovery(t *testing.T) {
	owner, connection := commercialPostgres(t, "postgres:16-alpine")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	open := func(database, role string) *gorm.DB {
		dsn := fmt.Sprintf("host=127.0.0.1 port=%d dbname=%s user=%s password=daily-fixture sslmode=disable", connection.Port, database, role)
		if role == "issue347" {
			dsn = strings.ReplaceAll(dsn, "password=daily-fixture", "password=issue347-fixture")
		}
		db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		pool.SetMaxOpenConns(8)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	require.NoError(t, owner.Exec("CREATE DATABASE daily_workbench").Error)
	require.NoError(t, owner.Exec("CREATE DATABASE reports").Error)
	require.NoError(t, owner.Exec(`CREATE ROLE ai_workbench_runtime LOGIN PASSWORD 'daily-fixture' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;CREATE ROLE report_center_runtime LOGIN PASSWORD 'daily-fixture' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;CREATE ROLE daily_store_runtime LOGIN PASSWORD 'daily-fixture' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`).Error)
	wbOwner, reportOwner := open("daily_workbench", "issue347"), open("reports", "issue347")
	require.NoError(t, workstore.InstallSchema(wbOwner))
	require.NoError(t, workstore.InstallDailyReportSchema(wbOwner))
	require.NoError(t, workstore.GrantRuntime(wbOwner, "ai_workbench_runtime"))
	require.NoError(t, workstore.GrantDailyReportRuntime(wbOwner, "ai_workbench_runtime"))
	require.NoError(t, reportOwner.Exec(`REVOKE CREATE ON DATABASE reports FROM PUBLIC;REVOKE CREATE ON SCHEMA public FROM PUBLIC`).Error)
	require.NoError(t, reportstore.InstallSchema(reportOwner))
	require.NoError(t, reportstore.InstallDailyReportSchema(reportOwner))
	require.NoError(t, reportstore.GrantRuntime(reportOwner, "report_center_runtime"))
	sqlDB, e := owner.DB()
	require.NoError(t, e)
	tx, e := sqlDB.BeginTx(ctx, nil)
	require.NoError(t, e)
	require.NoError(t, storecenter.InstallCurrentSchemaTx(ctx, tx))
	require.NoError(t, tx.Commit())
	require.NoError(t, cockpitstore.Install(ctx, owner))
	require.NoError(t, owner.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC;GRANT USAGE ON SCHEMA public TO daily_store_runtime;GRANT SELECT ON ALL TABLES IN SCHEMA public TO daily_store_runtime;GRANT UPDATE(active) ON workbench_store_member_grants TO daily_store_runtime`).Error)
	require.NoError(t, cockpitstore.GrantRuntime(ctx, owner, "daily_store_runtime"))
	now := time.Now().UTC().Truncate(time.Second)
	identity := authidentity.AuthenticatedIdentity{UserID: "daily-user", TenantID: "daily-org", EffectiveOrganizationID: "daily-org", EffectiveMemberID: "daily-member", HomeOrganizationID: "daily-org", TokenExpiresAt: now.Add(time.Hour), Roles: []string{authz.EnterpriseRoleKey("daily-org", 1)}}
	iam := &dailyIAM{member: identity.EffectiveMemberID, active: true, modules: []string{"tasks", "reports", "overview-stores", "agent-custom"}}
	policy, e := authz.NewListingKitAuthorizer(nil, nil)
	require.NoError(t, e)
	policy.SetRolePolicyReader(iam)
	require.False(t, authz.AllowedOrganization(ctx, policy, identity.UserID, identity.TenantID, identity.Roles, authz.PermissionLocalAgentWrite), "the report fixture has no product write authority")
	base, e := storecenter.NewMemberScopedStoreRepository(owner, dailyMember{storecenter.StoreMemberAccess{OrganizationID: identity.TenantID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID, CanWrite: true}})
	require.NoError(t, e)
	store, e := storecenter.NewStore(storecenter.CreateStoreInput{ID: uuid.NewString(), OrganizationID: identity.TenantID, ActorSubject: identity.UserID, Name: "日报合成测试店", Platform: "shein", Region: "US", CreateIdempotencyKey: uuid.NewString(), OccurredAt: now})
	require.NoError(t, e)
	_, _, e = base.CreateOrReplay(ctx, identity.TenantID, store)
	require.NoError(t, e)
	day := aiworkbench.DailyReportSpec{StoreID: store.ID(), Version: 1, PeriodPolicy: "previous_natural_day", BusinessZone: "Asia/Shanghai"}
	period := day.Period(now)
	require.NoError(t, owner.Exec(`INSERT INTO operations_cockpit.facts(organization_id,record_id,store_id,revision,start_date,end_date,amounts,note,updated_by,updated_at) VALUES(?,?,?,1,?,?,?::jsonb,'synthetic explicit zero',?,?)`, identity.TenantID, uuid.NewString(), store.ID(), period, period, `{"revenue":0,"refunds":0,"procurement":0,"logistics":0,"platform":0,"advertising":0,"other":0}`, identity.UserID, now).Error)
	workDB, reportDB, storeDB := open("daily_workbench", "ai_workbench_runtime"), open("reports", "report_center_runtime"), open("issue347", "daily_store_runtime")
	daily, e := scheduledreport.New(ctx, scheduledreport.Dependencies{StoreDB: storeDB, ReportDB: reportDB, IAM: iam, Users: iam, ServiceToken: func(context.Context) (string, error) { return "synthetic", nil }, ProjectID: "project", Policy: policy, OrganizationStatus: iam})
	require.NoError(t, e)
	resolver := &productReviewResolverStub{value: identity}
	deps := routeAuthDependencies{workbenchVerifier: mountedVerifierStub{identity: identity}, organizationResolver: resolver, authorizer: policy}
	app, e := buildAIWorkbenchApplication(ctx, AIWorkbenchDependencies{DB: workDB, ConversationOnly: true, StoreDailyReports: true, Schedules: &scheduledBackendFixture{}, AllowedOrganizationIDs: []string{identity.TenantID}, resolver: resolver, authorizer: policy}, nil)
	require.NoError(t, e)
	app.dailyReports = daily
	require.NoError(t, daily.Authorize(ctx, rc.ScheduledOrigin{OrganizationID: identity.TenantID, ActorID: identity.UserID, MemberID: identity.EffectiveMemberID, StoreID: store.ID()}, true))
	require.True(t, app.dailyReportReady())
	reports, e := reportapp.New(ctx, reportapp.Dependencies{DB: reportDB, Resolver: resolver, Policy: policy})
	require.NoError(t, e)
	reports.Service.DailyVisibility = daily.Visibility
	routes := append(aiWorkbenchScheduleRoutes(app), reporthttp.Routes(reports)...)
	running := httptest.NewServer(buildCurrentApplicationHTTPServer(routes, deps).Handler)
	defer running.Close()
	request := func(method, path, key string, body any, status int) []byte {
		var reader io.Reader
		if body != nil {
			raw, e := json.Marshal(body)
			require.NoError(t, e)
			reader = strings.NewReader(string(raw))
		}
		r, e := http.NewRequest(method, running.URL+path, reader)
		require.NoError(t, e)
		r.Header.Set("Authorization", "Bearer synthetic")
		r.Header.Set("X-Requested-Organization-ID", identity.TenantID)
		r.Header.Set("Content-Type", "application/json")
		if key != "" {
			r.Header.Set("Idempotency-Key", key)
		}
		response, e := http.DefaultClient.Do(r)
		require.NoError(t, e)
		defer response.Body.Close()
		raw, e := io.ReadAll(response.Body)
		require.NoError(t, e)
		require.Equal(t, status, response.StatusCode, string(raw))
		return raw
	}
	request("GET", workbenchScheduleBase+"/availability", "", nil, 200)
	input := scheduledInput{Kind: aiworkbench.DailyReportKind, Report: &day, Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "Asia/Shanghai", AtUTC: now.Add(time.Minute)}, MaxOccurrences: "1"}
	var quote scheduledQuote
	require.NoError(t, json.Unmarshal(request("POST", workbenchScheduleBase+"/quote", "", input, 200), &quote))
	require.Equal(t, "none", quote.Unit)
	require.Nil(t, quote.Selection)
	input.Confirmation = quote.Confirmation
	key := uuid.NewString()
	raw := request("POST", workbenchScheduleBase, key, input, 200)
	var result struct{ Plan scheduledPlanView }
	require.NoError(t, json.Unmarshal(raw, &result))
	id := result.Plan.ID
	require.Equal(t, raw, request("POST", workbenchScheduleBase, key, input, 200), "same create command replays without another plan")
	plan, e := app.store.ReadScheduledPlan(ctx, aiworkbench.Scope{OrganizationID: identity.TenantID, ActorID: identity.UserID}, id)
	require.NoError(t, e)
	// Callback's nominal slot is current; use a bounded synthetic future now only
	// for the persistence claim. The business day still comes from the exact slot.
	_, execution, e := app.store.ClaimScheduledReport(ctx, plan.Scope, id, plan.Revision, input.Rule.AtUTC, input.Rule.AtUTC)
	require.NoError(t, e)
	require.NoError(t, app.store.BeginReportCapture(ctx, execution.Origin))
	// Crash before Report call, beyond Temporal retry lifetime: the ordinary scan
	// resumes this same frozen occurrence rather than creating a new one.
	require.NoError(t, app.recoverScheduledPlans(ctx))
	history := request("GET", workbenchScheduleBase+"/"+id+"/occurrences", "", nil, 200)
	var page struct {
		Occurrences []struct {
			ReportID string `json:"reportId"`
			State    string
		}
	}
	require.NoError(t, json.Unmarshal(history, &page))
	require.Len(t, page.Occurrences, 1)
	require.Equal(t, "RECONCILED", page.Occurrences[0].State)
	reportID := page.Occurrences[0].ReportID
	require.NotEmpty(t, reportID)
	saved := request("GET", reporthttp.Base+"/"+reportID, "", nil, 200)
	require.Contains(t, string(saved), "0.00 元")
	require.Contains(t, string(saved), period)
	request("GET", reporthttp.Base+"?view=all&limit=20&kind=STORE_DAILY_REPORT", "", nil, 200)
	// Revocation affects list/search/count/detail/favorite, even an already saved
	// personal report. Exact metadata reconciliation remains available internally.
	iam.mu.Lock()
	iam.member = "replacement-member"
	iam.mu.Unlock()
	request("GET", reporthttp.Base+"/"+reportID, "", nil, 404)
	require.NotContains(t, string(request("GET", reporthttp.Base+"?view=all&limit=20&search=日报", "", nil, 200)), reportID)
	require.Contains(t, string(request("GET", reporthttp.Base+"/summary", "", nil, 200)), `"saved":0`)
	request("POST", reporthttp.Base+"/"+reportID+"/favorite", uuid.NewString(), map[string]bool{"favorite": true}, 404)
	receipt, found, e := daily.Capture.ReadScheduledCaptureReceipt(ctx, execution.Origin)
	require.NoError(t, e)
	require.True(t, found)
	require.Equal(t, reportID, receipt.ReportID)
	iam.mu.Lock()
	iam.member = identity.EffectiveMemberID
	iam.mu.Unlock()
	require.Equal(t, saved, request("GET", reporthttp.Base+"/"+reportID, "", nil, 200), "refresh reads the same immutable report")
	t.Run("live user and organization authorization", func(t *testing.T) {
		future := plan.ScheduledPlanDraft
		future.ID = uuid.NewString()
		require.NoError(t, app.store.CreateScheduledPlan(ctx, future))
		require.NoError(t, app.store.MarkScheduleRegistered(ctx, future.Scope, future.ID, 1))
		iam.mu.Lock()
		iam.active = false
		iam.mu.Unlock()
		require.ErrorIs(t, daily.Authorize(ctx, execution.Origin, true), rc.ErrForbidden)
		iam.mu.Lock()
		iam.active = true
		iam.suspended = true
		iam.mu.Unlock()
		require.ErrorIs(t, daily.Authorize(ctx, execution.Origin, true), rc.ErrForbidden)
		iam.mu.Lock()
		iam.suspended = false
		iam.modules = []string{"tasks", "agent-custom", "overview-stores"}
		iam.mu.Unlock()
		request("POST", workbenchScheduleBase+"/quote", "", input, 403)
		// Losing report generation authority does not trap a member in an
		// enabled schedule: pausing their own future triggers remains possible.
		request("POST", workbenchScheduleBase+"/"+future.ID+"/pause", uuid.NewString(), map[string]string{"expectedRevision": "2"}, 200)
		iam.mu.Lock()
		iam.modules = []string{"tasks", "reports", "overview-stores", "agent-custom"}
		iam.mu.Unlock()
		require.NoError(t, owner.Exec("UPDATE workbench_store_member_grants SET active=FALSE WHERE store_id=?", store.ID()).Error)
		require.ErrorIs(t, daily.Authorize(ctx, execution.Origin, true), rc.ErrForbidden)
		request("GET", reporthttp.Base+"/"+reportID, "", nil, 404)
		require.NoError(t, owner.Exec("UPDATE workbench_store_member_grants SET active=TRUE WHERE store_id=?", store.ID()).Error)
	})
	t.Run("original recovery scan resumes after the Temporal window", func(t *testing.T) {
		slot := now.Add(-30 * time.Minute)
		p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: plan.Scope, MemberID: identity.EffectiveMemberID, Report: &day, GoalSummary: "店铺经营日报", CreatedAt: slot.Add(-time.Minute), Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "Asia/Shanghai", AtUTC: slot}, Budget: aiworkbench.ScheduleBudget{Unit: "none", MaxOccurrences: 1}}
		require.NoError(t, app.store.CreateScheduledPlan(ctx, p))
		require.NoError(t, app.store.MarkScheduleRegistered(ctx, p.Scope, p.ID, 1))
		o, x, e := app.store.ClaimScheduledReport(ctx, p.Scope, p.ID, 2, slot, slot)
		require.NoError(t, e)
		require.NoError(t, app.store.BeginReportCapture(ctx, x.Origin))
		app.dailyReportsEnabled = false
		require.Error(t, app.recoverScheduledPlans(ctx))
		pending, e := app.store.ReadScheduledPlan(ctx, p.Scope, p.ID)
		require.NoError(t, e)
		require.Equal(t, o.ID, pending.ActiveOccurrenceID)
		require.EqualValues(t, 1, pending.ConsumedOccurrences)
		app.dailyReportsEnabled = true
		require.NoError(t, app.recoverScheduledPlans(ctx))
		finished, e := app.store.ReadReportExecution(ctx, p.Scope, o.ID)
		require.NoError(t, e)
		require.NotEmpty(t, finished.ReportID)
		require.NoError(t, app.recoverScheduledPlans(ctx))
		replayed, e := app.store.ReadReportExecution(ctx, p.Scope, o.ID)
		require.NoError(t, e)
		require.Equal(t, finished, replayed)
	})
	if dir := os.Getenv("SCHEDULED_REPORT_BROWSER_FIXTURE_DIR"); dir != "" {
		require.True(t, filepath.IsAbs(dir))
		control := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			if r.URL.Path == "/dispatch" {
				var command struct {
					PlanID string `json:"planId"`
				}
				if json.NewDecoder(r.Body).Decode(&command) != nil {
					w.WriteHeader(400)
					return
				}
				c, stop := context.WithTimeout(context.Background(), 20*time.Second)
				defer stop()
				p, e := app.store.ReadScheduledExecutionPlan(c, command.PlanID)
				if e == nil {
					slot := p.Rule.AtUTC
					if p.Rule.Frequency != aiworkbench.ScheduleOnce {
						slot, e = p.Rule.NextAfter(p.CreatedAt)
					}
					if e == nil {
						_, _, e = app.store.ClaimScheduledReport(c, p.Scope, p.ID, p.Revision, slot, slot)
					}
				}
				if e == nil {
					slot := p.Rule.AtUTC
					if p.Rule.Frequency != aiworkbench.ScheduleOnce {
						slot, e = p.Rule.NextAfter(p.CreatedAt)
					}
					if e == nil {
						e = app.DispatchScheduled(c, p.ID, slot)
					}
				}
				if e != nil {
					w.WriteHeader(500)
					return
				}
				w.WriteHeader(204)
				return
			}
			w.WriteHeader(404)
		}))
		defer control.Close()
		data, _ := json.Marshal(map[string]any{"goOrigin": running.URL, "controlOrigin": control.URL, "storeId": store.ID(), "storeName": store.Name(), "period": period})
		require.NoError(t, os.WriteFile(filepath.Join(dir, "go.json"), data, 0600))
		// Bounded optional handoff follows the existing account browser fixture; it
		// is not a runner or a permanent acceptance service.
		deadline := time.NewTimer(20 * time.Minute)
		defer deadline.Stop()
		tick := time.NewTicker(200 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-deadline.C:
				t.Fatal("browser handoff timed out")
			case <-tick.C:
				if _, e := os.Stat(filepath.Join(dir, "stop")); e == nil {
					return
				}
			}
		}
	}
}
