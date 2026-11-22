package aiworkbenchpersistence

import (
	"context"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
)

// ChatImageCommand resolves an attachment against the exact source message and
// original member. The API separately checks the live conversation authority.
func (s *Store) ChatImageCommand(ctx context.Context, scope aiworkbench.Scope, conversation, message, member string) (aiworkbench.PlanningCommand, error) {
	if s == nil || s.db == nil || ctx == nil || !validScope(scope) || !validKey(conversation) || !validKey(message) || !validName(member) {
		return aiworkbench.PlanningCommand{}, aiworkbench.ErrInvalid
	}
	var row commandRow
	if err := s.db.WithContext(ctx).Where("organization_id=? AND actor_id=? AND conversation_id=? AND member_id=? AND user_message_id=? AND operation=? AND jsonb_typeof(work_scope->'chat_images')='array'",
		scope.OrganizationID, scope.ActorID, conversation, member, message, messageOperation).Take(&row).Error; err != nil {
		return aiworkbench.PlanningCommand{}, missing(err)
	}
	result := command(row)
	if !aiworkbench.ValidChatImageSelection(result.WorkScope.ChatImages) || !result.ScopeRef().Valid() {
		return aiworkbench.PlanningCommand{}, aiworkbench.ErrUnavailable
	}
	return result, nil
}
