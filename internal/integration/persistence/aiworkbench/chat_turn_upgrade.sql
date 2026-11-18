ALTER TABLE ai_workbench.commands ADD COLUMN turn_input jsonb CHECK (turn_input IS NULL OR (jsonb_typeof(turn_input)='object' AND turn_input->>'version'='1' AND octet_length(turn_input::text)<=98304)), ADD COLUMN tool_state text CHECK (tool_state IS NULL OR tool_state IN ('PENDING','BOUND','NOT_STARTED')), ADD COLUMN tool_key uuid;
ALTER TABLE ai_workbench.messages ADD COLUMN turn_key uuid;
ALTER TABLE ai_workbench.messages DROP CONSTRAINT messages_message_kind_check; ALTER TABLE ai_workbench.messages ADD CONSTRAINT messages_message_kind_check CHECK (message_kind IN ('TEXT','STORE_OVERVIEW_REF','AGENT_SKILL_REF'));
ALTER TABLE ai_workbench.messages DROP CONSTRAINT messages_check; ALTER TABLE ai_workbench.messages ADD CONSTRAINT ai_workbench_message_presentation CHECK (((message_kind = 'TEXT' AND store_query_status IS NULL AND turn_key IS NULL) OR
           (message_kind = 'STORE_OVERVIEW_REF' AND author_kind = 'ASSISTANT' AND content = '店铺概况' AND store_query_status IN ('ALL_NON_DELETED','ACTIVE') AND turn_key IS NULL) OR
           (message_kind = 'AGENT_SKILL_REF' AND author_kind = 'ASSISTANT' AND content = '技能草稿' AND store_query_status IS NULL AND turn_key IS NOT NULL)) IS TRUE);
ALTER TABLE ai_workbench.commands DROP CONSTRAINT commands_route_result_check; ALTER TABLE ai_workbench.commands ADD CONSTRAINT commands_route_result_check CHECK (route_result IS NULL OR route_result IN ('STORE_OVERVIEW','PRODUCT_TASK','CLARIFY','CREATE_TITLE_SKILL'));
ALTER TABLE ai_workbench.commands DROP CONSTRAINT commands_check; ALTER TABLE ai_workbench.commands ADD CONSTRAINT ai_workbench_command_shape CHECK ((operation = 'conversation_create' AND state = 'COMPLETE'
            AND user_message_id IS NULL AND planner_invocation_id IS NULL)
        OR (operation IN ('chat_message_plan','chat_intent_route') AND user_message_id IS NOT NULL
            AND source_sequence IS NOT NULL AND planner_invocation_id IS NOT NULL
            AND member_id IS NOT NULL
            AND work_scope IS NOT NULL
            AND (state = 'FAILED_BEFORE_DISPATCH' OR
                (planner_input_hash IS NOT NULL AND planner_model_profile IS NOT NULL
                 AND planner_started_at IS NOT NULL AND planner_deadline IS NOT NULL)))
        AND (operation <> 'chat_intent_route' OR route_commit_deadline IS NOT NULL OR (state='FAILED_BEFORE_DISPATCH' AND planner_model_profile IS NULL AND planner_input_hash IS NULL AND planner_started_at IS NULL AND planner_deadline IS NULL AND assistant_message_id IS NOT NULL))
        OR (operation = 'chat_store_overview' AND state = 'COMPLETE' AND member_id IS NOT NULL AND user_message_id IS NOT NULL AND source_sequence IS NOT NULL AND assistant_message_id IS NOT NULL AND planner_invocation_id IS NULL AND planner_input_hash IS NULL AND planner_model_profile IS NULL AND route_result = 'STORE_OVERVIEW')
        OR (operation='chat_intent_route' AND turn_input IS NOT NULL AND jsonb_typeof(turn_input->'input'->'skill')='object' AND state='COMPLETE' AND route_result='CREATE_TITLE_SKILL' AND tool_state IS NOT NULL AND member_id IS NOT NULL AND user_message_id IS NOT NULL AND source_sequence IS NOT NULL AND assistant_message_id IS NOT NULL AND planner_invocation_id IS NULL AND planner_input_hash IS NULL AND planner_model_profile IS NULL AND planner_started_at IS NULL AND planner_deadline IS NULL AND route_commit_deadline IS NULL));
ALTER TABLE ai_workbench.commands ADD CONSTRAINT ai_workbench_turn_tool CHECK (((tool_key IS NULL AND tool_state IS DISTINCT FROM 'BOUND') OR (tool_key IS NOT NULL AND tool_state='BOUND')) AND
        (tool_state IS NULL OR (turn_input IS NOT NULL AND state='COMPLETE' AND route_result='CREATE_TITLE_SKILL')) AND
        (route_result IS DISTINCT FROM 'CREATE_TITLE_SKILL' OR turn_input IS NOT NULL));
ALTER TABLE ai_workbench.commands ADD UNIQUE (organization_id,actor_id,conversation_id,idempotency_key), ADD UNIQUE (organization_id,actor_id,conversation_id,tool_key), ADD CONSTRAINT ai_workbench_turn_authoring FOREIGN KEY (organization_id,actor_id,conversation_id,tool_key) REFERENCES ai_workbench.skill_authoring_drafts (organization_id,actor_id,conversation_id,key);
ALTER TABLE ai_workbench.messages ADD CONSTRAINT ai_workbench_message_turn FOREIGN KEY (organization_id,owner_user_id,conversation_id,turn_key) REFERENCES ai_workbench.commands (organization_id,actor_id,conversation_id,idempotency_key) DEFERRABLE INITIALLY DEFERRED;
