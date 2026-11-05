package aiworkbenchpersistence

import (
	"context"
	"gorm.io/gorm"

	"task-processor/internal/agent"
	"task-processor/internal/aiworkbench"
)

func verifyBusinessScopeSchema(ctx context.Context, db *gorm.DB) error {
	for table, columns := range map[string]string{
		"conversations":       "business_scope",
		"commands":            "frozen_scope, scope_digest",
		"execution_proposals": "workbench_context_ref",
	} {
		if err := db.WithContext(ctx).Table("ai_workbench." + table).Select(columns).Limit(0).Find(&[]map[string]any{}).Error; err != nil {
			return aiworkbench.ErrUnavailable
		}
	}
	var found []struct{ Name, Definition string }
	const checks = "SELECT c.conname AS name,pg_get_constraintdef(c.oid) AS definition FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace JOIN pg_class r ON r.oid=c.conrelid WHERE n.nspname='ai_workbench' AND c.contype='c' AND c.convalidated AND ((r.relname='conversations' AND c.conname='ai_workbench_fixed_scope') OR (r.relname='commands' AND c.conname='ai_workbench_frozen_scope') OR (r.relname='execution_proposals' AND c.conname='ai_workbench_proposal_context'))"
	if err := db.WithContext(ctx).Raw(checks).Scan(&found).Error; err != nil || len(found) != 3 {
		return aiworkbench.ErrUnavailable
	}
	expected := map[string]string{
		"ai_workbench_fixed_scope":      `CHECK (((business_scope IS NULL) OR ((jsonb_typeof(business_scope) = 'object'::text) AND (octet_length((business_scope)::text) <= 1024) AND (NOT ((business_scope ->> 'version'::text) IS DISTINCT FROM 'workbench-scope-v1'::text)) AND (((((business_scope ->> 'kind'::text) = 'general'::text) AND (NOT (business_scope ?| ARRAY['projectId'::text, 'storeId'::text]))) OR (((business_scope ->> 'kind'::text) = 'project'::text) AND (COALESCE((business_scope ->> 'projectId'::text), ''::text) <> ''::text)) OR (((business_scope ->> 'kind'::text) = 'store'::text) AND (NOT (business_scope ? 'projectId'::text)) AND (COALESCE((business_scope ->> 'storeId'::text), ''::text) <> ''::text))) IS TRUE))))`,
		"ai_workbench_frozen_scope":     `CHECK ((((frozen_scope IS NULL) AND (scope_digest IS NULL)) OR ((operation = 'chat_message_plan'::text) AND (frozen_scope IS NOT NULL) AND (jsonb_typeof(frozen_scope) = 'object'::text) AND (octet_length((frozen_scope)::text) <= 16384) AND (scope_digest IS NOT NULL) AND (scope_digest ~ '^[a-f0-9]{64}$'::text))))`,
		"ai_workbench_proposal_context": `CHECK (((workbench_context_ref IS NULL) OR ((jsonb_typeof(workbench_context_ref) = 'object'::text) AND (octet_length((workbench_context_ref)::text) <= 1024))))`,
	}
	for _, check := range found {
		if expected[check.Name] != check.Definition {
			return aiworkbench.ErrUnavailable
		}
	}
	return nil
}

func (s *Store) ReplayCreate(ctx context.Context, owner aiworkbench.Scope, key string, input aiworkbench.CreateInput) (aiworkbench.Conversation, bool, error) {
	if !validScope(owner) || !validKey(key) {
		return aiworkbench.Conversation{}, false, aiworkbench.ErrInvalid
	}
	db := s.db.WithContext(ctx)
	row, found, err := s.lookupCommand(db, owner, key)
	if err != nil || !found {
		return aiworkbench.Conversation{}, found, unavailable(err)
	}
	c, err := s.createdReceipt(db, owner, row, createFingerprint(input))
	return c, true, err
}

func (s *Store) createdReceipt(db *gorm.DB, owner aiworkbench.Scope, row commandRow, fingerprint string) (aiworkbench.Conversation, error) {
	if row.Operation != createOperation || row.RequestFingerprint != fingerprint {
		return aiworkbench.Conversation{}, aiworkbench.ErrIdempotencyConflict
	}
	c, err := s.lookupConversation(db, owner, row.ConversationID, false)
	if err != nil {
		return aiworkbench.Conversation{}, err
	}
	return conversation(c), nil
}

func (s *Store) ConversationMember(ctx context.Context, owner aiworkbench.Scope, id, member string) error {
	if !validScope(owner) || !validKey(id) || !validName(member) {
		return aiworkbench.ErrInvalid
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&commandRow{}).
		Where("organization_id = ? AND actor_id = ? AND conversation_id = ? AND operation = ? AND member_id = ?", owner.OrganizationID, owner.ActorID, id, createOperation, member).
		Count(&count).Error; err != nil || count != 1 {
		return aiworkbench.ErrNotFound
	}
	return nil
}

func proposalValueRef(row proposalRow) agent.WorkbenchContextRef {
	if row.WorkbenchContextRef == nil {
		return agent.WorkbenchContextRef{}
	}
	return *row.WorkbenchContextRef
}

func (s *Store) TaskForExecution(ctx context.Context, owner aiworkbench.Scope, key string) (aiworkbench.BusinessTask, error) {
	if !validScope(owner) || !validKey(key) {
		return aiworkbench.BusinessTask{}, aiworkbench.ErrInvalid
	}
	var row taskRow
	if err := s.db.WithContext(ctx).Where("organization_id = ? AND owner_user_id = ? AND execution_request_key = ?", owner.OrganizationID, owner.ActorID, key).Take(&row).Error; err != nil {
		return aiworkbench.BusinessTask{}, missing(err)
	}
	return task(row)
}

func (s *Store) ReadFrozenScope(ctx context.Context, owner aiworkbench.Scope, ref agent.WorkbenchContextRef) (aiworkbench.FrozenScope, error) {
	if !ref.Valid() {
		return aiworkbench.FrozenScope{}, aiworkbench.ErrUnavailable
	}
	cmd, err := s.GetCommand(ctx, owner, ref.CommandKey)
	if err != nil {
		return aiworkbench.FrozenScope{}, err
	}
	if cmd.ScopeRef() != ref || cmd.FrozenScope == nil {
		return aiworkbench.FrozenScope{}, aiworkbench.ErrUnavailable
	}
	conversation, err := s.Get(ctx, owner, ref.ConversationID)
	if err != nil || conversation.BusinessScope == nil || *conversation.BusinessScope != cmd.FrozenScope.Scope {
		return aiworkbench.FrozenScope{}, aiworkbench.ErrUnavailable
	}
	return *cmd.FrozenScope, nil
}
