CREATE TABLE IF NOT EXISTS skill_capability.skills (
 id uuid PRIMARY KEY,
 owner_kind varchar(16) NOT NULL CHECK(owner_kind IN ('USER','ORGANIZATION','PLATFORM')),
 owner_id varchar(128) NOT NULL CHECK(owner_id<>''), creator varchar(128) NOT NULL CHECK(creator<>''),
 head_version bigint NOT NULL CHECK(head_version>0), revision bigint NOT NULL CHECK(revision>0),
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 CHECK(owner_kind<>'PLATFORM' OR owner_id='official')
);
CREATE TABLE IF NOT EXISTS skill_capability.versions (
 skill_id uuid NOT NULL REFERENCES skill_capability.skills(id), version bigint NOT NULL CHECK(version>0),
 files bytea NOT NULL CHECK(octet_length(files) BETWEEN 2 AND 49152),
 digest varchar(64) NOT NULL CHECK(digest ~ '^[0-9a-f]{64}$'),
 state varchar(8) NOT NULL CHECK(state IN ('DRAFT','READY','ENABLED','DISABLED','REVOKED')),
 visibility varchar(8) NOT NULL CHECK(visibility IN ('PRIVATE','SHARED')),
 allow_copy boolean NOT NULL, allow_export boolean NOT NULL,
 source_ref bytea CHECK(source_ref IS NULL OR octet_length(source_ref) BETWEEN 2 AND 1024),
 created_at timestamptz NOT NULL, PRIMARY KEY(skill_id,version)
);
CREATE TABLE IF NOT EXISTS skill_capability.commands (
 owner_kind varchar(16) NOT NULL,owner_id varchar(128) NOT NULL,actor_id varchar(128) NOT NULL,
 key uuid NOT NULL, fingerprint varchar(64) NOT NULL CHECK(fingerprint ~ '^[0-9a-f]{64}$'),
 receipt bytea NOT NULL CHECK(octet_length(receipt) BETWEEN 2 AND 2048),
 PRIMARY KEY(owner_kind,owner_id,actor_id,key)
);
CREATE TABLE IF NOT EXISTS skill_capability.share_requests (
 id uuid PRIMARY KEY, source_skill_id uuid NOT NULL,source_version bigint NOT NULL,
 source_digest varchar(64) NOT NULL CHECK(source_digest ~ '^[0-9a-f]{64}$'),
 submitted_by varchar(128) NOT NULL CHECK(submitted_by<>''), organization_id varchar(128) NOT NULL CHECK(organization_id<>''),
 consent boolean NOT NULL CHECK(consent), state varchar(16) NOT NULL CHECK(state IN ('SUBMITTED','WITHDRAWN','APPROVED','REJECTED')),
 result_skill_id uuid REFERENCES skill_capability.skills(id), created_at timestamptz NOT NULL,updated_at timestamptz NOT NULL,
 FOREIGN KEY(source_skill_id,source_version) REFERENCES skill_capability.versions(skill_id,version),
 CHECK((state='APPROVED')=(result_skill_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS skill_owner_directory ON skill_capability.skills(owner_kind,owner_id,id);
CREATE INDEX IF NOT EXISTS skill_share_organization ON skill_capability.share_requests(organization_id,id);
