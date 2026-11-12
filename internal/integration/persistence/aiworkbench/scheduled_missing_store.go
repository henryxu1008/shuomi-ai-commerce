package aiworkbenchpersistence

import (
	"errors"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"time"
)

// Caller holds the plan row lock. Every callback insertion path advances the
// same history first, so a later rejection cannot hide earlier missing slots.
func recordScheduledMissingBefore(tx *gorm.DB, plan scheduledPlanRow, policy aiworkbench.ScheduledPlanDraft, slot, now time.Time) (bool, error) {
	var previous scheduledOccurrenceRow
	err := tx.Where("plan_id=?", plan.ID).Order("scheduled_at_utc DESC").Take(&previous).Error
	anchor := policy.CreatedAt
	if plan.RuleStartAt != nil {
		anchor = plan.RuleStartAt.UTC()
	}
	if err == nil {
		if previous.ScheduledAtUTC.After(anchor) {
			anchor = previous.ScheduledAtUTC
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, unavailable(err)
	}
	valid, err := policy.Rule.NextAfter(slot.Add(-time.Nanosecond))
	if err != nil || !valid.Equal(slot) || !slot.After(anchor) {
		return false, aiworkbench.ErrInvalid
	}
	expected, err := policy.Rule.NextAfter(anchor)
	if err != nil {
		return false, aiworkbench.ErrInvalid
	}
	for missed := 0; expected.Before(slot); missed++ {
		if missed == 256 {
			return true, nil
		}
		receipt := newScheduledOccurrence(plan, policy, expected, now, "SKIPPED")
		if err := tx.Create(&receipt).Error; err != nil {
			return false, unavailable(err)
		}
		expected, err = policy.Rule.NextAfter(expected)
		if err != nil {
			return false, aiworkbench.ErrInvalid
		}
	}
	if !expected.Equal(slot) {
		return false, aiworkbench.ErrInvalid
	}
	return false, nil
}
