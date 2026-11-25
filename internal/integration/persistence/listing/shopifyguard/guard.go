// Package shopifyguard holds the Product owner's Shopify generation and create guards.
package shopifyguard

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"gorm.io/gorm"
)

const (
	requiredSetting = "listing:shopify-isolated-privacy-required"
	commandTable    = "public.listing_shopify_frozen_generations"
	sourceTable     = "public.listing_shopify_restricted_sources"
)

var (
	ErrFrozen      = errors.New("Shopify authorization generation frozen")
	ErrRestricted  = errors.New("Shopify source requires verification")
	ErrUnavailable = errors.New("Shopify generation protection unavailable")
	ErrConflict    = errors.New("Shopify protection command conflicts with original")
)

// RequireIsolated pins the guard requirement for one borrowed test connection.
func RequireIsolated(db *gorm.DB) *gorm.DB {
	return db.Set(requiredSetting, true).Session(&gorm.Session{})
}

// ProjectionEnabled checks the guard capability before building a guarded read query.
func ProjectionEnabled(db *gorm.DB) (bool, error) { return installed(db) }

func installed(db *gorm.DB) (bool, error) {
	var commands, sources, dispatches bool
	err := db.Raw(`SELECT to_regclass('public.listing_shopify_frozen_generations') IS NOT NULL,
		to_regclass('public.listing_shopify_restricted_sources') IS NOT NULL,
		to_regclass('public.listing_shopify_dispatches') IS NOT NULL`).Row().Scan(&commands, &sources, &dispatches)
	_, required := db.Get(requiredSetting)
	if err != nil || commands != sources || commands != dispatches || required && !commands {
		return false, ErrUnavailable
	}
	return commands, nil
}

// Check takes the generation lock and, for a new send, the stable source lock.
// The caller keeps its existing Product-transaction boundary until commit.
func Check(tx *gorm.DB, org, shop string, generation int64, source string) error {
	enabled, err := installed(tx)
	if err != nil || !enabled {
		return err
	}
	if err := lockGeneration(tx, org, shop, generation); err != nil {
		return err
	}
	if err := checkGeneration(tx, org, shop, generation); err != nil || source == "" {
		return err
	}
	if err := lock(tx, "source", org, source); err != nil {
		return err
	}
	var restricted bool
	if err := tx.Raw("SELECT EXISTS (SELECT 1 FROM "+sourceTable+" WHERE organization_id=? AND source_key=?)", org, source).Row().Scan(&restricted); err != nil {
		return ErrUnavailable
	}
	if restricted {
		return ErrRestricted
	}
	return nil
}

// Visible performs a fresh read check without taking a lock in the reverse
// order when an existing reader already holds a Product row lock.
func Visible(db *gorm.DB, org, shop string, generation int64) error {
	enabled, err := installed(db)
	if err != nil || !enabled {
		return err
	}
	return checkGeneration(db, org, shop, generation)
}

// CheckRecordReference gates a projection using its immutable Product record
// scope. A missing record cannot authorize restoring a purged reference.
func CheckRecordReference(db *gorm.DB, org, actor, member, id string, write bool) error {
	enabled, supportErr := installed(db)
	if supportErr == nil && !enabled {
		return nil
	}
	var site, shop string
	var generation int64
	err := db.Raw("SELECT site, coalesce(body_json#>>'{merchant,shopify,shopKey}',''), "+
		"coalesce(body_json#>>'{merchant,connection_revision}','0')::bigint "+
		"FROM public.listing_target_records WHERE organization_id=? AND actor_id=? AND member_id=? AND id=?", org, actor, member, id).
		Row().Scan(&site, &shop, &generation)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrFrozen
	}
	if err != nil {
		return ErrUnavailable
	}
	if site != "shopify" {
		return nil
	}
	if supportErr != nil {
		return supportErr
	}
	if _, err := uuid.Parse(shop); err != nil || generation < 1 {
		return ErrUnavailable
	}
	if write {
		if err := lockGeneration(db, org, shop, generation); err != nil {
			return err
		}
	}
	return checkGeneration(db, org, shop, generation)
}

func checkGeneration(db *gorm.DB, org, shop string, generation int64) error {
	var frozen bool
	if err := db.Raw("SELECT EXISTS (SELECT 1 FROM "+commandTable+" WHERE organization_id=? AND shop_key=? AND generation=?)", org, shop, generation).Row().Scan(&frozen); err != nil {
		return ErrUnavailable
	}
	if frozen {
		return ErrFrozen
	}
	return nil
}

func lockGeneration(tx *gorm.DB, org, shop string, generation int64) error {
	if !canonicalUUID(shop) || generation < 1 {
		return ErrUnavailable
	}
	return lock(tx, "generation", org, shop, strconv.FormatInt(generation, 10))
}

func lock(tx *gorm.DB, parts ...string) error {
	if _, open := tx.Statement.ConnPool.(gorm.TxCommitter); !open {
		return ErrUnavailable
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", "listing-shopify-privacy:"+collection.Digest(parts)).Error; err != nil {
		return ErrUnavailable
	}
	return nil
}

// InstallIsolated is used only by disposable verification. No normal owner
// initializer or serving role is given permission to freeze or clear data.
func InstallIsolated(ctx context.Context, db *gorm.DB) (*LiveIsolation, error) {
	pool, err := db.DB()
	if err != nil {
		return nil, ErrUnavailable
	}
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE public.listing_submission_execution_attempts IN SHARE ROW EXCLUSIVE MODE").Error; err != nil {
			return ErrUnavailable
		}
		var prior bool
		if err := tx.Raw("SELECT EXISTS(SELECT 1 FROM public.listing_submission_execution_attempts WHERE platform='shopify')").Row().Scan(&prior); err != nil {
			return ErrUnavailable
		}
		if prior {
			return ErrConflict
		}
		return tx.Exec(`CREATE TABLE public.listing_shopify_frozen_generations (
			id uuid PRIMARY KEY, organization_id text NOT NULL, shop_key uuid NOT NULL, generation bigint NOT NULL CHECK(generation>0),
			source_keys jsonb NOT NULL CHECK(jsonb_typeof(source_keys)='array'), sealed_locally boolean NOT NULL,
			UNIQUE(organization_id,shop_key,generation));
			CREATE TABLE public.listing_shopify_restricted_sources (
			organization_id text NOT NULL, source_key text NOT NULL, PRIMARY KEY(organization_id,source_key));
			CREATE TABLE public.listing_shopify_dispatches (
			organization_id text NOT NULL, shop_key uuid NOT NULL, generation bigint NOT NULL CHECK(generation>0),
			attempt_id uuid NOT NULL, epoch bigint NOT NULL CHECK(epoch>0), response_observed boolean NOT NULL DEFAULT false,
			PRIMARY KEY(organization_id,attempt_id));
			CREATE INDEX listing_shopify_dispatches_open ON public.listing_shopify_dispatches(organization_id,shop_key,generation)
			WHERE NOT response_observed;`).Error
	})
	if err != nil {
		return nil, err
	}
	return newLiveIsolation(pool), nil
}

// Command uses only local owner identifiers. Installation attribution is a
// Store responsibility. The isolated writer never receives a webhook.
type Command struct {
	ID, Organization, ShopKey string
	Generation                int64
	Sources                   []string
}

// SealIsolated records the exact guarded generation and captures potentially
// sent sources from the current Product owner in the same transaction.
func SealIsolated(ctx context.Context, db *gorm.DB, command Command) (Command, error) {
	return apply(ctx, db, command, false)
}

// ReplayIsolated receives the already sealed command from the isolated
// restore authority, including sources absent from an older Product image.
func ReplayIsolated(ctx context.Context, db *gorm.DB, command Command) error {
	_, err := apply(ctx, db, command, true)
	return err
}

func canonicalUUID(value string) bool {
	parsed, err := uuid.Parse(value)
	return err == nil && parsed.String() == value
}

func apply(ctx context.Context, db *gorm.DB, command Command, replay bool) (Command, error) {
	if !canonicalUUID(command.ID) || !canonicalUUID(command.ShopKey) || !authidentity.IsBoundedIdentifier(command.Organization) || command.Generation < 1 {
		return Command{}, ErrConflict
	}
	err := RequireIsolated(db).WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := installed(tx); err != nil {
			return err
		}
		if err := lockGeneration(tx, command.Organization, command.ShopKey, command.Generation); err != nil {
			return err
		}
		prior, err := readCommand(tx, command.ID)
		if err == nil {
			if prior.Organization != command.Organization || prior.ShopKey != command.ShopKey || prior.Generation != command.Generation || replay && !slices.Equal(prior.Sources, command.Sources) {
				return ErrConflict
			}
			command = prior
			return nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return ErrUnavailable
		}
		if !replay {
			command.Sources, err = captureSources(tx, command)
			if err != nil {
				return err
			}
		}
		return insertCommand(tx, command, !replay)
	})
	if err != nil {
		return Command{}, err
	}
	return command, nil
}

func captureSources(tx *gorm.DB, command Command) ([]string, error) {
	var attempts []struct{ Shop, Revision, Source string }
	err := tx.Raw(`SELECT DISTINCT
			coalesce(i.body_json#>>'{binding,shopify,shopKey}','') AS shop,
			coalesce(i.body_json#>>'{binding,connection_revision}','') AS revision,
			coalesce(i.body_json#>>'{source,source,productKey}','') AS source
		FROM public.listing_submission_execution_attempts a
		LEFT JOIN public.listing_submission_official_intents i ON i.organization_id=a.organization_id AND i.intent_key=a.intent_key
		WHERE a.organization_id=? AND a.platform='shopify' AND a.store_id=? AND a.action='create_shopify_draft'
		  AND a.status IN ('claimed','outcome_unknown','succeeded') ORDER BY source`, command.Organization, command.ShopKey).Scan(&attempts).Error
	if err != nil {
		return nil, ErrUnavailable
	}
	sources := []string{}
	for _, attempt := range attempts {
		revision, err := strconv.ParseInt(attempt.Revision, 10, 64)
		if err != nil || revision < 1 || attempt.Shop != command.ShopKey || !authidentity.IsBoundedIdentifier(attempt.Source) {
			return nil, ErrUnavailable
		}
		if revision == command.Generation {
			sources = append(sources, attempt.Source)
		}
	}
	var mediaPlans bool
	if err := tx.Raw("SELECT to_regclass(?) IS NOT NULL", MediaSyncPlanTable).Row().Scan(&mediaPlans); err != nil {
		return nil, ErrUnavailable
	}
	if mediaPlans {
		var mediaSources []string
		err := tx.Raw(`SELECT DISTINCT p.source_key FROM `+MediaSyncPlanTable+` p
			JOIN `+MediaSyncStepTable+` s USING (organization_id,plan_id)
			JOIN `+dispatchTable+` d USING (organization_id,attempt_id)
			WHERE p.organization_id=? AND p.shop_key=? AND p.generation=? AND d.shop_key=p.shop_key AND d.generation=p.generation`,
			command.Organization, command.ShopKey, command.Generation).Scan(&mediaSources).Error
		if err != nil {
			return nil, ErrUnavailable
		}
		sources = append(sources, mediaSources...)
	}
	slices.Sort(sources)
	return slices.Compact(sources), nil
}

func readCommand(tx *gorm.DB, id string) (Command, error) {
	command := Command{ID: id}
	var sources []byte
	err := tx.Raw("SELECT organization_id,shop_key::text,generation,source_keys FROM "+commandTable+" WHERE id=?", id).
		Row().Scan(&command.Organization, &command.ShopKey, &command.Generation, &sources)
	if err != nil {
		return Command{}, err
	}
	if json.Unmarshal(sources, &command.Sources) != nil {
		return Command{}, ErrUnavailable
	}
	return command, nil
}

func insertCommand(tx *gorm.DB, command Command, locallySealed bool) error {
	if command.Sources == nil || !slices.IsSorted(command.Sources) {
		return ErrConflict
	}
	for i, source := range command.Sources {
		if !authidentity.IsBoundedIdentifier(source) || i > 0 && source == command.Sources[i-1] {
			return ErrConflict
		}
		if err := lock(tx, "source", command.Organization, source); err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO "+sourceTable+" (organization_id,source_key) VALUES (?,?) ON CONFLICT DO NOTHING", command.Organization, source).Error; err != nil {
			return ErrUnavailable
		}
	}
	raw, err := json.Marshal(command.Sources)
	if err != nil {
		return ErrUnavailable
	}
	res := tx.Exec("INSERT INTO "+commandTable+" (id,organization_id,shop_key,generation,source_keys,sealed_locally) VALUES (?,?,?,?,?::jsonb,?) ON CONFLICT DO NOTHING",
		command.ID, command.Organization, command.ShopKey, command.Generation, string(raw), locallySealed)
	if res.Error != nil {
		return ErrUnavailable
	}
	if res.RowsAffected != 1 {
		return ErrConflict
	}
	return nil
}
