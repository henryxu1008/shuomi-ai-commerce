package storecenter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/pgschema"
)

const amazonStorePlatformDefinition = `CHECK ((platform = ANY (ARRAY['shein'::text, 'temu'::text, 'tiktokshop'::text, 'walmart'::text, 'amazon'::text])))`

var amazonAuthorizationColumnNames = map[string][]string{
	"workbench_store_amazon_attempts": strings.Fields("session_digest organization_id store_id attempt_id actor_id member_id store_version app_id app_revision redirect_uri beta flow_digest state_digest code_digest seller_id generation phase started_at expires_at"),
	"workbench_store_amazon_grants":   strings.Fields("organization_id store_id attempt_id app_id app_revision beta seller_id generation key_id ciphertext bound_at"),
}

// InstallAmazonAuthorizationSchemaTx augments a fresh Store schema in an
// isolated test database. The normal Store installation never invokes it.
func InstallAmazonAuthorizationSchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Amazon Store schema requires an installation transaction")
	}
	// Use the same relation lock as the Google owner installer so the
	// platform precondition and constraint replacement see one schema.
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.workbench_stores IN ACCESS EXCLUSIVE MODE"); err != nil {
		return err
	}
	var definition string
	err := tx.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint
		WHERE conrelid='public.workbench_stores'::regclass AND contype='c' AND conname='store_platform' AND convalidated`).Scan(&definition)
	if err != nil || !pgschema.SchemaConstraintEqual(definition, currentConstraintDefinitions["store_platform"]) {
		return errors.New("Amazon authorization requires the current base Store platform constraint")
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE public.workbench_stores DROP CONSTRAINT store_platform,
		ADD CONSTRAINT store_platform `+amazonStorePlatformDefinition); err != nil {
		return err
	}
	for _, statement := range amazonAuthorizationSchema {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func verifyAmazonAuthorizationSchema(ctx context.Context, db coreSchemaQueryer) error {
	const storeForeignKey = `FOREIGN KEY (organization_id, store_id) REFERENCES workbench_stores(organization_id, id)`
	for _, constraint := range []struct{ table, name, definition string }{
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_pkey", "PRIMARY KEY (session_digest)"},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_attempt_id_key", "UNIQUE (attempt_id)"},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_flow_digest_key", "UNIQUE (flow_digest)"},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_organization_id_store_id_fkey", storeForeignKey},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_check", `CHECK (((octet_length((session_digest)::text) = 64) AND (octet_length((flow_digest)::text) = 64) AND (store_version > 0)))`},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_check1", `CHECK (((expires_at > started_at) AND (expires_at <= (started_at + '00:09:00'::interval))))`},
		{"workbench_store_amazon_attempts", "workbench_store_amazon_attempts_check2", `CHECK (((((phase)::text = ANY ((ARRAY['awaiting_login'::character varying, 'abandoned'::character varying])::text[])) AND ((state_digest)::text = ''::text) AND ((code_digest)::text = ''::text) AND ((seller_id)::text = ''::text) AND (generation = 0)) OR (((phase)::text = 'awaiting_redirect'::text) AND (octet_length((state_digest)::text) = 64) AND ((seller_id)::text <> ''::text) AND ((code_digest)::text = ''::text) AND (generation = 0)) OR (((phase)::text = ANY ((ARRAY['exchange_claimed'::character varying, 'bound'::character varying])::text[])) AND (octet_length((state_digest)::text) = 64) AND (octet_length((code_digest)::text) = 64) AND ((seller_id)::text <> ''::text) AND (generation > 0))))`},
		{"workbench_store_amazon_grants", "workbench_store_amazon_grants_pkey", "PRIMARY KEY (organization_id, store_id)"},
		{"workbench_store_amazon_grants", "workbench_store_amazon_grants_beta_seller_id_key", "UNIQUE (beta, seller_id)"},
		{"workbench_store_amazon_grants", "workbench_store_amazon_grants_organization_id_store_id_fkey", storeForeignKey},
		{"workbench_store_amazon_grants", "workbench_store_amazon_grants_check", `CHECK ((((seller_id)::text <> ''::text) AND (generation > 0) AND ((key_id)::text <> ''::text) AND ((octet_length(ciphertext) >= 1) AND (octet_length(ciphertext) <= 16384))))`},
	} {
		var definition string
		err := db.QueryRowContext(ctx, `SELECT pg_get_constraintdef(c.oid) FROM pg_catalog.pg_constraint c
			LEFT JOIN pg_catalog.pg_index i ON i.indexrelid=c.conindid
			WHERE c.conrelid=to_regclass('public.'||$1) AND c.conname=$2 AND c.convalidated AND NOT c.condeferrable
			  AND (c.contype NOT IN ('p','u') OR (i.indisvalid AND i.indisready))`,
			constraint.table, constraint.name).Scan(&definition)
		if err != nil || !pgschema.SchemaConstraintEqual(definition, constraint.definition) {
			return fmt.Errorf("Amazon authorization constraint unavailable: %s", constraint.name)
		}
	}
	var definition string
	err := db.QueryRowContext(ctx, `SELECT pg_get_indexdef(indexrelid) FROM pg_catalog.pg_index
		WHERE indexrelid=to_regclass('public.workbench_store_amazon_active_attempt') AND indisvalid AND indisready`).Scan(&definition)
	if err != nil || !pgschema.SchemaIndexEqual(definition, amazonActiveAttemptIndexDefinition) {
		return errors.New("Amazon authorization active-attempt index unavailable")
	}
	return nil
}

const amazonActiveAttemptIndexDefinition = `CREATE UNIQUE INDEX workbench_store_amazon_active_attempt ON public.workbench_store_amazon_attempts USING btree (organization_id, store_id) WHERE ((phase)::text = ANY ((ARRAY['awaiting_login'::character varying, 'awaiting_redirect'::character varying, 'exchange_claimed'::character varying])::text[]))`

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
