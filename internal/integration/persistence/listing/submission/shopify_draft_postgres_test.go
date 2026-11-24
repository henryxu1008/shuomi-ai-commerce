package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	officialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/official"
	prepstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/preparation"
	recordstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record"
	effectsstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	assetstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/asset"
	catalogstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/catalog"
	collectionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/collection"
	record "github.com/henryxu1008/shuomi-ai-commerce/internal/listing/record/target"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/marketplace/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/storecenter"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresShopifyDraftRetainsOriginalAttemptAndBlocksDuplicate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("shopify_product"), tcpostgres.WithUsername("shopify_fixture"), tcpostgres.WithPassword("isolated-fixture"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema, recordstore.InstallSchema, effectsstore.InstallSchema, officialstore.InstallOfficialSchema, assetstore.AutoMigrate} {
		require.NoError(t, install(db))
	}
	require.NoError(t, recordstore.VerifyShopifySchema(ctx, db))
	require.NoError(t, effectsstore.VerifyShopifySchema(ctx, db))
	saved, inventory := temuEffectRecordFixture(t, ctx, db, db)
	prior := saved.Merchant
	shopKey := uuid.NewString()
	saved.Merchant = storecenter.ProductMerchantBinding{
		Platform: storecenter.PlatformShopify, OrganizationID: prior.OrganizationID, StoreID: prior.StoreID, Site: "shopify",
		StoreVersion: 2, ConnectionRevision: 1, ApplicationID: "gid://shopify/App/32", ApplicationRevision: "shopify-admin-2026-10",
		SupplierIdentityHash: collection.Digest(shopKey), ServiceExpiresAt: prior.ServiceExpiresAt,
		Shopify: storecenter.ShopifyMerchantBinding{ShopKey: shopKey, Currency: "USD"},
	}
	inventory.Scope.TargetPlatform = "product"
	input := shopify.Input{Currency: "USD", Prices: []string{"12.50"}}
	draft, err := shopify.BuildDraft(input, catalog.ProductSnapshot{Title: "Fixture"}, inventory, "USD")
	require.NoError(t, err)
	saved.Input = record.TargetInput{SourceID: saved.Source.ID, StoreID: saved.Merchant.StoreID, EffectiveVersion: 1, Shopify: &input}
	saved.Temu, saved.TemuRuleSnapshot, saved.TemuImageObservations = nil, nil, nil
	saved.Shopify, saved.RulesHash, saved.InventoryHash = &draft, collection.Digest(saved.Merchant.Shopify), collection.Digest(inventory)
	replaceEffectFixtureRecord(t, db, saved)
	scope := collection.Scope{OrganizationID: prior.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}
	executionScope := submission.ExecutionScope{OrganizationID: scope.OrganizationID}
	repository, err := effectsstore.NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(repository)
	require.NoError(t, err)
	official, err := officialstore.NewOfficialRepository(ctx, db)
	require.NoError(t, err)
	target, err := submission.ShopifyDraftTarget(saved.Merchant, saved.Source.Source.ProductKey)
	require.NoError(t, err)
	_, payload, err := submission.ShopifyDraftPayload(saved)
	require.NoError(t, err)
	command := submission.AcquireExecutionCommand{Scope: executionScope, Target: target, Action: submission.ShopifyDraftAction, Payload: payload, ClaimOwnerID: "fixture-worker", Lease: time.Minute}
	seedIntent := func() submission.OfficialIntent {
		command.IntentKey = uuid.NewString()
		id, idErr := uuid.NewV7()
		require.NoError(t, idErr)
		reservation, reserveErr := submission.NewExecutionReservation(command, id.String(), "fixture-claim", time.Now())
		require.NoError(t, reserveErr)
		intent := submission.OfficialIntent{Key: command.IntentKey, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: saved.Source, Target: target, Binding: saved.Merchant, PayloadFingerprint: reservation.Attempt.PayloadFingerprint, CreatedAt: saved.CreatedAt}
		intent.InputHash = submission.OfficialIntentInputHash(intent)
		require.NoError(t, submission.ValidateOfficialIntent(intent))
		raw, marshalErr := json.Marshal(intent)
		require.NoError(t, marshalErr)
		require.NoError(t, db.Exec("INSERT INTO listing_submission_official_intents(organization_id,actor_id,member_id,intent_key,record_id,kind,input_hash,body_hash,body_json) VALUES(?,?,?,?,?,'publish',?,?,?::jsonb)", scope.OrganizationID, scope.ActorID, scope.MemberID, intent.Key, saved.ID, intent.InputHash, collection.Digest(intent), string(raw)).Error)
		return intent
	}
	command.IntentKey = uuid.NewString()
	_, err = kernel.PrepareShopifyExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
	seedIntent()
	prepared, err := kernel.PrepareShopifyExecution(ctx, command)
	require.NoError(t, err)
	stopped, err := kernel.StopShopifyBeforeSend(ctx, prepared.Preparation, submission.NoSendAccessRecheckFailed)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionNotSent, stopped.Status)
	intent := seedIntent()
	prepared, err = kernel.PrepareShopifyExecution(ctx, command)
	require.NoError(t, err)
	permit, err := kernel.BeginShopifySend(ctx, prepared.Preparation)
	require.NoError(t, err)
	claim := submission.ExecutionClaim{Scope: executionScope, AttemptID: permit.AttemptID, FenceEpoch: permit.FenceEpoch, OwnerID: permit.ClaimOwnerID, Token: permit.ClaimToken}
	lost, err := kernel.MarkUnknown(ctx, claim, submission.UnknownResponseLost)
	require.NoError(t, err)
	seedIntent()
	_, err = kernel.PrepareShopifyExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
	observation := shopify.ProductObservation{ID: "gid://shopify/Product/42", Status: "DRAFT", Media: []string{"PROCESSING"}}
	resolution, err := submission.NewShopifyDraftResolution(intent, saved, lost, observation)
	require.NoError(t, err)
	receipt, err := official.ResolveOfficial(ctx, resolution)
	require.NoError(t, err)
	require.Equal(t, lost.AttemptID, receipt.ID)
	require.Equal(t, &observation, receipt.Shopify)
	rebound := saved.Merchant
	rebound.StoreID, rebound.ConnectionRevision = uuid.NewString(), 2
	reboundTarget, err := submission.ShopifyDraftTarget(rebound, saved.Source.Source.ProductKey)
	require.NoError(t, err)
	require.Equal(t, target, reboundTarget)
	retained, err := official.FindOfficialTarget(ctx, scope.OrganizationID, reboundTarget)
	require.NoError(t, err)
	require.Equal(t, receipt.ID, retained.ID)
	_, err = kernel.PrepareShopifyExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded)
}
