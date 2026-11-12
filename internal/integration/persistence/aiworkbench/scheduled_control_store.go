package aiworkbenchpersistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"task-processor/internal/aiworkbench"
)

type scheduledControlRow struct {
	OrganizationID  string
	OwnerUserID     string
	IdempotencyKey  string
	PlanID          string `gorm:"default:null"`
	Outcome         string `gorm:"default:COMPLETED"`
	RequestDigest   string
	AppliedRevision int64
	CreatedAt       time.Time
}

func (scheduledControlRow) TableName() string { return "ai_workbench.scheduled_plan_commands" }

func scheduledControlDigest(id string, expected int64, action string, rule *aiworkbench.ScheduleRule, confirmation string) string {
	return digest(struct {
		Plan         string
		Revision     int64
		Action       string
		Rule         *aiworkbench.ScheduleRule
		Confirmation string
	}{id, expected, action, rule, confirmation})
}

// ReadScheduledControlReceipt is a scoped read of an already committed command.
// Current access authorization belongs to the caller; execution readiness is
// deliberately not repeated for an immutable durable result.
func (s *Store) ReadScheduledControlReceipt(ctx context.Context, scope aiworkbench.Scope, id, key string, expected int64, action string, rule *aiworkbench.ScheduleRule, confirmation string) (aiworkbench.ScheduledPlan, bool, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || !validKey(key) || expected < 1 {
		return aiworkbench.ScheduledPlan{}, false, aiworkbench.ErrInvalid
	}
	return s.readScheduledCommand(ctx, scope, id, key, scheduledControlDigest(id, expected, action, rule, confirmation))
}

// ChangeScheduledPlan records a bounded control receipt in the same transaction
// as the new desired configuration. Temporal is synchronized after commit.
func (s *Store) ChangeScheduledPlan(ctx context.Context, scope aiworkbench.Scope, id, key string, expected int64, action string, rule *aiworkbench.ScheduleRule, confirmation string, now time.Time) (aiworkbench.ScheduledPlan, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || !validKey(key) || expected < 1 || now.IsZero() ||
		(action != "pause" && action != "resume" && action != "close" && action != "edit") || (action == "edit") != (rule != nil) {
		return aiworkbench.ScheduledPlan{}, aiworkbench.ErrInvalid
	}
	fingerprint := scheduledControlDigest(id, expected, action, rule, confirmation)
	var result aiworkbench.ScheduledPlan
	var catchupPending bool
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockTaskKey(tx, scope, key); err != nil {
			return err
		}
		var row scheduledPlanRow
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND organization_id=? AND owner_user_id=?", id, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
			return missing(err)
		}
		var prior scheduledControlRow
		err := tx.Where("organization_id=? AND owner_user_id=? AND idempotency_key=?", scope.OrganizationID, scope.ActorID, key).Take(&prior).Error
		if err == nil {
			if prior.RequestDigest != fingerprint {
				return aiworkbench.ErrIdempotencyConflict
			}
			if prior.Outcome == "REJECTED" {
				return aiworkbench.ErrScheduleCommandRejected
			}
			if prior.PlanID != id {
				return aiworkbench.ErrIdempotencyConflict
			}
			result, err = scheduledPlanRecord(row)
			return err
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return unavailable(err)
		}
		if row.Revision != expected || row.State == "CLOSED" {
			return aiworkbench.ErrRevisionMismatch
		}
		if rule != nil {
			if _, err := rule.NextAfter(now); err != nil {
				return aiworkbench.ErrInvalid
			}
		}
		updates := map[string]any{"revision": expected + 1, "updated_at": now.UTC(), "registration_state": "PENDING"}
		switch action {
		case "pause":
			updates["state"] = "PAUSED"
		case "close":
			updates["state"] = "CLOSED"
		case "resume":
			if row.State != "PAUSED" {
				return aiworkbench.ErrRevisionMismatch
			}
			policy, err := schedulePlanValue(row)
			if err != nil {
				return err
			}
			if _, err := policy.Rule.NextAfter(now); err != nil || row.ConsumedOccurrences >= row.MaxOccurrences {
				return aiworkbench.ErrRevisionMismatch
			}
			catchupPending, err = recordScheduledPausedSlots(tx, row, policy, now)
			if err != nil || catchupPending {
				// Commit only the bounded history progress. Admission and the
				// command receipt remain paused until every past slot is fenced.
				return err
			}
			updates["state"] = "PENDING_REGISTRATION"
		case "edit":
			if rule.Frequency == aiworkbench.ScheduleOnce && row.MaxOccurrences != 1 {
				return aiworkbench.ErrInvalid
			}
			updates["frequency"], updates["time_zone"], updates["rule_start_at"] = string(rule.Frequency), rule.TimeZone, now.UTC()
			updates["once_at_utc"], updates["until_utc"] = nil, nil
			updates["local_hour"], updates["local_minute"], updates["local_weekday"] = nil, nil, nil
			if !rule.UntilUTC.IsZero() {
				updates["until_utc"] = rule.UntilUTC.UTC()
			}
			if rule.Frequency == aiworkbench.ScheduleOnce {
				updates["once_at_utc"] = rule.AtUTC.UTC()
			} else {
				updates["local_hour"], updates["local_minute"] = rule.Hour, rule.Minute
				if rule.Frequency == aiworkbench.ScheduleWeekly {
					updates["local_weekday"] = int(rule.Weekday)
				}
			}
			if row.State != "PAUSED" {
				updates["state"] = "PENDING_REGISTRATION"
			}
		}
		if err := tx.Model(&scheduledPlanRow{}).Where("id=? AND revision=?", id, expected).Updates(updates).Error; err != nil {
			return unavailable(err)
		}
		if err := tx.Create(&scheduledControlRow{OrganizationID: scope.OrganizationID, OwnerUserID: scope.ActorID, IdempotencyKey: key, PlanID: id, RequestDigest: fingerprint, AppliedRevision: expected + 1, CreatedAt: now.UTC()}).Error; err != nil {
			return unavailable(err)
		}
		result, err = (&Store{db: tx}).ReadScheduledPlan(ctx, scope, id)
		return err
	})
	if err == nil && catchupPending {
		err = errors.Join(aiworkbench.ErrUnavailable, aiworkbench.ErrScheduleCatchupPending)
	}
	return result, err
}

func (s *Store) ListScheduledPlans(ctx context.Context, scope aiworkbench.Scope, after string, limit int) ([]aiworkbench.ScheduledPlan, string, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || limit < 1 || limit > 50 || after != "" && !validKey(after) {
		return nil, "", aiworkbench.ErrInvalid
	}
	query := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=?", scope.OrganizationID, scope.ActorID)
	if after != "" {
		query = query.Where("id > ?", after)
	}
	var rows []scheduledPlanRow
	if err := query.Order("id").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, "", unavailable(err)
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = rows[len(rows)-1].ID
	}
	values := make([]aiworkbench.ScheduledPlan, 0, len(rows))
	for _, row := range rows {
		value, err := scheduledPlanRecord(row)
		if err != nil {
			return nil, "", err
		}
		values = append(values, value)
	}
	return values, next, nil
}

func (s *Store) ListScheduledOccurrences(ctx context.Context, scope aiworkbench.Scope, id, after string, limit int) ([]aiworkbench.ScheduledOccurrence, string, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || after != "" && !validKey(after) || limit < 1 || limit > 50 {
		return nil, "", aiworkbench.ErrInvalid
	}
	if _, err := s.ReadScheduledPlan(ctx, scope, id); err != nil {
		return nil, "", err
	}
	query := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=? AND plan_id=?", scope.OrganizationID, scope.ActorID, id)
	if after != "" {
		var cursor scheduledOccurrenceRow
		if err := query.Session(&gorm.Session{}).Where("id=?", after).Take(&cursor).Error; err != nil {
			return nil, "", missing(err)
		}
		query = query.Where("scheduled_at_utc < ?", cursor.ScheduledAtUTC)
	}
	var rows []scheduledOccurrenceRow
	if err := query.Order("scheduled_at_utc DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, "", unavailable(err)
	}
	next := ""
	if len(rows) > limit {
		rows = rows[:limit]
		next = rows[len(rows)-1].ID
	}
	values := make([]aiworkbench.ScheduledOccurrence, 0, len(rows))
	for _, row := range rows {
		values = append(values, occurrenceReceipt(row))
	}
	return values, next, nil
}
