package storecenter

import (
	"context"
	"database/sql"
	"errors"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/pkg/pgschema"
)

const shopifyInstallSchema = `CREATE TABLE public.workbench_store_shopify_installs (
 attempt_id CHAR(36) NOT NULL, app_id VARCHAR(128) NOT NULL, domain VARCHAR(253) NOT NULL,
 session_digest CHAR(64) NOT NULL, state_digest CHAR(64) NOT NULL, code_digest VARCHAR(64) NOT NULL, state VARCHAR(16) NOT NULL,
 shop_id VARCHAR(128) NOT NULL, installation_id VARCHAR(128) NOT NULL, currency VARCHAR(3) NOT NULL,
 generation BIGINT NOT NULL, key_id VARCHAR(128) NOT NULL, ciphertext TEXT NOT NULL,
 organization_id VARCHAR(200) NOT NULL, store_id VARCHAR(36) NOT NULL, actor_id VARCHAR(200) NOT NULL,
 member_id VARCHAR(200) NOT NULL, store_version BIGINT NOT NULL, shop_key VARCHAR(36) NOT NULL,
 created_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
 CONSTRAINT shopify_install_pk PRIMARY KEY (attempt_id),
 CONSTRAINT shopify_install_bounds CHECK (app_id <> '' AND generation >= 0 AND octet_length(ciphertext) <= 16384
  AND ((state IN ('created','confirmed') AND code_digest = '') OR (state IN ('exchanging','staged','selected') AND length(code_digest) = 64))
  AND expires_at > created_at AND expires_at <= created_at + INTERVAL '10 minutes'),
 CONSTRAINT shopify_install_selection CHECK (
  (state IN ('selected','confirmed') AND organization_id <> '' AND store_id <> '' AND actor_id <> '' AND member_id <> '' AND store_version > 0)
  OR (state IN ('created','exchanging','staged') AND organization_id = '' AND store_id = '' AND actor_id = '' AND member_id = '' AND store_version = 0)),
 CONSTRAINT shopify_install_state CHECK (
  (state IN ('created','exchanging') AND domain <> '' AND shop_id = '' AND installation_id = '' AND currency = ''
   AND generation = 0 AND shop_key = '' AND key_id = '' AND ciphertext = '')
  OR (state IN ('staged','selected') AND domain <> '' AND shop_id <> '' AND installation_id <> '' AND length(currency) = 3
   AND shop_key = '' AND key_id <> '' AND ciphertext <> '')
  OR (state = 'confirmed' AND domain = '' AND shop_id = '' AND installation_id = '' AND currency = ''
   AND generation > 0 AND shop_key <> '' AND key_id = '' AND ciphertext = ''))
)`

var shopifySchemaStatements = []string{
	`CREATE TABLE public.workbench_store_shopify_shops (
 organization_id VARCHAR(200) NOT NULL, shop_key CHAR(36) NOT NULL, store_id CHAR(36) NOT NULL,
 app_id VARCHAR(128) NOT NULL, shop_id VARCHAR(128) NOT NULL, generation BIGINT NOT NULL,
 CONSTRAINT shopify_shop_pk PRIMARY KEY (organization_id,shop_key),
 CONSTRAINT shopify_store_unique UNIQUE (organization_id,store_id),
 CONSTRAINT shopify_identity_unique UNIQUE (app_id,shop_id),
 CONSTRAINT shopify_shop_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT shopify_shop_shape CHECK (app_id<>'' AND shop_id<>'' AND generation>0)
)`,
	`CREATE TABLE public.workbench_store_shopify_authorizations (
 organization_id VARCHAR(200) NOT NULL, shop_key CHAR(36) NOT NULL, generation BIGINT NOT NULL,
 installation_id VARCHAR(128) NOT NULL, domain VARCHAR(253) NOT NULL, currency VARCHAR(3) NOT NULL,
 approved_at TIMESTAMPTZ NOT NULL, revoked_at TIMESTAMPTZ,
 CONSTRAINT shopify_authorization_pk PRIMARY KEY (organization_id,shop_key,generation),
 CONSTRAINT shopify_authorization_shop FOREIGN KEY (organization_id,shop_key)
 REFERENCES public.workbench_store_shopify_shops(organization_id,shop_key),
 CONSTRAINT shopify_authorization_shape CHECK (generation>0 AND installation_id<>'' AND domain<>'' AND length(currency)=3
 AND (revoked_at IS NULL OR revoked_at>=approved_at))
)`,
	`ALTER TABLE public.workbench_store_shopify_shops ADD CONSTRAINT shopify_current_authorization
 FOREIGN KEY (organization_id,shop_key,generation)
 REFERENCES public.workbench_store_shopify_authorizations(organization_id,shop_key,generation)
 DEFERRABLE INITIALLY DEFERRED`,
	`CREATE TABLE public.workbench_store_shopify_pairings (
 organization_id VARCHAR(200) NOT NULL, store_id CHAR(36) NOT NULL, attempt_id CHAR(36) NOT NULL,
 actor_id VARCHAR(200) NOT NULL, member_id VARCHAR(200) NOT NULL, store_version BIGINT NOT NULL,
 app_id VARCHAR(128) NOT NULL, state_digest VARCHAR(64) NOT NULL, session_digest VARCHAR(64) NOT NULL,
 state VARCHAR(20) NOT NULL, shop_id VARCHAR(128) NOT NULL DEFAULT '', installation_id VARCHAR(128) NOT NULL DEFAULT '',
 domain VARCHAR(253) NOT NULL DEFAULT '', currency VARCHAR(3) NOT NULL DEFAULT '', generation BIGINT NOT NULL DEFAULT 0,
 key_id VARCHAR(128) NOT NULL DEFAULT '', ciphertext TEXT NOT NULL DEFAULT '', shop_key CHAR(36),
 created_at TIMESTAMPTZ NOT NULL, expires_at TIMESTAMPTZ NOT NULL,
 CONSTRAINT shopify_pairing_pk PRIMARY KEY (organization_id,attempt_id),
 CONSTRAINT shopify_pairing_store FOREIGN KEY (organization_id,store_id) REFERENCES public.workbench_stores(organization_id,id),
 CONSTRAINT shopify_pairing_bounds CHECK (store_version>0 AND app_id<>'' AND actor_id<>'' AND member_id<>''
 AND length(state_digest)=64 AND length(session_digest)=64 AND expires_at>created_at
 AND expires_at<=created_at+INTERVAL '10 minutes' AND generation>=0 AND octet_length(ciphertext)<=16384),
 CONSTRAINT shopify_pairing_state CHECK (
 (state IN ('created','exchanging','cancelled') AND shop_key IS NULL AND key_id='' AND ciphertext='') OR
 (state='staged' AND shop_key IS NULL AND key_id<>'' AND ciphertext<>'' AND shop_id<>'' AND installation_id<>'' AND domain<>'' AND length(currency)=3) OR
 (state='confirmed' AND shop_key IS NOT NULL AND generation>0 AND key_id='' AND ciphertext='' AND shop_id='' AND installation_id='' AND domain='' AND currency=''))
)`,
	`CREATE TABLE public.workbench_store_shopify_lifecycle_events (
 app_id VARCHAR(128) NOT NULL, shop_id VARCHAR(128) NOT NULL, revision BIGINT NOT NULL,
 topic VARCHAR(32) NOT NULL, event_id VARCHAR(128) NOT NULL, body_hash CHAR(64) NOT NULL,
 organization_id VARCHAR(200) NOT NULL, shop_key VARCHAR(36) NOT NULL, generation BIGINT NOT NULL,
 status VARCHAR(16) NOT NULL, received_at TIMESTAMPTZ NOT NULL, retry_at TIMESTAMPTZ, attempt BIGINT NOT NULL,
 CONSTRAINT shopify_lifecycle_pk PRIMARY KEY (app_id,shop_id,revision),
 CONSTRAINT shopify_lifecycle_event UNIQUE (app_id,shop_id,topic,event_id),
 CONSTRAINT shopify_lifecycle_bounds CHECK (revision>0 AND attempt>=0 AND app_id<>'' AND shop_id<>'' AND event_id<>''
   AND topic IN ('app/uninstalled','app/scopes_update') AND length(body_hash)=64
   AND ((status='unbound' AND generation=0 AND organization_id='' AND shop_key='' AND retry_at IS NULL AND attempt=0)
     OR (generation>0 AND organization_id<>'' AND shop_key<>''
       AND ((status='pending' AND retry_at IS NOT NULL) OR (status IN ('healthy','revoked','superseded') AND retry_at IS NULL)))) )
)`,
	`CREATE INDEX shopify_lifecycle_due ON public.workbench_store_shopify_lifecycle_events(app_id,retry_at,shop_id,revision) WHERE status='pending'`,
	`CREATE INDEX shopify_lifecycle_pending ON public.workbench_store_shopify_lifecycle_events(organization_id,shop_key,generation) WHERE status='pending'`,
	shopifyInstallSchema,
	`CREATE INDEX shopify_install_expiry ON public.workbench_store_shopify_installs(expires_at)`,
}

// InstallShopifySchemaTx admits the optional Shopify capability on an empty Store
// database. The caller owns the transaction and any runtime grants.
func InstallShopifySchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return errors.New("Store Shopify schema transaction unavailable")
	}
	if _, err := tx.ExecContext(ctx, "LOCK TABLE public.workbench_stores IN ACCESS EXCLUSIVE MODE"); err != nil {
		return err
	}
	var installed, hasStores bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.workbench_store_shopify_shops') IS NOT NULL, EXISTS (SELECT 1 FROM public.workbench_stores)`).Scan(&installed, &hasStores); err != nil {
		return err
	}
	var definition string
	if err := tx.QueryRowContext(ctx, `SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint
		WHERE conrelid='public.workbench_stores'::regclass AND contype='c' AND conname='store_platform' AND convalidated`).Scan(&definition); err != nil {
		return err
	}
	googleInstalled, err := googleMerchantSchemaPresent(ctx, tx)
	if err != nil {
		return err
	}
	capabilities := RuntimeCapabilities{TikTokShop: true, WalmartUS: true, Shopify: true}
	capabilities.AmazonAuthorization = pgschema.SchemaConstraintEqual(definition, storePlatformDefinition(true, googleInstalled, false)) ||
		pgschema.SchemaConstraintEqual(definition, storePlatformDefinition(true, googleInstalled, true))
	if installed {
		return verifyCurrentSchema(ctx, tx, capabilities)
	}
	if hasStores {
		return errors.New("Store Shopify schema requires an empty Store database")
	}
	before := capabilities
	before.Shopify = false
	if err := verifyCurrentSchema(ctx, tx, before); err != nil {
		return err
	}
	platformConstraint := storePlatformDefinition(capabilities.AmazonAuthorization, googleInstalled, true)
	if _, err := tx.ExecContext(ctx, `ALTER TABLE public.workbench_stores DROP CONSTRAINT store_platform, ADD CONSTRAINT store_platform `+platformConstraint); err != nil {
		return err
	}
	for _, statement := range shopifySchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	capabilities.ShopifyStartedInstallation = true
	return verifyCurrentSchema(ctx, tx, capabilities)
}

func verifyShopifyStartedInstallSchema(ctx context.Context, db coreSchemaQueryer) error {
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT count(*) = 4 FROM pg_catalog.pg_constraint c
 JOIN pg_catalog.pg_class r ON r.oid=c.conrelid
 WHERE c.conrelid=to_regclass('public.workbench_store_shopify_installs') AND r.relkind='r'
 AND NOT r.relrowsecurity AND r.relpersistence='p' AND c.convalidated AND NOT c.condeferrable
 AND ((c.conname='shopify_install_pk' AND c.contype='p' AND pg_get_constraintdef(c.oid)='PRIMARY KEY (attempt_id)')
  OR (c.conname IN ('shopify_install_bounds','shopify_install_selection','shopify_install_state') AND c.contype='c'))`).Scan(&ready)
	if err != nil || !ready {
		return errors.New("Store Shopify-started installation schema unavailable")
	}
	return nil
}

func verifyShopifySchema(ctx context.Context, db coreSchemaQueryer) error {
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT NOT EXISTS (
 SELECT 1 FROM (VALUES
   ('workbench_store_shopify_shops','shopify_identity_unique','UNIQUE (app_id, shop_id)'),
   ('workbench_store_shopify_shops','shopify_store_unique','UNIQUE (organization_id, store_id)'),
   ('workbench_store_shopify_shops','shopify_current_authorization','FOREIGN KEY (organization_id, shop_key, generation) REFERENCES workbench_store_shopify_authorizations(organization_id, shop_key, generation) DEFERRABLE INITIALLY DEFERRED'),
   ('workbench_store_shopify_authorizations','shopify_authorization_pk','PRIMARY KEY (organization_id, shop_key, generation)'),
   ('workbench_store_shopify_pairings','shopify_pairing_pk','PRIMARY KEY (organization_id, attempt_id)'),
   ('workbench_store_shopify_lifecycle_events','shopify_lifecycle_pk','PRIMARY KEY (app_id, shop_id, revision)'),
   ('workbench_store_shopify_lifecycle_events','shopify_lifecycle_event','UNIQUE (app_id, shop_id, topic, event_id)')
 ) AS expected(table_name, constraint_name, definition)
 WHERE NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint c JOIN pg_catalog.pg_class r ON r.oid=c.conrelid
  JOIN pg_catalog.pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname='public' AND r.relname=expected.table_name AND r.relkind='r'
  AND NOT r.relrowsecurity AND r.relpersistence='p' AND c.conname=expected.constraint_name AND c.convalidated
  AND pg_catalog.pg_get_constraintdef(c.oid)=expected.definition)
)`).Scan(&ready)
	if err != nil || !ready {
		return errors.New("Store Shopify authorization schema unavailable")
	}
	return nil
}
