package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"strings"
)

var walmartSchemaStatements = []string{
	`CREATE TABLE public.workbench_store_walmart_oauth_grants (
 organization_id VARCHAR(200) NOT NULL,id CHAR(36) NOT NULL,store_id CHAR(36) NOT NULL,
 app_id VARCHAR(200) NOT NULL,app_revision VARCHAR(128) NOT NULL,environment VARCHAR(16) NOT NULL,
 merchant_identity_hash VARCHAR(64) NOT NULL,seller_id VARCHAR(200) NOT NULL,merchant_name VARCHAR(500) NOT NULL,
 generation BIGINT NOT NULL,version BIGINT NOT NULL,status VARCHAR(32) NOT NULL,key_id VARCHAR(128) NOT NULL,ciphertext TEXT NOT NULL,scopes_hash VARCHAR(64) NOT NULL,
 access_expires_at TIMESTAMPTZ NOT NULL,refresh_expires_at TIMESTAMPTZ NOT NULL,
 refresh_operation_id VARCHAR(36) NOT NULL,refresh_input_hash VARCHAR(64) NOT NULL,refresh_actor_id VARCHAR(200) NOT NULL,refresh_member_id VARCHAR(200) NOT NULL,refresh_generation BIGINT NOT NULL,claim_expires_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL,updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(organization_id,id),CONSTRAINT walmart_grant_store UNIQUE(organization_id,store_id),
 CONSTRAINT walmart_grant_merchant UNIQUE(environment,app_id,merchant_identity_hash),
 CONSTRAINT walmart_grant_store_fk FOREIGN KEY(organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT walmart_grant_identity CHECK(environment IN ('production','sandbox') AND app_id<>'' AND app_revision<>'' AND seller_id<>'' AND octet_length(merchant_identity_hash)=64 AND octet_length(scopes_hash)=64),
 CONSTRAINT walmart_grant_bounds CHECK(generation>0 AND version>=generation AND key_id<>'' AND octet_length(ciphertext) BETWEEN 1 AND 16384 AND access_expires_at>created_at AND refresh_expires_at>created_at AND updated_at>=created_at),
 CONSTRAINT walmart_grant_state CHECK(status IN ('active','refresh_dispatched','refresh_unknown','revoked','expired') AND (status NOT IN ('refresh_dispatched','refresh_unknown') OR (refresh_operation_id<>'' AND octet_length(refresh_input_hash)=64 AND refresh_generation=generation AND refresh_actor_id<>'' AND refresh_member_id<>'')))
 )`,
	`CREATE TABLE public.workbench_store_walmart_oauth_attempts (
 organization_id VARCHAR(200) NOT NULL,attempt_id CHAR(36) NOT NULL,store_id CHAR(36) NOT NULL,actor_id VARCHAR(200) NOT NULL,member_id VARCHAR(200) NOT NULL,
 app_id VARCHAR(200) NOT NULL,app_revision VARCHAR(128) NOT NULL,environment VARCHAR(16) NOT NULL,callback_url TEXT NOT NULL,state_hash VARCHAR(64) NOT NULL,nonce VARCHAR(128) NOT NULL,
 store_version BIGINT NOT NULL,connection_version BIGINT NOT NULL,grant_id VARCHAR(36) NOT NULL,grant_generation BIGINT NOT NULL,grant_version BIGINT NOT NULL,
 seller_id VARCHAR(200) NOT NULL,state VARCHAR(32) NOT NULL,key_id VARCHAR(128) NOT NULL,ciphertext TEXT NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL,created_at TIMESTAMPTZ NOT NULL,updated_at TIMESTAMPTZ NOT NULL,
 PRIMARY KEY(organization_id,attempt_id),CONSTRAINT walmart_attempt_store FOREIGN KEY(organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT walmart_attempt_bounds CHECK(store_version>0 AND connection_version>0 AND actor_id<>'' AND member_id<>'' AND app_id<>'' AND app_revision<>'' AND environment IN ('production','sandbox') AND callback_url<>'' AND octet_length(state_hash)=64 AND nonce<>'' AND expires_at>created_at AND expires_at<=created_at+INTERVAL '5 minutes' AND updated_at>=created_at),
 CONSTRAINT walmart_attempt_state CHECK((state IN ('created','exchange_dispatched','bound','rejected','unknown') AND key_id='' AND ciphertext='') OR (state='credential_received' AND key_id<>'' AND octet_length(ciphertext) BETWEEN 1 AND 16384 AND seller_id<>''))
 )`,
}

func installWalmartSchemaTx(ctx context.Context, tx *sql.Tx) error {
	for _, s := range walmartSchemaStatements {
		if _, e := tx.ExecContext(ctx, s); e != nil {
			return e
		}
	}
	return nil
}

// Exact PostgreSQL 16 contract: same-name weakened constraints are rejected.
var walmartSchemaConstraints = []struct{ table, name, definition string }{
	{"workbench_store_merchant_bindings", "store_merchant_binding_shape", `CHECK (((octet_length((identity_hash)::text) = 64) AND ((app_id)::text <> ''::text) AND ((((platform)::text = 'shein'::text) AND ((open_key_id)::text <> ''::text)) OR (((platform)::text = ANY ((ARRAY['temu'::character varying, 'walmart'::character varying])::text[])) AND ((open_key_id)::text = ''::text)))))`},
	{"workbench_store_walmart_oauth_attempts", "walmart_attempt_bounds", `CHECK (((store_version > 0) AND (connection_version > 0) AND ((actor_id)::text <> ''::text) AND ((member_id)::text <> ''::text) AND ((app_id)::text <> ''::text) AND ((app_revision)::text <> ''::text) AND ((environment)::text = ANY ((ARRAY['production'::character varying, 'sandbox'::character varying])::text[])) AND (callback_url <> ''::text) AND (octet_length((state_hash)::text) = 64) AND ((nonce)::text <> ''::text) AND (expires_at > created_at) AND (expires_at <= (created_at + '00:05:00'::interval)) AND (updated_at >= created_at)))`},
	{"workbench_store_walmart_oauth_attempts", "walmart_attempt_state", `CHECK (((((state)::text = ANY ((ARRAY['created'::character varying, 'exchange_dispatched'::character varying, 'bound'::character varying, 'rejected'::character varying, 'unknown'::character varying])::text[])) AND ((key_id)::text = ''::text) AND (ciphertext = ''::text)) OR (((state)::text = 'credential_received'::text) AND ((key_id)::text <> ''::text) AND ((octet_length(ciphertext) >= 1) AND (octet_length(ciphertext) <= 16384)) AND ((seller_id)::text <> ''::text))))`},
	{"workbench_store_walmart_oauth_attempts", "walmart_attempt_store", `FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_bounds", `CHECK (((generation > 0) AND (version >= generation) AND ((key_id)::text <> ''::text) AND ((octet_length(ciphertext) >= 1) AND (octet_length(ciphertext) <= 16384)) AND (access_expires_at > created_at) AND (refresh_expires_at > created_at) AND (updated_at >= created_at)))`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_identity", `CHECK ((((environment)::text = ANY ((ARRAY['production'::character varying, 'sandbox'::character varying])::text[])) AND ((app_id)::text <> ''::text) AND ((app_revision)::text <> ''::text) AND ((seller_id)::text <> ''::text) AND (octet_length((merchant_identity_hash)::text) = 64) AND (octet_length((scopes_hash)::text) = 64)))`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_merchant", `UNIQUE (environment, app_id, merchant_identity_hash)`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_state", `CHECK ((((status)::text = ANY ((ARRAY['active'::character varying, 'refresh_dispatched'::character varying, 'refresh_unknown'::character varying, 'revoked'::character varying, 'expired'::character varying])::text[])) AND (((status)::text <> ALL ((ARRAY['refresh_dispatched'::character varying, 'refresh_unknown'::character varying])::text[])) OR (((refresh_operation_id)::text <> ''::text) AND (octet_length((refresh_input_hash)::text) = 64) AND (refresh_generation = generation) AND ((refresh_actor_id)::text <> ''::text) AND ((refresh_member_id)::text <> ''::text)))))`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_store", `UNIQUE (organization_id, store_id)`},
	{"workbench_store_walmart_oauth_grants", "walmart_grant_store_fk", `FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)`},
	{"workbench_store_walmart_oauth_attempts", "workbench_store_walmart_oauth_attempts_pkey", `PRIMARY KEY (organization_id, attempt_id)`},
	{"workbench_store_walmart_oauth_grants", "workbench_store_walmart_oauth_grants_pkey", `PRIMARY KEY (organization_id, id)`},
}

func verifyWalmartSchema(ctx context.Context, db *sql.DB) error {
	for _, c := range walmartSchemaConstraints {
		var definition string
		if e := db.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint WHERE conrelid=('public.'||$1)::regclass AND conname=$2 AND convalidated`, c.table, c.name).Scan(&definition); e != nil || strings.Join(strings.Fields(definition), " ") != c.definition {
			return errors.New("Walmart Store schema boundary unavailable")
		}
	}

	for _, c := range []struct{ table, name, kind string }{
		{"workbench_store_walmart_oauth_grants", "workbench_store_walmart_oauth_grants_pkey", "p"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_store", "u"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_merchant", "u"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_store_fk", "f"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_identity", "c"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_bounds", "c"}, {"workbench_store_walmart_oauth_grants", "walmart_grant_state", "c"},
		{"workbench_store_walmart_oauth_attempts", "workbench_store_walmart_oauth_attempts_pkey", "p"}, {"workbench_store_walmart_oauth_attempts", "walmart_attempt_store", "f"}, {"workbench_store_walmart_oauth_attempts", "walmart_attempt_bounds", "c"}, {"workbench_store_walmart_oauth_attempts", "walmart_attempt_state", "c"},
	} {
		var valid bool
		if e := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_catalog.pg_constraint WHERE conrelid=('public.'||$1)::regclass AND conname=$2 AND contype::text=$3 AND convalidated)`, c.table, c.name, c.kind).Scan(&valid); e != nil || !valid {
			return errors.New("Walmart Store schema boundary unavailable")
		}
	}
	return nil
}
