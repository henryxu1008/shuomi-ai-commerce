CREATE TABLE IF NOT EXISTS agent_configuration.organization_agents (
 organization_id varchar(128) NOT NULL CHECK (organization_id <> ''),
 agent_id varchar(128) NOT NULL CHECK (agent_id <> ''),
 activation varchar(8) NOT NULL CHECK (activation IN ('ENABLED','DISABLED')),
 activation_epoch bigint NOT NULL CHECK (activation_epoch>0),
 revision bigint NOT NULL CHECK (revision>0),
 default_template_id uuid, default_template_revision bigint,
 created_by varchar(128) NOT NULL, updated_by varchar(128) NOT NULL,
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,agent_id),
 CHECK ((default_template_id IS NULL)=(default_template_revision IS NULL))
);
CREATE TABLE IF NOT EXISTS agent_configuration.templates (
 organization_id varchar(128) NOT NULL, agent_id varchar(128) NOT NULL,
 template_id uuid NOT NULL, lifecycle varchar(8) NOT NULL CHECK(lifecycle IN ('ACTIVE','ARCHIVED')),
 head_revision bigint NOT NULL CHECK(head_revision>0), revision bigint NOT NULL CHECK(revision>0),
 created_by varchar(128) NOT NULL, updated_by varchar(128) NOT NULL,
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,agent_id,template_id),
 FOREIGN KEY(organization_id,agent_id) REFERENCES agent_configuration.organization_agents(organization_id,agent_id)
);
CREATE TABLE IF NOT EXISTS agent_configuration.template_revisions (
 organization_id varchar(128) NOT NULL, agent_id varchar(128) NOT NULL, template_id uuid NOT NULL,
 version bigint NOT NULL CHECK(version>0), name varchar(512) NOT NULL CHECK(length(name) BETWEEN 1 AND 120),
 schema_version varchar(64) NOT NULL CHECK(schema_version IN ('title-config-v1','image-config-v1')),
 target_platform varchar(8) NOT NULL CHECK(target_platform IN ('product','shein','temu','amazon')),
 image_parameters bytea,
 default_knowledge_base_id uuid, created_by varchar(128) NOT NULL, created_at timestamptz NOT NULL,
 CHECK ((schema_version='title-config-v1' AND agent_id<>'product.image.agent' AND target_platform<>'product' AND image_parameters IS NULL)
 OR (schema_version='image-config-v1' AND agent_id='product.image.agent' AND default_knowledge_base_id IS NULL AND image_parameters IS NOT NULL AND octet_length(image_parameters) BETWEEN 2 AND 65536)),
 PRIMARY KEY(organization_id,agent_id,template_id,version),
 FOREIGN KEY(organization_id,agent_id,template_id) REFERENCES agent_configuration.templates(organization_id,agent_id,template_id)
);
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='agent_default_template_fk' AND conrelid='agent_configuration.organization_agents'::regclass) THEN
 ALTER TABLE agent_configuration.organization_agents ADD CONSTRAINT agent_default_template_fk
 FOREIGN KEY(organization_id,agent_id,default_template_id,default_template_revision)
 REFERENCES agent_configuration.template_revisions(organization_id,agent_id,template_id,version);
 END IF;
END $$;
CREATE TABLE IF NOT EXISTS agent_configuration.start_snapshots (
 id uuid PRIMARY KEY, organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL,
 context_kind varchar(128) NOT NULL, context_id varchar(128) NOT NULL, request_key varchar(128) NOT NULL,
 agent_id varchar(128) NOT NULL, agent_version varchar(128) NOT NULL,
 fingerprint varchar(64) NOT NULL, payload bytea NOT NULL CHECK(octet_length(payload)<=8192),
 digest varchar(64) NOT NULL, activation_epoch bigint NOT NULL CHECK(activation_epoch>0),
 agent_revision bigint NOT NULL CHECK(agent_revision>0), template_id uuid, template_revision bigint,
 created_at timestamptz NOT NULL,
 UNIQUE(organization_id,actor_id,context_kind,context_id,request_key),
 UNIQUE(id,organization_id,actor_id),
 CHECK((template_id IS NULL)=(template_revision IS NULL)),
 FOREIGN KEY(organization_id,agent_id) REFERENCES agent_configuration.organization_agents(organization_id,agent_id),
 FOREIGN KEY(organization_id,agent_id,template_id,template_revision) REFERENCES agent_configuration.template_revisions(organization_id,agent_id,template_id,version)
);
CREATE TABLE IF NOT EXISTS agent_configuration.commands (
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL, idempotency_key uuid NOT NULL,
 command_id uuid NOT NULL, operation varchar(32) NOT NULL, agent_id varchar(128) NOT NULL,
 template_id uuid, fingerprint varchar(64) NOT NULL,
 before_revision bigint NOT NULL, after_revision bigint NOT NULL,
 receipt bytea NOT NULL CHECK(octet_length(receipt)<=4096), committed_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,actor_id,idempotency_key), UNIQUE(command_id)
);
CREATE INDEX IF NOT EXISTS agent_templates_list ON agent_configuration.templates(organization_id,agent_id,lifecycle,created_at,template_id);
CREATE TABLE IF NOT EXISTS agent_configuration.image_run_admissions (
 snapshot_id uuid PRIMARY KEY,
 organization_id varchar(128) NOT NULL, actor_id varchar(128) NOT NULL,
 member_id varchar(128) NOT NULL CHECK(member_id<>''), run_id uuid NOT NULL, action_id uuid NOT NULL,
 id uuid NOT NULL UNIQUE, fingerprint varchar(64) NOT NULL, digest varchar(64) NOT NULL,
 payload bytea NOT NULL CHECK(octet_length(payload)<=8192),
 admitted_at timestamptz NOT NULL, deadline timestamptz NOT NULL CHECK(deadline>admitted_at),
 UNIQUE(organization_id,run_id), UNIQUE(organization_id,actor_id,action_id),
 FOREIGN KEY(snapshot_id,organization_id,actor_id) REFERENCES agent_configuration.start_snapshots(id,organization_id,actor_id)
);
CREATE INDEX IF NOT EXISTS agent_snapshots_recent ON agent_configuration.start_snapshots(organization_id,actor_id,agent_id,created_at,id);
CREATE INDEX IF NOT EXISTS agent_commands_audit ON agent_configuration.commands(organization_id,committed_at,command_id);
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
