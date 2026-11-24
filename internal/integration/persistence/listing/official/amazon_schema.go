package officialpersistence

import (
	"context"
	"database/sql"

	submissionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
)

// InstallIsolatedAmazonSchemaTx is used only in explicitly selected synthetic
// Product databases. The default Product schema and serving mode omit it.
func InstallIsolatedAmazonSchemaTx(ctx context.Context, tx *sql.Tx) error {
	if tx == nil {
		return submission.ErrExecutionUnavailable
	}
	for _, statement := range []string{
		`ALTER TABLE public.listing_target_records DROP CONSTRAINT listing_target_records_sites_v2,
			ADD CONSTRAINT listing_target_records_sites_v2 CHECK(site IN ('shein-us','temu-supplier','temu-local','tiktokshop-us','walmart-us','shopify','amazon-us'))`,
		`ALTER TABLE public.listing_preparation_targets DROP CONSTRAINT listing_preparation_targets_sites_v2,
			ADD CONSTRAINT listing_preparation_targets_sites_v2 CHECK(site IN ('shein-us','temu-supplier','temu-local','tiktokshop-us','walmart-us','shopify','amazon-us'))`,
		`CREATE TABLE public.listing_amazon_sku_occupancy (
			environment text NOT NULL CHECK (environment IN ('production','beta')),
			marketplace text NOT NULL CHECK (marketplace='ATVPDKIKX0DER'),
			seller_partition text NOT NULL CHECK (seller_partition ~ '^[0-9a-f]{64}$'),
			identity_kind text NOT NULL,
			identity_value text NOT NULL CHECK ((identity_kind='sku' AND identity_value ~ '^shuomi-[0-9a-f]{32}$') OR (identity_kind='gtin' AND identity_value ~ '^[0-9]{14}$')),
			organization_id text NOT NULL, store_id text NOT NULL, product_key text NOT NULL,
			PRIMARY KEY(environment,marketplace,seller_partition,identity_kind,identity_value)
		)`,
		`CREATE TABLE public.listing_amazon_receipts (
			organization_id varchar(128) NOT NULL, attempt_id uuid NOT NULL,
			body jsonb NOT NULL CHECK (jsonb_typeof(body)='object' AND octet_length(body::text)<=32768),
			PRIMARY KEY(organization_id,attempt_id),
			FOREIGN KEY(organization_id,attempt_id) REFERENCES public.listing_submission_execution_attempts(organization_id,attempt_id)
		)`,
		`CREATE TABLE public.listing_amazon_observations (
			organization_id varchar(128) NOT NULL, attempt_id uuid NOT NULL, observed_at timestamptz NOT NULL,
			body jsonb NOT NULL CHECK (jsonb_typeof(body)='object' AND octet_length(body::text)<=32768),
			PRIMARY KEY(organization_id,attempt_id),
			FOREIGN KEY(organization_id,attempt_id) REFERENCES public.listing_submission_execution_attempts(organization_id,attempt_id)
		)`,
	} {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return submissionstore.InstallIsolatedAmazonNoSendSchemaTx(ctx, tx)
}
