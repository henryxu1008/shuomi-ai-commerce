package aiworkbenchpersistence

import (
	"context"
	_ "embed"
	"fmt"
	"gorm.io/gorm"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/pkg/pgschema"
)

//go:embed chat_turn_upgrade.sql
var chatTurnUpgradeSQL string

// Original current checks are retained here solely to verify the bounded
// installation target; this is not a legacy reader or data migration.
const chatTurnOriginalChecks = `ALTER TABLE ai_workbench.messages DROP CONSTRAINT IF EXISTS messages_message_kind_check; ALTER TABLE ai_workbench.messages ADD CONSTRAINT messages_message_kind_check CHECK (message_kind IN ('TEXT','STORE_OVERVIEW_REF'));
ALTER TABLE ai_workbench.messages DROP CONSTRAINT IF EXISTS messages_check; ALTER TABLE ai_workbench.messages ADD CONSTRAINT messages_check CHECK (((message_kind = 'TEXT' AND store_query_status IS NULL) OR
           (message_kind = 'STORE_OVERVIEW_REF' AND author_kind = 'ASSISTANT' AND content = '店铺概况' AND store_query_status IN ('ALL_NON_DELETED','ACTIVE'))) IS TRUE);
ALTER TABLE ai_workbench.commands DROP CONSTRAINT IF EXISTS commands_route_result_check; ALTER TABLE ai_workbench.commands ADD CONSTRAINT commands_route_result_check CHECK (route_result IS NULL OR route_result IN ('STORE_OVERVIEW','PRODUCT_TASK','CLARIFY'));
ALTER TABLE ai_workbench.commands DROP CONSTRAINT IF EXISTS commands_check; ALTER TABLE ai_workbench.commands ADD CONSTRAINT commands_check CHECK ((operation = 'conversation_create' AND state = 'COMPLETE'
            AND user_message_id IS NULL AND planner_invocation_id IS NULL)
        OR (operation IN ('chat_message_plan','chat_intent_route') AND user_message_id IS NOT NULL
            AND source_sequence IS NOT NULL AND planner_invocation_id IS NOT NULL
            AND member_id IS NOT NULL
            AND work_scope IS NOT NULL
            AND (state = 'FAILED_BEFORE_DISPATCH' OR
                (planner_input_hash IS NOT NULL AND planner_model_profile IS NOT NULL
                 AND planner_started_at IS NOT NULL AND planner_deadline IS NOT NULL)))
        AND (operation <> 'chat_intent_route' OR route_commit_deadline IS NOT NULL OR (state='FAILED_BEFORE_DISPATCH' AND planner_model_profile IS NULL AND planner_input_hash IS NULL AND planner_started_at IS NULL AND planner_deadline IS NULL AND assistant_message_id IS NOT NULL))
        OR (operation = 'chat_store_overview' AND state = 'COMPLETE' AND member_id IS NOT NULL AND user_message_id IS NOT NULL AND source_sequence IS NOT NULL AND assistant_message_id IS NOT NULL AND planner_invocation_id IS NULL AND planner_input_hash IS NULL AND planner_model_profile IS NULL AND route_result = 'STORE_OVERVIEW'));`

func InstallChatTurnSchema(ctx context.Context, db *gorm.DB, role string) error {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" || role != "ai_workbench_runtime" {
		return aiworkbench.ErrInvalid
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL lock_timeout='5s'").Error; err != nil {
			return err
		}
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(current_database()||':workbench-chat-turn-schema',0))").Error; err != nil {
			return err
		}
		if err := validateBoundedWorkbenchPrivileges(tx, role); err != nil {
			return err
		}
		layout, err := chatTurnLayout(tx)
		if err != nil {
			return err
		}
		if layout == 1 {
			if !(&Store{db: tx}).CanChatTurns(ctx) {
				return aiworkbench.ErrUnavailable
			}
			return nil
		}
		for _, check := range []struct{ Table, Name, Definition string }{
			{"messages", "messages_message_kind_check", `CHECK ((message_kind = ANY (ARRAY['TEXT'::text, 'STORE_OVERVIEW_REF'::text])))`},
			{"messages", "messages_check", `CHECK (((((message_kind = 'TEXT'::text) AND (store_query_status IS NULL)) OR ((message_kind = 'STORE_OVERVIEW_REF'::text) AND (author_kind = 'ASSISTANT'::text) AND (content = '店铺概况'::text) AND (store_query_status = ANY (ARRAY['ALL_NON_DELETED'::text, 'ACTIVE'::text])))) IS TRUE))`},
			{"commands", "commands_route_result_check", `CHECK (((route_result IS NULL) OR (route_result = ANY (ARRAY['STORE_OVERVIEW'::text, 'PRODUCT_TASK'::text, 'CLARIFY'::text]))))`},
			{"commands", "commands_check", `CHECK ((((operation = 'conversation_create'::text) AND (state = 'COMPLETE'::text) AND (user_message_id IS NULL) AND (planner_invocation_id IS NULL)) OR ((operation = ANY (ARRAY['chat_message_plan'::text, 'chat_intent_route'::text])) AND (user_message_id IS NOT NULL) AND (source_sequence IS NOT NULL) AND (planner_invocation_id IS NOT NULL) AND (member_id IS NOT NULL) AND (work_scope IS NOT NULL) AND ((state = 'FAILED_BEFORE_DISPATCH'::text) OR ((planner_input_hash IS NOT NULL) AND (planner_model_profile IS NOT NULL) AND (planner_started_at IS NOT NULL) AND (planner_deadline IS NOT NULL))) AND ((operation <> 'chat_intent_route'::text) OR (route_commit_deadline IS NOT NULL) OR ((state = 'FAILED_BEFORE_DISPATCH'::text) AND (planner_model_profile IS NULL) AND (planner_input_hash IS NULL) AND (planner_started_at IS NULL) AND (planner_deadline IS NULL) AND (assistant_message_id IS NOT NULL)))) OR ((operation = 'chat_store_overview'::text) AND (state = 'COMPLETE'::text) AND (member_id IS NOT NULL) AND (user_message_id IS NOT NULL) AND (source_sequence IS NOT NULL) AND (assistant_message_id IS NOT NULL) AND (planner_invocation_id IS NULL) AND (planner_input_hash IS NULL) AND (planner_model_profile IS NULL) AND (route_result = 'STORE_OVERVIEW'::text))))`},
		} {
			var actual struct {
				Definition string
				Validated  bool
			}
			if tx.Raw("SELECT pg_get_constraintdef(oid) AS definition,convalidated AS validated FROM pg_constraint WHERE conrelid=to_regclass(?) AND conname=? AND contype='c'", "ai_workbench."+check.Table, check.Name).Scan(&actual).Error != nil || !actual.Validated || !pgschema.SchemaConstraintEqual(actual.Definition, check.Definition) {
				return fmt.Errorf("chat turn target check %s.%s: %w", check.Table, check.Name, aiworkbench.ErrUnavailable)
			}
		}
		if err := tx.Exec(chatTurnUpgradeSQL).Error; err != nil {
			return err
		}
		if err := tx.Exec("GRANT UPDATE(tool_state,tool_key) ON ai_workbench.commands TO ai_workbench_runtime").Error; err != nil {
			return err
		}
		if err := validateBoundedWorkbenchPrivileges(tx, role); err != nil {
			return err
		}
		if !(&Store{db: tx}).CanChatTurns(ctx) {
			return aiworkbench.ErrUnavailable
		}
		return nil
	})
}
