//go:build integration

package aiworkbenchpersistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"strings"
	"sync"
	"task-processor/internal/aiworkbench"
	reportstore "task-processor/internal/integration/persistence/reportcenter"
	reviewproposal "task-processor/internal/product/review/proposal"
	rc "task-processor/internal/reportcenter"
	"testing"
	"time"
)

func TestScheduledReportTwoDatabaseProtocol(t *testing.T) {
	owner := workbenchDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dsn := owner.Dialector.(*postgres.Dialector).Config.DSN
	open := func(raw string) *gorm.DB {
		db, e := gorm.Open(postgres.Open(raw), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, e)
		pool, e := db.DB()
		require.NoError(t, e)
		pool.SetMaxOpenConns(12)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	require.NoError(t, owner.Exec("CREATE DATABASE reports").Error)
	u, e := url.Parse(dsn)
	require.NoError(t, e)
	u.Path = "/reports"
	reportOwner := open(u.String())
	require.NoError(t, reportstore.InstallSchema(reportOwner))
	require.NoError(t, reportstore.InstallDailyReportSchema(reportOwner))
	require.NoError(t, reportOwner.Exec(`RESET ROLE; REVOKE CREATE ON DATABASE reports FROM PUBLIC;REVOKE CREATE ON SCHEMA public FROM PUBLIC;CREATE ROLE report_center_runtime LOGIN PASSWORD 'daily-report-test' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`).Error)
	require.NoError(t, reportstore.GrantRuntime(reportOwner, "report_center_runtime"))
	u.User = url.UserPassword("report_center_runtime", "daily-report-test")
	reportDB := open(u.String())
	require.NoError(t, reportstore.VerifyDailyReportSchema(ctx, reportDB))
	reports, e := reportstore.New(reportDB)
	require.NoError(t, e)

	oldStore, e := New(owner)
	require.NoError(t, e)
	oldIDs := []string{}
	oldBefore := map[string]string{}
	for _, frequency := range []aiworkbench.ScheduleFrequency{aiworkbench.ScheduleOnce, aiworkbench.ScheduleDaily, aiworkbench.ScheduleWeekly} {
		created := time.Now().UTC().Truncate(time.Second)
		p := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: aiworkbench.Scope{OrganizationID: "existing-org", ActorID: "existing-user"}, MemberID: "existing-member", GoalSummary: "Existing content", Selection: aiworkbench.TaskSelection{Subject: aiworkbench.Subject{Kind: "own_product", ID: uuid.NewString(), CatalogVersion: "1", PublicationID: "existing-publication"}, Capability: aiworkbench.ContentTaskKind, Parameters: reviewproposal.ContentSelection{TargetLanguage: "en-US", Description: true}}, Rule: aiworkbench.ScheduleRule{Frequency: frequency, TimeZone: "UTC", Hour: 9, Weekday: time.Monday}, Budget: aiworkbench.ScheduleBudget{MaxOccurrences: 3, PerRunUpperBoundMicros: 1}, CreatedAt: created}
		if frequency == aiworkbench.ScheduleOnce {
			p.Rule.AtUTC = created.Add(time.Hour)
			p.Budget.MaxOccurrences = 1
		}
		require.NoError(t, oldStore.CreateScheduledPlan(ctx, p))
		oldIDs = append(oldIDs, p.ID)
		state := "PAUSED"
		if frequency == aiworkbench.ScheduleWeekly {
			state = "REGISTRATION_UNKNOWN"
		}
		require.NoError(t, owner.Exec("UPDATE ai_workbench.scheduled_plans SET state=? WHERE id=?", state, p.ID).Error)
		var raw string
		require.NoError(t, owner.Raw("SELECT (to_jsonb(p)-'report_spec')::text FROM ai_workbench.scheduled_plans p WHERE id=?", p.ID).Scan(&raw).Error)
		oldBefore[p.ID] = raw
	}
	require.NoError(t, InstallDailyReportSchema(owner))
	for _, id := range oldIDs {
		var raw string
		require.NoError(t, owner.Raw("SELECT (to_jsonb(p)-'report_spec')::text FROM ai_workbench.scheduled_plans p WHERE id=?", id).Scan(&raw).Error)
		require.Equal(t, oldBefore[id], raw, "existing plan columns must not be rewritten")
	}

	require.NoError(t, owner.Exec(`REVOKE CREATE ON SCHEMA public FROM PUBLIC;CREATE ROLE ai_workbench_runtime LOGIN PASSWORD 'daily-report-test' NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS`).Error)
	require.NoError(t, GrantRuntime(owner, "ai_workbench_runtime"))
	require.NoError(t, GrantDailyReportRuntime(owner, "ai_workbench_runtime"))
	u, e = url.Parse(dsn)
	require.NoError(t, e)
	u.User = url.UserPassword("ai_workbench_runtime", "daily-report-test")
	db := open(u.String())
	require.NoError(t, VerifyDailyReportSchema(ctx, db))
	store, e := New(db)
	require.NoError(t, e)
	now := time.Now().UTC().Truncate(time.Second)
	scope := aiworkbench.Scope{OrganizationID: "daily-org", ActorID: "daily-actor"}
	plan := aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Scope: scope, MemberID: "daily-member", Report: &aiworkbench.DailyReportSpec{StoreID: uuid.NewString(), Version: 1, PeriodPolicy: "previous_natural_day", BusinessZone: "Asia/Shanghai"}, GoalSummary: "店铺经营日报", CreatedAt: now.Add(-time.Minute), Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "Asia/Shanghai", AtUTC: now}, Budget: aiworkbench.ScheduleBudget{Unit: "none", MaxOccurrences: 1}}
	plan.CreationFingerprint = digest(plan)
	require.NoError(t, store.CreateScheduledPlanForCommand(ctx, plan, uuid.NewString()))
	require.NoError(t, store.MarkScheduleRegistered(ctx, scope, plan.ID, 1))
	var wg sync.WaitGroup
	failures := make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, e := store.ClaimScheduledReport(ctx, scope, plan.ID, 2, now, now)
			failures <- e
		}()
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		require.NoError(t, e)
	}
	occurrence, e := store.ReadScheduledOccurrence(ctx, scope, plan.ID, now)
	require.NoError(t, e)
	require.Empty(t, occurrence.BusinessTaskID)
	execution, e := store.ReadReportExecution(ctx, scope, occurrence.ID)
	require.NoError(t, e)
	require.NoError(t, store.BeginReportCapture(ctx, execution.Origin))
	require.Error(t, store.BlockUnstartedReport(ctx, execution.Origin), "a concurrent/old started writer cannot be declared no-write")
	makeSnapshot := func(text string) rc.Snapshot {
		return rc.Snapshot{SourceInfo: rc.SourceInfo{Ref: rc.SourceRef{Kind: rc.StoreDailyReportKind, ID: occurrence.ID, Version: "1"}, Title: "店铺经营日报", StoreID: plan.Report.StoreID, ScheduledOrigin: &execution.Origin}, Content: rc.Document{SchemaVersion: 1, Sections: []rc.Section{{Title: "经营财务", Fields: []rc.Field{{Label: "收入", Value: text}}}}}}
	}
	receipts := make(chan rc.CaptureReceipt, 12)
	failures = make(chan error, 12)
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r, e := reports.SaveScheduledCapture(ctx, execution.Origin, makeSnapshot(strings.Repeat("0", i+1)), func(context.Context, *rc.Snapshot) error { return nil })
			receipts <- r
			failures <- e
		}(i)
	}
	wg.Wait()
	close(receipts)
	close(failures)
	for e := range failures {
		require.NoError(t, e)
	}
	var winner rc.CaptureReceipt
	for r := range receipts {
		if winner.ReportID == "" {
			winner = r
		}
		require.Equal(t, winner, r, "first committed snapshot wins even when readers saw different source revisions")
	}
	// Report committed but the worker lost its response / died before finalize.
	recovered, found, e := reports.ReadScheduledCaptureReceipt(ctx, execution.Origin)
	require.NoError(t, e)
	require.True(t, found)
	require.Equal(t, winner, recovered)
	require.NoError(t, store.FinishReportCapture(ctx, recovered))
	require.NoError(t, store.FinishReportCapture(ctx, recovered))
	saved, e := reports.Read(ctx, execution.Origin.Scope(), winner.ReportID)
	require.NoError(t, e)
	require.Equal(t, winner.Digest, saved.Digest)
	current, e := store.ReadScheduledPlan(ctx, scope, plan.ID)
	require.NoError(t, e)
	require.EqualValues(t, 1, current.ConsumedOccurrences)
	require.Empty(t, current.ActiveOccurrenceID)
	var count int64
	require.NoError(t, reportOwner.Table("report_center.saved_reports").Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, owner.Table("ai_workbench.business_tasks").Count(&count).Error)
	require.Zero(t, count, "no fabricated agent task or model path")
	t.Run("permission change sanitizes the actual committed bytes", func(t *testing.T) {
		o := execution.Origin
		o.OccurrenceID = uuid.NewString()
		snap := makeSnapshot("sensitive optional orders")
		snap.ScheduledOrigin = &o
		snap.Ref.ID = o.OccurrenceID
		r, e := reports.SaveScheduledCapture(ctx, o, snap, func(_ context.Context, snapshot *rc.Snapshot) error {
			snapshot.Content.Sections[0].Fields[0].Value = "来源当前不可用"
			return nil
		})
		require.NoError(t, e)
		saved, e := reports.Read(ctx, o.Scope(), r.ReportID)
		require.NoError(t, e)
		require.Equal(t, "来源当前不可用", saved.Content.Sections[0].Fields[0].Value)
	})
	for _, update := range []string{"budget_unit='ai-micropoint-v1',per_run_upper_bound_micros=1,total_upper_bound_micros=1", "task_selection='{}'::jsonb", "execution_spec='{\"model\":\"forbidden\"}'::jsonb", "report_spec='{}'::jsonb"} {
		require.Error(t, owner.Exec("UPDATE ai_workbench.scheduled_plans SET "+update+" WHERE id=?", plan.ID).Error)
	}
	t.Run("readiness rejects relaxed constraints and privilege drift", func(t *testing.T) {
		require.NoError(t, owner.Exec("ALTER TABLE ai_workbench.report_execution ALTER COLUMN capture_started DROP NOT NULL").Error)
		require.Error(t, VerifyDailyReportSchema(ctx, db))
		require.NoError(t, owner.Exec("ALTER TABLE ai_workbench.report_execution ALTER COLUMN capture_started SET NOT NULL").Error)
		require.NoError(t, owner.Exec("ALTER TABLE ai_workbench.scheduled_plans DROP CONSTRAINT scheduled_report_capability;ALTER TABLE ai_workbench.scheduled_plans ADD CONSTRAINT scheduled_report_capability CHECK(TRUE)").Error)
		require.Error(t, VerifyDailyReportSchema(ctx, db))
		require.NoError(t, InstallDailyReportSchema(owner))
		require.NoError(t, VerifyDailyReportSchema(ctx, db))
		require.NoError(t, owner.Exec("GRANT UPDATE(fingerprint) ON ai_workbench.report_execution TO ai_workbench_runtime").Error)
		require.Error(t, VerifyDailyReportSchema(ctx, db))
		require.NoError(t, owner.Exec("REVOKE UPDATE(fingerprint) ON ai_workbench.report_execution FROM ai_workbench_runtime").Error)
		require.NoError(t, VerifyDailyReportSchema(ctx, db))
		require.NoError(t, reportOwner.Exec("SET ROLE report_center_owner;ALTER TABLE report_center.saved_reports DROP CONSTRAINT saved_report_daily_origin;ALTER TABLE report_center.saved_reports ADD CONSTRAINT saved_report_daily_origin CHECK(TRUE);RESET ROLE").Error)
		require.Error(t, reportstore.VerifyDailyReportSchema(ctx, reportDB))
		require.NoError(t, reportstore.InstallDailyReportSchema(reportOwner))
		require.NoError(t, reportstore.VerifyDailyReportSchema(ctx, reportDB))
	})

}
