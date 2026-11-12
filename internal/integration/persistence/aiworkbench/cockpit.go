package aiworkbenchpersistence

import (
	"context"
	"time"

	"task-processor/internal/aiworkbench"
)

// ListNewStoreTasks uses the source's fixed scope as a candidate index.
// The consumer must still verify each task's frozen scope and live grants.
func (s *Store) ListNewStoreTasks(ctx context.Context, owner aiworkbench.Scope, member string, stores []string, start, end time.Time, limit int) ([]aiworkbench.BusinessTask, error) {
	if !validScope(owner) || !validName(member) || len(stores) == 0 || !start.Before(end) || limit < 1 {
		return nil, aiworkbench.ErrInvalid
	}
	var rows []taskRow
	err := s.scopedTasks(ctx, owner, aiworkbench.ScopeSelection{Kind: "all"}).
		Select("t.*").
		Where("((t.source_kind='CHAT_PROPOSAL' AND EXISTS (SELECT 1 FROM ai_workbench.commands cmd WHERE cmd.organization_id = t.organization_id AND cmd.actor_id = t.owner_user_id AND cmd.conversation_id = t.conversation_id AND cmd.operation = ? AND cmd.member_id = ?)) OR (t.source_kind='SCHEDULE_OCCURRENCE' AND f.frozen_scope->>'MemberID'=?))", createOperation, member, member).
		Where("t.created_at >= ? AND t.created_at < ? AND "+taskBusinessScopeSQL+"->>'kind' IN ('store','project') AND "+taskBusinessScopeSQL+"->>'storeId' IN ?", start, end, stores).
		Limit(limit).Find(&rows).Error
	if err != nil {
		return nil, unavailable(err)
	}
	result := make([]aiworkbench.BusinessTask, 0, len(rows))
	for _, row := range rows {
		item, err := task(row)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}
