package workbenchschedules

import (
	"context"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"task-processor/internal/aiworkbench"
	"testing"
	"time"
)

func TestScheduleShapePreservesTimeAndOwnerAdmission(t *testing.T) {
	now := time.Date(2026, 11, 12, 1, 0, 0, 0, time.UTC)
	plan := aiworkbench.ScheduledPlan{ScheduledPlanDraft: aiworkbench.ScheduledPlanDraft{ID: uuid.NewString(), Rule: aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleOnce, TimeZone: "Asia/Shanghai", AtUTC: now.Add(time.Hour)}, CreatedAt: now}, Revision: 1, State: "PENDING_REGISTRATION"}
	opts, err := scheduleOptions(plan)
	require.NoError(t, err)
	require.Equal(t, "UTC", opts.Spec.TimeZoneName)
	require.Equal(t, 2, opts.Spec.Calendars[0].Hour[0].Start)
	require.Equal(t, enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL, opts.Overlap, "the Workbench transaction records every skipped occurrence and owns actual Agent overlap")
	require.Equal(t, 15*time.Minute, opts.CatchupWindow)
	plan.Rule = aiworkbench.ScheduleRule{Frequency: aiworkbench.ScheduleWeekly, TimeZone: "America/New_York", Weekday: time.Sunday, Hour: 2, Minute: 30}
	opts, err = scheduleOptions(plan)
	require.NoError(t, err)
	require.Equal(t, plan.Rule.TimeZone, opts.Spec.TimeZoneName)
	require.Equal(t, 0, opts.Spec.Calendars[0].DayOfWeek[0].Start)
	plan.State = "PAUSED"
	opts, err = scheduleOptions(plan)
	require.NoError(t, err)
	require.True(t, opts.Paused)
}

func TestWorkflowUsesNominalSlotAndRejectsUnboundStarts(t *testing.T) {
	for _, scheduled := range []bool{false, true} {
		t.Run(map[bool]string{false: "unbound", true: "scheduled"}[scheduled], func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			id := uuid.NewString()
			slot := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
			if scheduled {
				env.RegisterActivityWithOptions(func(context.Context, string, time.Time) error { return nil }, activity.RegisterOptions{Name: dispatchActivity})
				env.RegisterActivityWithOptions(func(context.Context, string, time.Time) (bool, error) { return true, nil }, activity.RegisterOptions{Name: reconcileActivity})
				require.NoError(t, env.SetTypedSearchAttributesOnStart(temporal.NewSearchAttributes(
					temporal.NewSearchAttributeKeyTime("TemporalScheduledStartTime").ValueSet(slot),
					temporal.NewSearchAttributeKeyKeyword("TemporalScheduledById").ValueSet(scheduleID(id)))))
				env.OnActivity(dispatchActivity, mock.Anything, id, slot).Return(nil).Once()
				env.OnActivity(reconcileActivity, mock.Anything, id, slot).Return(true, nil).Once()
			}
			env.ExecuteWorkflow(Workflow, id)
			if scheduled {
				require.NoError(t, env.GetWorkflowError())
				env.AssertExpectations(t)
			} else {
				require.Error(t, env.GetWorkflowError())
			}
		})
	}
}
