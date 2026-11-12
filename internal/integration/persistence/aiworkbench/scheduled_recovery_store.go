package aiworkbenchpersistence

import (
	"context"
	"task-processor/internal/aiworkbench"
	"time"
)

func (s *Store) DueScheduledRecovery(ctx context.Context, after string, limit int) ([]aiworkbench.ScheduledPlan, string, error) {
	if s == nil || s.db == nil || ctx == nil || after != "" && !validKey(after) || limit < 1 || limit > 50 {
		return nil, "", aiworkbench.ErrInvalid
	}
	var rows []scheduledPlanRow
	query := s.db.WithContext(ctx).Where("(registration_state <> 'MATCHED' OR active_occurrence_id IS NOT NULL)")
	if after != "" {
		query = query.Where("id > ?", after)
	}
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

func (s *Store) NoteScheduledRegistrationUnknown(ctx context.Context, scope aiworkbench.Scope, id string, revision int64) error {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(id) || revision < 1 {
		return aiworkbench.ErrInvalid
	}
	return s.db.WithContext(ctx).Model(&scheduledPlanRow{}).Where("id=? AND organization_id=? AND owner_user_id=? AND revision=?", id, scope.OrganizationID, scope.ActorID, revision).Updates(map[string]any{"registration_state": "UNKNOWN", "updated_at": time.Now().UTC()}).Error
}

func (s *Store) ReadActiveScheduledOccurrence(ctx context.Context, scope aiworkbench.Scope, planID, id string) (aiworkbench.ScheduledOccurrence, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(planID) || !validKey(id) {
		return aiworkbench.ScheduledOccurrence{}, aiworkbench.ErrInvalid
	}
	var row scheduledOccurrenceRow
	if err := s.db.WithContext(ctx).Where("id=? AND plan_id=? AND organization_id=? AND owner_user_id=?", id, planID, scope.OrganizationID, scope.ActorID).Take(&row).Error; err != nil {
		return aiworkbench.ScheduledOccurrence{}, missing(err)
	}
	return occurrenceReceipt(row), nil
}
