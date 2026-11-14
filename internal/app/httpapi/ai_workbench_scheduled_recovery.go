package httpapi

import (
	"context"
	"errors"
	"task-processor/internal/aiworkbench"
	"time"
)

func (a *aiWorkbenchApplication) synchronizeScheduledPlan(ctx context.Context, plan aiworkbench.ScheduledPlan) error {
	if a == nil || a.schedules == nil || !a.schedules.Ready() {
		return aiworkbench.ErrUnavailable
	}
	if err := a.schedules.Synchronize(ctx, plan); err != nil {
		_ = a.store.NoteScheduledRegistrationUnknown(ctx, plan.Scope, plan.ID, plan.Revision)
		return err
	}
	return a.store.MarkScheduleRegistered(ctx, plan.Scope, plan.ID, plan.Revision)
}

func (a *aiWorkbenchApplication) recoverScheduledPlans(ctx context.Context) error {
	if a == nil || a.schedules == nil || !a.schedules.Ready() {
		return aiworkbench.ErrUnavailable
	}
	a.scheduleRecoveryMu.Lock()
	defer a.scheduleRecoveryMu.Unlock()
	plans, next, err := a.store.DueScheduledRecovery(ctx, a.scheduleRecoveryAfter, 20)
	if err != nil {
		return err
	}
	var failures []error
	for _, plan := range plans {
		if ctx.Err() != nil {
			return errors.Join(append(failures, ctx.Err())...)
		}
		// Advance even when authorization/Run evidence remains unresolved;
		// occupancy is untouched and the bounded scan wraps after the last page.
		a.scheduleRecoveryAfter = plan.ID
		// Temporal may omit an action outside its catch-up window entirely.
		// Repair missed history without dispatching a replacement paid task.
		plan, err = a.store.RecordExpiredScheduledSlots(ctx, plan.Scope, plan.ID, time.Now().UTC())
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if plan.RegistrationState != "MATCHED" {
			err = a.synchronizeScheduledPlan(ctx, plan)
			if err != nil {
				failures = append(failures, err)
			}
		}
		if plan.ActiveOccurrenceID != "" {
			occurrence, readErr := a.store.ReadActiveScheduledOccurrence(ctx, plan.Scope, plan.ID, plan.ActiveOccurrenceID)
			if readErr != nil {
				failures = append(failures, readErr)
				continue
			}
			_, err = a.ReconcileScheduled(ctx, plan.ID, occurrence.ScheduledAtUTC)
			if err != nil {
				failures = append(failures, err)
			}
		}
	}
	if next == "" {
		a.scheduleRecoveryAfter = ""
	}
	return errors.Join(failures...)
}

const scheduledRecoveryInterval = 30 * time.Second
