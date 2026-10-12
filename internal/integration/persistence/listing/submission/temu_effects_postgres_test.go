package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"sync"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	effectsstore "task-processor/internal/integration/persistence/listing/submission"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

func TestPostgresTemuImageRenewalRequiresLatestKnownExpiredSuccess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, err := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("temu31"), tcpostgres.WithUsername("temu_owner"), tcpostgres.WithPassword("isolated-temu-test"), tcpostgres.BasicWaitStrategies())
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
	require.NoError(t, db.Exec("CREATE DATABASE temu_assets31").Error)
	parsed, err := url.Parse(dsn)
	require.NoError(t, err)
	parsed.Path = "/temu_assets31"
	assetDB, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	assetPool, err := assetDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, assetPool.Close()) })
	require.NoError(t, assetstore.AutoMigrate(assetDB))
	saved, _ := temuEffectRecordFixture(t, ctx, db, assetDB)
	records, err := recordstore.NewRepository(ctx, db)
	require.NoError(t, err)
	retained, err := records.ReadTargetHead(ctx, collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}, saved.TargetID)
	require.NoError(t, err)
	require.Equal(t, saved, retained)
	require.Equal(t, collection.Digest(saved), collection.Digest(retained))
	original := seedTemuImageSuccess(t, db, saved, time.Now().UTC().Truncate(time.Microsecond))
	require.NoError(t, db.Exec("CREATE ROLE temu_runtime31 LOGIN PASSWORD 'synthetic-runtime-only' NOSUPERUSER NOINHERIT").Error)
	require.NoError(t, db.Exec("GRANT USAGE ON SCHEMA public TO temu_runtime31").Error)
	require.NoError(t, db.Exec("GRANT SELECT,INSERT,UPDATE ON listing_target_records,listing_preparation_targets,listing_target_record_commands TO temu_runtime31").Error)
	for _, table := range []string{effectsstore.AttemptTable, effectsstore.TargetFenceTable, "listing_submission_official_intents", "listing_submission_official_receipts"} {
		require.NoError(t, db.Exec("GRANT SELECT,INSERT,UPDATE ON "+table+" TO temu_runtime31").Error)
	}
	runtimeDSN, err := url.Parse(dsn)
	require.NoError(t, err)
	runtimeDSN.User = url.UserPassword("temu_runtime31", "synthetic-runtime-only")
	runtimeDB, err := gorm.Open(postgres.Open(runtimeDSN.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	runtimePool, err := runtimeDB.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, runtimePool.Close()) })
	var assetTableExists bool
	require.NoError(t, runtimeDB.Raw("SELECT to_regclass('public.product_approved_inventory_version_heads') IS NOT NULL").Scan(&assetTableExists).Error)
	require.False(t, assetTableExists, "Product serving DB has no copied Asset tables")
	repo, err := effectsstore.NewOfficialEffectsRepository(runtimeDB)
	require.NoError(t, err)
	kernel, err := submission.NewExecutionKernel(repo)
	require.NoError(t, err)
	command := seedTemuImageIntent(t, db, saved, "unexpired-renewal", original.ID)
	_, err = kernel.Acquire(ctx, command)
	require.ErrorIs(t, err, submission.ErrExecutionTargetSucceeded, "a live URL is reused; a new intent cannot reset its fence")
	resetTemuEffectRows(t, db)
	original = seedTemuImageSuccess(t, db, saved, time.Now().UTC().Add(-8*24*time.Hour).Truncate(time.Microsecond))
	first := seedTemuImageIntent(t, db, saved, "expired-a", original.ID)
	second := seedTemuImageIntent(t, db, saved, "expired-b", original.ID)
	type result struct {
		a   submission.ExecutionAcquisition
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, command := range []submission.AcquireExecutionCommand{first, second} {
		wg.Add(1)
		go func(c submission.AcquireExecutionCommand) {
			defer wg.Done()
			<-start
			a, err := kernel.Acquire(ctx, c)
			results <- result{a, err}
		}(command)
	}
	close(start)
	wg.Wait()
	close(results)
	var winner submission.ExecutionAcquisition
	permits := 0
	for got := range results {
		if got.err == nil {
			require.NotNil(t, got.a.Permit)
			winner = got.a
			permits++
		} else {
			require.True(t, errors.Is(got.err, submission.ErrExecutionTargetClaimed), got.err)
		}
	}
	require.Equal(t, 1, permits)
	require.EqualValues(t, 2, winner.Attempt.FenceEpoch)
	claim := submission.ExecutionClaim{Scope: submission.ExecutionScope{OrganizationID: "org-a"}, AttemptID: winner.Attempt.AttemptID, FenceEpoch: winner.Permit.FenceEpoch, OwnerID: winner.Permit.ClaimOwnerID, Token: winner.Permit.ClaimToken}
	_, err = kernel.MarkUnknown(ctx, claim, submission.UnknownResponseLost)
	require.NoError(t, err)
	blocked := seedTemuImageIntent(t, db, saved, "unknown-renewal", winner.Attempt.AttemptID)
	_, err = kernel.Acquire(ctx, blocked)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed, "an UNKNOWN generation never inherits the old success TTL")
	originals, err := officialstore.NewOfficialRepository(ctx, runtimeDB)
	require.NoError(t, err)
	retainedReceipt, err := originals.ReadOfficial(ctx, original.Owner, original.ID)
	require.NoError(t, err)
	require.Equal(t, collection.Digest(original), collection.Digest(retainedReceipt))
	_, err = originals.FindOfficialTarget(ctx, "org-a", original.Target)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed, "latest UNKNOWN cannot return an older success")
	resetTemuEffectRows(t, db)
	original = seedTemuImageSuccess(t, db, saved, time.Now().UTC().Add(-8*24*time.Hour).Truncate(time.Microsecond))
	renewedCommand := seedTemuImageIntent(t, db, saved, "known-renewal", original.ID)
	renewed, err := kernel.Acquire(ctx, renewedCommand)
	require.NoError(t, err)
	require.NotNil(t, renewed.Permit)
	claim = submission.ExecutionClaim{Scope: submission.ExecutionScope{OrganizationID: "org-a"}, AttemptID: renewed.Attempt.AttemptID, FenceEpoch: renewed.Permit.FenceEpoch, OwnerID: renewed.Permit.ClaimOwnerID, Token: renewed.Permit.ClaimToken}
	var imageInput model.LocalImageUpload
	require.NoError(t, json.Unmarshal(renewedCommand.Payload, &imageInput))
	proof, err := submission.NewTemuImageCompletion(original.Owner, saved.Merchant, renewed.Attempt, claim, imageInput, saved.TemuImageObservations[0], model.LocalImageUploaded{Images: []model.LocalUploadedImage{{URL: "https://temu-image.example.org/new.png"}}}, collection.Digest("new synthetic response"))
	require.NoError(t, err)
	latest, err := originals.CompleteOfficial(ctx, proof)
	require.NoError(t, err)
	retainedReceipt, err = originals.ReadOfficial(ctx, original.Owner, original.ID)
	require.NoError(t, err)
	require.Equal(t, collection.Digest(original), collection.Digest(retainedReceipt))
	currentReceipt, err := originals.FindOfficialTarget(ctx, "org-a", original.Target)
	require.NoError(t, err)
	require.Equal(t, latest.ID, currentReceipt.ID)
	require.EqualValues(t, 2, currentReceipt.TemuImage.FenceEpoch)
	var receiptCount int64
	require.NoError(t, db.Table("listing_submission_official_receipts").Count(&receiptCount).Error)
	require.EqualValues(t, 2, receiptCount)
	originalInput := model.LocalImageUpload{FileURL: saved.TemuImageObservations[0].SourceURL, CatID: 34, Usage: 3}
	originalPayload, _ := json.Marshal(originalInput)
	oldReplay, err := kernel.Acquire(ctx, submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: "org-a"}, IntentKey: original.IntentKey, Target: original.Target, Action: submission.TemuLocalImageAction, Payload: originalPayload, ClaimOwnerID: "another-worker", Lease: time.Minute})
	require.NoError(t, err)
	require.True(t, oldReplay.Replayed)
	require.Nil(t, oldReplay.Permit)
	require.Equal(t, original.ID, oldReplay.Attempt.AttemptID)
	productCommand := seedTemuProductIntent(t, db, saved, latest, time.Now().UTC())
	product, err := kernel.Acquire(ctx, productCommand)
	require.NoError(t, err)
	require.NotNil(t, product.Permit)
	blockedImage := seedTemuImageIntent(t, db, saved, "product-pending-image", latest.ID)
	_, err = kernel.Acquire(ctx, blockedImage)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed, "the canonical product fence gates all new image effects")
	productClaim := submission.ExecutionClaim{Scope: submission.ExecutionScope{OrganizationID: "org-a"}, AttemptID: product.Attempt.AttemptID, FenceEpoch: product.Permit.FenceEpoch, OwnerID: product.Permit.ClaimOwnerID, Token: product.Permit.ClaimToken}
	_, err = kernel.MarkUnknown(ctx, productClaim, submission.UnknownResponseLost)
	require.NoError(t, err)
	_, err = kernel.Acquire(ctx, blockedImage)
	require.ErrorIs(t, err, submission.ErrExecutionTargetClaimed, "product UNKNOWN cannot begin image renewal")
	for _, remaining := range []time.Duration{-time.Hour, 5 * time.Second} {
		resetTemuEffectRows(t, db)
		created := time.Now().UTC().Add(-submission.TemuLocalImageLifetime).Add(remaining).Truncate(time.Microsecond)
		original = seedTemuImageSuccess(t, db, saved, created)
		productCommand := seedTemuProductIntent(t, db, saved, original, original.ObservedAt)
		_, err = kernel.Acquire(ctx, productCommand)
		require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired, "a durable intent cannot acquire a product permit after image expiry or without its complete sending window")
	}
	require.NoError(t, db.Exec("DROP INDEX listing_submission_official_permanent_effect").Error)
	_, err = officialstore.NewOfficialRepository(ctx, db)
	require.ErrorIs(t, err, submission.ErrExecutionUnavailable, "serving must reject a missing permanent-effect constraint")
}

func seedTemuProductIntent(t *testing.T, db *gorm.DB, saved record.TargetRecord, image submission.OfficialReceipt, preparedAt time.Time) submission.AcquireExecutionCommand {
	t.Helper()
	payload, refs, err := submission.ResolveTemuPublicationPayload(saved, []submission.OfficialReceipt{image}, preparedAt)
	require.NoError(t, err)
	subject, err := submission.TemuProductSubjectID(saved.Source.Source.ProductKey, saved.Merchant)
	require.NoError(t, err)
	command := submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: "org-a"}, IntentKey: "product-original", Target: submission.ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID, SubjectID: subject}, Action: submission.TemuPublishAction, Payload: payload, ClaimOwnerID: "worker-a", Lease: time.Minute}
	id, _ := uuid.NewV7()
	reservation, err := submission.NewExecutionReservation(command, id.String(), "test-claim", preparedAt)
	require.NoError(t, err)
	body := submission.OfficialIntent{Key: command.IntentKey, Owner: image.Owner, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: saved.Source, Target: command.Target, Binding: saved.Merchant, PayloadFingerprint: reservation.Attempt.PayloadFingerprint, TemuImages: refs, CreatedAt: preparedAt}
	body.InputHash = submission.OfficialIntentInputHash(body)
	require.NoError(t, submission.ValidateOfficialIntent(body))
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO listing_submission_official_intents(organization_id,actor_id,member_id,intent_key,record_id,kind,input_hash,body_hash,body_json) VALUES(?,?,?,?,?,'publish',?,?,?::jsonb)", body.Owner.OrganizationID, body.Owner.ActorID, body.Owner.MemberID, body.Key, body.RecordID, body.InputHash, collection.Digest(body), string(raw)).Error)
	return command
}

func resetTemuEffectRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, table := range []string{effectsstore.TargetFenceTable, "listing_submission_official_receipts", effectsstore.AttemptTable, "listing_submission_official_intents"} {
		require.NoError(t, db.Exec("DELETE FROM "+table).Error)
	}
}

func temuEffectRecordFixture(t *testing.T, ctx context.Context, db, assetDB *gorm.DB) (record.TargetRecord, asset.ApprovedAssetInventory) {
	t.Helper()
	p := func(v int64) *int64 { return &v }
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	now := time.Now().UTC().Add(-9 * 24 * time.Hour).Truncate(time.Microsecond)
	storeID, sourceID, prepID, batchID, itemID, publication := uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString(), uuid.NewString()
	source := preparation.SourceItem{ID: sourceID, PreparationID: prepID, CollectionItemID: itemID, CollectionRevision: 1, Source: collection.Source{ProductKey: "product-a", PublicationID: publication, Version: 1, Kind: "own"}}
	require.NoError(t, db.Create(&catalogstore.SnapshotVersionRecord{TenantID: scope.OrganizationID, ProductKey: "product-a", Version: 1, PublicationID: publication, PayloadHash: collection.Digest("snapshot"), SnapshotJSON: []byte(`{"title":"Fixture"}`)}).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_batches(organization_id,actor_id,member_id,id,name,kind,revision,created_at) VALUES(?,?,?,?,?,'manual',1,?)", scope.OrganizationID, scope.ActorID, scope.MemberID, batchID, "Fixture", now).Error)
	require.NoError(t, db.Exec("INSERT INTO product_collection_items(organization_id,actor_id,member_id,id,batch_id,product_key,publication_id,original_version,source_kind,source_operation_id,revision,created_at) VALUES(?,?,?,?,?,?,?,1,'own','',1,?)", scope.OrganizationID, scope.ActorID, scope.MemberID, itemID, batchID, "product-a", publication, now).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparations(organization_id,actor_id,member_id,id,command_key,input_hash,input_json,source_batch_id,source_revision,name,item_count,revision,created_at) VALUES(?,?,?,?,?,?,'{}',?,1,'Fixture',1,1,?)", scope.OrganizationID, scope.ActorID, scope.MemberID, prepID, uuid.NewString(), collection.Digest("prep"), batchID, now).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparation_sources(organization_id,actor_id,member_id,id,preparation_id,collection_item_id,collection_revision,product_key,publication_id,original_version,source_kind,source_operation_id) VALUES(?,?,?,?,?,?,1,?,?,1,'own','')", scope.OrganizationID, scope.ActorID, scope.MemberID, sourceID, prepID, itemID, "product-a", publication).Error)
	assets, err := assetstore.NewRepository(assetDB)
	require.NoError(t, err)
	approved := asset.ApprovedAsset{ID: "asset-a", RunID: "synthetic-run", PlanRevision: 1, SlotID: "gallery", Attempt: 1, Role: asset.RoleGallery, URL: "https://images.example.org/a.png", Width: 900, Height: 900}
	_, err = assets.CommitApproval(ctx, asset.ApprovalCommit{TenantID: scope.OrganizationID, ProductKey: "product-a", TargetPlatform: "temu", ActionID: "synthetic-approval", SourceSnapshotVersion: 1, Assets: []asset.ApprovedAsset{approved}})
	require.NoError(t, err)
	inventory, err := assets.GetApprovedInventory(ctx, asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: "product-a", TargetPlatform: "temu", SourceSnapshotVersion: 1})
	require.NoError(t, err)
	binding := storecenter.ProductMerchantBinding{Platform: storecenter.PlatformTemu, IdentityPartition: storecenter.PartitionUS, MerchantMode: storecenter.MerchantLocal, RegionID: 211, OrganizationID: scope.OrganizationID, StoreID: storeID, Site: "temu-local", StoreVersion: 1, ConnectionRevision: 1, ApplicationID: "app-a", ApplicationRevision: "v1:temu:US:manual_token:1:r1", SupplierIdentityHash: collection.Digest("merchant"), ScopesHash: collection.Digest("scopes"), VerificationRevision: "official-v1", ServiceExpiresAt: time.Now().UTC().Add(time.Hour), CredentialExpiresAt: time.Now().UTC().Add(time.Hour)}
	input := temugoods.DraftInput{Mode: "local", Selection: model.RuleSelection{CatID: 34, CostTemplateID: "freight-a"}, PackageUnits: rules.LocalPackageUnits{Weight: "lb", Dimension: "in"}, Images: []temugoods.ImageSlot{{Field: "sku", AssetID: approved.ID, Usage: 3}}, Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: "Fixture", CatID: 34}, ServicePromise: model.LocalServicePromise{ShipmentLimitDay: p(1), CostTemplateID: "freight-a"}, SKUs: []model.LocalSKU{{ExternalSKUID: "sku-a", Quantity: p(0), Price: model.LocalSKUPrice{BasePrice: model.LocalMoney{Amount: "2.00", Currency: "USD"}}, PackageInfo: model.LocalPackageInfo{Weight: "0.1", Length: "0.1", Width: "0.1", Height: "0.1"}, Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), Name: "Blue"}}}}}}
	catalog, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	target := catalog.LocalTarget("US", 211)
	snapshot := temugoods.RuleSnapshot{Mode: "local", Selection: input.Selection, LocalTarget: &target, SpecificationRevision: catalog.Revision(), SelectableLeaf: true, ObservedAt: now, FreightTemplates: []model.LocalFreightTemplate{{ID: "freight-a", Name: "Fixture"}}, LocalAttributes: &model.LocalAttributeTemplate{CatID: p(34), Attributes: []model.LocalAttribute{}}, LocalFieldRules: &model.LocalCategoryRules{ExpandCatType: p(1)}, LocalVariations: &model.LocalVariationTemplate{CatID: p(34), Variations: []model.LocalVariation{{ParentSpecID: p(10), Name: "Color", Type: p(1), Required: boolPtr(true)}}}}
	observed := temugoods.ImageObservation{AssetID: approved.ID, SourceURL: approved.URL, CatID: 34, Usage: 3, ContentHash: collection.Digest("exact PNG"), Width: 900, Height: 900, Bytes: 1234, MediaType: "image/png"}
	draft := temugoods.Build(input, snapshot, inventory, []temugoods.ImageObservation{observed})
	require.True(t, draft.ReadyForUpload, draft.Issues)
	saved := record.TargetRecord{ID: uuid.NewString(), TargetID: record.TargetIdentity(scope, sourceID, storeID), Revision: 1, Source: source, EffectiveVersion: 1, Merchant: binding, ProductHash: collection.Digest("product"), InventoryHash: collection.Digest(inventory), RulesHash: temugoods.RuleHash(snapshot), Input: record.TargetInput{SourceID: sourceID, StoreID: storeID, EffectiveVersion: 1, Temu: &input}, Temu: &draft, TemuRuleSnapshot: &snapshot, TemuImageObservations: []temugoods.ImageObservation{observed}, CreatedAt: now}
	raw, err := json.Marshal(saved)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO listing_target_records(organization_id,actor_id,member_id,id,target_id,source_id,store_id,site,revision,input_hash,body_hash,body_json,created_at) VALUES(?,?,?,?,?,?,?,?,1,?,?,?::jsonb,?)", scope.OrganizationID, scope.ActorID, scope.MemberID, saved.ID, saved.TargetID, sourceID, storeID, "temu-local", collection.Digest(saved.Input), collection.Digest(saved), string(raw), now).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparation_targets(organization_id,actor_id,member_id,id,source_id,store_id,site,current_record_id,revision) VALUES(?,?,?,?,?,?,?,?,1)", scope.OrganizationID, scope.ActorID, scope.MemberID, saved.TargetID, sourceID, storeID, "temu-local", saved.ID).Error)
	return saved, inventory
}

func boolPtr(v bool) *bool { return &v }

func seedTemuImageIntent(t *testing.T, db *gorm.DB, saved record.TargetRecord, key, prior string) submission.AcquireExecutionCommand {
	t.Helper()
	image := saved.TemuImageObservations[0]
	raw, err := json.Marshal(model.LocalImageUpload{FileURL: image.SourceURL, CatID: image.CatID, Usage: image.Usage})
	require.NoError(t, err)
	subject, err := submission.TemuLocalImageSubjectID(image.ContentHash, image.CatID, image.Usage, saved.Merchant)
	require.NoError(t, err)
	command := submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: saved.Merchant.OrganizationID}, IntentKey: key, Target: submission.ExecutionTarget{Platform: "temu", StoreID: saved.Merchant.StoreID, SubjectID: subject}, Action: submission.TemuLocalImageAction, Payload: raw, ClaimOwnerID: "worker-a", Lease: time.Minute}
	id, _ := uuid.NewV7()
	reserved, err := submission.NewExecutionReservation(command, id.String(), "test-claim", time.Now().UTC())
	require.NoError(t, err)
	body := submission.OfficialIntent{Key: key, Owner: collection.Scope{OrganizationID: saved.Merchant.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}, Kind: "image", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: saved.Source, Target: command.Target, Binding: saved.Merchant, PayloadFingerprint: reserved.Attempt.PayloadFingerprint, TemuImage: &image, TemuRenewalFrom: prior, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
	body.InputHash = submission.OfficialIntentInputHash(body)
	require.NoError(t, submission.ValidateOfficialIntent(body))
	encoded, err := json.Marshal(body)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO listing_submission_official_intents(organization_id,actor_id,member_id,intent_key,record_id,kind,input_hash,body_hash,body_json) VALUES(?,?,?,?,?,'image',?,?,?::jsonb)", body.Owner.OrganizationID, body.Owner.ActorID, body.Owner.MemberID, key, saved.ID, body.InputHash, collection.Digest(body), string(encoded)).Error)
	return command
}

func seedTemuImageSuccess(t *testing.T, db *gorm.DB, saved record.TargetRecord, created time.Time) submission.OfficialReceipt {
	t.Helper()
	command := seedTemuImageIntent(t, db, saved, "image-original", "")
	id, _ := uuid.NewV7()
	reserved, err := submission.NewExecutionReservation(command, id.String(), "test-claim", created)
	require.NoError(t, err)
	a := reserved.Attempt
	a.FenceEpoch = 1
	observedAt := created.Add(time.Second)
	responseHash := collection.Digest("synthetic image response")
	image := saved.TemuImageObservations[0]
	image.RemoteURLs = []string{"https://temu-image.example.org/a.png"}
	image.ResponseHash = responseHash
	r := submission.OfficialReceipt{ID: a.AttemptID, Owner: collection.Scope{OrganizationID: saved.Merchant.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}, Kind: "image", Target: a.Target, IntentKey: a.IntentKey, PayloadFingerprint: a.PayloadFingerprint, ResponseHash: responseHash, Binding: saved.Merchant, TemuImage: &submission.TemuImageReceipt{Image: image, AttemptCreatedAt: created, FenceEpoch: 1, UseBefore: created.Add(submission.TemuLocalImageLifetime)}, ObservedAt: observedAt}
	require.NoError(t, submission.ValidateOfficialReceipt(r))
	a, err = submission.CompleteClaimedExecution(a, submission.ExecutionEvidence{Kind: submission.EvidenceProviderResponse, Outcome: submission.ExecutionSucceeded, Reference: r.ID, Fingerprint: responseHash, ObservedAt: observedAt}, created.Add(2*time.Second))
	require.NoError(t, err)
	require.NoError(t, db.Table(effectsstore.AttemptTable).Create(map[string]any{"organization_id": a.OrganizationID, "attempt_id": a.AttemptID, "intent_key": a.IntentKey, "platform": a.Target.Platform, "store_id": a.Target.StoreID, "subject_id": a.Target.SubjectID, "action": a.Action, "payload_fingerprint": a.PayloadFingerprint, "provider_execution_key": a.ProviderExecutionKey, "claim_owner_id": a.ClaimOwnerID, "claim_token_hash": reserved.ClaimTokenHash, "fence_epoch": 1, "status": a.Status, "lease_expires_at": a.LeaseExpiresAt, "created_at": a.CreatedAt, "updated_at": a.UpdatedAt, "finished_at": a.FinishedAt, "evidence_kind": a.Evidence.Kind, "evidence_outcome": a.Evidence.Outcome, "evidence_reference": a.Evidence.Reference, "evidence_fingerprint": a.Evidence.Fingerprint, "evidence_observed_at": a.Evidence.ObservedAt}).Error)
	require.NoError(t, db.Table(effectsstore.TargetFenceTable).Create(map[string]any{"organization_id": a.OrganizationID, "platform": a.Target.Platform, "store_id": a.Target.StoreID, "subject_id": a.Target.SubjectID, "epoch": 1, "current_attempt_id": a.AttemptID, "current_status": a.Status, "updated_at": a.UpdatedAt}).Error)
	encoded, err := json.Marshal(r)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO listing_submission_official_receipts(organization_id,actor_id,member_id,attempt_id,intent_key,store_id,subject_id,kind,body_hash,body_json) VALUES(?,?,?,?,?,?,?,'image',?,?::jsonb)", r.Owner.OrganizationID, r.Owner.ActorID, r.Owner.MemberID, r.ID, r.IntentKey, r.Target.StoreID, r.Target.SubjectID, collection.Digest(r), string(encoded)).Error)
	return r
}
