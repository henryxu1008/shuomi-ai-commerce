CREATE SCHEMA browser_kernel;
CREATE TABLE browser_kernel.releases (
 release_id uuid PRIMARY KEY,
 version text NOT NULL CHECK (length(version) BETWEEN 1 AND 64),
 os text NOT NULL CHECK (os IN ('windows','linux','darwin')),
 arch text NOT NULL CHECK (arch IN ('amd64','arm64')),
 min_client_version text NOT NULL,
 max_client_version_exclusive text NOT NULL,
 archive_format text NOT NULL CHECK (archive_format IN ('zip','tar.gz','tar.xz')),
 size bigint NOT NULL CHECK (size BETWEEN 1 AND 2147483648),
 sha256 text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
 download_url text NOT NULL CHECK (length(download_url) BETWEEN 1 AND 2048),
 recommended boolean NOT NULL DEFAULT false,
 published_at timestamptz NOT NULL DEFAULT clock_timestamp(),
 published_by varchar(128) NOT NULL,
 withdrawn_at timestamptz,
 withdrawn_by varchar(128),
 recommendation_updated_at timestamptz,
 recommendation_updated_by varchar(128),
 CHECK ((withdrawn_at IS NULL AND withdrawn_by IS NULL) OR
        (withdrawn_at IS NOT NULL AND withdrawn_by IS NOT NULL AND NOT recommended)),
 CHECK ((recommendation_updated_at IS NULL) = (recommendation_updated_by IS NULL))
);
CREATE INDEX releases_catalog ON browser_kernel.releases(os,arch,release_id) WHERE withdrawn_at IS NULL;
