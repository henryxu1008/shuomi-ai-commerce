package aiworkbenchpersistence

import (
	"context"
	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
)

// StoreDependencies ties every consumed assistant turn to its original command.
// It must not infer dependencies from the currently selected store.
func (s *Store) StoreDependencies(ctx context.Context, scope aiworkbench.Scope, conversation string, through uint64) ([]agent.WorkbenchContextRef, error) {
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
	return planningStoreDependencies(messages, rows)
}

func planningStoreDependencies(messages []messageRow, rows []commandRow) ([]agent.WorkbenchContextRef, error) {
	byID := map[string]commandRow{}
	for _, row := range rows {
		if row.AssistantMessageID == nil || *row.AssistantMessageID == "" {
			return nil, aiworkbench.ErrUnavailable
		}
		id := *row.AssistantMessageID
		if _, ok := byID[id]; ok {
			return nil, aiworkbench.ErrUnavailable
		}
		byID[id] = row
	}
	var refs []agent.WorkbenchContextRef
	for _, message := range messages {
		row, ok := byID[message.ID]
		if !ok {
			return nil, aiworkbench.ErrUnavailable
		}
		if row.Operation == routeOperation || row.Operation == explicitOverviewOperation {
			continue
		}
		if row.Operation != messageOperation {
			return nil, aiworkbench.ErrUnavailable
		}
		if len(row.FrozenScopeJSON) == 0 {
			continue
		}
		cmd := command(row)
		if cmd.FrozenScope == nil || cmd.FrozenScope.Validate() != nil {
			return nil, aiworkbench.ErrUnavailable
		}
		if cmd.FrozenScope.StoreChoice == nil || cmd.FrozenScope.StoreChoice.Kind != "store" {
			continue
		}
		ref := cmd.ScopeRef()
		if !ref.Valid() {
			return nil, aiworkbench.ErrUnavailable
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
