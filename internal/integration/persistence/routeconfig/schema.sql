CREATE TABLE IF NOT EXISTS public.ai_route_config_revisions (
 revision_id varchar(128) NOT NULL,
 organization_id varchar(128) NOT NULL,
 capability varchar(128) NOT NULL,
 parent_revision varchar(128) NOT NULL,
 schema_version integer NOT NULL CHECK(schema_version=1),
 payload jsonb NOT NULL CHECK(jsonb_typeof(payload)='object'),
 payload_digest varchar(64) NOT NULL,
 actor_id varchar(128) NOT NULL,
 published_at timestamptz NOT NULL,
 idempotency_key varchar(128) NOT NULL,
 request_fingerprint varchar(128) NOT NULL,
 rollback_from varchar(128) NOT NULL DEFAULT '',
 CONSTRAINT ai_route_config_revision_scope PRIMARY KEY(organization_id,capability,revision_id),
 CONSTRAINT ai_route_config_request_scope UNIQUE(organization_id,capability,idempotency_key)
);
CREATE TABLE IF NOT EXISTS public.ai_route_config_current (
 organization_id varchar(128) NOT NULL,
 capability varchar(128) NOT NULL,
 revision_id varchar(128) NOT NULL,
 CONSTRAINT ai_route_config_current_scope PRIMARY KEY(organization_id,capability),
 CONSTRAINT ai_route_config_current_revision FOREIGN KEY(organization_id,capability,revision_id)
 REFERENCES public.ai_route_config_revisions(organization_id,capability,revision_id)
);
CREATE TABLE IF NOT EXISTS public.ai_route_config_options (
 organization_id varchar(128) NOT NULL,
 capability varchar(128) NOT NULL,
 binding_id varchar(64) NOT NULL,
 revision_id varchar(128) NOT NULL,
 CONSTRAINT ai_route_config_option_scope PRIMARY KEY(organization_id,capability,binding_id),
 CONSTRAINT ai_route_config_option_revision FOREIGN KEY(organization_id,capability,revision_id)
 REFERENCES public.ai_route_config_revisions(organization_id,capability,revision_id)
);
