package storecenter

import (
	"context"
	"database/sql"
	"errors"
)

var googleMerchantSchemaStatements = []string{
	`CREATE TABLE public.workbench_store_google_oauth_attempts (
		organization_id VARCHAR(200) NOT NULL,
		store_id CHAR(36) NOT NULL,
		attempt_id CHAR(36) NOT NULL,
		actor_id VARCHAR(200) NOT NULL,
		member_id VARCHAR(200) NOT NULL,
		app_id VARCHAR(128) NOT NULL,
		app_revision VARCHAR(128) NOT NULL,
		state_hash CHAR(64) NOT NULL,
		state VARCHAR(24) NOT NULL CHECK (state IN ('created','exchanging','candidate','bound','superseded')),
		key_id VARCHAR(128) NOT NULL,
		ciphertext TEXT NOT NULL,
		grant_id CHAR(36) NOT NULL,
		store_version BIGINT NOT NULL CHECK (store_version > 0),
		generation BIGINT NOT NULL CHECK (generation > 0),
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (organization_id, attempt_id),
		FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
		CHECK ((state = 'candidate') = (key_id <> '' AND ciphertext <> ''))
	)`,
	`CREATE TABLE public.workbench_store_google_oauth_grants (
		organization_id VARCHAR(200) NOT NULL,
		store_id CHAR(36) NOT NULL,
		id CHAR(36) NOT NULL,
		attempt_id CHAR(36) NOT NULL,
		app_id VARCHAR(128) NOT NULL,
		app_revision VARCHAR(128) NOT NULL,
		generation BIGINT NOT NULL CHECK (generation > 0),
		account_id VARCHAR(19) NOT NULL UNIQUE,
		data_source_id VARCHAR(19) NOT NULL,
		content_language CHAR(2) NOT NULL,
		feed_label VARCHAR(20) NOT NULL,
		offer_prefix CHAR(24) NOT NULL UNIQUE,
		key_id VARCHAR(128) NOT NULL,
		ciphertext TEXT NOT NULL,
		revoked BOOLEAN NOT NULL,
		access_expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ NOT NULL,
		updated_at TIMESTAMPTZ NOT NULL,
		PRIMARY KEY (organization_id,store_id),
		FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
		FOREIGN KEY (organization_id,attempt_id) REFERENCES public.workbench_store_google_oauth_attempts(organization_id,attempt_id),
		CHECK (revoked = (key_id = '' AND ciphertext = ''))
	)`,
}

const googleRefreshTable = "public.workbench_store_google_oauth_refresh"

const googleRefreshTableDDL = `CREATE TABLE ` + googleRefreshTable + ` (
	organization_id VARCHAR(200) NOT NULL,
	store_id CHAR(36) NOT NULL,
	generation BIGINT NOT NULL,
	revision BIGINT NOT NULL,
	phase VARCHAR(24) NOT NULL,
	claim_id CHAR(36),
	deadline TIMESTAMPTZ,
	CONSTRAINT google_refresh_key PRIMARY KEY (organization_id,store_id),
	CONSTRAINT google_refresh_grant FOREIGN KEY (organization_id,store_id)
		REFERENCES public.workbench_store_google_oauth_grants(organization_id,store_id),
	CONSTRAINT google_refresh_version CHECK (generation > 0 AND revision > 0),
	CONSTRAINT google_refresh_phase CHECK (
		(phase = 'idle' AND claim_id IS NULL)
		OR (phase IN ('claimed','possible') AND claim_id IS NOT NULL AND deadline IS NOT NULL)
		OR (phase = 'reauthorize' AND claim_id IS NULL AND deadline IS NULL))
)`

// InstallGoogleMerchantSchemaTx adds this channel through an explicit owner
// operation. Runtime clients never install or alter Store schema.
func InstallGoogleMerchantSchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Google Merchant Store schema requires its owner transaction")
	}
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.workbench_stores IN ACCESS EXCLUSIVE MODE"); err != nil {
		return err
	}
	var amazonInstalled, shopifyInstalled bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.workbench_store_amazon_attempts') IS NOT NULL
		OR to_regclass('public.workbench_store_amazon_grants') IS NOT NULL,
		to_regclass('public.workbench_store_shopify_shops') IS NOT NULL
		OR to_regclass('public.workbench_store_shopify_authorizations') IS NOT NULL
		OR to_regclass('public.workbench_store_shopify_pairings') IS NOT NULL`).Scan(&amazonInstalled, &shopifyInstalled); err != nil {
		return err
	}
	if err := verifyCurrentSchema(ctx, tx, RuntimeCapabilities{
		TikTokShop: true, WalmartUS: true, AmazonAuthorization: amazonInstalled, Shopify: shopifyInstalled,
	}); err != nil {
		return err
	}
	installed, err := googleMerchantSchemaPresent(ctx, tx)
	if err != nil {
		return err
	}
	if !installed {
		next := storePlatformDefinition(amazonInstalled, true, shopifyInstalled)
		if _, err := tx.ExecContext(ctx, "ALTER TABLE public.workbench_stores DROP CONSTRAINT store_platform, ADD CONSTRAINT store_platform "+next); err != nil {
			return err
		}
		for _, statement := range googleMerchantSchemaStatements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
	}
	var refreshInstalled bool
	if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", googleRefreshTable).Scan(&refreshInstalled); err != nil {
		return err
	}
	if !refreshInstalled {
		if _, err := tx.ExecContext(ctx, googleRefreshTableDDL); err != nil {
			return err
		}
	}
	return verifyGoogleRefreshSchema(ctx, tx)
}

func googleMerchantSchemaPresent(ctx context.Context, db coreSchemaQueryer) (bool, error) {
	var installed bool
	err := db.QueryRowContext(ctx, `SELECT to_regclass('public.workbench_store_google_oauth_grants') IS NOT NULL
		OR to_regclass('public.workbench_store_google_oauth_attempts') IS NOT NULL`).Scan(&installed)
	return installed, err
}

func verifyGoogleSchema(ctx context.Context, db coreSchemaQueryer) error {
	var valid bool
	err := db.QueryRowContext(ctx, `SELECT (SELECT count(*) = 2 FROM pg_constraint
		WHERE conrelid = to_regclass('public.workbench_store_google_oauth_grants') AND contype = 'u'
		AND pg_get_constraintdef(oid) IN ('UNIQUE (account_id)', 'UNIQUE (offer_prefix)'))
		AND to_regclass('public.workbench_store_google_oauth_attempts') IS NOT NULL`).Scan(&valid)
	if err != nil || !valid {
		return errors.New("Google Merchant Store authorization schema unavailable")
	}
	return nil
}

// Check types, nullability and check definitions on the read-only runtime path.
func verifyGoogleRefreshSchema(ctx context.Context, db coreSchemaQueryer) error {
	var valid bool
	err := db.QueryRowContext(ctx, `SELECT
		(SELECT array_agg(attname::text || ':' || format_type(atttypid,atttypmod) || ':' || attnotnull::text ORDER BY attnum)
		FROM pg_attribute WHERE attrelid=to_regclass($1) AND attnum>0 AND NOT attisdropped)
		= ARRAY['organization_id:character varying(200):true','store_id:character(36):true',
			'generation:bigint:true','revision:bigint:true','phase:character varying(24):true',
			'claim_id:character(36):false','deadline:timestamp with time zone:false']
		AND (SELECT count(*)=4 FROM pg_constraint WHERE conrelid=to_regclass($1) AND convalidated AND (
			(conname='google_refresh_key' AND contype='p' AND pg_get_constraintdef(oid)='PRIMARY KEY (organization_id, store_id)')
			OR (conname='google_refresh_grant' AND contype='f' AND confrelid=to_regclass('public.workbench_store_google_oauth_grants')
				AND conkey=ARRAY[1,2]::smallint[] AND confkey=ARRAY[1,2]::smallint[])
			OR (conname='google_refresh_version' AND contype='c' AND pg_get_expr(conbin,conrelid)='((generation > 0) AND (revision > 0))')
			OR (conname='google_refresh_phase' AND contype='c' AND pg_get_expr(conbin,conrelid)=
				'((((phase)::text = ''idle''::text) AND (claim_id IS NULL)) OR (((phase)::text = ANY ((ARRAY[''claimed''::character varying, ''possible''::character varying])::text[])) AND (claim_id IS NOT NULL) AND (deadline IS NOT NULL)) OR (((phase)::text = ''reauthorize''::text) AND (claim_id IS NULL) AND (deadline IS NULL)))')
		))`, googleRefreshTable).Scan(&valid)
	if err != nil || !valid {
		return errors.New("Google Merchant Store renewal schema unavailable")
	}
	return nil
}
