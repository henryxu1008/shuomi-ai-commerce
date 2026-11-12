package aiworkbenchpersistence

import (
	"context"

	"gorm.io/gorm"

	"task-processor/internal/aiworkbench"
)

type ScopedTaskPage struct {
	Tasks      []aiworkbench.BusinessTask
	Next       string
	Incomplete bool
}

type taskScopeRow struct {
	TaskRow       taskRow                    `gorm:"embedded"`
	BusinessScope *aiworkbench.BusinessScope `gorm:"serializer:json;type:jsonb"`
}

// Scheduled tasks carry a genuine immutable scope reference, not a fabricated
// Chat ancestor. Direct schedules without that reference remain unrecorded.
const taskBusinessScopeSQL = "(CASE WHEN t.source_kind = 'SCHEDULE_OCCURRENCE' THEN f.frozen_scope->'Scope' ELSE c.business_scope END)"

func (s *Store) scopedTasks(ctx context.Context, owner aiworkbench.Scope, filter aiworkbench.ScopeSelection) *gorm.DB {
	q := s.db.WithContext(ctx).Table("ai_workbench.business_tasks AS t").
		Select("t.*, "+taskBusinessScopeSQL+" AS business_scope").
		Joins("LEFT JOIN ai_workbench.conversations AS c ON c.id = t.conversation_id AND c.organization_id = t.organization_id AND c.owner_user_id = t.owner_user_id AND t.source_kind='CHAT_PROPOSAL'").
		Joins("LEFT JOIN ai_workbench.commands AS f ON t.source_kind='SCHEDULE_OCCURRENCE' AND f.organization_id=t.organization_id AND f.actor_id=t.owner_user_id AND f.idempotency_key::text=t.execution_request->'WorkbenchContextRef'->>'CommandKey' AND f.conversation_id::text=t.execution_request->'WorkbenchContextRef'->>'ConversationID' AND f.scope_digest=t.execution_request->'WorkbenchContextRef'->>'Digest' AND f.operation=?", messageOperation).
		Where("(t.source_kind='SCHEDULE_OCCURRENCE' OR c.id IS NOT NULL)").
		Where("t.organization_id = ? AND t.owner_user_id = ?", owner.OrganizationID, owner.ActorID)
	switch filter.Kind {
	case "general":
		q = q.Where(taskBusinessScopeSQL + "->>'kind' = 'general'")
	case "project":
		q = q.Where(taskBusinessScopeSQL+"->>'kind' = 'project' AND "+taskBusinessScopeSQL+"->>'projectId' = ?", filter.ProjectID)
	case "store":
		q = q.Where(taskBusinessScopeSQL+"->>'kind' IN ('store','project') AND "+taskBusinessScopeSQL+"->>'storeId' = ?", filter.StoreID)
	case "unrecorded":
		q = q.Where(taskBusinessScopeSQL + " IS NULL")
	}
	return q
}

func (s *Store) ListScopedTasks(ctx context.Context, owner aiworkbench.Scope, filter aiworkbench.ScopeSelection, after string, limit int,
	visible func(*aiworkbench.BusinessScope, aiworkbench.BusinessTask) bool) (ScopedTaskPage, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(owner) || !filter.ValidTaskFilter() || limit < 1 || limit > 50 || visible == nil {
		return ScopedTaskPage{}, aiworkbench.ErrInvalid
	}
	query := s.scopedTasks(ctx, owner, filter)
	first := ""
	read := func(row taskScopeRow) (aiworkbench.BusinessTask, bool) {
		item, err := task(row.TaskRow)
		if err != nil || (row.BusinessScope == nil) != item.WorkbenchContextRef.Absent() ||
			row.BusinessScope != nil && !row.BusinessScope.Valid() {
			return aiworkbench.BusinessTask{}, false
		}
		return item, visible(row.BusinessScope, item)
	}
	if after != "" {
		firstID, lastID, err := aiworkbench.DecodeTaskScopeCursor(owner, filter, after)
		if err != nil {
			return ScopedTaskPage{}, err
		}
		var anchors [2]taskScopeRow
		for i, id := range []string{firstID, lastID} {
			if err = s.scopedTasks(ctx, owner, filter).Where("t.id = ?", id).Take(&anchors[i]).Error; err != nil {
				return ScopedTaskPage{}, aiworkbench.ErrInvalid
			}
			if _, ok := read(anchors[i]); !ok {
				return ScopedTaskPage{}, aiworkbench.ErrInvalid
			}
		}
		head, tail := anchors[0], anchors[1]
		if tail.TaskRow.CreatedAt.After(head.TaskRow.CreatedAt) || tail.TaskRow.CreatedAt.Equal(head.TaskRow.CreatedAt) && tail.TaskRow.ID > head.TaskRow.ID {
			return ScopedTaskPage{}, aiworkbench.ErrInvalid
		}
		first = firstID
		query = query.Where("(t.created_at,t.id) <= (?, ?::uuid) AND (t.created_at,t.id) < (?, ?::uuid)", head.TaskRow.CreatedAt, head.TaskRow.ID, tail.TaskRow.CreatedAt, tail.TaskRow.ID)
	}
	var rows []taskScopeRow
	if err := query.Order("t.created_at DESC, t.id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return ScopedTaskPage{}, unavailable(err)
	}
	result := ScopedTaskPage{Tasks: make([]aiworkbench.BusinessTask, 0, limit)}
	var last string
	var lastPosition int
	for i, row := range rows {
		if i == limit {
			// This is lookahead only. Its identity must never be returned.
			break
		}
		item, ok := read(row)
		if !ok {
			result.Incomplete = true
			continue
		}
		if first == "" {
			first = item.ID
		}
		result.Tasks = append(result.Tasks, item)
		last, lastPosition = item.ID, i
	}
	if len(rows) > limit && last != "" {
		if lastPosition+1 < limit {
			result.Incomplete = true
		}
		next, err := aiworkbench.EncodeTaskScopeCursor(owner, filter, first, last)
		if err != nil {
			return ScopedTaskPage{}, err
		}
		result.Next = next
	} else if len(rows) > limit {
		result.Incomplete = true
	}
	return result, nil
}
