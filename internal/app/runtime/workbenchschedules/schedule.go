// Package workbenchschedules adapts bounded Workbench plans to Temporal Schedules.
package workbenchschedules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	common "go.temporal.io/api/common/v1"
	enums "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"task-processor/internal/aiworkbench"
)

const (
	TaskQueue         = "ai-workbench-scheduled-v1"
	workflowName      = "ai-workbench.scheduled.v1"
	dispatchActivity  = "ai-workbench.scheduled.dispatch.v1"
	reconcileActivity = "ai-workbench.scheduled.reconcile.v1"
)

type Backend struct {
	Client client.Client
	ready  atomic.Bool
}

func (b *Backend) Ready() bool    { return b != nil && b.ready.Load() }
func scheduleID(id string) string { return "ai-workbench-schedule-" + id }

func Workflow(ctx workflow.Context, id string) error {
	attributes := workflow.GetTypedSearchAttributes(ctx)
	slot, hasSlot := attributes.GetTime(temporal.NewSearchAttributeKeyTime("TemporalScheduledStartTime"))
	by, hasBy := attributes.GetKeyword(temporal.NewSearchAttributeKeyKeyword("TemporalScheduledById"))
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.String() != id || parsed == uuid.Nil || !hasSlot || slot.IsZero() || !hasBy || by != scheduleID(id) {
		return temporal.NewNonRetryableApplicationError("unbound scheduled execution", "INVALID_SCHEDULE", nil)
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 4 * time.Minute, ScheduleToCloseTimeout: 15 * time.Minute, RetryPolicy: &temporal.RetryPolicy{InitialInterval: time.Second, MaximumInterval: 30 * time.Second, MaximumAttempts: 8}})
	if err := workflow.ExecuteActivity(ctx, dispatchActivity, id, slot.UTC()).Get(ctx, nil); err != nil {
		return err
	}
	return workflow.ExecuteActivity(ctx, reconcileActivity, id, slot.UTC()).Get(ctx, nil)
}

func (b *Backend) Start(dispatcher aiworkbench.ScheduledDispatcher) (func(), error) {
	if b == nil || b.Client == nil || dispatcher == nil {
		return nil, aiworkbench.ErrUnavailable
	}
	w := worker.New(b.Client, TaskQueue, worker.Options{MaxConcurrentActivityExecutionSize: 2, WorkerStopTimeout: 10 * time.Second, OnFatalError: func(error) { b.ready.Store(false) }})
	w.RegisterWorkflowWithOptions(Workflow, workflow.RegisterOptions{Name: workflowName})
	w.RegisterActivityWithOptions(dispatcher.DispatchScheduled, activity.RegisterOptions{Name: dispatchActivity})
	w.RegisterActivityWithOptions(dispatcher.ReconcileScheduled, activity.RegisterOptions{Name: reconcileActivity})
	if err := w.Start(); err != nil {
		return nil, err
	}
	b.ready.Store(true)
	var stopOnce sync.Once
	return func() { stopOnce.Do(func() { b.ready.Store(false); w.Stop() }) }, nil
}

func exact(n int) []client.ScheduleRange { return []client.ScheduleRange{{Start: n, End: n, Step: 1}} }
func scheduleOptions(plan aiworkbench.ScheduledPlan) (client.ScheduleOptions, error) {
	if !plan.Rule.Valid() || plan.Revision < 1 {
		return client.ScheduleOptions{}, aiworkbench.ErrInvalidSchedule
	}
	start := plan.CreatedAt
	if !plan.RuleStartAt.IsZero() {
		start = plan.RuleStartAt
	}
	spec := client.ScheduleSpec{TimeZoneName: plan.Rule.TimeZone, StartAt: start.UTC(), EndAt: plan.Rule.UntilUTC.UTC()}
	calendar := client.ScheduleCalendarSpec{Second: exact(0), Minute: exact(plan.Rule.Minute), Hour: exact(plan.Rule.Hour)}
	if plan.Rule.Frequency == aiworkbench.ScheduleWeekly {
		calendar.DayOfWeek = exact(int(plan.Rule.Weekday))
	}
	if plan.Rule.Frequency == aiworkbench.ScheduleOnce {
		slot := plan.Rule.AtUTC.UTC()
		if slot.Nanosecond() != 0 {
			return client.ScheduleOptions{}, aiworkbench.ErrInvalidSchedule
		}
		spec.TimeZoneName = "UTC"
		calendar.Second, calendar.Minute, calendar.Hour = exact(slot.Second()), exact(slot.Minute()), exact(slot.Hour())
		calendar.DayOfMonth, calendar.Month, calendar.Year = exact(slot.Day()), exact(int(slot.Month())), exact(slot.Year())
	}
	spec.Calendars = []client.ScheduleCalendarSpec{calendar}
	paused := plan.State == "PAUSED" || plan.State == "CLOSED"
	wire, _ := json.Marshal(struct {
		Draft  aiworkbench.ScheduledPlanDraft
		Paused bool
	}{plan.ScheduledPlanDraft, paused})
	sum := sha256.Sum256(wire)
	return client.ScheduleOptions{ID: scheduleID(plan.ID), Spec: spec, Overlap: enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL, CatchupWindow: 15 * time.Minute, Paused: paused,
		Action: &client.ScheduleWorkflowAction{ID: "ai-workbench-occurrence-" + plan.ID, Workflow: workflowName, TaskQueue: TaskQueue, Args: []interface{}{plan.ID}, WorkflowExecutionTimeout: 20 * time.Minute, WorkflowRunTimeout: 20 * time.Minute,
			Memo: map[string]interface{}{"workbench_plan": plan.ID, "revision": plan.Revision, "intent_digest": hex.EncodeToString(sum[:])}}}, nil
}

func decode(value interface{}, target interface{}) error {
	if payload, ok := value.(*common.Payload); ok {
		return converter.GetDefaultDataConverter().FromPayload(payload, target)
	}
	wire, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(wire, target)
}

func actionIdentity(action client.ScheduleAction) (string, int64, string, error) {
	a, ok := action.(*client.ScheduleWorkflowAction)
	if !ok || a == nil || len(a.Args) != 1 || a.Workflow != workflowName || a.TaskQueue != TaskQueue {
		return "", 0, "", aiworkbench.ErrUnavailable
	}
	var id, arg, hash string
	var revision int64
	if decode(a.Memo["workbench_plan"], &id) != nil || decode(a.Args[0], &arg) != nil || id != arg || decode(a.Memo["revision"], &revision) != nil || decode(a.Memo["intent_digest"], &hash) != nil || revision < 1 || len(hash) != 64 {
		return "", 0, "", aiworkbench.ErrUnavailable
	}
	return id, revision, hash, nil
}

func canonicalRanges(value []client.ScheduleRange, fallback []client.ScheduleRange) []client.ScheduleRange {
	if len(value) == 0 {
		value = fallback
	}
	if len(value) == 0 {
		return nil
	}
	result := append([]client.ScheduleRange(nil), value...)
	for i := range result {
		if result[i].End < result[i].Start {
			result[i].End = result[i].Start
		}
		if result[i].Step == 0 {
			result[i].Step = 1
		}
	}
	return result
}
func canonicalCalendar(c client.ScheduleCalendarSpec) client.ScheduleCalendarSpec {
	c.Comment = ""
	c.Second = canonicalRanges(c.Second, exact(0))
	c.Minute = canonicalRanges(c.Minute, exact(0))
	c.Hour = canonicalRanges(c.Hour, exact(0))
	c.DayOfMonth = canonicalRanges(c.DayOfMonth, []client.ScheduleRange{{Start: 1, End: 31, Step: 1}})
	c.Month = canonicalRanges(c.Month, []client.ScheduleRange{{Start: 1, End: 12, Step: 1}})
	c.DayOfWeek = canonicalRanges(c.DayOfWeek, []client.ScheduleRange{{Start: 0, End: 6, Step: 1}})
	c.Year = canonicalRanges(c.Year, nil)
	return c
}
func sameSpec(a, b client.ScheduleSpec) bool {
	if a.TimeZoneName != b.TimeZoneName || !a.StartAt.Equal(b.StartAt) || !a.EndAt.Equal(b.EndAt) || a.Jitter != b.Jitter || len(a.Calendars) != len(b.Calendars) || len(a.Intervals) != 0 || len(a.CronExpressions) != 0 || len(a.Skip) != 0 {
		return false
	}
	for i := range a.Calendars {
		if !reflect.DeepEqual(canonicalCalendar(a.Calendars[i]), canonicalCalendar(b.Calendars[i])) {
			return false
		}
	}
	return true
}

// Synchronize verifies the exact stable schedule after create/update, including
// response loss. SDK optimistic updates reject stale projection revisions.
func (b *Backend) Synchronize(ctx context.Context, plan aiworkbench.ScheduledPlan) error {
	if b == nil || b.Client == nil || ctx == nil {
		return aiworkbench.ErrUnavailable
	}
	options, err := scheduleOptions(plan)
	if err != nil {
		return err
	}
	schedules := b.Client.ScheduleClient()
	handle := schedules.GetHandle(ctx, options.ID)
	_, createErr := schedules.Create(ctx, options)
	if createErr != nil {
		err = handle.Update(ctx, client.ScheduleUpdateOptions{DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			id, revision, _, err := actionIdentity(in.Description.Schedule.Action)
			if err != nil || id != plan.ID {
				return nil, aiworkbench.ErrUnavailable
			}
			if revision > plan.Revision {
				return nil, aiworkbench.ErrRevisionMismatch
			}
			return &client.ScheduleUpdate{Schedule: &client.Schedule{Action: options.Action, Spec: &options.Spec, Policy: &client.SchedulePolicies{Overlap: options.Overlap, CatchupWindow: options.CatchupWindow}, State: &client.ScheduleState{Paused: options.Paused}}}, nil
		}})
		if err != nil {
			return errors.Join(createErr, err)
		}
	}
	description, err := handle.Describe(ctx)
	if err != nil {
		return err
	}
	id, revision, hash, err := actionIdentity(description.Schedule.Action)
	_, _, expectedHash, _ := actionIdentity(options.Action)
	if err != nil || id != plan.ID || revision != plan.Revision || hash != expectedHash || description.Schedule.Spec == nil || !sameSpec(*description.Schedule.Spec, options.Spec) || description.Schedule.State == nil || description.Schedule.State.Paused != options.Paused || description.Schedule.Policy == nil || description.Schedule.Policy.Overlap != options.Overlap || description.Schedule.Policy.CatchupWindow != options.CatchupWindow {
		return aiworkbench.ErrUnavailable
	}
	return nil
}
