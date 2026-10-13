CREATE TABLE IF NOT EXISTS agent_configuration.execution_modes (
 organization_id varchar(128) NOT NULL CHECK(organization_id<>''),
 subject_kind varchar(32) NOT NULL CHECK(subject_kind IN ('registered_agent','private_delivery')),
 subject_id varchar(128) NOT NULL CHECK(subject_id<>''),
 mode varchar(16) NOT NULL CHECK(mode IN ('default','auto_approve','read_only','full_access')),
 revision bigint NOT NULL CHECK(revision>0),
 created_by varchar(128) NOT NULL CHECK(created_by<>''), updated_by varchar(128) NOT NULL CHECK(updated_by<>''),
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,subject_kind,subject_id),
 CHECK(subject_kind<>'private_delivery' OR subject_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
);
CREATE TABLE IF NOT EXISTS agent_configuration.execution_mode_snapshots (
 id uuid PRIMARY KEY,
 organization_id varchar(128) NOT NULL CHECK(organization_id<>''), actor_id varchar(128) NOT NULL CHECK(actor_id<>''),
 member_id varchar(128) NOT NULL CHECK(member_id<>''),
 subject_kind varchar(32) NOT NULL CHECK(subject_kind IN ('registered_agent','private_delivery')),
 subject_id varchar(128) NOT NULL CHECK(subject_id<>''),
 definition_id varchar(128) NOT NULL CHECK(definition_id<>''), definition_version varchar(128) NOT NULL CHECK(definition_version<>''),
 context_kind varchar(128) NOT NULL CHECK(context_kind<>''), context_id varchar(128) NOT NULL CHECK(context_id<>''), request_key uuid NOT NULL,
 mode varchar(16) NOT NULL CHECK(mode IN ('default','auto_approve','read_only','full_access')), mode_revision bigint NOT NULL CHECK(mode_revision>=0),
 policy_version varchar(128) NOT NULL CHECK(policy_version<>''),
 fingerprint varchar(64) NOT NULL CHECK(fingerprint ~ '^[0-9a-f]{64}$'),
 digest varchar(64) NOT NULL CHECK(digest ~ '^[0-9a-f]{64}$'), payload bytea NOT NULL CHECK(octet_length(payload) BETWEEN 2 AND 8192),
 created_at timestamptz NOT NULL,
 UNIQUE(organization_id,actor_id,subject_kind,subject_id,context_kind,context_id,request_key),
 CHECK(subject_kind<>'private_delivery' OR subject_id ~ '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$')
);
