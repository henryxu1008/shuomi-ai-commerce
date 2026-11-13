package aiworkbenchpersistence

import (
	"context"
	"encoding/json"

	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
)

const productContextPromptVersion = "ai-workbench-chat-plan-v4"

// AssistantSequence binds a durable reply to its owner and conversation.
func (s *Store) AssistantSequence(ctx context.Context, scope aiworkbench.Scope, conversation, id string) (uint64, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversation) || !validKey(id) {
		return 0, aiworkbench.ErrInvalid
	}
	var row messageRow
	if err := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=? AND conversation_id=? AND id=? AND author_kind=?", scope.OrganizationID, scope.ActorID, conversation, id, string(aiworkbench.AuthorAssistant)).Take(&row).Error; err != nil {
		return 0, missing(err)
	}
	if row.Sequence == 0 {
		return 0, aiworkbench.ErrUnavailable
	}
	return row.Sequence, nil
}

// ProductDependencies resolves each assistant message to exactly one original
// command before extracting frozen product selections. Other tenants and users
// are excluded by both queries. A malformed association fails closed.
func (s *Store) ProductDependencies(ctx context.Context, scope aiworkbench.Scope, conversation string, through uint64) ([]*aiworkbench.TaskSelection, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversation) || through == 0 {
		return nil, aiworkbench.ErrInvalid
	}
	var messages []messageRow
	if err := s.db.WithContext(ctx).Where("organization_id=? AND owner_user_id=? AND conversation_id=? AND author_kind=? AND sequence<=?", scope.OrganizationID, scope.ActorID, conversation, string(aiworkbench.AuthorAssistant), through).Order("sequence").Limit(5001).Find(&messages).Error; err != nil {
		return nil, unavailable(err)
	}
	if len(messages) > 5000 {
		return nil, aiworkbench.ErrUnavailable
	}
	if len(messages) == 0 {
		return nil, nil
	}
	ids := make([]string, len(messages))
	for i := range messages {
		ids[i] = messages[i].ID
	}
	var rows []commandRow
	if err := s.db.WithContext(ctx).Where("organization_id=? AND actor_id=? AND conversation_id=? AND assistant_message_id IN ?", scope.OrganizationID, scope.ActorID, conversation, ids).Find(&rows).Error; err != nil {
		return nil, unavailable(err)
	}
	return planningProductDependencies(messages, rows)
}

func planningProductDependencies(messages []messageRow, rows []commandRow) ([]*aiworkbench.TaskSelection, error) {
	byID := map[string]commandRow{}
	for _, row := range rows {
		if row.AssistantMessageID == nil || *row.AssistantMessageID == "" {
			return nil, aiworkbench.ErrUnavailable
		}
		id := *row.AssistantMessageID
		if _, exists := byID[id]; exists {
			return nil, aiworkbench.ErrUnavailable
		}
		byID[id] = row
	}
	var result []*aiworkbench.TaskSelection
	for _, m := range messages {
		row, ok := byID[m.ID]
		if !ok {
			return nil, aiworkbench.ErrUnavailable
		}
		if row.Operation != messageOperation && row.Operation != routeOperation && row.Operation != explicitOverviewOperation {
			return nil, aiworkbench.ErrUnavailable
		}
		if row.Operation != messageOperation {
			continue
		}
		var profile aicapability.ModelProfile
		if len(row.PlannerModelProfile) == 0 || json.Unmarshal(row.PlannerModelProfile, &profile) != nil || profile.PromptVersion == "" {
			return nil, aiworkbench.ErrUnavailable
		}
		if profile.PromptVersion != productContextPromptVersion {
			continue
		}
		var scope aiworkbench.WorkScope
		if json.Unmarshal(row.WorkScopeJSON, &scope) != nil || scope.TaskSelection == nil || !scope.TaskSelection.Valid() {
			return nil, aiworkbench.ErrUnavailable
		}
		result = append(result, scope.TaskSelection)
	}
	return result, nil
}
