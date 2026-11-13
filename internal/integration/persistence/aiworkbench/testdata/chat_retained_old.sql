CREATE SCHEMA IF NOT EXISTS ai_workbench;

CREATE TABLE IF NOT EXISTS ai_workbench.conversations (
    id uuid PRIMARY KEY,
    organization_id text NOT NULL CHECK (octet_length(organization_id) BETWEEN 1 AND 128),
    owner_user_id text NOT NULL CHECK (octet_length(owner_user_id) BETWEEN 1 AND 128),
    business_scope jsonb CONSTRAINT ai_workbench_fixed_scope CHECK (business_scope IS NULL OR (jsonb_typeof(business_scope) = 'object' AND octet_length(business_scope::text) <= 1024 AND business_scope->>'version' IS NOT DISTINCT FROM 'workbench-scope-v1' AND ((business_scope->>'kind' = 'general' AND NOT (business_scope ?| array['projectId','storeId'])) OR (business_scope->>'kind' = 'project' AND coalesce(business_scope->>'projectId','') <> '') OR (business_scope->>'kind' = 'store' AND NOT (business_scope ? 'projectId') AND coalesce(business_scope->>'storeId','') <> '')) IS TRUE)),
    title text NOT NULL DEFAULT '' CHECK (octet_length(title) <= 256),
    favorite boolean NOT NULL DEFAULT false,
    lifecycle text NOT NULL CHECK (lifecycle IN ('ACTIVE', 'ARCHIVED')),
    metadata_revision bigint NOT NULL CHECK (metadata_revision > 0),
    next_sequence bigint NOT NULL CHECK (next_sequence > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (id, organization_id, owner_user_id)
);
CREATE INDEX IF NOT EXISTS ai_workbench_conversations_owner_recent
    ON ai_workbench.conversations
    (organization_id, owner_user_id, lifecycle, updated_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS ai_workbench_conversations_owner_saved
    ON ai_workbench.conversations
    (organization_id, owner_user_id, lifecycle, favorite, updated_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ai_workbench.metadata_audit (
    conversation_id uuid NOT NULL,
    organization_id text NOT NULL,
    actor_id text NOT NULL,
    metadata_revision bigint NOT NULL CHECK (metadata_revision > 1),
    change_mask smallint NOT NULL CHECK (change_mask BETWEEN 1 AND 7),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (conversation_id, metadata_revision),
    FOREIGN KEY (conversation_id, organization_id, actor_id)
        REFERENCES ai_workbench.conversations (id, organization_id, owner_user_id)
);

CREATE TABLE IF NOT EXISTS ai_workbench.messages (
    id uuid PRIMARY KEY,
    organization_id text NOT NULL,
    owner_user_id text NOT NULL,
    conversation_id uuid NOT NULL,
    sequence bigint NOT NULL CHECK (sequence > 0),
    author_kind text NOT NULL CHECK (author_kind IN ('USER', 'ASSISTANT')),
    content text NOT NULL CHECK (octet_length(content) BETWEEN 1 AND 16384),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, owner_user_id, conversation_id, sequence),
    UNIQUE (organization_id, owner_user_id, conversation_id, id),
    FOREIGN KEY (conversation_id, organization_id, owner_user_id)
        REFERENCES ai_workbench.conversations (id, organization_id, owner_user_id)
);
CREATE INDEX IF NOT EXISTS ai_workbench_messages_latest_user
    ON ai_workbench.messages
    (organization_id, owner_user_id, conversation_id, author_kind, sequence DESC);

CREATE TABLE IF NOT EXISTS ai_workbench.commands (
    organization_id text NOT NULL,
    actor_id text NOT NULL,
    idempotency_key uuid NOT NULL,
    operation text NOT NULL CHECK (operation IN ('conversation_create', 'chat_message_plan')),
    request_fingerprint char(64) NOT NULL,
    conversation_id uuid NOT NULL,
    state text NOT NULL CHECK (state IN ('READY_TO_DISPATCH', 'COMPLETE', 'FAILED_BEFORE_DISPATCH', 'PLANNER_INVALID_OUTPUT', 'PLANNER_UNKNOWN')),
    user_message_id uuid,
    source_sequence bigint,
    planner_invocation_id char(64),
    member_id text CHECK (member_id IS NULL OR octet_length(member_id) BETWEEN 1 AND 128),
    planner_input_hash char(64),
    work_scope jsonb CHECK (work_scope IS NULL OR octet_length(work_scope::text) <= 1024),
    planner_model_profile jsonb,
    frozen_scope jsonb,
    scope_digest char(64),
    CONSTRAINT ai_workbench_frozen_scope CHECK ((frozen_scope IS NULL AND scope_digest IS NULL) OR (operation = 'chat_message_plan' AND frozen_scope IS NOT NULL AND jsonb_typeof(frozen_scope) = 'object' AND octet_length(frozen_scope::text) <= 16384 AND scope_digest IS NOT NULL AND scope_digest ~ '^[a-f0-9]{64}$')),
    planner_started_at timestamptz,
    planner_deadline timestamptz,
    assistant_message_id uuid,
    proposal_id uuid,
    terminal_digest char(64),
    mode text CHECK (mode IS NULL OR mode IN ('CLARIFY', 'READY')),
    created_at timestamptz NOT NULL,
    committed_at timestamptz,
    PRIMARY KEY (organization_id, actor_id, idempotency_key),
    UNIQUE (planner_invocation_id),
    FOREIGN KEY (conversation_id, organization_id, actor_id)
        REFERENCES ai_workbench.conversations (id, organization_id, owner_user_id),
    FOREIGN KEY (organization_id, actor_id, conversation_id, user_message_id)
        REFERENCES ai_workbench.messages (organization_id, owner_user_id, conversation_id, id),
    FOREIGN KEY (organization_id, actor_id, conversation_id, assistant_message_id)
        REFERENCES ai_workbench.messages (organization_id, owner_user_id, conversation_id, id),
    CHECK ((operation = 'conversation_create' AND state = 'COMPLETE'
            AND user_message_id IS NULL AND planner_invocation_id IS NULL)
        OR (operation = 'chat_message_plan' AND user_message_id IS NOT NULL
            AND source_sequence IS NOT NULL AND planner_invocation_id IS NOT NULL
            AND member_id IS NOT NULL
            AND work_scope IS NOT NULL
            AND (state = 'FAILED_BEFORE_DISPATCH' OR
                (planner_input_hash IS NOT NULL AND planner_model_profile IS NOT NULL
                 AND planner_started_at IS NOT NULL AND planner_deadline IS NOT NULL))))
);

CREATE TABLE IF NOT EXISTS ai_workbench.execution_proposals (
    id uuid PRIMARY KEY,
    workbench_context_ref jsonb CONSTRAINT ai_workbench_proposal_context CHECK (workbench_context_ref IS NULL OR (jsonb_typeof(workbench_context_ref) = 'object' AND octet_length(workbench_context_ref::text) <= 1024)),
    digest char(64) NOT NULL,
    organization_id text NOT NULL,
    owner_user_id text NOT NULL,
    conversation_id uuid NOT NULL,
    source_user_message_id uuid NOT NULL,
    assistant_message_id uuid NOT NULL,
    source_sequence bigint NOT NULL CHECK (source_sequence > 0),
    kind text NOT NULL,
    goal_summary text NOT NULL CHECK (octet_length(goal_summary) BETWEEN 1 AND 512),
    operation_id text NOT NULL,
    product_key text NOT NULL,
    catalog_version text NOT NULL,
    publication_id text NOT NULL,
    target_platform text NOT NULL,
    agent_id text NOT NULL,
    agent_version text NOT NULL,
    observed_agent_revision text NOT NULL,
    observed_mode_revision text NOT NULL CHECK(observed_mode_revision ~ '^(0|[1-9][0-9]{0,18})$' AND observed_mode_revision::numeric <= 9223372036854775807),
    observed_activation_epoch text NOT NULL,
    template_id text,
    template_revision text,
    knowledge_base_id text,
    knowledge_revision_set_digest text,
    context_kind text,
    content_selection jsonb CHECK (octet_length(content_selection::text) <= 1024),
    execution_limits jsonb CHECK (octet_length(execution_limits::text) <= 512),
    CONSTRAINT ai_workbench_proposal_capability CHECK (
      (kind = 'product.title.optimize' AND target_platform IN ('shein','temu','amazon') AND context_kind IS NULL AND content_selection IS NULL AND execution_limits IS NULL) OR
      (kind = 'product.content.localize' AND target_platform = 'product' AND context_kind IS NOT NULL AND context_kind IN ('own_product','acquisition') AND content_selection IS NOT NULL AND jsonb_typeof(content_selection) = 'object' AND content_selection <> '{}'::jsonb AND execution_limits IS NOT NULL AND jsonb_typeof(execution_limits) = 'object' AND execution_limits <> '{}'::jsonb)),
    execution_model_profile jsonb NOT NULL CHECK (octet_length(execution_model_profile::text) <= 8192),
    created_at timestamptz NOT NULL,
    UNIQUE (id, organization_id, owner_user_id, conversation_id),
    FOREIGN KEY (conversation_id, organization_id, owner_user_id)
        REFERENCES ai_workbench.conversations (id, organization_id, owner_user_id),
    FOREIGN KEY (organization_id, owner_user_id, conversation_id, source_user_message_id)
        REFERENCES ai_workbench.messages (organization_id, owner_user_id, conversation_id, id),
    FOREIGN KEY (organization_id, owner_user_id, conversation_id, assistant_message_id)
        REFERENCES ai_workbench.messages (organization_id, owner_user_id, conversation_id, id)
);
CREATE INDEX IF NOT EXISTS ai_workbench_proposals_conversation
    ON ai_workbench.execution_proposals (organization_id, owner_user_id, conversation_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ai_workbench.business_tasks (
    id uuid PRIMARY KEY,
    organization_id text NOT NULL,
    owner_user_id text NOT NULL,
    source_kind text NOT NULL DEFAULT 'CHAT_PROPOSAL' CHECK (source_kind IN ('CHAT_PROPOSAL','SCHEDULE_OCCURRENCE')),
    scheduled_occurrence_id uuid,
    conversation_id uuid,
    source_message_id uuid,
    proposal_id uuid,
    CHECK ((source_kind = 'CHAT_PROPOSAL' AND conversation_id IS NOT NULL AND source_message_id IS NOT NULL AND proposal_id IS NOT NULL AND scheduled_occurrence_id IS NULL)
      OR (source_kind = 'SCHEDULE_OCCURRENCE' AND conversation_id IS NULL AND source_message_id IS NULL AND proposal_id IS NULL AND scheduled_occurrence_id IS NOT NULL)),
    proposal_digest char(64) NOT NULL,
    confirmation_fingerprint char(64) NOT NULL,
    kind text NOT NULL,
    title text NOT NULL CHECK (octet_length(title) BETWEEN 1 AND 256),
    goal_summary text NOT NULL CHECK (octet_length(goal_summary) BETWEEN 1 AND 512),
    operation_id text NOT NULL,
    product_key text NOT NULL,
    target_platform text NOT NULL,
    agent_id text NOT NULL,
    agent_version text NOT NULL,
    CONSTRAINT ai_workbench_task_capability CHECK (
      (kind = 'product.title.optimize' AND target_platform IN ('shein','temu','amazon')) OR
      (kind = 'product.content.localize' AND target_platform = 'product')),
    execution_request_key uuid NOT NULL,
    configuration_snapshot_ref jsonb NOT NULL CHECK (octet_length(configuration_snapshot_ref::text) <= 512),
    context_snapshot_ref jsonb NOT NULL CHECK (octet_length(context_snapshot_ref::text) <= 512),
    execution_request_digest char(64) NOT NULL,
    execution_request jsonb NOT NULL CHECK (octet_length(execution_request::text) <= 8192),
    created_at timestamptz NOT NULL,
    UNIQUE (organization_id, owner_user_id, execution_request_key),
    UNIQUE (id, organization_id, owner_user_id),
    UNIQUE (scheduled_occurrence_id, organization_id, owner_user_id),
    UNIQUE (id, organization_id, owner_user_id, scheduled_occurrence_id),
    FOREIGN KEY (conversation_id, organization_id, owner_user_id)
        REFERENCES ai_workbench.conversations (id, organization_id, owner_user_id),
    FOREIGN KEY (proposal_id, organization_id, owner_user_id, conversation_id)
        REFERENCES ai_workbench.execution_proposals (id, organization_id, owner_user_id, conversation_id),
    FOREIGN KEY (organization_id, owner_user_id, conversation_id, source_message_id)
        REFERENCES ai_workbench.messages (organization_id, owner_user_id, conversation_id, id)
);
CREATE INDEX IF NOT EXISTS ai_workbench_tasks_owner_recent
    ON ai_workbench.business_tasks (organization_id, owner_user_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ai_workbench.task_action_receipts (
    organization_id text NOT NULL,
    actor_id text NOT NULL,
    idempotency_key uuid NOT NULL,
    task_id uuid NOT NULL,
    action text NOT NULL CHECK (action IN ('start', 'resume', 'review')),
    expected_revision bigint NOT NULL CHECK (expected_revision >= 0),
    request_fingerprint char(64) NOT NULL,
    state text NOT NULL CHECK (state IN ('CLAIMED', 'COMPLETE', 'FAILED', 'UNKNOWN')),
    error_code text CHECK (error_code IS NULL OR error_code IN ('REVISION_MISMATCH', 'DEPENDENCY_UNAVAILABLE')),
    created_at timestamptz NOT NULL,
    finished_at timestamptz,
    PRIMARY KEY (organization_id, actor_id, idempotency_key),
    FOREIGN KEY (task_id, organization_id, actor_id)
        REFERENCES ai_workbench.business_tasks (id, organization_id, owner_user_id),
    CHECK ((state = 'CLAIMED' AND finished_at IS NULL AND error_code IS NULL) OR
           (state = 'COMPLETE' AND finished_at IS NOT NULL AND error_code IS NULL) OR
           (state = 'FAILED' AND finished_at IS NOT NULL AND error_code IS NOT NULL) OR
           (state = 'UNKNOWN' AND finished_at IS NOT NULL AND error_code IS NULL))
);


-- A scheduled plan owns only user-approved execution intent; business execution,
-- review decisions and commercial usage remain with their existing owners.
-- Installation is explicit and is never attempted by the serving constructor.
CREATE TABLE IF NOT EXISTS ai_workbench.scheduled_plans (
    id uuid PRIMARY KEY,
    organization_id text NOT NULL CHECK (octet_length(organization_id) BETWEEN 1 AND 128),
    owner_user_id text NOT NULL CHECK (octet_length(owner_user_id) BETWEEN 1 AND 128),
    member_id text NOT NULL CHECK (octet_length(member_id) BETWEEN 1 AND 128),
    kind text NOT NULL CHECK (kind = 'product.content.localize'),
    task_selection jsonb NOT NULL CHECK (jsonb_typeof(task_selection) = 'object' AND octet_length(task_selection::text) BETWEEN 2 AND 2048),
    execution_spec jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(execution_spec) = 'object' AND octet_length(execution_spec::text) <= 16384),
    rule_start_at timestamptz,
    registration_state text NOT NULL DEFAULT 'PENDING' CHECK (registration_state IN ('PENDING','MATCHED','UNKNOWN')),
    creation_fingerprint char(64) NOT NULL DEFAULT repeat('0',64),
    goal_summary text NOT NULL CHECK (octet_length(goal_summary) BETWEEN 1 AND 512),
    frequency text NOT NULL CHECK (frequency IN ('once', 'daily', 'weekly')),
    time_zone text NOT NULL CHECK (octet_length(time_zone) BETWEEN 1 AND 128),
    once_at_utc timestamptz,
    local_hour smallint,
    local_minute smallint,
    local_weekday smallint,
    until_utc timestamptz,
    max_occurrences bigint NOT NULL CHECK (max_occurrences BETWEEN 1 AND 9223372036854775807),
    budget_unit text NOT NULL DEFAULT 'ai-micropoint-v1' CHECK (budget_unit = 'ai-micropoint-v1'),
    per_run_upper_bound_micros bigint NOT NULL CHECK (per_run_upper_bound_micros > 0),
    total_upper_bound_micros bigint NOT NULL CHECK (total_upper_bound_micros > 0),
    consumed_occurrences bigint NOT NULL DEFAULT 0 CHECK (consumed_occurrences >= 0 AND consumed_occurrences <= max_occurrences),
    active_occurrence_id uuid,
    state text NOT NULL CHECK (state IN ('PENDING_REGISTRATION','ACTIVE','PAUSED','CLOSED','REGISTRATION_UNKNOWN')),
    revision bigint NOT NULL CHECK (revision > 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (id, organization_id, owner_user_id),
    CHECK (
      (frequency = 'once' AND once_at_utc IS NOT NULL AND local_hour IS NULL AND local_minute IS NULL AND local_weekday IS NULL) OR
      (frequency = 'daily' AND once_at_utc IS NULL AND local_hour IS NOT NULL AND local_minute IS NOT NULL AND local_hour BETWEEN 0 AND 23 AND local_minute BETWEEN 0 AND 59 AND local_weekday IS NULL) OR
      (frequency = 'weekly' AND once_at_utc IS NULL AND local_hour IS NOT NULL AND local_minute IS NOT NULL AND local_weekday IS NOT NULL AND local_hour BETWEEN 0 AND 23 AND local_minute BETWEEN 0 AND 59 AND local_weekday BETWEEN 0 AND 6)
    ),
    CHECK (until_utc IS NULL OR once_at_utc IS NULL OR once_at_utc <= until_utc),
    CHECK (max_occurrences::numeric * per_run_upper_bound_micros::numeric = total_upper_bound_micros::numeric)
);

CREATE INDEX IF NOT EXISTS ai_workbench_scheduled_plans_owner
    ON ai_workbench.scheduled_plans (organization_id, owner_user_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS ai_workbench_scheduled_plans_status
    ON ai_workbench.scheduled_plans (state, updated_at, id);

-- An occurrence is unique per actual UTC slot, independent of retries and
-- plan revisions. No occurrence result is ever an alternative AgentRun fact.
CREATE TABLE IF NOT EXISTS ai_workbench.scheduled_occurrences (
    id uuid PRIMARY KEY,
    organization_id text NOT NULL,
    owner_user_id text NOT NULL,
    plan_id uuid NOT NULL,
    scheduled_at_utc timestamptz NOT NULL,
    plan_revision bigint NOT NULL CHECK (plan_revision > 0),
    payload_digest char(64) NOT NULL CHECK (payload_digest ~ '^[a-f0-9]{64}$'),
    state text NOT NULL CHECK (state IN ('CLAIMED','DISPATCHED','SKIPPED','BLOCKED','UNKNOWN','RECONCILED')),
    business_task_id uuid,
    reconciliation_digest char(64),
    reason text NOT NULL DEFAULT '' CHECK (octet_length(reason) <= 128),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (organization_id, plan_id, scheduled_at_utc),
    UNIQUE (id, organization_id, owner_user_id, plan_id),
    UNIQUE (id, organization_id, owner_user_id),
    FOREIGN KEY (plan_id, organization_id, owner_user_id)
        REFERENCES ai_workbench.scheduled_plans (id, organization_id, owner_user_id),
    FOREIGN KEY (business_task_id, organization_id, owner_user_id, id)
        REFERENCES ai_workbench.business_tasks (id, organization_id, owner_user_id, scheduled_occurrence_id)
);
CREATE INDEX IF NOT EXISTS ai_workbench_scheduled_occurrences_plan
    ON ai_workbench.scheduled_occurrences (organization_id, owner_user_id, plan_id, scheduled_at_utc DESC);

DO $scheduled_plan$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'ai_workbench_plan_active_occurrence'
          AND conrelid = 'ai_workbench.scheduled_plans'::regclass
    ) THEN
        ALTER TABLE ai_workbench.scheduled_plans
            ADD CONSTRAINT ai_workbench_plan_active_occurrence
            FOREIGN KEY (active_occurrence_id, organization_id, owner_user_id, id)
            REFERENCES ai_workbench.scheduled_occurrences (id, organization_id, owner_user_id, plan_id);
    END IF;
END
$scheduled_plan$;

DO $scheduled_task$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'ai_workbench_task_scheduled_source'
      AND conrelid = 'ai_workbench.business_tasks'::regclass) THEN
        ALTER TABLE ai_workbench.business_tasks ADD CONSTRAINT ai_workbench_task_scheduled_source
          FOREIGN KEY (scheduled_occurrence_id, organization_id, owner_user_id)
          REFERENCES ai_workbench.scheduled_occurrences (id, organization_id, owner_user_id);
    END IF;
END
$scheduled_task$;

CREATE TABLE IF NOT EXISTS ai_workbench.scheduled_plan_commands (
    organization_id text NOT NULL,
    owner_user_id text NOT NULL,
    idempotency_key uuid NOT NULL,
    plan_id uuid,
    request_digest char(64) NOT NULL,
    outcome text NOT NULL DEFAULT 'COMPLETED' CHECK (outcome IN ('COMPLETED','REJECTED')),
    applied_revision bigint NOT NULL,
    CHECK ((outcome='COMPLETED' AND plan_id IS NOT NULL AND applied_revision>0)
      OR (outcome='REJECTED' AND plan_id IS NULL AND applied_revision=0)),
    created_at timestamptz NOT NULL,
    PRIMARY KEY (organization_id, owner_user_id, idempotency_key),
    FOREIGN KEY (plan_id, organization_id, owner_user_id)
      REFERENCES ai_workbench.scheduled_plans (id, organization_id, owner_user_id)
);
