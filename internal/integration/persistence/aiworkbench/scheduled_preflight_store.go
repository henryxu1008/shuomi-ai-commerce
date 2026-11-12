package aiworkbenchpersistence

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
	"time"
)

// RecordScheduledPreflightReceipt makes a rejected nominal callback terminal
// without claiming quota or changing an existing task's durable occupancy.
func (s *Store) RecordScheduledPreflightReceipt(ctx context.Context, scope aiworkbench.Scope, id string, slot, now time.Time, state, reason string) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || slot.IsZero() || slot.After(now.Add(5*time.Second)) || (state != "SKIPPED" && state != "BLOCKED") || reason == "" {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
			return missing(err)
		}
		var prior scheduledOccurrenceRow
		err := tx.Where("plan_id=? AND scheduled_at_utc=?", id, slot.UTC()).Take(&prior).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		policy, err := schedulePlanValue(row)
		if err != nil {
			return err
		}
		expected, err := policy.Rule.NextAfter(slot.Add(-time.Nanosecond))
		if err != nil || !expected.Equal(slot) || !slot.After(policy.CreatedAt) {
			return aiworkbench.ErrInvalid
		}
		receipt := newScheduledOccurrence(row, policy, slot, now, state)
		receipt.Reason = reason
		return unavailable(tx.Create(&receipt).Error)
	})
}
