package submissionpersistence_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	official "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/official"
	prepstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/preparation"
	recordstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/shopifyguard"
	effects "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	catalogstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/catalog"
	collectionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestIsolatedShopifyMediaServingPoolHandoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("shopify_media"),
		tcpostgres.WithUsername("fixture"), tcpostgres.WithPassword("isolated-fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable", "search_path=public")
	require.NoError(t, err)
	openDatabase := func(dsn string) *gorm.DB {
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		require.NoError(t, err)
		pool, err := db.DB()
		require.NoError(t, err)
		t.Cleanup(func() { _ = pool.Close() })
		return db
	}
	install := func(owner *gorm.DB) *shopifyguard.LiveIsolation {
		require.NoError(t, effects.InstallSchema(owner))
		require.NoError(t, owner.Exec("CREATE TABLE public.listing_submission_official_intents (organization_id text, intent_key text, body_json jsonb)").Error)
		isolation, err := shopifyguard.InstallIsolated(ctx, owner)
		require.NoError(t, err)
		require.NoError(t, effects.InstallIsolatedShopifyMediaSchema(ctx, owner, isolation))
		return isolation
	}
	const grants = `
		GRANT SELECT, INSERT, UPDATE ON public.listing_submission_execution_attempts,public.listing_submission_target_fences,
			public.listing_shopify_dispatches,public.listing_shopify_media_sync_plans TO shopify_media_serve;
		GRANT SELECT,INSERT ON public.listing_shopify_media_sync_steps,public.listing_shopify_frozen_generations,
			public.listing_shopify_restricted_sources TO shopify_media_serve;
		GRANT SELECT ON public.listing_submission_official_intents TO shopify_media_serve;
	`
	admin := openDatabase(dsn)
	isolation := install(admin)
	require.NoError(t, admin.Exec("CREATE ROLE shopify_media_serve LOGIN PASSWORD 'isolated-serving-fixture'").Error)
	require.NoError(t, admin.Exec("REVOKE TEMPORARY ON DATABASE shopify_media FROM PUBLIC").Error)
	require.NoError(t, admin.Exec(grants).Error)
	require.NoError(t, admin.Exec("CREATE DATABASE shopify_media_other TEMPLATE shopify_media").Error)
	require.NoError(t, admin.Exec("CREATE DATABASE shopify_media_pinned").Error)
	servingURL, err := url.Parse(dsn)
	require.NoError(t, err)
	pinnedAdminURL := *servingURL
	pinnedAdminURL.Path = "/shopify_media_pinned"
	pinned := openDatabase(pinnedAdminURL.String())
	pinnedIsolation := install(pinned)
	require.NoError(t, pinned.Exec("REVOKE TEMPORARY ON DATABASE shopify_media_pinned FROM PUBLIC").Error)
	require.NoError(t, pinned.Exec(grants).Error)
	servingURL.User = url.UserPassword("shopify_media_serve", "isolated-serving-fixture")
	pinnedServingURL := *servingURL
	pinnedServingURL.Path = pinnedAdminURL.Path
	pinnedServing := openDatabase(pinnedServingURL.String())
	_, err = effects.NewIsolatedShopifyMediaJournal(ctx, pinned, pinnedIsolation)
	require.NoError(t, err)
	require.Error(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, pinned, pinnedServing, pinnedIsolation),
		"do not transfer a witness once a journal exists on the installer pool")
	require.NoError(t, shopifyguard.VerifyLiveIsolation(pinned, pinnedIsolation))

	serving := openDatabase(servingURL.String())
	_, err = effects.NewIsolatedShopifyMediaJournal(ctx, serving, isolation)
	require.Error(t, err)
	require.Error(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, admin, openDatabase(dsn), isolation))
	otherURL := *servingURL
	otherURL.Path = "/shopify_media_other"
	other := openDatabase(otherURL.String())
	require.NoError(t, admin.Exec("REVOKE TEMPORARY ON DATABASE shopify_media_other FROM PUBLIC").Error)
	require.Error(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, admin, other, isolation))
	require.NoError(t, admin.Exec("GRANT DELETE ON public.listing_shopify_media_sync_plans TO shopify_media_serve").Error)
	require.Error(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, admin, serving, isolation))
	require.NoError(t, admin.Exec("REVOKE DELETE ON public.listing_shopify_media_sync_plans FROM shopify_media_serve").Error)
	require.NoError(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, admin, serving, isolation))
	require.Error(t, shopifyguard.HandoffIsolatedMediaServingPool(ctx, admin, serving, isolation))
	require.Error(t, shopifyguard.VerifyLiveIsolation(admin, isolation))
	journal, err := effects.NewIsolatedShopifyMediaJournal(ctx, serving, isolation)
	require.NoError(t, err)

	plan := fixtureMediaPlan("serving-handoff")
	shop := fixtureMediaProvider(plan)
	shop.onAppend = func() {
		freeze, err := shopifyguard.SealIsolated(ctx, serving, shopifyguard.Command{ID: uuid.NewString(),
			Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation})
		require.NoError(t, err)
		drained, err := shopifyguard.DrainReadyIsolated(ctx, serving, isolation, freeze)
		require.NoError(t, err)
		require.False(t, drained)
	}
	service := submission.ShopifyMediaSync{Journal: journal, Provider: shop}
	_, err = service.Confirm(ctx, plan)
	require.Error(t, err)
	require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
	require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
	var commandID string
	require.NoError(t, serving.Raw("SELECT id FROM public.listing_shopify_frozen_generations WHERE organization_id=?", plan.Owner.OrganizationID).Row().Scan(&commandID))
	drained, err := shopifyguard.DrainReadyIsolated(ctx, serving, isolation, shopifyguard.Command{ID: commandID,
		Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation, Sources: []string{plan.SourceKey}})
	require.NoError(t, err)
	require.True(t, drained)
}

func TestIsolatedShopifyMediaSync(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("shopify_media"),
		tcpostgres.WithUsername("fixture"), tcpostgres.WithPassword("isolated-fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema,
		recordstore.InstallSchema, effects.InstallSchema, official.InstallOfficialSchema} {
		require.NoError(t, install(db))
	}
	_, err = effects.NewIsolatedShopifyMediaJournal(ctx, db, nil)
	require.Error(t, err)
	isolation, err := shopifyguard.InstallIsolated(ctx, db)
	require.NoError(t, err)
	require.NoError(t, effects.InstallIsolatedShopifyMediaSchema(ctx, db, isolation))
	open := func() submission.ShopifyMediaJournal {
		repo, err := effects.NewIsolatedShopifyMediaJournal(ctx, db, isolation)
		require.NoError(t, err)
		return repo
	}

	t.Run("adds missing originals and changes only the intended six variants", func(t *testing.T) {
		plan := fixtureMediaPlan("normal")
		shop := fixtureMediaProvider(plan)
		shop.processing = true
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		result, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, result)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
		shop.processing = false
		// A fresh service has only the retained intent and response proof.
		service.Journal = open()
		result, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaComplete, result)
		require.Equal(t, shop.desiredCovers(), shop.covers)
		require.Len(t, shop.gallery, 7)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaPrepare))
		require.Equal(t, 3, shop.count(submission.ShopifyMediaUpload))
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAssociate))
		result, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaComplete, result)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAssociate))
		changed := plan
		changed.ApprovalHash = collection.Digest("other approval")
		_, err = service.Confirm(ctx, changed)
		require.ErrorIs(t, err, submission.ErrExecutionIntentConflict)
	})

	t.Run("repeat confirmation reuses the frozen plan after the gallery changes", func(t *testing.T) {
		plan := fixtureMediaPlan("repeat-operation")
		shop := fixtureMediaProvider(plan)
		shop.processing = true
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		operation := plan.Operation()
		builds := 0
		build := func(context.Context) (submission.ShopifyMediaPlan, error) {
			builds++
			return plan, nil
		}
		status, err := service.ConfirmOperation(ctx, operation, build)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, status)
		require.Len(t, shop.gallery, 7)

		shop.processing = false
		service.Journal = open()
		plan.BeforeHash = collection.Digest("gallery after append")
		wrongMember := operation
		wrongMember.Owner.MemberID = "another-member"
		_, err = service.ConfirmOperation(ctx, wrongMember, build)
		require.ErrorIs(t, err, submission.ErrExecutionIntentConflict)
		wrongSource := operation
		wrongSource.SourceKey = "another-source"
		_, err = service.ConfirmOperation(ctx, wrongSource, build)
		require.ErrorIs(t, err, submission.ErrExecutionIntentConflict)
		wrongProduct := operation
		wrongProduct.ProductID = "gid://shopify/Product/45"
		_, err = service.ConfirmOperation(ctx, wrongProduct, build)
		require.ErrorIs(t, err, submission.ErrExecutionIntentConflict)
		require.Equal(t, 1, builds)

		shop.denied = true
		_, err = service.ConfirmOperation(ctx, operation, build)
		require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
		shop.denied = false

		for range 2 {
			status, err = service.ConfirmOperation(ctx, operation, build)
			require.NoError(t, err)
			require.Equal(t, submission.ShopifyMediaComplete, status)
		}
		for _, id := range plan.GalleryIDs {
			require.Contains(t, shop.gallery, id)
		}
		require.Equal(t, shop.desiredCovers(), shop.covers)
		require.Len(t, shop.gallery, 7)
		require.Equal(t, 1, builds)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaPrepare))
		require.Equal(t, 3, shop.count(submission.ShopifyMediaUpload))
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAssociate))

		require.NoError(t, db.Exec("UPDATE "+shopifyguard.MediaSyncPlanTable+" SET body_hash=? WHERE organization_id=? AND plan_id=?",
			collection.Digest("invalid stored fingerprint"), plan.Owner.OrganizationID, plan.ID).Error)
		_, err = service.ConfirmOperation(ctx, operation, build)
		require.ErrorIs(t, err, submission.ErrExecutionUnavailable)
		require.Equal(t, 1, builds)
	})

	t.Run("rejects a gallery beyond the verification limit before staging", func(t *testing.T) {
		plan := fixtureMediaPlan("gallery-limit")
		for i := len(plan.GalleryIDs) + 1; i <= 57; i++ {
			plan.GalleryIDs = append(plan.GalleryIDs, fmt.Sprintf("gid://shopify/MediaImage/%d", i))
		}
		require.True(t, plan.Valid())
		plan.GalleryIDs = append(plan.GalleryIDs, "gid://shopify/MediaImage/58")
		shop := fixtureMediaProvider(plan)
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		_, err := service.Confirm(ctx, plan)
		require.ErrorIs(t, err, submission.ErrExecutionInvalid)
		require.Empty(t, shop.calls)
	})

	t.Run("lost append response cannot authorize duplicate transfer or drain", func(t *testing.T) {
		plan := fixtureMediaPlan("lost")
		shop := fixtureMediaProvider(plan)
		shop.lose = submission.ShopifyMediaAppend
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		result, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaUncertain, result)
		require.Len(t, shop.gallery, 7, "provider performed the append before the response was lost")
		service.Journal = open()
		result, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaUncertain, result)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
		var attempts int64
		require.NoError(t, db.Table(effects.AttemptTable).Where("organization_id=? AND action=? AND status=?",
			plan.Owner.OrganizationID, submission.ShopifyMediaAppend, submission.ExecutionOutcomeUnknown).Count(&attempts).Error)
		require.EqualValues(t, 1, attempts)
		differentKey := plan
		differentKey.ID = uuid.NewString()
		_, err = service.Confirm(ctx, differentKey)
		require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
		freeze, err := shopifyguard.SealIsolated(ctx, db, shopifyguard.Command{ID: uuid.NewString(),
			Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation})
		require.NoError(t, err)
		require.Contains(t, freeze.Sources, plan.SourceKey)
		drained, err := shopifyguard.DrainReadyIsolated(ctx, db, isolation, freeze)
		require.NoError(t, err)
		require.False(t, drained)
	})

	t.Run("lost variant response remains uncertain after the covers appear", func(t *testing.T) {
		plan := fixtureMediaPlan("lost-variant")
		shop := fixtureMediaProvider(plan)
		shop.lose = submission.ShopifyMediaAssociate
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		result, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaUncertain, result)
		require.Equal(t, shop.desiredCovers(), shop.covers)
		result, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaUncertain, result)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAssociate))
	})

	t.Run("a change to the selected source is checked before sending", func(t *testing.T) {
		plan := fixtureMediaPlan("changed")
		shop := fixtureMediaProvider(plan)
		shop.denied = true
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		_, err := service.Confirm(ctx, plan)
		require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
		require.Empty(t, shop.calls)
	})

	t.Run("restages only if every old staging request returned and append was never sent", func(t *testing.T) {
		plan := fixtureMediaPlan("restage")
		shop := fixtureMediaProvider(plan)
		shop.stopAfterStaging = true
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		result, err := service.Confirm(ctx, plan)
		require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
		require.Equal(t, submission.ShopifyMediaPending, result)
		require.Zero(t, shop.count(submission.ShopifyMediaAppend))
		service.Journal = open()
		result, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaComplete, result)
		require.Equal(t, 2, shop.count(submission.ShopifyMediaPrepare))
		require.Equal(t, 6, shop.count(submission.ShopifyMediaUpload))
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
	})

	t.Run("a repeated confirmation cannot send while the first is in flight", func(t *testing.T) {
		plan := fixtureMediaPlan("concurrent")
		shop := fixtureMediaProvider(plan)
		entered, resume := make(chan struct{}), make(chan struct{})
		shop.onAppend = func() {
			close(entered)
			select {
			case <-resume:
			case <-ctx.Done():
			}
		}
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		type reply struct {
			status submission.ShopifyMediaStatus
			err    error
		}
		finished := make(chan reply, 1)
		go func() {
			status, err := service.Confirm(ctx, plan)
			finished <- reply{status, err}
		}()
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		status, err := service.Confirm(ctx, plan)
		close(resume)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, status)
		original := <-finished
		require.NoError(t, original.err)
		require.Equal(t, submission.ShopifyMediaComplete, original.status)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
	})

	t.Run("freeze permits only the original reply to drain", func(t *testing.T) {
		plan := fixtureMediaPlan("frozen")
		shop := fixtureMediaProvider(plan)
		var frozen shopifyguard.Command
		shop.onAppend = func() {
			var err error
			frozen, err = shopifyguard.SealIsolated(ctx, db, shopifyguard.Command{
				ID: uuid.NewString(), Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation})
			require.NoError(t, err)
		}
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		status, err := service.Confirm(ctx, plan)
		require.ErrorIs(t, err, shopifyguard.ErrFrozen)
		require.Equal(t, submission.ShopifyMediaUncertain, status)
		drained, err := shopifyguard.DrainReadyIsolated(ctx, db, isolation, frozen)
		require.NoError(t, err)
		require.True(t, drained)
		_, err = service.Confirm(ctx, plan)
		require.ErrorIs(t, err, shopifyguard.ErrFrozen)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
	})

	t.Run("freeze before finalizing suppresses a stale success", func(t *testing.T) {
		plan := fixtureMediaPlan("freeze-final-read")
		shop := fixtureMediaProvider(plan)
		shop.onRead = func() {
			if len(shop.calls) > 0 && shop.calls[len(shop.calls)-1] == submission.ShopifyMediaAssociate {
				_, err := shopifyguard.SealIsolated(ctx, db, shopifyguard.Command{
					ID: uuid.NewString(), Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation})
				require.NoError(t, err)
			}
		}
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		status, err := service.Confirm(ctx, plan)
		require.ErrorIs(t, err, shopifyguard.ErrFrozen)
		require.Equal(t, submission.ShopifyMediaUncertain, status)
		require.Equal(t, shop.desiredCovers(), shop.covers)
		var retained string
		require.NoError(t, db.Raw("SELECT status FROM "+shopifyguard.MediaSyncPlanTable+" WHERE organization_id=? AND plan_id=?", plan.Owner.OrganizationID, plan.ID).Row().Scan(&retained))
		require.NotEqual(t, string(submission.ShopifyMediaComplete), retained)
	})

	t.Run("an intervening variant edit is preserved", func(t *testing.T) {
		plan := fixtureMediaPlan("variant-edit")
		shop := fixtureMediaProvider(plan)
		variant := plan.Covers[0].VariantID
		shop.onAppend = func() { shop.covers[variant] = "gid://shopify/MediaImage/2" }
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		status, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, status)
		require.Equal(t, "gid://shopify/MediaImage/2", shop.covers[variant])
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
	})

	t.Run("an extra unverified append candidate blocks association", func(t *testing.T) {
		plan := fixtureMediaPlan("uncertain-original")
		shop := fixtureMediaProvider(plan)
		shop.extraCandidate = true
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		status, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, status)
		status, err = service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaPending, status)
		require.Equal(t, 1, shop.count(submission.ShopifyMediaAppend))
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
	})

	t.Run("a full rejection response drains without recording a gallery change", func(t *testing.T) {
		plan := fixtureMediaPlan("reject")
		shop := fixtureMediaProvider(plan)
		shop.reject = submission.ShopifyMediaAppend
		service := submission.ShopifyMediaSync{Journal: open(), Provider: shop}
		status, err := service.Confirm(ctx, plan)
		require.NoError(t, err)
		require.Equal(t, submission.ShopifyMediaUncertain, status)
		require.Len(t, shop.gallery, 4)
		require.Zero(t, shop.count(submission.ShopifyMediaAssociate))
		freeze, err := shopifyguard.SealIsolated(ctx, db, shopifyguard.Command{ID: uuid.NewString(),
			Organization: plan.Owner.OrganizationID, ShopKey: plan.ShopKey, Generation: plan.Generation})
		require.NoError(t, err)
		drained, err := shopifyguard.DrainReadyIsolated(ctx, db, isolation, freeze)
		require.NoError(t, err)
		require.True(t, drained)
	})
}

func fixtureMediaPlan(name string) submission.ShopifyMediaPlan {
	plan := submission.ShopifyMediaPlan{
		ID: uuid.NewString(), Owner: collection.Scope{OrganizationID: "fixture-" + name, ActorID: "actor-a", MemberID: "member-a"},
		ShopKey: uuid.NewString(), Generation: 1, ProductID: "gid://shopify/Product/44", SourceKey: "own-fixture",
		SourceVersion: 1, ApprovalHash: collection.Digest("approval"), BeforeHash: collection.Digest("remote-draft"), ExpiresAt: time.Now().Add(15 * time.Minute).UTC(),
	}
	for i := 1; i <= 4; i++ {
		plan.GalleryIDs = append(plan.GalleryIDs, fmt.Sprintf("gid://shopify/MediaImage/%d", i))
	}
	for i := 0; i < 3; i++ {
		hash := collection.Digest(fmt.Sprintf("fixture-color-%d", i))
		plan.Images = append(plan.Images, hash)
		for size := 0; size < 2; size++ {
			plan.Covers = append(plan.Covers, submission.ShopifyMediaCover{
				VariantID: fmt.Sprintf("gid://shopify/ProductVariant/%d", 100+i*2+size), ImageHash: hash,
				CurrentMediaID: "gid://shopify/MediaImage/1",
			})
		}
	}
	return plan
}

type fakeMediaProvider struct {
	plan               submission.ShopifyMediaPlan
	mu                 sync.Mutex
	calls              []submission.ShopifyMediaPhase
	gallery            map[string]submission.ShopifyMediaOriginal
	covers             map[string]string
	processing, denied bool
	stopAfterStaging   bool
	extraCandidate     bool
	lose               submission.ShopifyMediaPhase
	reject             submission.ShopifyMediaPhase
	onAppend           func()
	onRead             func()
}

func fixtureMediaProvider(plan submission.ShopifyMediaPlan) *fakeMediaProvider {
	p := &fakeMediaProvider{plan: plan, gallery: map[string]submission.ShopifyMediaOriginal{}, covers: map[string]string{}}
	for i := 1; i <= 4; i++ {
		id := fmt.Sprintf("gid://shopify/MediaImage/%d", i)
		p.gallery[id] = submission.ShopifyMediaOriginal{ID: id, Hash: collection.Digest(id), Ready: true}
	}
	for _, cover := range plan.Covers {
		p.covers[cover.VariantID] = cover.CurrentMediaID
	}
	return p
}

func (p *fakeMediaProvider) Verify(_ context.Context, _ submission.ShopifyMediaPlan, beforeAppend bool) error {
	if p.denied {
		return submission.ErrExecutionEvidenceRequired
	}
	if beforeAppend && p.stopAfterStaging && p.count(submission.ShopifyMediaUpload) == len(p.plan.Images) {
		p.stopAfterStaging = false
		return submission.ErrExecutionEvidenceRequired
	}
	return nil
}

func (p *fakeMediaProvider) Send(_ context.Context, request submission.ShopifyMediaRequest) (submission.ShopifyMediaResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, request.Phase)
	reply := submission.ShopifyMediaResponse{OriginalResponse: true}
	if p.reject == request.Phase {
		reply.Rejected = true
		return reply, nil
	}
	switch request.Phase {
	case submission.ShopifyMediaPrepare:
		for i := range request.Plan.Images {
			reply.Staged = append(reply.Staged, fmt.Sprintf("fixture-staged-%d", i))
		}
	case submission.ShopifyMediaUpload:
		if request.Staged[0] != fmt.Sprintf("fixture-staged-%d", request.Index) {
			return submission.ShopifyMediaResponse{}, errors.New("wrong temporary resource")
		}
	case submission.ShopifyMediaAppend:
		for i, hash := range request.Plan.Images {
			id := fmt.Sprintf("gid://shopify/MediaImage/%d", 10+i)
			p.gallery[id] = submission.ShopifyMediaOriginal{ID: id, Hash: hash, Ready: true}
			reply.Candidates = append(reply.Candidates, id)
		}
		if p.extraCandidate {
			const id = "gid://shopify/MediaImage/99"
			p.gallery[id] = submission.ShopifyMediaOriginal{ID: id}
			reply.Candidates = append(reply.Candidates, id)
		}
		if p.onAppend != nil {
			p.onAppend()
		}
	case submission.ShopifyMediaAssociate:
		for _, item := range request.Associations {
			p.covers[item.VariantID] = item.MediaID
		}
	}
	if p.lose == request.Phase {
		return submission.ShopifyMediaResponse{}, errors.New("fixture original response lost")
	}
	return reply, nil
}

func (p *fakeMediaProvider) Read(_ context.Context, _ submission.ShopifyMediaPlan) (submission.ShopifyMediaObservation, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.onRead != nil {
		p.onRead()
	}
	observation := submission.ShopifyMediaObservation{Draft: true, Complete: true, ProductID: p.plan.ProductID, Associations: make(map[string]string)}
	for _, item := range p.gallery {
		item.Ready = item.Ready && !p.processing
		observation.Gallery = append(observation.Gallery, item)
	}
	for variant, media := range p.covers {
		observation.Associations[variant] = media
	}
	return observation, nil
}

func (p *fakeMediaProvider) desiredCovers() map[string]string {
	out := make(map[string]string)
	for _, cover := range p.plan.Covers {
		for id, image := range p.gallery {
			if cover.ImageHash == image.Hash {
				out[cover.VariantID] = id
			}
		}
	}
	return out
}

func (p *fakeMediaProvider) count(phase submission.ShopifyMediaPhase) int {
	count := 0
	for _, value := range p.calls {
		if value == phase {
			count++
		}
	}
	return count
}
