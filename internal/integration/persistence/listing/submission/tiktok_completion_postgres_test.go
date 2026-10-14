package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	effectsstore "task-processor/internal/integration/persistence/listing/submission"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestPostgresTikTokStagesUseAtomicCompletionAndRetainDraft(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("tiktok669"), tcpostgres.WithUsername("tiktok_owner"), tcpostgres.WithPassword("isolated-tiktok-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	pool, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, pool.Close()) })
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema, recordstore.InstallSchema, effectsstore.InstallSchema, officialstore.InstallOfficialSchema} {
		require.NoError(t, install(db))
	}
	require.NoError(t, db.Exec("CREATE DATABASE tiktok_assets669").Error)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	parsed.Path = "/tiktok_assets669"
	assetDB, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	assetPool, err := assetDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, assetPool.Close()) })
	require.NoError(t, assetstore.AutoMigrate(assetDB))
	saved, inventory := temuEffectRecordFixture(t, ctx, db, assetDB)
	b := saved.Merchant
	b.Platform = storecenter.PlatformTikTokShop
	b.Site = "tiktokshop-us"
	b.ApplicationType = ""
	b.IdentityPartition = ""
	b.MerchantMode = ""
	b.RegionID = 0
	b.VerificationRevision = ""
	b.ScopesHash = collection.Digest("granted scopes")
	b.CredentialExpiresAt = time.Now().UTC().Add(time.Hour)
	b.TikTok = storecenter.TikTokMerchantBinding{GrantID: uuid.NewString(), ShopID: "shop", GrantGeneration: 1, GrantRevision: 1, BindingRevision: 1}
	inventory.Scope.TargetPlatform = "tiktokshop"
	image := inventory.Assets[0]
	image.Role = asset.RoleMain
	inventory.Assets = []asset.ApprovedAsset{image}
	observed := marketplace.ImageObservation{AssetID: image.ID, UseCase: "MAIN_IMAGE", SourceURL: image.URL, ContentHash: collection.Digest("original bytes"), Width: 600, Height: 600, Bytes: 123, MediaType: "image/png"}
	yes, no := true, false
	rules := marketplace.RuleSnapshot{Selection: marketplace.RuleSelection{Method: marketplace.GlobalPublishing, CategoryID: "cat"}, Listing: marketplace.ListingRules{Methods: []marketplace.ListingMethod{marketplace.GlobalPublishing}, Fingerprint: collection.Digest("rules")}, Category: marketplace.Category{ID: "cat", Leaf: &yes, Permissions: []string{"AVAILABLE"}}, Rules: marketplace.CategoryRules{SizeChart: &marketplace.SizeChartRule{Required: &no, Supported: &no}}, Warehouses: []marketplace.Warehouse{{ID: "US-wh", Type: "SALES_WAREHOUSE", EffectStatus: "ENABLED", SubType: "CB_DIRECT_SHIPPING_WAREHOUSE"}}, GlobalWarehouses: []marketplace.GlobalWarehouse{{ID: "global-wh", Ownership: "SELLER"}}}
	input := marketplace.DraftInput{Selection: rules.Selection, Title: "A human title", Description: "A human description", SaveMode: "LISTING", MainImages: []string{image.ID}, PackageWeight: marketplace.Weight{Value: "0.500", Unit: "KILOGRAM"}, SKUs: []marketplace.SKUInput{{SellerSKU: "SKU1", SalePrice: "10.00", GlobalBasePrice: "8.00", Inventory: []marketplace.InventoryInput{{WarehouseID: "US-wh", Quantity: 3}}, GlobalInventory: []marketplace.GlobalInventoryInput{{WarehouseID: "global-wh", Quantity: 3}}}}}
	draft := marketplace.BuildDraft(input, rules, inventory, []marketplace.ImageObservation{observed})
	require.True(t, draft.ReadyForUpload, "%v", draft.Issues)
	saved.Merchant = b
	saved.Input = record.TargetInput{SourceID: saved.Source.ID, StoreID: b.StoreID, EffectiveVersion: 1, TikTok: &input}
	saved.Temu = nil
	saved.TemuRuleSnapshot = nil
	saved.TemuImageObservations = nil
	saved.TikTok = &draft
	saved.TikTokRuleSnapshot = &rules
	saved.TikTokImageObservations = []marketplace.ImageObservation{observed}
	saved.RulesHash = marketplace.RuleHash(rules)
	saved.InventoryHash = collection.Digest(inventory)
	raw, err := json.Marshal(saved)
	require.NoError(t, err)
	require.NoError(t, db.Exec("UPDATE listing_target_records SET site=?,input_hash=?,body_hash=?,body_json=?::jsonb WHERE id=?", b.Site, collection.Digest(saved.Input), collection.Digest(saved), string(raw), saved.ID).Error)
	require.NoError(t, db.Exec("UPDATE listing_preparation_targets SET site=? WHERE id=?", b.Site, saved.TargetID).Error)
	records, err := recordstore.NewRepository(ctx, db)
	require.NoError(t, err)
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	actual, err := records.ReadTargetHead(ctx, scope, saved.TargetID)
	require.NoError(t, err)
	require.Equal(t, saved, actual)
	repo, err := effectsstore.NewOfficialEffectsRepository(db)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(repo)
	require.NoError(t, err)
	official, err := officialstore.NewOfficialRepository(ctx, db)
	require.NoError(t, err)
	acquire := func(stage string, img *marketplace.ImageObservation, images []submission.OfficialReceipt, global *submission.OfficialReceipt) (submission.OfficialIntent, []byte, submission.ExecutionAcquisition, submission.ExecutionClaim) {
		key := uuid.NewString()
		payload, effect, e := submission.ResolveTikTokPayload(saved, stage, key, img, images, global)
		require.NoError(t, e)
		subject, e := submission.TikTokProductSubjectID(saved.Source.Source.ProductKey, stage)
		if img != nil {
			subject, e = submission.TikTokImageSubjectID(*img, b)
		}
		require.NoError(t, e)
		command := submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: key, Target: submission.ExecutionTarget{Platform: "tiktokshop", StoreID: b.StoreID, SubjectID: subject}, Action: submission.TikTokStageAction(stage), Payload: payload, ClaimOwnerID: "test-worker", Lease: time.Minute}
		id, _ := uuid.NewV7()
		reservation, e := submission.NewExecutionReservation(command, id.String(), "test-claim", time.Now().UTC())
		require.NoError(t, e)
		intent := submission.OfficialIntent{Key: key, Owner: scope, Kind: stage, RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: saved.Source, Target: command.Target, Binding: b, PayloadFingerprint: reservation.Attempt.PayloadFingerprint, TikTok: effect, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
		intent.InputHash = submission.OfficialIntentInputHash(intent)
		require.NoError(t, submission.ValidateOfficialIntent(intent))
		encoded, e := json.Marshal(intent)
		require.NoError(t, e)
		require.NoError(t, db.Exec("INSERT INTO listing_submission_official_intents(organization_id,actor_id,member_id,intent_key,record_id,kind,input_hash,body_hash,body_json) VALUES(?,?,?,?,?,?,?,?,?::jsonb)", scope.OrganizationID, scope.ActorID, scope.MemberID, key, saved.ID, stage, intent.InputHash, collection.Digest(intent), string(encoded)).Error)
		acquired, e := kernel.Acquire(ctx, command)
		require.NoError(t, e)
		require.NotNil(t, acquired.Permit)
		permit := acquired.Permit
		claim := submission.ExecutionClaim{Scope: command.Scope, AttemptID: permit.AttemptID, FenceEpoch: permit.FenceEpoch, OwnerID: permit.ClaimOwnerID, Token: permit.ClaimToken}
		return intent, payload, acquired, claim
	}
	intent, payload, acquired, claim := acquire(submission.TikTokImageStage, &observed, nil, nil)
	uploaded := marketplace.UploadedImage{URI: "actual-uri", URL: "https://images.example.org/uploaded.png", Width: 600, Height: 600, UseCase: "MAIN_IMAGE", ResponseHash: collection.Digest("actual upload response")}
	proof, err := submission.NewTikTokCompletion(scope, saved, intent, acquired.Attempt, claim, payload, submission.TikTokEffectReceipt{DraftHash: intent.TikTok.DraftHash, Image: &observed, Uploaded: &uploaded})
	require.NoError(t, err)
	imageReceipt, err := official.CompleteOfficial(ctx, proof)
	require.NoError(t, err)
	intent, payload, acquired, claim = acquire(submission.TikTokGlobalCreateStage, nil, []submission.OfficialReceipt{imageReceipt}, nil)
	global := marketplace.CreatedGlobalProduct{ID: "global-1", SKUs: []marketplace.CreatedSKU{{ID: "global-sku-1", SellerSKU: "SKU1"}}, ResponseHash: collection.Digest("actual global response")}
	proof, err = submission.NewTikTokCompletion(scope, saved, intent, acquired.Attempt, claim, payload, submission.TikTokEffectReceipt{DraftHash: intent.TikTok.DraftHash, Global: &global})
	require.NoError(t, err)
	require.NoError(t, db.Exec("CREATE FUNCTION reject_tiktok_receipt669() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN IF NEW.kind='global_create' THEN RAISE EXCEPTION 'test receipt insert failure'; END IF; RETURN NEW; END$$; CREATE TRIGGER reject_tiktok_receipt669 BEFORE INSERT ON listing_submission_official_receipts FOR EACH ROW EXECUTE FUNCTION reject_tiktok_receipt669();").Error)
	_, err = official.CompleteOfficial(ctx, proof)
	require.Error(t, err)
	attempt, err := kernel.Get(ctx, claim.Scope, claim.AttemptID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionClaimed, attempt.Status, "receipt failure rolls back kernel success in the same transaction")
	_, err = official.ReadOfficial(ctx, scope, claim.AttemptID)
	require.ErrorIs(t, err, submission.ErrExecutionNotFound)
	require.NoError(t, db.Exec("DROP TRIGGER reject_tiktok_receipt669 ON listing_submission_official_receipts").Error)
	globalReceipt, err := official.CompleteOfficial(ctx, proof)
	require.NoError(t, err)
	intent, payload, acquired, claim = acquire(submission.TikTokPublishStage, nil, nil, &globalReceipt)
	publication := marketplace.GlobalPublishResult{State: marketplace.RemoteDraft, ShopID: "shop", ProductID: "US-product", SKUs: []marketplace.PublishedSKU{{LocalSKUID: "US-sku", GlobalSKUID: "global-sku-1"}}, FailureReason: "official_validation_failed_saved_as_draft", FailureCount: 1, Fingerprint: collection.Digest("actual publish DRAFT response")}
	proof, err = submission.NewTikTokCompletion(scope, saved, intent, acquired.Attempt, claim, payload, submission.TikTokEffectReceipt{DraftHash: intent.TikTok.DraftHash, Publication: &publication})
	require.NoError(t, err)
	retained, err := official.CompleteOfficial(ctx, proof)
	require.NoError(t, err)
	reread, err := official.FindOfficialTarget(ctx, scope.OrganizationID, retained.Target)
	require.NoError(t, err)
	require.Equal(t, "US-product", reread.TikTok.Publication.ProductID)
	require.Equal(t, marketplace.RemoteDraft, reread.TikTok.Publication.State)
	_, err = kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: claim.Scope, IntentKey: uuid.NewString(), Target: retained.Target, Action: submission.TikTokPublishAction, Payload: []byte("changed approved data"), ClaimOwnerID: "test-worker", Lease: time.Minute})
	require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded)
	require.NoError(t, db.Exec("ALTER TABLE listing_submission_official_intents DROP CONSTRAINT listing_submission_official_intents_kind_check; ALTER TABLE listing_submission_official_intents ADD CHECK(kind IN ('publish','image')) NOT VALID").Error)
	require.Error(t, officialstore.VerifyOfficialSchema(ctx, db), "serving must reject a schema that cannot persist a global_create stage")
}
