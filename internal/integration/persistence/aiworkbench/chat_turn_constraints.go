package aiworkbenchpersistence

import (
	"gorm.io/gorm"
	"task-processor/internal/pkg/pgschema"
)

// These are the installed PostgreSQL definitions of the current bounded Chat
// extension, checked read-only before new-format traffic is admitted.
func verifyChatTurnConstraints(db *gorm.DB) bool {
	for _, check := range []struct{ Table, Name, Definition string }{
		{"ai_workbench.commands", "ai_workbench_command_shape", `CHECK ((((operation = 'conversation_create'::text) AND (state = 'COMPLETE'::text) AND (user_message_id IS NULL) AND (planner_invocation_id IS NULL)) OR ((operation = ANY (ARRAY['chat_message_plan'::text, 'chat_intent_route'::text])) AND (user_message_id IS NOT NULL) AND (source_sequence IS NOT NULL) AND (planner_invocation_id IS NOT NULL) AND (member_id IS NOT NULL) AND (work_scope IS NOT NULL) AND ((state = 'FAILED_BEFORE_DISPATCH'::text) OR ((planner_input_hash IS NOT NULL) AND (planner_model_profile IS NOT NULL) AND (planner_started_at IS NOT NULL) AND (planner_deadline IS NOT NULL))) AND ((operation <> 'chat_intent_route'::text) OR (route_commit_deadline IS NOT NULL) OR ((state = 'FAILED_BEFORE_DISPATCH'::text) AND (planner_model_profile IS NULL) AND (planner_input_hash IS NULL) AND (planner_started_at IS NULL) AND (planner_deadline IS NULL) AND (assistant_message_id IS NOT NULL)))) OR ((operation = 'chat_store_overview'::text) AND (state = 'COMPLETE'::text) AND (member_id IS NOT NULL) AND (user_message_id IS NOT NULL) AND (source_sequence IS NOT NULL) AND (assistant_message_id IS NOT NULL) AND (planner_invocation_id IS NULL) AND (planner_input_hash IS NULL) AND (planner_model_profile IS NULL) AND (route_result = 'STORE_OVERVIEW'::text)) OR ((operation = 'chat_intent_route'::text) AND (turn_input IS NOT NULL) AND (jsonb_typeof(((turn_input -> 'input'::text) -> 'skill'::text)) = 'object'::text) AND (state = 'COMPLETE'::text) AND (route_result = 'CREATE_TITLE_SKILL'::text) AND (tool_state IS NOT NULL) AND (member_id IS NOT NULL) AND (user_message_id IS NOT NULL) AND (source_sequence IS NOT NULL) AND (assistant_message_id IS NOT NULL) AND (planner_invocation_id IS NULL) AND (planner_input_hash IS NULL) AND (planner_model_profile IS NULL) AND (planner_started_at IS NULL) AND (planner_deadline IS NULL) AND (route_commit_deadline IS NULL))))`},
		{"ai_workbench.messages", "ai_workbench_message_presentation", `CHECK (((((message_kind = 'TEXT'::text) AND (store_query_status IS NULL) AND (turn_key IS NULL)) OR ((message_kind = 'STORE_OVERVIEW_REF'::text) AND (author_kind = 'ASSISTANT'::text) AND (content = '店铺概况'::text) AND (store_query_status = ANY (ARRAY['ALL_NON_DELETED'::text, 'ACTIVE'::text])) AND (turn_key IS NULL)) OR ((message_kind = 'AGENT_SKILL_REF'::text) AND (author_kind = 'ASSISTANT'::text) AND (content = '技能草稿'::text) AND (store_query_status IS NULL) AND (turn_key IS NOT NULL))) IS TRUE))`},
		{"ai_workbench.messages", "ai_workbench_message_turn", `FOREIGN KEY (organization_id, owner_user_id, conversation_id, turn_key) REFERENCES ai_workbench.commands(organization_id, actor_id, conversation_id, idempotency_key) DEFERRABLE INITIALLY DEFERRED`},
		{"ai_workbench.commands", "ai_workbench_turn_authoring", `FOREIGN KEY (organization_id, actor_id, conversation_id, tool_key) REFERENCES ai_workbench.skill_authoring_drafts(organization_id, actor_id, conversation_id, key)`},
		{"ai_workbench.commands", "ai_workbench_turn_tool", `CHECK (((((tool_key IS NULL) AND (tool_state IS DISTINCT FROM 'BOUND'::text)) OR ((tool_key IS NOT NULL) AND (tool_state = 'BOUND'::text))) AND ((tool_state IS NULL) OR ((turn_input IS NOT NULL) AND (state = 'COMPLETE'::text) AND (route_result = 'CREATE_TITLE_SKILL'::text))) AND ((route_result IS DISTINCT FROM 'CREATE_TITLE_SKILL'::text) OR (turn_input IS NOT NULL))))`},
		{"ai_workbench.commands", "commands_route_result_check", `CHECK (((route_result IS NULL) OR (route_result = ANY (ARRAY['STORE_OVERVIEW'::text, 'PRODUCT_TASK'::text, 'CLARIFY'::text, 'CREATE_TITLE_SKILL'::text]))))`},
		{"ai_workbench.commands", "commands_tool_state_check", `CHECK (((tool_state IS NULL) OR (tool_state = ANY (ARRAY['PENDING'::text, 'BOUND'::text, 'NOT_STARTED'::text]))))`},
		{"ai_workbench.commands", "commands_turn_input_check", `CHECK (((turn_input IS NULL) OR ((jsonb_typeof(turn_input) = 'object'::text) AND ((turn_input ->> 'version'::text) = '1'::text) AND (octet_length((turn_input)::text) <= 98304))))`},
		{"ai_workbench.messages", "messages_message_kind_check", `CHECK ((message_kind = ANY (ARRAY['TEXT'::text, 'STORE_OVERVIEW_REF'::text, 'AGENT_SKILL_REF'::text])))`},
	} {
		var actual struct {
			Definition string
			Validated  bool
			Triggers   bool
		}
		if db.Raw(`SELECT pg_get_constraintdef(c.oid) AS definition,c.convalidated AS validated,NOT EXISTS(SELECT 1 FROM pg_trigger t WHERE t.tgconstraint=c.oid AND t.tgenabled<>'O') AS triggers FROM pg_constraint c WHERE c.conrelid=to_regclass(?) AND c.conname=?`, check.Table, check.Name).Scan(&actual).Error != nil || !actual.Validated || !actual.Triggers || !pgschema.SchemaConstraintEqual(actual.Definition, check.Definition) {
			return false
		}
	}
	return true
}
