package officialpersistence

import (
	"context"
	"gorm.io/gorm"
	"strings"
	"task-processor/internal/listing/submission"
)

const WalmartSKUStateTable = "listing_walmart_sku_bindings"
const WalmartObservationTable = "listing_walmart_feed_observations"

func installWalmartSchema(db *gorm.DB) error {
	return db.Exec(`
CREATE TABLE IF NOT EXISTS listing_walmart_sku_bindings (
 environment varchar(16) NOT NULL CHECK(environment IN ('production','sandbox')),market varchar(2) NOT NULL CHECK(market='US'),merchant_identity_hash varchar(64) NOT NULL CHECK(merchant_identity_hash ~ '^[0-9a-f]{64}$'),seller_sku varchar(128) NOT NULL,
 organization_id varchar(128) NOT NULL,store_id uuid NOT NULL,product_key varchar(128) NOT NULL,source_sku_key varchar(512) NOT NULL,subject_id varchar(128) NOT NULL,first_intent_key varchar(128) NOT NULL,last_attempt_id uuid NOT NULL,
 read_generation bigint NOT NULL DEFAULT 0 CHECK(read_generation>=0),latest_observation_id varchar(36) NOT NULL DEFAULT '',
 PRIMARY KEY(environment,market,merchant_identity_hash,seller_sku),
 UNIQUE(environment,market,merchant_identity_hash,product_key,source_sku_key),
 CONSTRAINT walmart_sku_intent FOREIGN KEY(organization_id,first_intent_key) REFERENCES listing_submission_official_intents(organization_id,intent_key),
 CONSTRAINT walmart_sku_attempt FOREIGN KEY(organization_id,last_attempt_id) REFERENCES listing_submission_execution_attempts(organization_id,attempt_id) DEFERRABLE INITIALLY DEFERRED,
 CHECK(seller_sku<>'' AND source_sku_key<>'' AND product_key<>'' AND subject_id LIKE 'walmart-offer-us-%')
);
CREATE TABLE IF NOT EXISTS listing_walmart_feed_observations (
 organization_id varchar(128) NOT NULL,id uuid NOT NULL,receipt_id uuid NOT NULL,generation bigint NOT NULL CHECK(generation>0),body_json jsonb NOT NULL,body_hash varchar(64) NOT NULL CHECK(body_hash ~ '^[0-9a-f]{64}$'),observed_at timestamptz NOT NULL,
 PRIMARY KEY(organization_id,id),UNIQUE(organization_id,receipt_id,generation),
 CONSTRAINT walmart_observation_receipt FOREIGN KEY(organization_id,receipt_id) REFERENCES listing_submission_official_receipts(organization_id,attempt_id),
 CHECK(octet_length(body_json::text)<=2097152 AND jsonb_typeof(body_json)='object')
);`).Error
}

// Exact PostgreSQL 16 contract: same-name weakened constraints are rejected.
var walmartSchemaConstraints = []struct{ table, name, definition string }{
	{"listing_walmart_feed_observations", "listing_walmart_feed_observat_organization_id_receipt_id_ge_key", `UNIQUE (organization_id, receipt_id, generation)`},
	{"listing_walmart_feed_observations", "listing_walmart_feed_observations_body_hash_check", `CHECK (((body_hash)::text ~ '^[0-9a-f]{64}$'::text))`},
	{"listing_walmart_feed_observations", "listing_walmart_feed_observations_body_json_check", `CHECK (((octet_length((body_json)::text) <= 2097152) AND (jsonb_typeof(body_json) = 'object'::text)))`},
	{"listing_walmart_feed_observations", "listing_walmart_feed_observations_generation_check", `CHECK ((generation > 0))`},
	{"listing_walmart_feed_observations", "listing_walmart_feed_observations_pkey", `PRIMARY KEY (organization_id, id)`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_check", `CHECK ((((seller_sku)::text <> ''::text) AND ((source_sku_key)::text <> ''::text) AND ((product_key)::text <> ''::text) AND ((subject_id)::text ~~ 'walmart-offer-us-%'::text)))`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_environment_check", `CHECK (((environment)::text = ANY ((ARRAY['production'::character varying, 'sandbox'::character varying])::text[])))`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_environment_market_merchant_id_key", `UNIQUE (environment, market, merchant_identity_hash, product_key, source_sku_key)`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_market_check", `CHECK (((market)::text = 'US'::text))`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_merchant_identity_hash_check", `CHECK (((merchant_identity_hash)::text ~ '^[0-9a-f]{64}$'::text))`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_pkey", `PRIMARY KEY (environment, market, merchant_identity_hash, seller_sku)`},
	{"listing_walmart_sku_bindings", "listing_walmart_sku_bindings_read_generation_check", `CHECK ((read_generation >= 0))`},
	{"listing_walmart_feed_observations", "walmart_observation_receipt", `FOREIGN KEY (organization_id, receipt_id) REFERENCES listing_submission_official_receipts(organization_id, attempt_id)`},
	{"listing_walmart_sku_bindings", "walmart_sku_attempt", `FOREIGN KEY (organization_id, last_attempt_id) REFERENCES listing_submission_execution_attempts(organization_id, attempt_id) DEFERRABLE INITIALLY DEFERRED`},
	{"listing_walmart_sku_bindings", "walmart_sku_intent", `FOREIGN KEY (organization_id, first_intent_key) REFERENCES listing_submission_official_intents(organization_id, intent_key)`},
}

func verifyWalmartSchema(ctx context.Context, db *gorm.DB) error {
	for _, c := range walmartSchemaConstraints {
		var definition string
		if e := db.WithContext(ctx).Raw(`SELECT pg_get_constraintdef(oid) FROM pg_catalog.pg_constraint WHERE conrelid=to_regclass(?) AND conname=? AND convalidated`, c.table, c.name).Row().Scan(&definition); e != nil || strings.Join(strings.Fields(definition), " ") != c.definition {
			return submission.ErrExecutionUnavailable
		}
	}

	for _, q := range []string{"SELECT environment,market,merchant_identity_hash,seller_sku,organization_id,store_id,product_key,source_sku_key,subject_id,first_intent_key,last_attempt_id,read_generation,latest_observation_id FROM listing_walmart_sku_bindings LIMIT 0", "SELECT organization_id,id,receipt_id,generation,body_json,body_hash,observed_at FROM listing_walmart_feed_observations LIMIT 0"} {
		if db.WithContext(ctx).Exec(q).Error != nil {
			return submission.ErrExecutionUnavailable
		}
	}
	var valid bool
	if db.WithContext(ctx).Raw(`SELECT count(*)=3 FROM pg_constraint WHERE convalidated AND conname IN ('walmart_sku_intent','walmart_sku_attempt','walmart_observation_receipt') AND contype='f' AND conrelid IN ('listing_walmart_sku_bindings'::regclass,'listing_walmart_feed_observations'::regclass)`).Scan(&valid).Error != nil || !valid {
		return submission.ErrExecutionUnavailable
	}
	return nil
}
