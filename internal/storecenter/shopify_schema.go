package storecenter

import (
	"context"
	"database/sql"
	"errors"
)

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
}

func installShopifySchemaTx(ctx context.Context, tx *sql.Tx) error {
	for _, statement := range shopifySchemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
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
   ('workbench_store_shopify_pairings','shopify_pairing_pk','PRIMARY KEY (organization_id, attempt_id)')
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
