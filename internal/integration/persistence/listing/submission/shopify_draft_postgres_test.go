package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	officialstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/official"
	prepstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/preparation"
	recordstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/record"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/shopifyguard"
	effectsstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/listing/submission"
	assetstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/asset"
	catalogstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/catalog"
	collectionstore "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/persistence/product/collection"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/preparation"
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
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema, prepstore.InstallOperationSchema, prepstore.InstallUploadPlanSchema, recordstore.InstallSchema, effectsstore.InstallSchema, officialstore.InstallOfficialSchema, assetstore.AutoMigrate} {
		require.NoError(t, install(db))
	}
	require.NoError(t, recordstore.VerifyShopifySchema(ctx, db))
	require.NoError(t, effectsstore.InstallGooglePlanSchema(ctx, db))
	require.NoError(t, effectsstore.VerifyShopifySchema(ctx, db))
	require.NoError(t, shopifyguard.InstallIsolated(ctx, db))
	db = shopifyguard.RequireIsolated(db)
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
	verifyRestored := prepareShopifyGuardRestore(t, ctx, container, db, saved.Merchant)
	scope := collection.Scope{OrganizationID: prior.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}
	executionScope := submission.ExecutionScope{OrganizationID: scope.OrganizationID}
	repository, err := effectsstore.NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	const liveClaimToken = "isolated-live-claim"
	kernel, err := submission.NewExecutionKernel(repository, submission.WithExecutionClaimTokenGenerator(func() (string, error) {
		return liveClaimToken, nil
	}))
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
	claim := submission.ExecutionClaim{Scope: executionScope, AttemptID: prepared.Attempt.AttemptID,
		FenceEpoch: prepared.Attempt.FenceEpoch, OwnerID: prepared.Attempt.ClaimOwnerID, Token: liveClaimToken}
	require.NoError(t, repository.VerifyShopifySend(ctx, claim))
	lost, err := kernel.MarkUnknown(ctx, claim, submission.UnknownResponseLost)
	require.NoError(t, err)
	permit, err := kernel.BeginShopifySend(ctx, prepared.Preparation)
	require.ErrorIs(t, err, submission.ErrExecutionClaimRejected)
	require.Nil(t, permit, "the locally held preparation cannot override a persisted UNKNOWN")
	seedIntent()
	_, err = kernel.PrepareShopifyExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
	observation := shopify.ProductObservation{ID: "gid://shopify/Product/42", Status: "DRAFT", Media: []string{"PROCESSING"}}
	completion, err := submission.NewShopifyDraftCompletion(scope, saved, prepared.Attempt, claim, payload, observation)
	require.NoError(t, err)
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

	preparationRepo, err := prepstore.NewRepository(ctx, db)
	require.NoError(t, err)
	readStage := func() []preparation.SourceStageFacts {
		stage, err := preparationRepo.ListStageFacts(ctx, scope, saved.Source.PreparationID, saved.Merchant.StoreID, shopKey, saved.Merchant.SupplierIdentityHash, "")
		require.NoError(t, err)
		require.Len(t, stage, 1)
		return stage
	}
	require.Equal(t, saved.ID, readStage()[0].PublishedRecordID)
	verifyOldOperation := prepareOldShopifyOperation(t, ctx, db, scope, saved)
	restore, err := shopifyguard.SealIsolated(ctx, db, shopifyguard.Command{ID: uuid.NewString(),
		Organization: scope.OrganizationID, ShopKey: shopKey, Generation: saved.Merchant.ConnectionRevision})
	require.NoError(t, err)
	require.Equal(t, []string{saved.Source.Source.ProductKey}, restore.Sources)
	verifyOldOperation()
	verifyRestored(restore)
	stage := readStage()[0]
	require.Empty(t, stage.PublishedRecordID)
	require.Empty(t, stage.RecordID)
	require.False(t, stage.Ready)
	targets, err := recordstore.NewRepository(ctx, db)
	require.NoError(t, err)
	revision, err := targets.ReadFrozenShopifyRevision(ctx, scope, saved.TargetID)
	require.NoError(t, err)
	require.Equal(t, saved.Revision, revision)
	_, err = official.CompleteOfficial(ctx, completion)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	_, err = kernel.MarkUnknown(ctx, claim, submission.UnknownResponseLost)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	_, err = official.ResolveOfficial(ctx, resolution)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	_, err = official.ReadOfficial(ctx, scope, receipt.ID)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	_, err = official.FindOfficialTarget(ctx, scope.OrganizationID, target)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)

	// A new Store identity for the same source must not create a second draft.
	saved.Merchant.Shopify.ShopKey = uuid.NewString()
	saved.Merchant.ConnectionRevision++
	saved.Merchant.SupplierIdentityHash = collection.Digest(saved.Merchant.Shopify.ShopKey)
	saved.RulesHash = collection.Digest(saved.Merchant.Shopify)
	saved.ID = uuid.NewString()
	saved.Revision++
	saved.Input.ExpectedRevision = saved.Revision - 1
	saved.CreatedAt = time.Now().UTC().Truncate(time.Microsecond)
	rawRecord, err := json.Marshal(saved)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`INSERT INTO listing_target_records
		(organization_id,actor_id,member_id,id,target_id,source_id,store_id,site,revision,input_hash,body_hash,body_json,created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?::jsonb,?)`,
		scope.OrganizationID, scope.ActorID, scope.MemberID, saved.ID, saved.TargetID, saved.Source.ID, saved.Merchant.StoreID,
		saved.Merchant.Site, saved.Revision, collection.Digest(saved.Input), collection.Digest(saved), string(rawRecord), saved.CreatedAt).Error)
	require.NoError(t, db.Exec(`UPDATE listing_preparation_targets SET current_record_id=?,revision=? WHERE id=?`,
		saved.ID, saved.Revision, saved.TargetID).Error)
	target, err = submission.ShopifyDraftTarget(saved.Merchant, saved.Source.Source.ProductKey)
	require.NoError(t, err)
	_, newPayload, err := submission.ShopifyDraftPayload(saved)
	require.NoError(t, err)
	command.Target, command.Payload = target, newPayload
	seedIntent()
	_, err = kernel.PrepareShopifyExecution(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed)
}

type shopifyOperationAccessFixture struct {
	preparation.CollectionSelections
	preparation.CollectionOwnerAuthority
	scope collection.Scope
}

func (a shopifyOperationAccessFixture) Authorize(context.Context, string) (collection.Scope, error) {
	return a.scope, nil
}

func prepareOldShopifyOperation(t *testing.T, ctx context.Context, db *gorm.DB, scope collection.Scope, saved record.TargetRecord) func() {
	t.Helper()
	key := uuid.NewString()
	id := preparation.OperationCommandID(scope, key)
	input := preparation.OperationInput{Action: preparation.OperationUpload, StoreID: saved.Merchant.StoreID,
		PreparationID: saved.Source.PreparationID, ExpectedRevision: 1, SourceIDs: []string{saved.Source.ID}}
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`INSERT INTO listing_preparation_operations
		(organization_id,actor_id,member_id,id,command_key,input_hash,input_json,preparation_id,store_id,action,item_count,status,execution,created_at)
		VALUES (?,?,?,?,?,?,?::jsonb,?,?,'upload',1,'running','started',now())`,
		scope.OrganizationID, scope.ActorID, scope.MemberID, id, key, collection.Digest(input), string(raw), saved.Source.PreparationID, saved.Merchant.StoreID).Error)
	require.NoError(t, db.Exec(`INSERT INTO listing_preparation_operation_items
		(organization_id,actor_id,member_id,operation_id,source_id,record_id,record_revision,status)
		VALUES (?,?,?,?,?,?,?,'running')`, scope.OrganizationID, scope.ActorID, scope.MemberID, id, saved.Source.ID, saved.ID, saved.Revision).Error)
	base, err := prepstore.NewRepository(ctx, db)
	require.NoError(t, err)
	operations, err := prepstore.NewOperationRepository(ctx, db)
	require.NoError(t, err)
	plans, err := prepstore.NewUploadPlanRepository(ctx, db)
	require.NoError(t, err)
	access := shopifyOperationAccessFixture{scope: scope}
	service, err := preparation.NewService(base, access, access)
	require.NoError(t, err)
	opService, err := preparation.NewOperationService(service, access, operations)
	require.NoError(t, err)
	ctx = authidentity.WithAuthenticatedIdentity(ctx, authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID,
		EffectiveOrganizationID: scope.OrganizationID, EffectiveMemberID: scope.MemberID, UserID: scope.ActorID,
		TokenExpiresAt: time.Now().Add(time.Minute)})
	_, prior, err := plans.ReadUploadPlanOperation(ctx, scope, id, saved.Source.ID)
	require.NoError(t, err)
	require.Equal(t, saved.ID, prior.RecordID)
	return func() {
		t.Helper()
		proof, err := opService.RequestAccess(ctx, id)
		require.NoError(t, err)
		prior.Status, prior.ResultReference = preparation.ItemSucceeded, uuid.NewString()
		require.ErrorIs(t, operations.FinishOperationItem(ctx, proof, prior), preparation.ErrNotFound)
		_, item, err := plans.ReadUploadPlanOperation(ctx, scope, id, saved.Source.ID)
		require.NoError(t, err)
		require.Equal(t, preparation.ItemMissing, item.Status)
		require.Empty(t, item.RecordID)
		require.Empty(t, item.ResultReference)
		page, err := operations.ListOperationItems(ctx, scope, id, collection.Query{Limit: 1})
		require.NoError(t, err)
		require.Equal(t, []preparation.OperationItem{item}, page.Items)
	}
}
