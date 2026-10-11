package storecenter

import (
	"context"
	"database/sql"
)

// Installed only by the Store schema owner. Serving never creates these tables.
var metadataSchemaStatements = []string{
	`CREATE TABLE public.workbench_store_groups (
 organization_id VARCHAR(200) NOT NULL,id CHAR(36) NOT NULL,name TEXT NOT NULL,
 version BIGINT NOT NULL,updated_by VARCHAR(200) NOT NULL,updated_at TIMESTAMPTZ NOT NULL,deleted_at TIMESTAMPTZ,
 PRIMARY KEY(organization_id,id),
 CONSTRAINT store_group_version CHECK(version BETWEEN 1 AND 9007199254740991),
 CONSTRAINT store_group_name CHECK(char_length(name) BETWEEN 1 AND 60),
 CONSTRAINT store_group_times CHECK(deleted_at IS NULL OR deleted_at>=updated_at))`,
	`CREATE TABLE public.workbench_store_profiles (
 organization_id VARCHAR(200) NOT NULL,store_id CHAR(36) NOT NULL,group_id CHAR(36),responsible_member_id VARCHAR(200),low_stock_threshold BIGINT,
 version BIGINT NOT NULL,updated_by VARCHAR(200) NOT NULL,updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(organization_id,store_id),
 CONSTRAINT store_profile_store FOREIGN KEY(organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT store_profile_group FOREIGN KEY(organization_id,group_id) REFERENCES public.workbench_store_groups(organization_id,id),
 CONSTRAINT store_profile_version CHECK(version BETWEEN 1 AND 9007199254740991),
 CONSTRAINT store_profile_threshold CHECK(low_stock_threshold IS NULL OR low_stock_threshold BETWEEN 0 AND 9007199254740991))`,
	`CREATE INDEX idx_store_profiles_group ON public.workbench_store_profiles(organization_id,group_id,store_id)`,
	`CREATE TABLE public.workbench_store_metadata_operations (
 organization_id VARCHAR(200) NOT NULL,operation_id CHAR(36) NOT NULL,actor_id VARCHAR(200) NOT NULL,member_id VARCHAR(200) NOT NULL,
 action VARCHAR(64) NOT NULL,fingerprint VARCHAR(64) NOT NULL,command_json TEXT NOT NULL,result_json TEXT NOT NULL,created_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(organization_id,operation_id),
 CONSTRAINT store_metadata_action CHECK(action IN('create_group','rename_group','delete_group','set_profile','assign_group')),
 CONSTRAINT store_metadata_bound CHECK(octet_length(command_json)<=16384 AND octet_length(result_json)<=131072 AND octet_length(fingerprint)=64))`,
}

func InstallMetadataSchemaTx(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range metadataSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}
