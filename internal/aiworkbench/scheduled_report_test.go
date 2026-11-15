package aiworkbench

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestDailyReportPlanHasNoProductOrModelBudget(t *testing.T) {
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	plan := ScheduledPlanDraft{ID: "e70eb8a5-1776-4514-8cda-d7f4a78dc001", Scope: Scope{OrganizationID: "org", ActorID: "actor"}, MemberID: "member", Report: &DailyReportSpec{StoreID: "e70eb8a5-1776-4514-8cda-d7f4a78dc002", Version: 1, PeriodPolicy: "previous_natural_day", BusinessZone: "Asia/Shanghai"}, GoalSummary: "店铺经营日报", CreatedAt: now, Rule: ScheduleRule{Frequency: ScheduleDaily, TimeZone: "Asia/Shanghai", Hour: 9}, Budget: ScheduleBudget{Unit: "none", MaxOccurrences: 2}}
	require.True(t, plan.Valid())
	require.Equal(t, "2026-11-13", plan.Report.Period(now.Add(time.Hour)))
	bad := plan
	bad.Budget.PerRunUpperBoundMicros = 1
	require.False(t, bad.Valid())
	bad = plan
	bad.ExecutionSpec = json.RawMessage(`{"model":"forbidden"}`)
	require.False(t, bad.Valid())
	bad = plan
	bad.Rule.Frequency = ScheduleWeekly
	require.False(t, bad.Valid())
	bad = plan
	bad.Selection.Capability = ContentTaskKind
	require.False(t, bad.Valid())
	require.Equal(t, plan.Report.Period(now.Add(time.Hour)), plan.Report.Period(now.Add(time.Hour)), "period is derived from nominal slot, never retry clock")
}
