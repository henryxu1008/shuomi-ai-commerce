package storecenter

import (
	"context"
	"database/sql"
	"errors"
)

// InstallAmazonAuthorizationSchemaTx installs the Store-owned tables in an
// explicitly selected database. The retained/runtime installation does not
// invoke it until Amazon's enablement and schema preflight are wired.
func InstallAmazonAuthorizationSchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Amazon Store schema requires an installation transaction")
	}
	for _, statement := range amazonAuthorizationSchema {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

var amazonAuthorizationSchema = []string{
	`CREATE TABLE public.workbench_store_amazon_attempts (
		session_digest VARCHAR(64) PRIMARY KEY,
		organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL, attempt_id CHAR(36) NOT NULL UNIQUE,
		actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL, store_version BIGINT NOT NULL,
		app_id VARCHAR(200) NOT NULL, app_revision VARCHAR(128) NOT NULL, redirect_uri VARCHAR(1024) NOT NULL, beta BOOLEAN NOT NULL,
		flow_digest VARCHAR(64) NOT NULL UNIQUE, state_digest VARCHAR(64) NOT NULL, code_digest VARCHAR(64) NOT NULL,
		seller_id VARCHAR(64) NOT NULL, generation BIGINT NOT NULL, phase VARCHAR(32) NOT NULL,
		started_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
		FOREIGN KEY (organization_id, store_id) REFERENCES public.workbench_stores (organization_id, id),
		CHECK (octet_length(session_digest) = 64 AND octet_length(flow_digest) = 64 AND store_version > 0),
		CHECK (expires_at > started_at AND expires_at <= started_at + INTERVAL '9 minutes'),
		CHECK (
			(phase IN ('awaiting_login', 'abandoned') AND state_digest = '' AND code_digest = '' AND seller_id = '' AND generation = 0)
			OR (phase = 'awaiting_redirect' AND octet_length(state_digest) = 64 AND seller_id <> '' AND code_digest = '' AND generation = 0)
			OR (phase IN ('exchange_claimed', 'bound') AND octet_length(state_digest) = 64 AND octet_length(code_digest) = 64 AND seller_id <> '' AND generation > 0)
		)
	)`,
	`CREATE UNIQUE INDEX workbench_store_amazon_active_attempt
		ON public.workbench_store_amazon_attempts (organization_id, store_id)
		WHERE phase IN ('awaiting_login', 'awaiting_redirect', 'exchange_claimed')`,
	`CREATE TABLE public.workbench_store_amazon_grants (
		organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL, attempt_id CHAR(36) NOT NULL,
		app_id VARCHAR(200) NOT NULL, app_revision VARCHAR(128) NOT NULL, beta BOOLEAN NOT NULL,
		seller_id VARCHAR(64) NOT NULL, generation BIGINT NOT NULL, key_id VARCHAR(128) NOT NULL,
		ciphertext TEXT NOT NULL, bound_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (organization_id, store_id), UNIQUE (beta, seller_id),
		FOREIGN KEY (organization_id, store_id) REFERENCES public.workbench_stores (organization_id, id),
		CHECK (seller_id <> '' AND generation > 0 AND key_id <> '' AND octet_length(ciphertext) BETWEEN 1 AND 16384)
	)`,
}
