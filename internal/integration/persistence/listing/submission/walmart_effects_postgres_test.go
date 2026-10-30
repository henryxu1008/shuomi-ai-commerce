package submissionpersistence_test

import (
	"context"
	"encoding/json"
	"fmt"
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

type walmartPostgresFixture struct {
	ctx        context.Context
	db         *gorm.DB
	scope      collection.Scope
	binding    storecenter.ProductMerchantBinding
	saved      *record.TargetRecord
	input      *marketplace.DraftInput
	kernel     *submission.ExecutionKernel
	official   *officialstore.OfficialRepository
	seed       func() (submission.OfficialIntent, submission.AcquireExecutionCommand)
	saveRecord func()
	saveRead   func(submission.WalmartResultRequest, marketplace.FeedObservation) (submission.WalmartResultRecord, error)
}

func newWalmartPostgresFixture(t *testing.T, count int) walmartPostgresFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
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
	keys := []string{"a", "b", "c"}
	for n := 3; n < count; n++ {
		key := fmt.Sprint("source-", n)
		keys = append(keys, key)
		in.SKUs = append(in.SKUs, marketplace.SKUInput{SourceSKUKey: key, Identifier: marketplace.Identifier{Type: "GTIN", ID: fmt.Sprintf("%014d", 12345678901234+int64(n))}, Condition: "New", Price: "10", ShippingWeight: "1"})
	}
	saved.Input = record.TargetInput{SourceID: saved.Source.ID, StoreID: binding.StoreID, EffectiveVersion: 1, Walmart: &in}
	saved.Merchant = binding
	saved.Temu = nil
	saved.TemuRuleSnapshot = nil
	saved.TemuImageObservations = nil
	saved.WalmartRuleSnapshot = &rules
	saved.RulesHash = rules.Hash()
	saveRecord := func() {
		draft, e := marketplace.BuildDraft(in, rules, marketplace.SKUNamespace(binding.SupplierIdentityHash, binding.Walmart.GrantID), saved.Source.Source.ProductKey, keys)
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
	saveRead := func(req submission.WalmartResultRequest, observation marketplace.FeedObservation) (submission.WalmartResultRecord, error) {
		if len(observation.ResponseHashes) == 0 {
			observation.ResponseHashes = []string{collection.Digest(observation)}
		}
		if len(req.ReadProgress.Pages) == 0 {
			if err := req.ObserveWalmartPage(req.ReadProgress.StartOffset, observation); err != nil {
				return submission.WalmartResultRecord{}, err
			}
		}
		return official.SaveWalmartRead(ctx, scope, req, observation)
	}
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
	return walmartPostgresFixture{ctx, db, scope, binding, &saved, &in, kernel, official, seed, saveRecord, saveRead}
}

func TestPostgresWalmartFeedAdmissionAndPartialRepairAreAtomic(t *testing.T) {
	f := newWalmartPostgresFixture(t, 3)
	ctx, db, scope, binding, saved, in, kernel, official := f.ctx, f.db, f.scope, f.binding, f.saved, f.input, f.kernel, f.official
	seed, saveRecord, saveRead := f.seed, f.saveRecord, f.saveRead
	var e error
	t.Run("definitive rejection uses original attempt fence and SKU owner", func(t *testing.T) {
		var previousAttempt string
		var previousEpoch int64
		for range 2 {
			intent, cmd := seed()
			acquired, err := kernel.Acquire(ctx, cmd)
			require.NoError(t, err)
			require.NotNil(t, acquired.Permit)
			if previousAttempt != "" {
				require.NotEqual(t, previousAttempt, acquired.Attempt.AttemptID)
				require.Greater(t, acquired.Attempt.FenceEpoch, previousEpoch)
			}
			claim := submission.ExecutionClaim{Scope: cmd.Scope, AttemptID: acquired.Attempt.AttemptID, FenceEpoch: acquired.Attempt.FenceEpoch, OwnerID: acquired.Permit.ClaimOwnerID, Token: acquired.Permit.ClaimToken}
			evidence, err := submission.NewWalmartRejectionEvidence(intent, acquired.Attempt, time.Now())
			require.NoError(t, err)
			failed, err := kernel.Complete(ctx, claim, evidence)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionFailedDefinitive, failed.Status)
			replayed, err := kernel.Acquire(ctx, cmd)
			require.NoError(t, err)
			require.Nil(t, replayed.Permit, "the original command must never send again")
			require.Equal(t, failed, replayed.Attempt)
			_, err = official.ReadOfficial(ctx, scope, failed.AttemptID)
			require.ErrorIs(t, err, submission.ErrExecutionNotFound, "rejection is not Feed acceptance")
			var occupied []effectsstore.WalmartSKUState
			require.NoError(t, db.Where("organization_id=?", scope.OrganizationID).Find(&occupied).Error)
			require.Len(t, occupied, len(saved.Walmart.SKUs))
			for _, sku := range occupied {
				require.Equal(t, saved.Source.Source.ProductKey, sku.ProductKey)
				require.Equal(t, failed.AttemptID, sku.LastAttemptID)
			}
			previousAttempt, previousEpoch = failed.AttemptID, failed.FenceEpoch
		}
	})
	intent, cmd := seed()
	acquired, e := kernel.Acquire(ctx, cmd)
	require.NoError(t, e)
	require.NotNil(t, acquired.Permit)
	_, duplicate := seed()
	_, e = kernel.Acquire(ctx, duplicate)
	require.ErrorIs(t, e, submission.ErrExecutionTargetClaimed)
	completion, e := submission.NewWalmartCompletion(scope, *saved, intent, acquired.Attempt, submission.ExecutionClaim{Scope: cmd.Scope, AttemptID: acquired.Attempt.AttemptID, FenceEpoch: acquired.Attempt.FenceEpoch, OwnerID: acquired.Permit.ClaimOwnerID, Token: acquired.Permit.ClaimToken}, marketplace.AcceptedFeed{FeedID: "feed@original", CorrelationID: submission.WalmartCorrelationID(intent)})
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
	latest, e := saveRead(later, full)
	require.NoError(t, e)
	require.True(t, latest.Latest)
	stale, e := saveRead(req, full)
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

	t.Run("rejected repair retains its owner and admits only the approved payload", func(t *testing.T) {
		claim := submission.ExecutionClaim{Scope: repair.Scope, AttemptID: next.Attempt.AttemptID, FenceEpoch: next.Attempt.FenceEpoch, OwnerID: next.Permit.ClaimOwnerID, Token: next.Permit.ClaimToken}
		evidence, err := submission.NewWalmartRejectionEvidence(repairIntent, next.Attempt, time.Now())
		require.NoError(t, err)
		failed, err := kernel.Complete(ctx, claim, evidence)
		require.NoError(t, err)
		require.Equal(t, submission.ExecutionFailedDefinitive, failed.Status)
		replayed, err := kernel.Acquire(ctx, repair)
		require.NoError(t, err)
		require.Nil(t, replayed.Permit)
		require.Equal(t, failed, replayed.Attempt)
		in.SKUs[1].Price = "22"
		saveRecord()
		_, changed := seed()
		_, err = kernel.Acquire(ctx, changed)
		require.Error(t, err, "a rejected repair cannot silently admit a different payload")
		in.SKUs[1].Price = "21"
		saveRecord()
		repairIntent, repair = seed()
		next, err = kernel.Acquire(ctx, repair)
		require.NoError(t, err)
		require.NotNil(t, next.Permit, "an explicit same-payload repair rechecks the original accepted evidence")
		require.Greater(t, next.Attempt.FenceEpoch, failed.FenceEpoch)
	})

	repairCompletion, e := submission.NewWalmartCompletion(scope, *saved, repairIntent, next.Attempt, submission.ExecutionClaim{Scope: repair.Scope, AttemptID: next.Attempt.AttemptID, FenceEpoch: next.Attempt.FenceEpoch, OwnerID: next.Permit.ClaimOwnerID, Token: next.Permit.ClaimToken}, marketplace.AcceptedFeed{FeedID: "feed-b-repair", CorrelationID: submission.WalmartCorrelationID(repairIntent)})
	require.NoError(t, e)
	repairReceipt, e := official.CompleteOfficial(ctx, repairCompletion)
	require.NoError(t, e)

	historical, e := official.BeginWalmartRead(ctx, scope, receipt.ID)
	require.NoError(t, e)
	require.Len(t, historical.TrackedSKUs, 2)
	_, e = saveRead(historical, full)
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
	_, e = saveRead(repairRead, marketplace.FeedObservation{FeedID: repairReceipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Succeeded: 1, ObservedAt: time.Now().UTC(), Items: []marketplace.ItemResult{{SKU: repairReceipt.Walmart.SKUs[0].SellerSKU, IngestionStatus: "SUCCESS"}}})
	require.NoError(t, e)

	for range 101 {
		read, e := official.BeginWalmartRead(ctx, scope, repairReceipt.ID)
		require.NoError(t, e)
		_, e = saveRead(read, marketplace.FeedObservation{FeedID: repairReceipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Succeeded: 1, ObservedAt: time.Now().UTC(), Items: []marketplace.ItemResult{{SKU: repairReceipt.Walmart.SKUs[0].SellerSKU, IngestionStatus: "SUCCESS"}}})
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
	for range 2 {
		rejectedIntent, err := official.ReadOfficialIntent(ctx, scope, repairC.IntentKey)
		require.NoError(t, err)
		require.Equal(t, receipt.ID, rejectedIntent.Walmart.RepairFrom)
		claim := submission.ExecutionClaim{Scope: repairC.Scope, AttemptID: third.Attempt.AttemptID, FenceEpoch: third.Attempt.FenceEpoch, OwnerID: third.Permit.ClaimOwnerID, Token: third.Permit.ClaimToken}
		evidence, err := submission.NewWalmartRejectionEvidence(rejectedIntent, third.Attempt, time.Now())
		require.NoError(t, err)
		failed, err := kernel.Complete(ctx, claim, evidence)
		require.NoError(t, err)
		_, repairC = seed()
		third, err = kernel.Acquire(ctx, repairC)
		require.NoError(t, err, "consecutive rejected repairs recheck the original ancestor and a different last accepted Feed")
		require.NotNil(t, third.Permit)
		require.Greater(t, third.Attempt.FenceEpoch, failed.FenceEpoch)
	}
	retained, e := official.ReadOfficial(ctx, scope, receipt.ID)
	require.NoError(t, e)
	require.Equal(t, receipt, retained, "repair preserves original accepted Feed")
	require.NoError(t, db.Exec(`ALTER TABLE listing_walmart_sku_bindings DROP CONSTRAINT walmart_sku_attempt; ALTER TABLE listing_walmart_sku_bindings ADD CONSTRAINT walmart_sku_attempt CHECK(true)`).Error)
	require.Error(t, officialstore.VerifyOfficialSchema(ctx, db), "same-name non-FK must fail preflight")

}

func TestPostgresWalmartReadProgressSurvivesReconstructionAndRejectsTampering(t *testing.T) {
	f := newWalmartPostgresFixture(t, 1000)
	intent, command := f.seed()
	acquired, err := f.kernel.Acquire(f.ctx, command)
	require.NoError(t, err)
	claim := submission.ExecutionClaim{Scope: command.Scope, AttemptID: acquired.Attempt.AttemptID, FenceEpoch: acquired.Attempt.FenceEpoch, OwnerID: acquired.Permit.ClaimOwnerID, Token: acquired.Permit.ClaimToken}
	completion, err := submission.NewWalmartCompletion(f.scope, *f.saved, intent, acquired.Attempt, claim, marketplace.AcceptedFeed{FeedID: "large-feed", CorrelationID: submission.WalmartCorrelationID(intent)})
	require.NoError(t, err)
	receipt, err := f.official.CompleteOfficial(f.ctx, completion)
	require.NoError(t, err)
	page := func(offset int) marketplace.FeedObservation {
		out := marketplace.FeedObservation{FeedID: receipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1000, Failed: 1000, ObservedAt: time.Now().UTC(), ResponseHashes: []string{collection.Digest(offset)}}
		for _, sku := range receipt.Walmart.SKUs[offset : offset+50] {
			out.Items = append(out.Items, marketplace.ItemResult{SKU: sku.SellerSKU, IngestionStatus: "DATA_ERROR"})
		}
		return out
	}
	begin := func(offset int) submission.WalmartResultRequest {
		t.Helper()
		request, err := f.official.BeginWalmartRead(f.ctx, f.scope, receipt.ID)
		require.NoError(t, err)
		require.Equal(t, offset, request.ReadProgress.StartOffset)
		return request
	}
	first := begin(0)
	require.NoError(t, f.db.Exec(`CREATE FUNCTION reject_window() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'task-owned save failure'; END $$; CREATE TRIGGER reject_window BEFORE INSERT ON listing_walmart_feed_observations FOR EACH ROW EXECUTE FUNCTION reject_window()`).Error)
	firstPage := page(0)
	_, err = f.saveRead(first, firstPage)
	require.Error(t, err)
	require.NoError(t, f.db.Exec(`DROP TRIGGER reject_window ON listing_walmart_feed_observations; DROP FUNCTION reject_window()`).Error)
	winner := begin(0)
	win, err := f.saveRead(winner, firstPage)
	require.NoError(t, err)
	require.True(t, win.Latest)
	old, err := f.official.SaveWalmartRead(f.ctx, f.scope, first, firstPage)
	require.NoError(t, err)
	require.False(t, old.Latest, "a late window cannot move the committed cursor")
	late, current := begin(50), begin(50)
	currentPage := page(50)
	win, err = f.saveRead(current, currentPage)
	require.NoError(t, err)
	old, err = f.saveRead(late, page(50))
	require.NoError(t, err)
	require.False(t, old.Latest)
	view, err := f.official.ReadWalmartResults(f.ctx, f.scope, receipt.Target)
	require.NoError(t, err)
	require.Equal(t, win.ID, view[0].ID)
	require.Len(t, view[0].ObservedPages, 1)
	require.True(t, view[0].ObservedPages[0].Observation.ObservedAt.Equal(firstPage.ObservedAt))
	_ = begin(100) // Process stops after Begin and never saves this generation.
	f.official, err = officialstore.NewOfficialRepository(f.ctx, f.db)
	require.NoError(t, err)
	interrupted := begin(100)
	_, err = f.official.SaveWalmartRead(f.ctx, f.scope, interrupted, marketplace.FeedObservation{FeedID: receipt.Walmart.FeedID, Partial: true, ObservedAt: time.Now().UTC()})
	require.NoError(t, err, "a bounded read with no HTTP response preserves its anchor")
	var finalRequest submission.WalmartResultRequest
	var finalPage marketplace.FeedObservation
	var final submission.WalmartResultRecord
	for offset := 100; offset < 1000; offset += 50 {
		finalRequest, finalPage = begin(offset), page(offset)
		final, err = f.saveRead(finalRequest, finalPage)
		require.NoError(t, err)
	}
	replayed, err := f.official.SaveWalmartRead(f.ctx, f.scope, finalRequest, finalPage)
	require.NoError(t, err)
	require.Equal(t, final.ID, replayed.ID, "a lost save response reads the original immutable window")
	view, err = f.official.ReadWalmartResults(f.ctx, f.scope, receipt.Target)
	require.NoError(t, err)
	require.Len(t, view[0].ObservedPages, 19)
	require.True(t, view[0].Observation.Partial)
	require.Empty(t, view[0].RepairableSKUs, "a full set collected across queries never grants repair")
	fresh := begin(0)
	require.Equal(t, fresh.Generation, fresh.ReadProgress.Round)
	_, err = f.official.SaveWalmartRead(f.ctx, f.scope, fresh, marketplace.FeedObservation{FeedID: receipt.Walmart.FeedID, FeedStatus: "ERROR", Received: 1000, Failed: 1000, ObservedAt: time.Now().UTC()})
	require.Error(t, err, "a read without an HTTP page cannot manufacture a complete no-effect Feed result")
	require.NoError(t, f.db.Exec(`UPDATE listing_walmart_feed_observations SET body_json=jsonb_set(body_json,'{readProgress,nextOffset}','999'::jsonb) WHERE id=?`, final.ID).Error)
	_, err = f.official.ReadWalmartResults(f.ctx, f.scope, receipt.Target)
	require.Error(t, err, "projection hashes the entire persisted typed document")
	_, err = f.official.BeginWalmartRead(f.ctx, f.scope, receipt.ID)
	require.Error(t, err, "resumption rejects altered progress")
	_, err = f.official.SaveWalmartRead(f.ctx, f.scope, finalRequest, finalPage)
	require.Error(t, err, "save replay also validates the entire stored document")
}
