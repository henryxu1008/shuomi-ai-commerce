package aiworkbenchpersistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
	"time"
)

// RecordExpiredScheduledSlots advances only expired calendar history. It never
// claims quota, creates a BusinessTask, releases occupancy or sends paid work.
// The same plan lock serializes this path with callbacks, edits and pause.
func (s *Store) RecordExpiredScheduledSlots(ctx context.Context, scope aiworkbench.Scope, id string, now time.Time) (aiworkbench.ScheduledPlan, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || now.IsZero() {
		return aiworkbench.ScheduledPlan{}, aiworkbench.ErrInvalid
	}
	var result aiworkbench.ScheduledPlan
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
			return missing(err)
		}
		if row.State == "PAUSED" || row.State == "CLOSED" {
			var err error
			result, err = scheduledPlanRecord(row)
			return err
		}
		policy, err := schedulePlanValue(row)
		if err != nil {
			return err
		}
		anchor := policy.CreatedAt
		if !policy.RuleStartAt.IsZero() {
			anchor = policy.RuleStartAt
		}
		var previous scheduledOccurrenceRow
		err = tx.Where("plan_id=?", id).Order("scheduled_at_utc DESC").Take(&previous).Error
		if err == nil {
			if previous.ScheduledAtUTC.After(anchor) {
				anchor = previous.ScheduledAtUTC
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		cutoff := now.UTC().Add(-15 * time.Minute)
		for recorded := 0; recorded < 256; recorded++ {
			slot, nextErr := policy.Rule.NextAfter(anchor)
			if errors.Is(nextErr, aiworkbench.ErrScheduleEnded) {
				// No future calendar action remains. Closing future admission keeps
				// any existing unknown task and its consumed quota untouched.
				if err := tx.Model(&scheduledPlanRow{}).Where("id=? AND revision=?", id, row.Revision).Updates(map[string]any{"state": "CLOSED", "registration_state": "PENDING", "revision": row.Revision + 1, "updated_at": now.UTC()}).Error; err != nil {
					return unavailable(err)
				}
				break
			}
			if nextErr != nil {
				return nextErr
			}
			if !slot.Before(cutoff) {
				break
			}
			receipt := newScheduledOccurrence(row, policy, slot, now, "SKIPPED")
			receipt.Reason = "CATCHUP_EXPIRED"
			if err := tx.Create(&receipt).Error; err != nil {
				return unavailable(err)
			}
			anchor = slot
		}
		result, err = (&Store{db: tx}).ReadScheduledPlan(ctx, scope, id)
		return err
	})
	return result, err
}
