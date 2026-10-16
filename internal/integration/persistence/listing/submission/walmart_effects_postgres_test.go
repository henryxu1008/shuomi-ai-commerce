package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	officialstore "task-processor/internal/integration/persistence/listing/official"
	prepstore "task-processor/internal/integration/persistence/listing/preparation"
	recordstore "task-processor/internal/integration/persistence/listing/record"
	effectsstore "task-processor/internal/integration/persistence/listing/submission"
	assetstore "task-processor/internal/integration/persistence/product/asset"
	catalogstore "task-processor/internal/integration/persistence/product/catalog"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

func TestPostgresWalmartFeedAdmissionAndPartialRepairAreAtomic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	container, e := tcpostgres.Run(ctx, "postgres:16-alpine", tcpostgres.WithDatabase("walmart733"), tcpostgres.WithUsername("postgres"), tcpostgres.WithPassword("synthetic-walmart-test"), tcpostgres.BasicWaitStrategies())
	require.NoError(t, e)
	t.Cleanup(func() { require.NoError(t, container.Terminate(context.Background())) })
	dsn, e := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, e)
	db, e := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	pool, e := db.DB()
	require.NoError(t, e)
	t.Cleanup(func() { _ = pool.Close() })
	pool.SetMaxOpenConns(1)
	for _, install := range []func(*gorm.DB) error{catalogstore.AutoMigrate, collectionstore.InstallSchema, prepstore.InstallSchema, recordstore.InstallSchema, effectsstore.InstallSchema, officialstore.InstallOfficialSchema} {
		require.NoError(t, install(db))
	}

	require.NoError(t, db.Exec("CREATE DATABASE walmart_assets733").Error)
	parsed, e := url.Parse(dsn)
	require.NoError(t, e)
	parsed.Path = "/walmart_assets733"
	assets, e := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, e)
	assetPool, e := assets.DB()
	require.NoError(t, e)
	t.Cleanup(func() { _ = assetPool.Close() })
	require.NoError(t, assetstore.AutoMigrate(assets))
	saved, _ := temuEffectRecordFixture(t, ctx, db, assets)
	scope := collection.Scope{OrganizationID: "org-a", ActorID: "actor-a", MemberID: "member-a"}
	binding := storecenter.ProductMerchantBinding{Platform: storecenter.PlatformWalmart, Site: "walmart-us", OrganizationID: scope.OrganizationID, StoreID: saved.Merchant.StoreID, StoreVersion: 1, ConnectionRevision: 1, ApplicationID: "app", ApplicationRevision: "r1", SupplierIdentityHash: collection.Digest("walmart merchant"), ScopesHash: collection.Digest("scopes"), CredentialExpiresAt: time.Now().Add(time.Hour), ServiceExpiresAt: time.Now().Add(time.Hour), Walmart: storecenter.WalmartMerchantBinding{Market: "US", Environment: "production", GrantID: uuid.NewString(), Generation: 1, GrantVersion: 1}}
	rules := marketplace.RuleSnapshot{Selection: marketplace.RuleSelection{FeedType: marketplace.MatchFeed, Version: "5-test", ProductType: "Shirts"}, Header: marketplace.FeedHeader{BusinessUnit: "WALMART_US", Locale: "en", Version: "5-test"}, Schema: json.RawMessage(`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["MPItemFeedHeader","MPItem"],"properties":{"MPItem":{"type":"array","minItems":1,"items":{"type":"object","required":["Item"]}}}}`)}
	in := marketplace.DraftInput{Selection: rules.Selection, SKUs: []marketplace.SKUInput{{SourceSKUKey: "a", Identifier: marketplace.Identifier{Type: "GTIN", ID: "12345678901234"}, Condition: "New", Price: "10", ShippingWeight: "1"}, {SourceSKUKey: "b", Identifier: marketplace.Identifier{Type: "GTIN", ID: "12345678901235"}, Condition: "New", Price: "20", ShippingWeight: "1"}}}
	in.SKUs = append(in.SKUs, marketplace.SKUInput{SourceSKUKey: "c", Identifier: marketplace.Identifier{Type: "GTIN", ID: "12345678901236"}, Condition: "New", Price: "30", ShippingWeight: "1"})
	saved.Input = record.TargetInput{SourceID: saved.Source.ID, StoreID: binding.StoreID, EffectiveVersion: 1, Walmart: &in}
	saved.Merchant = binding
	saved.Temu = nil
	saved.TemuRuleSnapshot = nil
	saved.TemuImageObservations = nil
	saved.WalmartRuleSnapshot = &rules
	saved.RulesHash = rules.Hash()
	saveRecord := func() {
		draft, e := marketplace.BuildDraft(in, rules, marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, []string{"a", "b", "c"})
		require.NoError(t, e)
		saved.Walmart = &draft
		require.True(t, record.PublicationReady(saved))
		raw, e := json.Marshal(saved)
		require.NoError(t, e)
		require.NoError(t, db.Exec("UPDATE listing_target_records SET site=?,input_hash=?,body_hash=?,body_json=?::jsonb WHERE id=?", binding.Site, collection.Digest(saved.Input), collection.Digest(saved), string(raw), saved.ID).Error)
		require.NoError(t, db.Exec("UPDATE listing_preparation_targets SET site=? WHERE id=?", binding.Site, saved.TargetID).Error)
	}
	saveRecord()
	recordReader, e := recordstore.NewRepository(ctx, db)
	require.NoError(t, e)
	readBack, e := recordReader.ReadTargetHead(ctx, scope, saved.TargetID)
	require.NoError(t, e)
	require.Equal(t, collection.Digest(saved), collection.Digest(readBack))
	repo, e := effectsstore.NewOfficialEffectsRepository(db)
	require.NoError(t, e)
	kernel, e := submission.NewExecutionKernel(repo)
	require.NoError(t, e)
	official, e := officialstore.NewOfficialRepository(ctx, db)
	require.NoError(t, e)
	seed := func() (submission.OfficialIntent, submission.AcquireExecutionCommand) {
		subject, e := submission.WalmartSubjectID(saved.Source.Source.ProductKey, binding.SupplierIdentityHash)
		require.NoError(t, e)
		cmd := submission.AcquireExecutionCommand{Scope: submission.ExecutionScope{OrganizationID: scope.OrganizationID}, IntentKey: uuid.NewString(), Target: submission.ExecutionTarget{Platform: "walmart", StoreID: binding.StoreID, SubjectID: subject}, Action: submission.WalmartSubmitAction, Payload: saved.Walmart.Payload, ClaimOwnerID: "worker", Lease: time.Minute}
		id, _ := uuid.NewV7()
		reservation, e := submission.NewExecutionReservation(cmd, id.String(), "synthetic-token", time.Now())
		require.NoError(t, e)
		intent := submission.OfficialIntent{Key: cmd.IntentKey, Owner: scope, Kind: "publish", RecordID: saved.ID, RecordHash: collection.Digest(saved), Source: saved.Source, Target: cmd.Target, Binding: binding, PayloadFingerprint: reservation.Attempt.PayloadFingerprint, Walmart: &submission.WalmartFeedIntent{Selection: rules.Selection, SKUs: saved.Walmart.SKUs, WireHash: saved.Walmart.WireHash, RepairFrom: in.RepairFrom}, CreatedAt: time.Now().UTC().Truncate(time.Microsecond)}
		intent.InputHash = submission.OfficialIntentInputHash(intent)
		require.NoError(t, submission.ValidateOfficialIntent(intent))
		raw, e := json.Marshal(intent)
		require.NoError(t, e)
		require.NoError(t, db.Exec("INSERT INTO listing_submission_official_intents(organization_id,actor_id,member_id,intent_key,record_id,kind,input_hash,body_hash,body_json) VALUES(?,?,?,?,?,'publish',?,?,?::jsonb)", scope.OrganizationID, scope.ActorID, scope.MemberID, intent.Key, saved.ID, intent.InputHash, collection.Digest(intent), string(raw)).Error)
		return intent, cmd
	}
	intent, cmd := seed()
	acquired, e := kernel.Acquire(ctx, cmd)
	require.NoError(t, e)
	require.NotNil(t, acquired.Permit)
	_, duplicate := seed()
	_, e = kernel.Acquire(ctx, duplicate)
	require.ErrorIs(t, e, submission.ErrExecutionTargetClaimed)
	completion, e := submission.NewWalmartCompletion(scope, saved, intent, acquired.Attempt, submission.ExecutionClaim{Scope: cmd.Scope, AttemptID: acquired.Attempt.AttemptID, FenceEpoch: acquired.Attempt.FenceEpoch, OwnerID: acquired.Permit.ClaimOwnerID, Token: acquired.Permit.ClaimToken}, marketplace.AcceptedFeed{FeedID: "feed@original", CorrelationID: submission.WalmartCorrelationID(intent)})
	require.NoError(t, e)
	receipt, e := official.CompleteOfficial(ctx, completion)
	require.NoError(t, e)
	_, e = kernel.Acquire(ctx, duplicate)
	require.ErrorIs(t, e, submission.ErrExecutionTargetSucceeded, "accepted is not permission to resubmit")
	req, e := official.BeginWalmartRead(ctx, scope, receipt.ID)
	require.NoError(t, e)
	later, e := official.BeginWalmartRead(ctx, scope, receipt.ID)
	require.NoError(t, e)
	full := marketplace.FeedObservation{FeedID: receipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 3, Succeeded: 1, Failed: 2, ObservedAt: time.Now().UTC(), Items: []marketplace.ItemResult{{SKU: saved.Walmart.SKUs[0].SellerSKU, IngestionStatus: "SUCCESS", PublicationStatus: "UNPUBLISHED"}, {SKU: saved.Walmart.SKUs[1].SellerSKU, IngestionStatus: "DATA_ERROR"}, {SKU: saved.Walmart.SKUs[2].SellerSKU, IngestionStatus: "DATA_ERROR"}}}
	latest, e := official.SaveWalmartRead(ctx, scope, later, full)
	require.NoError(t, e)
	require.True(t, latest.Latest)
	stale, e := official.SaveWalmartRead(ctx, scope, req, full)
	require.NoError(t, e)
	require.False(t, stale.Latest)
	in.RepairFrom = receipt.ID
	in.RepairSKUs = []string{saved.Walmart.SKUs[0].SellerSKU}
	saveRecord()
	_, bad := seed()
	_, e = kernel.Acquire(ctx, bad)
	require.ErrorIs(t, e, submission.ErrExecutionTargetSucceeded, "successful SKU cannot be repaired")
	in.RepairSKUs = []string{marketplace.SellerSKU(marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, "b")}
	in.SKUs[1].Price = "21"
	saveRecord()
	repairIntent, repair := seed()
	next, e := kernel.Acquire(ctx, repair)
	require.NoError(t, e)
	require.NotNil(t, next.Permit)
	require.Equal(t, acquired.Attempt.FenceEpoch+1, next.Attempt.FenceEpoch)
	var occupied int64
	require.NoError(t, db.Table("listing_walmart_sku_bindings").Count(&occupied).Error)
	require.EqualValues(t, 3, occupied)

	repairCompletion, e := submission.NewWalmartCompletion(scope, saved, repairIntent, next.Attempt, submission.ExecutionClaim{Scope: repair.Scope, AttemptID: next.Attempt.AttemptID, FenceEpoch: next.Attempt.FenceEpoch, OwnerID: next.Permit.ClaimOwnerID, Token: next.Permit.ClaimToken}, marketplace.AcceptedFeed{FeedID: "feed-b-repair", CorrelationID: submission.WalmartCorrelationID(repairIntent)})
	require.NoError(t, e)
	repairReceipt, e := official.CompleteOfficial(ctx, repairCompletion)
	require.NoError(t, e)

	historical, e := official.BeginWalmartRead(ctx, scope, receipt.ID)
	require.NoError(t, e)
	require.Len(t, historical.TrackedSKUs, 2)
	_, e = official.SaveWalmartRead(ctx, scope, historical, full)
	require.NoError(t, e)
	results, e := official.ReadWalmartResults(ctx, scope, receipt.Target)
	require.NoError(t, e)
	require.Contains(t, results[0].RepairableSKUs, marketplace.SellerSKU(marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, "c"))
	require.NotContains(t, results[0].RepairableSKUs, marketplace.SellerSKU(marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, "b"))
	in.RepairFrom = receipt.ID
	in.RepairSKUs = []string{marketplace.SellerSKU(marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, "c")}
	in.SKUs[2].Price = "31"
	saveRecord()
	_, repairC := seed()
	_, e = kernel.Acquire(ctx, repairC)
	require.Error(t, e, "current accepted Feed without terminal ingestion must block another send")
	repairRead, e := official.BeginWalmartRead(ctx, scope, repairReceipt.ID)
	require.NoError(t, e)
	_, e = official.SaveWalmartRead(ctx, scope, repairRead, marketplace.FeedObservation{FeedID: repairReceipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Succeeded: 1, ObservedAt: time.Now().UTC(), Items: []marketplace.ItemResult{{SKU: repairReceipt.Walmart.SKUs[0].SellerSKU, IngestionStatus: "SUCCESS"}}})
	require.NoError(t, e)

	for range 101 {
		read, e := official.BeginWalmartRead(ctx, scope, repairReceipt.ID)
		require.NoError(t, e)
		_, e = official.SaveWalmartRead(ctx, scope, read, marketplace.FeedObservation{FeedID: repairReceipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Succeeded: 1, ObservedAt: time.Now().UTC(), Items: []marketplace.ItemResult{{SKU: repairReceipt.Walmart.SKUs[0].SellerSKU, IngestionStatus: "SUCCESS"}}})
		require.NoError(t, e)
	}
	visible, e := official.ReadWalmartResults(ctx, scope, receipt.Target)
	require.NoError(t, e)
	visibleReceipts := []string{}
	for _, r := range visible {
		visibleReceipts = append(visibleReceipts, r.ReceiptID)
	}
	require.Contains(t, visibleReceipts, receipt.ID, "frequent reads of repair Feed cannot evict still-owned original SKU facts")
	third, e := kernel.Acquire(ctx, repairC)
	require.NoError(t, e, "unselected original failed SKU remains repairable after another subset succeeds")
	require.NotNil(t, third.Permit)
	retained, e := official.ReadOfficial(ctx, scope, receipt.ID)
	require.NoError(t, e)
	require.Equal(t, receipt, retained, "repair preserves original accepted Feed")
	require.NoError(t, db.Exec(`ALTER TABLE listing_walmart_sku_bindings DROP CONSTRAINT walmart_sku_attempt; ALTER TABLE listing_walmart_sku_bindings ADD CONSTRAINT walmart_sku_attempt CHECK(true)`).Error)
	require.Error(t, officialstore.VerifyOfficialSchema(ctx, db), "same-name non-FK must fail preflight")

}
