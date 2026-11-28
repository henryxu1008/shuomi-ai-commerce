package shopifyguard

import (
	"context"
	"strconv"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"gorm.io/gorm"
)

// NewIsolatedMediaFreeze binds revocation to the serving pool and one exact
// installation generation in the isolated Product execution ledger.
func NewIsolatedMediaFreeze(ctx context.Context, db *gorm.DB, live *LiveIsolation, organization, shopKey string, generation int64) (func(context.Context, string, string, int64) error, error) {
	if ctx == nil || !authidentity.IsBoundedIdentifier(organization) || !collection.ValidID(shopKey) || generation < 1 {
		return nil, ErrConflict
	}
	if err := VerifyLiveIsolation(db, live); err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil || pool == live.state.installer {
		return nil, ErrUnavailable
	}
	var mediaJournal bool
	err = db.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL AND to_regclass(?) IS NOT NULL", MediaSyncPlanTable, MediaSyncStepTable).
		Row().Scan(&mediaJournal)
	if err != nil || !mediaJournal {
		return nil, ErrUnavailable
	}
	command := Command{ID: collection.StableID("shopify-media-trial-revocation", organization, shopKey, strconv.FormatInt(generation, 10)),
		Organization: organization, ShopKey: shopKey, Generation: generation}
	return func(ctx context.Context, organization, shopKey string, generation int64) error {
		if ctx == nil || organization != command.Organization || shopKey != command.ShopKey || generation != command.Generation {
			return ErrConflict
		}
		if err := VerifyLiveIsolation(db.WithContext(ctx), live); err != nil {
			return err
		}
		_, err := SealIsolated(ctx, db, command)
		return err
	}, nil
}

// NewIsolatedMediaFreezeOnly reconstructs the uninstall freeze action after a
// process restart. It exposes no media journal or Shopify dispatch methods.
func NewIsolatedMediaFreezeOnly(ctx context.Context, db *gorm.DB, organization, shopKey string, generation int64) (func(context.Context, string, string, int64) error, error) {
	if ctx == nil || db == nil || db.Dialector.Name() != "postgres" || !authidentity.IsBoundedIdentifier(organization) || !collection.ValidID(shopKey) || generation < 1 {
		return nil, ErrConflict
	}
	var ready bool
	if err := db.WithContext(ctx).Raw("SELECT to_regclass(?) IS NOT NULL AND to_regclass(?) IS NOT NULL AND to_regclass('public.listing_shopify_frozen_generations') IS NOT NULL", MediaSyncPlanTable, MediaSyncStepTable).Row().Scan(&ready); err != nil || !ready {
		return nil, ErrUnavailable
	}
	command := Command{ID: collection.StableID("shopify-media-trial-revocation", organization, shopKey, strconv.FormatInt(generation, 10)),
		Organization: organization, ShopKey: shopKey, Generation: generation}
	return func(ctx context.Context, actualOrganization, actualShopKey string, actualGeneration int64) error {
		if ctx == nil || actualOrganization != command.Organization || actualShopKey != command.ShopKey || actualGeneration != command.Generation {
			return ErrConflict
		}
		_, err := SealIsolated(ctx, db, command)
		return err
	}, nil
}
