CREATE TABLE IF NOT EXISTS public.ai_search_credential_versions (
    organization_id text NOT NULL,
    revision bigint NOT NULL CHECK (revision > 0),
    updated_at timestamptz NOT NULL DEFAULT clock_timestamp(),
    PRIMARY KEY (organization_id, revision)
);
CREATE TABLE IF NOT EXISTS public.ai_web_search_configuration (
    owner text PRIMARY KEY,
    revision bigint NOT NULL CHECK (revision > 0),
    payload jsonb NOT NULL
);
CREATE TABLE IF NOT EXISTS public.ai_web_search_config_receipts (
    owner text NOT NULL,
    actor text NOT NULL,
    request_key text NOT NULL,
    input_digest text NOT NULL,
    payload jsonb NOT NULL,
    PRIMARY KEY (owner, actor, request_key)
);
CREATE TABLE IF NOT EXISTS public.ai_web_search_requests (
    organization text NOT NULL,
    actor text NOT NULL,
    member text NOT NULL,
    conversation text NOT NULL,
    request_key text NOT NULL,
    query text NOT NULL,
    fingerprint text NOT NULL,
    source text NOT NULL CHECK (source IN ('platform', 'owned')),
    connection_owner text NOT NULL,
    preview text NOT NULL,
    credential_reference text NOT NULL,
    publication_revision bigint NOT NULL,
    choice_revision bigint NOT NULL,
    max_usd_micros bigint NOT NULL CHECK (max_usd_micros >= 0),
    day_start timestamptz NOT NULL,
    started_at timestamptz NOT NULL,
    deadline timestamptz NOT NULL,
    status text NOT NULL,
    usage integer,
    PRIMARY KEY (organization, actor, member, conversation, request_key),
    CHECK (started_at >= day_start AND deadline <= day_start + interval '24 hours' AND deadline > started_at),
    CHECK (
        (status IN ('held', 'claimed', 'unknown') AND usage IS NULL)
        OR (status = 'not_sent' AND usage IS NOT NULL AND usage = 0)
        OR (status IN ('complete', 'no_results') AND usage IS NOT NULL AND usage = 1)
    )
);
CREATE INDEX IF NOT EXISTS ai_web_search_requests_window ON public.ai_web_search_requests (connection_owner, day_start, organization);
CREATE INDEX IF NOT EXISTS ai_web_search_requests_history ON public.ai_web_search_requests (organization, actor, member, conversation, started_at DESC);
CREATE INDEX IF NOT EXISTS ai_web_search_requests_expiring ON public.ai_web_search_requests (connection_owner, deadline) WHERE status = 'held';
