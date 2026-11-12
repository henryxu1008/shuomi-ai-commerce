package aiworkbenchpersistence

import (
	"context"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
	"time"
)

// ReconcileScheduledTask consumes a proof produced only by the application
// after exact Run/invocation/resource-owner readback. A UI projection or error
// state is never an argument that can release the durable occupancy.
func (s *Store) ReconcileScheduledTask(ctx context.Context, scope aiworkbench.Scope, planID, occurrenceID, taskID, proof string) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(planID) || !validKey(occurrenceID) || !validKey(taskID) || !validDigest(proof) {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", planID, scope.OrganizationID, scope.ActorID).Take(&plan).Error; err != nil {
			return missing(err)
		}
		var occurrence scheduledOccurrenceRow
		if err := tx.Where("id=? AND plan_id=? AND organization_id=? AND owner_user_id=? AND business_task_id=?", occurrenceID, planID, scope.OrganizationID, scope.ActorID, taskID).Take(&occurrence).Error; err != nil {
			return missing(err)
		}
		if occurrence.State == "RECONCILED" {
			if occurrence.ReconciliationDigest == nil || *occurrence.ReconciliationDigest != proof {
				return aiworkbench.ErrIdempotencyConflict
			}
			return nil
		}
		if plan.ActiveOccurrenceID == nil || *plan.ActiveOccurrenceID != occurrenceID {
			return aiworkbench.ErrRevisionMismatch
		}
		task, err := (&Store{db: tx}).GetTask(ctx, scope, taskID)
		if err != nil || task.ScheduledOccurrenceID != occurrenceID {
			return aiworkbench.ErrRevisionMismatch
		}
		now := time.Now().UTC()
		if err := tx.Model(&scheduledOccurrenceRow{}).Where("id=?", occurrenceID).Updates(map[string]any{"state": "RECONCILED", "reconciliation_digest": proof, "reason": "RUN_QUIESCENT", "updated_at": now}).Error; err != nil {
			return unavailable(err)
		}
		updates := map[string]any{"active_occurrence_id": nil, "updated_at": now}
		if plan.State == "ACTIVE" && plan.ConsumedOccurrences >= plan.MaxOccurrences {
			updates["state"], updates["registration_state"], updates["revision"] = "CLOSED", "PENDING", plan.Revision+1
		}
		return tx.Model(&scheduledPlanRow{}).Where("id=? AND active_occurrence_id=?", planID, occurrenceID).Updates(updates).Error
	})
}

func (s *Store) NoteScheduledOccurrence(ctx context.Context, scope aiworkbench.Scope, id, state, reason string) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || !validName(reason) || len(reason) > 128 || (state != "CLAIMED" && state != "DISPATCHED" && state != "UNKNOWN") {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Model(&scheduledOccurrenceRow{}).Where("id=? AND organization_id=? AND owner_user_id=? AND state IN ?", id, scope.OrganizationID, scope.ActorID, []string{"CLAIMED", "DISPATCHED", "UNKNOWN"}).Updates(map[string]any{"state": state, "reason": reason, "updated_at": time.Now().UTC()}).Error
}
