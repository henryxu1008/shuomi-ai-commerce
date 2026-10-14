package storecenter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// These statements create only the admitted Store-owned facts during a fresh
// schema-owner installation. Serving never runs DDL or upgrades retained data.
var tiktokSchemaStatements = []string{
	`CREATE TABLE public.workbench_store_tiktok_oauth_grants (
 organization_id VARCHAR(200) NOT NULL, id CHAR(36) NOT NULL, app_key VARCHAR(128) NOT NULL, app_revision VARCHAR(128) NOT NULL,
 seller_identity_hash VARCHAR(64) NOT NULL, generation BIGINT NOT NULL, version BIGINT NOT NULL, status VARCHAR(32) NOT NULL,
 key_id VARCHAR(128) NOT NULL, ciphertext TEXT NOT NULL, access_expires_at TIMESTAMPTZ NOT NULL, refresh_expires_at TIMESTAMPTZ NOT NULL,
 refresh_operation_id VARCHAR(36), claim_expires_at TIMESTAMPTZ, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY (organization_id,id), CONSTRAINT tiktok_grant_seller UNIQUE (app_key,seller_identity_hash),
 CONSTRAINT tiktok_grant_app_identity UNIQUE (organization_id,id,app_key),
 CONSTRAINT tiktok_grant_bounds CHECK (app_key<>'' AND app_revision<>'' AND octet_length(seller_identity_hash)=64 AND generation>0 AND version>=generation AND key_id<>'' AND octet_length(ciphertext) BETWEEN 1 AND 16384 AND access_expires_at>created_at AND refresh_expires_at>created_at AND updated_at>=created_at),
 CONSTRAINT tiktok_grant_state CHECK ((status IN ('active','revoked','expired') AND refresh_operation_id IS NULL AND claim_expires_at IS NULL) OR (status IN ('refresh_dispatched','refresh_unknown') AND refresh_operation_id IS NOT NULL AND claim_expires_at IS NOT NULL))
)`,
	`CREATE TABLE public.workbench_store_tiktok_oauth_attempts (
 organization_id VARCHAR(200) NOT NULL, attempt_id CHAR(36) NOT NULL, store_id CHAR(36) NOT NULL, actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL,
 app_key VARCHAR(128) NOT NULL, app_revision VARCHAR(128) NOT NULL, state_hash VARCHAR(64) NOT NULL,
 store_version BIGINT NOT NULL, connection_version BIGINT NOT NULL, grant_id CHAR(36), grant_generation BIGINT NOT NULL, grant_revision BIGINT NOT NULL,
 seller_identity_hash VARCHAR(64) NOT NULL, state VARCHAR(32) NOT NULL, key_id VARCHAR(128) NOT NULL, ciphertext TEXT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY (organization_id,attempt_id),
 CONSTRAINT tiktok_attempt_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT tiktok_attempt_grant FOREIGN KEY (organization_id,grant_id,app_key) REFERENCES public.workbench_store_tiktok_oauth_grants(organization_id,id,app_key),
 CONSTRAINT tiktok_attempt_bounds CHECK (app_key<>'' AND app_revision<>'' AND actor_id<>'' AND member_id<>'' AND store_version>0 AND connection_version>0 AND octet_length(state_hash)=64 AND octet_length(seller_identity_hash) IN (0,64) AND octet_length(ciphertext)<=16384 AND updated_at>=created_at AND expires_at>created_at AND expires_at<=created_at+INTERVAL '5 minutes'),
 CONSTRAINT tiktok_attempt_state CHECK ((state IN ('created','exchange_dispatched','unknown','rejected','bound') AND key_id='' AND ciphertext='') OR (state='credential_received' AND key_id<>'' AND ciphertext<>'' AND octet_length(seller_identity_hash)=64)),
 CONSTRAINT tiktok_attempt_generation CHECK ((grant_id IS NULL AND grant_generation=0 AND grant_revision=0) OR (grant_id IS NOT NULL AND grant_generation>0 AND grant_revision>=grant_generation))
)`,
	`CREATE TABLE public.workbench_store_tiktok_shop_bindings (
 organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL, grant_id CHAR(36) NOT NULL, app_key VARCHAR(128) NOT NULL,
 shop_id VARCHAR(128) NOT NULL, shop_code VARCHAR(128) NOT NULL, shop_cipher TEXT NOT NULL, market TEXT NOT NULL, seller_type VARCHAR(32) NOT NULL,
 status VARCHAR(32) NOT NULL, version BIGINT NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY (organization_id,store_id), CONSTRAINT tiktok_binding_shop UNIQUE (shop_id),
 CONSTRAINT tiktok_binding_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT tiktok_binding_grant FOREIGN KEY (organization_id,grant_id,app_key) REFERENCES public.workbench_store_tiktok_oauth_grants(organization_id,id,app_key),
 CONSTRAINT tiktok_binding_shape CHECK (market='US' AND seller_type='CROSS_BORDER' AND shop_id<>'' AND shop_code<>'' AND octet_length(shop_cipher) BETWEEN 1 AND 8192 AND status IN ('active','disconnected','revoked') AND version>0 AND updated_at>=created_at)
)`,
	`CREATE TABLE public.workbench_store_tiktok_oauth_refresh_operations (
 organization_id VARCHAR(200) NOT NULL, grant_id CHAR(36) NOT NULL, operation_id VARCHAR(36) NOT NULL, store_id CHAR(36) NOT NULL,
 actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL, expected_generation BIGINT NOT NULL, expected_grant_revision BIGINT NOT NULL,
 input_hash VARCHAR(64) NOT NULL, state VARCHAR(32) NOT NULL, claim_expires_at TIMESTAMPTZ NOT NULL, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY (organization_id,grant_id,operation_id), CONSTRAINT tiktok_refresh_generation UNIQUE (organization_id,grant_id,expected_generation),
 CONSTRAINT tiktok_refresh_key UNIQUE (organization_id,operation_id),
 CONSTRAINT tiktok_refresh_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT tiktok_refresh_grant FOREIGN KEY (organization_id,grant_id) REFERENCES public.workbench_store_tiktok_oauth_grants(organization_id,id),
 CONSTRAINT tiktok_refresh_shape CHECK (expected_generation>0 AND expected_grant_revision>=expected_generation AND octet_length(input_hash)=64 AND actor_id<>'' AND member_id<>'' AND state IN ('dispatch','unknown','succeeded','rejected') AND claim_expires_at>created_at AND claim_expires_at<=created_at+INTERVAL '5 minutes' AND updated_at>=created_at)
)`,
	`ALTER TABLE public.workbench_store_tiktok_oauth_grants ADD CONSTRAINT tiktok_grant_operation FOREIGN KEY (organization_id,id,refresh_operation_id) REFERENCES public.workbench_store_tiktok_oauth_refresh_operations(organization_id,grant_id,operation_id) DEFERRABLE INITIALLY DEFERRED`,
	`CREATE TABLE public.workbench_store_tiktok_authorization_events (
 app_key VARCHAR(128) NOT NULL, notification_id VARCHAR(128) NOT NULL, organization_id VARCHAR(200) NOT NULL,
 store_id CHAR(36) NOT NULL, grant_id CHAR(36) NOT NULL, shop_id VARCHAR(128) NOT NULL, topic INTEGER NOT NULL,
 body_hash VARCHAR(64) NOT NULL, occurred_at TIMESTAMPTZ NOT NULL, processed_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY (app_key,notification_id),
 CONSTRAINT tiktok_event_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_store_tiktok_shop_bindings(organization_id,store_id),
 CONSTRAINT tiktok_event_grant FOREIGN KEY (organization_id,grant_id,app_key) REFERENCES public.workbench_store_tiktok_oauth_grants(organization_id,id,app_key),
 CONSTRAINT tiktok_event_shape CHECK (notification_id<>'' AND shop_id<>'' AND topic=6 AND octet_length(body_hash)=64)
)`,
}

func installTikTokSchemaTx(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range tiktokSchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

// Fixed composite keys are part of the authorization boundary, including the
// app and org in every binding to a canonical grant.
var tiktokSchemaConstraints = []struct{ table, name, definition string }{
	{"workbench_store_tiktok_oauth_attempts", "tiktok_attempt_bounds", `CHECK ((((app_key)::text <> ''::text) AND ((app_revision)::text <> ''::text) AND ((actor_id)::text <> ''::text) AND ((member_id)::text <> ''::text) AND (store_version > 0) AND (connection_version > 0) AND (octet_length((state_hash)::text) = 64) AND (octet_length((seller_identity_hash)::text) = ANY (ARRAY[0, 64])) AND (octet_length(ciphertext) <= 16384) AND (updated_at >= created_at) AND (expires_at > created_at) AND (expires_at <= (created_at + '00:05:00'::interval))))`},
	{"workbench_store_tiktok_oauth_attempts", "tiktok_attempt_generation", `CHECK ((((grant_id IS NULL) AND (grant_generation = 0) AND (grant_revision = 0)) OR ((grant_id IS NOT NULL) AND (grant_generation > 0) AND (grant_revision >= grant_generation))))`},
	{"workbench_store_tiktok_oauth_attempts", "tiktok_attempt_state", `CHECK (((((state)::text = ANY ((ARRAY['created'::character varying, 'exchange_dispatched'::character varying, 'unknown'::character varying, 'rejected'::character varying, 'bound'::character varying])::text[])) AND ((key_id)::text = ''::text) AND (ciphertext = ''::text)) OR (((state)::text = 'credential_received'::text) AND ((key_id)::text <> ''::text) AND (ciphertext <> ''::text) AND (octet_length((seller_identity_hash)::text) = 64))))`},
	{"workbench_store_tiktok_shop_bindings", "tiktok_binding_shape", `CHECK (((market = 'US'::text) AND ((seller_type)::text = 'CROSS_BORDER'::text) AND ((shop_id)::text <> ''::text) AND ((shop_code)::text <> ''::text) AND ((octet_length(shop_cipher) >= 1) AND (octet_length(shop_cipher) <= 8192)) AND ((status)::text = ANY ((ARRAY['active'::character varying, 'disconnected'::character varying, 'revoked'::character varying])::text[])) AND (version > 0) AND (updated_at >= created_at)))`},
	{"workbench_store_tiktok_authorization_events", "tiktok_event_shape", `CHECK ((((notification_id)::text <> ''::text) AND ((shop_id)::text <> ''::text) AND (topic = 6) AND (octet_length((body_hash)::text) = 64)))`},
	{"workbench_store_tiktok_oauth_grants", "tiktok_grant_bounds", `CHECK ((((app_key)::text <> ''::text) AND ((app_revision)::text <> ''::text) AND (octet_length((seller_identity_hash)::text) = 64) AND (generation > 0) AND (version >= generation) AND ((key_id)::text <> ''::text) AND ((octet_length(ciphertext) >= 1) AND (octet_length(ciphertext) <= 16384)) AND (access_expires_at > created_at) AND (refresh_expires_at > created_at) AND (updated_at >= created_at)))`},
	{"workbench_store_tiktok_oauth_grants", "tiktok_grant_state", `CHECK (((((status)::text = ANY ((ARRAY['active'::character varying, 'revoked'::character varying, 'expired'::character varying])::text[])) AND (refresh_operation_id IS NULL) AND (claim_expires_at IS NULL)) OR (((status)::text = ANY ((ARRAY['refresh_dispatched'::character varying, 'refresh_unknown'::character varying])::text[])) AND (refresh_operation_id IS NOT NULL) AND (claim_expires_at IS NOT NULL))))`},
	{"workbench_store_tiktok_oauth_refresh_operations", "tiktok_refresh_shape", `CHECK (((expected_generation > 0) AND (expected_grant_revision >= expected_generation) AND (octet_length((input_hash)::text) = 64) AND ((actor_id)::text <> ''::text) AND ((member_id)::text <> ''::text) AND ((state)::text = ANY ((ARRAY['dispatch'::character varying, 'unknown'::character varying, 'succeeded'::character varying, 'rejected'::character varying])::text[])) AND (claim_expires_at > created_at) AND (claim_expires_at <= (created_at + '00:05:00'::interval)) AND (updated_at >= created_at)))`},
	{"workbench_store_tiktok_oauth_grants", "workbench_store_tiktok_oauth_grants_pkey", "PRIMARY KEY (organization_id, id)"},
	{"workbench_store_tiktok_oauth_grants", "tiktok_grant_seller", "UNIQUE (app_key, seller_identity_hash)"},
	{"workbench_store_tiktok_oauth_grants", "tiktok_grant_app_identity", "UNIQUE (organization_id, id, app_key)"},
	{"workbench_store_tiktok_oauth_attempts", "workbench_store_tiktok_oauth_attempts_pkey", "PRIMARY KEY (organization_id, attempt_id)"},
	{"workbench_store_tiktok_oauth_attempts", "tiktok_attempt_store", "FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)"},
	{"workbench_store_tiktok_oauth_attempts", "tiktok_attempt_grant", "FOREIGN KEY (organization_id, grant_id, app_key) REFERENCES workbench_store_tiktok_oauth_grants(organization_id, id, app_key)"},
	{"workbench_store_tiktok_shop_bindings", "workbench_store_tiktok_shop_bindings_pkey", "PRIMARY KEY (organization_id, store_id)"},
	{"workbench_store_tiktok_shop_bindings", "tiktok_binding_shop", "UNIQUE (shop_id)"},
	{"workbench_store_tiktok_shop_bindings", "tiktok_binding_store", "FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)"},
	{"workbench_store_tiktok_shop_bindings", "tiktok_binding_grant", "FOREIGN KEY (organization_id, grant_id, app_key) REFERENCES workbench_store_tiktok_oauth_grants(organization_id, id, app_key)"},
	{"workbench_store_tiktok_oauth_refresh_operations", "workbench_store_tiktok_oauth_refresh_operations_pkey", "PRIMARY KEY (organization_id, grant_id, operation_id)"},
	{"workbench_store_tiktok_oauth_refresh_operations", "tiktok_refresh_generation", "UNIQUE (organization_id, grant_id, expected_generation)"},
	{"workbench_store_tiktok_oauth_refresh_operations", "tiktok_refresh_key", "UNIQUE (organization_id, operation_id)"},
	{"workbench_store_tiktok_oauth_refresh_operations", "tiktok_refresh_store", "FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)"},
	{"workbench_store_tiktok_oauth_refresh_operations", "tiktok_refresh_grant", "FOREIGN KEY (organization_id, grant_id) REFERENCES workbench_store_tiktok_oauth_grants(organization_id, id)"},
	{"workbench_store_tiktok_oauth_grants", "tiktok_grant_operation", "FOREIGN KEY (organization_id, id, refresh_operation_id) REFERENCES workbench_store_tiktok_oauth_refresh_operations(organization_id, grant_id, operation_id) DEFERRABLE INITIALLY DEFERRED"},
	{"workbench_store_tiktok_authorization_events", "workbench_store_tiktok_authorization_events_pkey", "PRIMARY KEY (app_key, notification_id)"},
	{"workbench_store_tiktok_authorization_events", "tiktok_event_store", "FOREIGN KEY (organization_id, store_id) REFERENCES workbench_store_tiktok_shop_bindings(organization_id, store_id)"},
	{"workbench_store_tiktok_authorization_events", "tiktok_event_grant", "FOREIGN KEY (organization_id, grant_id, app_key) REFERENCES workbench_store_tiktok_oauth_grants(organization_id, id, app_key)"},
}

func verifyTikTokSchema(ctx context.Context, db *sql.DB) error {
	for _, boundary := range tiktokSchemaConstraints {
		var definition string
		if err := db.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint WHERE conrelid=('public.'||$1)::regclass AND conname=$2 AND convalidated`, boundary.table, boundary.name).Scan(&definition); err != nil {
			return fmt.Errorf("TikTok Shop schema constraint unavailable: %s", boundary.name)
		}
		if strings.Join(strings.Fields(definition), " ") != boundary.definition {
			return fmt.Errorf("TikTok Shop schema constraint mismatch: %s", boundary.name)
		}
	}
	return nil
}
