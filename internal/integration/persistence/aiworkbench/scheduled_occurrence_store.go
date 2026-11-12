package aiworkbenchpersistence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"task-processor/internal/aiworkbench"
)

// ScheduledOccurrence is a durable Workbench dispatch-intent receipt, not a
// claim that the AgentRun has started or that paid usage is known.
type ScheduledOccurrence struct {
	ID             string
	PlanID         string
	Scope          aiworkbench.Scope
	ScheduledAtUTC time.Time
	PlanRevision   int64
	PayloadDigest  string
	State          string
}

type scheduledOccurrenceRow struct {
	ID             string
	OrganizationID string
	OwnerUserID    string
	PlanID         string
	ScheduledAtUTC time.Time
	PlanRevision   int64
	PayloadDigest  string
	State          string
	BusinessTaskID *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (scheduledOccurrenceRow) TableName() string { return "ai_workbench.scheduled_occurrences" }

func occurrenceReceipt(row scheduledOccurrenceRow) ScheduledOccurrence {
	return ScheduledOccurrence{ID: row.ID, PlanID: row.PlanID,
		Scope:          aiworkbench.Scope{OrganizationID: row.OrganizationID, ActorID: row.OwnerUserID},
		ScheduledAtUTC: row.ScheduledAtUTC.UTC(), PlanRevision: row.PlanRevision,
		PayloadDigest: row.PayloadDigest, State: row.State}
}

// ClaimScheduledOccurrence performs no external owner operations under its
// plan row lock. It is a narrow persistence primitive invoked only after a
// trusted Temporal callback has been checked by the application owner.
// The returned CLAIMED state is NOT permission to send AI requests: the
// original Agent/Resource owners must still authorize and bind one BusinessTask.
func (s *Store) ClaimScheduledOccurrence(ctx context.Context, scope aiworkbench.Scope, planID string, scheduledAtUTC, now time.Time) (ScheduledOccurrence, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) ||
		!validKey(planID) || scheduledAtUTC.IsZero() || now.IsZero() ||
		scheduledAtUTC.After(now.Add(5*time.Second)) {
		return ScheduledOccurrence{}, false, aiworkbench.ErrInvalid
	}
	scheduledAtUTC = scheduledAtUTC.UTC()
	var receipt ScheduledOccurrence
	replay := false
	catchupPending := false
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var plan scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND organization_id = ? AND owner_user_id = ?",
				planID, scope.OrganizationID, scope.ActorID).Take(&plan).Error; err != nil {
			return missing(err)
		}
		var existing scheduledOccurrenceRow
		err := tx.Where("organization_id = ? AND owner_user_id = ? AND plan_id = ? AND scheduled_at_utc = ?",
			scope.OrganizationID, scope.ActorID, planID, scheduledAtUTC).Take(&existing).Error
		if err == nil {
			receipt, replay = occurrenceReceipt(existing), true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		policy, err := schedulePlanValue(plan)
		if err != nil {
			return err
		}
		// A prior slot always advances the calendar sequence even if it was
		// skipped due to pause/overlap. Do not invent missing paid catch-ups.
		var previous scheduledOccurrenceRow
		err = tx.Where("organization_id = ? AND owner_user_id = ? AND plan_id = ?",
			scope.OrganizationID, scope.ActorID, planID).
			Order("scheduled_at_utc DESC").Take(&previous).Error
		anchor := policy.CreatedAt
		if err == nil {
			anchor = previous.ScheduledAtUTC
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		// Verify the supplied slot before recording any missed dates. Temporal
		// may omit callbacks while paused/offline; the next valid date remains
		// eligible, but omitted dates must never become delayed paid calls.
		validSlot, err := policy.Rule.NextAfter(scheduledAtUTC.Add(-time.Nanosecond))
		if err != nil || !validSlot.Equal(scheduledAtUTC) || !scheduledAtUTC.After(anchor) {
			return aiworkbench.ErrInvalid
		}
		expected, err := policy.Rule.NextAfter(anchor)
		if err != nil {
			return aiworkbench.ErrInvalid
		}
		// Persist bounded batches so a long outage cannot hold a plan lock for
		// an unbounded number of inserts. Retry the same callback to continue.
		for missed := 0; expected.Before(scheduledAtUTC); missed++ {
			if missed == 256 {
				catchupPending = true
				return nil
			}
			row := newScheduledOccurrence(plan, policy, expected, now, "SKIPPED")
			if err := tx.Create(&row).Error; err != nil {
				return unavailable(err)
			}
			expected, err = policy.Rule.NextAfter(expected)
			if err != nil {
				return aiworkbench.ErrInvalid
			}
		}
		if !expected.Equal(scheduledAtUTC) {
			return aiworkbench.ErrInvalid
		}

		state := "SKIPPED"
		claim := plan.State == "ACTIVE" && plan.ActiveOccurrenceID == nil &&
			plan.ConsumedOccurrences < plan.MaxOccurrences &&
			!scheduledAtUTC.Before(now.Add(-15*time.Minute))
		if claim {
			state = "CLAIMED"
		}
		row := newScheduledOccurrence(plan, policy, scheduledAtUTC, now, state)
		if err := tx.Create(&row).Error; err != nil {
			return unavailable(err)
		}
		if claim {
			res := tx.Model(&scheduledPlanRow{}).
				Where("id = ? AND organization_id = ? AND owner_user_id = ? AND state = 'ACTIVE' AND active_occurrence_id IS NULL AND consumed_occurrences < max_occurrences",
					planID, scope.OrganizationID, scope.ActorID).
				Updates(map[string]any{
					"active_occurrence_id": row.ID,
					"consumed_occurrences": gorm.Expr("consumed_occurrences + 1"),
					"updated_at":           now.UTC(),
				})
			if res.Error != nil {
				return unavailable(res.Error)
			}
			if res.RowsAffected != 1 {
				return aiworkbench.ErrRevisionMismatch
			}
		}
		receipt = occurrenceReceipt(row)
		return nil
	})
	if err != nil {
		return ScheduledOccurrence{}, false, err
	}
	if catchupPending {
		return ScheduledOccurrence{}, false, aiworkbench.ErrUnavailable
	}
	return receipt, replay, nil
}

func newScheduledOccurrence(plan scheduledPlanRow, policy aiworkbench.ScheduledPlanDraft, slot, now time.Time, state string) scheduledOccurrenceRow {
	return scheduledOccurrenceRow{
		ID:             uuid.NewSHA1(uuid.NameSpaceOID, []byte(plan.OrganizationID+"\x1f"+plan.ID+"\x1f"+slot.UTC().Format(time.RFC3339Nano))).String(),
		OrganizationID: plan.OrganizationID, OwnerUserID: plan.OwnerUserID, PlanID: plan.ID,
		ScheduledAtUTC: slot.UTC(), PlanRevision: plan.Revision,
		PayloadDigest: digest(struct {
			PlanID    string
			Slot      time.Time
			Revision  int64
			Selection aiworkbench.TaskSelection
			Budget    aiworkbench.ScheduleBudget
		}{plan.ID, slot.UTC(), plan.Revision, policy.Selection, policy.Budget}),
		State: state, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
}
