package supplychainapp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	marketplace "task-processor/internal/marketplace/walmart"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
)

type walmartFlowMerchant struct {
	binding   storecenter.ProductMerchantBinding
	rules     marketplace.RuleSnapshot
	submitted [][]byte
	lost      bool
	occupied  bool
	feed      marketplace.FeedObservation
	products  map[string]marketplace.Product
}

func (m *walmartFlowMerchant) Binding() storecenter.ProductMerchantBinding { return m.binding }
func (m *walmartFlowMerchant) Rules(context.Context, marketplace.RuleSelection) (marketplace.RuleSnapshot, error) {
	return m.rules, nil
}
func (m *walmartFlowMerchant) ProductBySKU(_ context.Context, sku string) (*marketplace.Product, error) {
	if p, ok := m.products[sku]; ok {
		return &p, nil
	}
	if m.occupied {
		return &marketplace.Product{SKU: "existing"}, nil
	}
	return nil, nil
}
func (m *walmartFlowMerchant) FeedPage(context.Context, string, int) (marketplace.FeedObservation, error) {
	if m.feed.FeedID == "" {
		return marketplace.FeedObservation{}, marketplace.ErrUnavailable
	}
	return m.feed, nil
}
func (m *walmartFlowMerchant) SubmitFeed(_ context.Context, d marketplace.Draft, correlation string) (marketplace.AcceptedFeed, error) {
	m.submitted = append(m.submitted, append([]byte(nil), d.Payload...))
	if m.lost {
		return marketplace.AcceptedFeed{}, marketplace.ErrUnknown
	}
	return marketplace.AcceptedFeed{FeedID: "accepted-feed", CorrelationID: correlation}, nil
}

type walmartFlowStores struct{ merchant *walmartFlowMerchant }

func (s walmartFlowStores) WalmartMerchant(_ context.Context, _ collection.Scope, id string, expected *storecenter.ProductMerchantBinding) (WalmartRulesMerchant, error) {
	if id != s.merchant.binding.StoreID || expected != nil && *expected != s.merchant.binding {
		return nil, record.ErrForbidden
	}
	return s.merchant, nil
}
func (s walmartFlowStores) WalmartExecutionMerchant(_ context.Context, _ collection.Scope, id string, expected *storecenter.ProductMerchantBinding) (WalmartExecutionMerchant, error) {
	if id != s.merchant.binding.StoreID || expected == nil || *expected != s.merchant.binding {
		return nil, record.ErrForbidden
	}
	return s.merchant, nil
}

type walmartFlowRecords struct {
	*uploadRecords
	key, hash string
}

func (r *walmartFlowRecords) FindTargetCommand(_ context.Context, _ collection.Scope, key, hash string) (record.TargetReceipt, error) {
	if key != r.key {
		return record.TargetReceipt{}, record.ErrNotFound
	}
	if hash != r.hash {
		return record.TargetReceipt{}, record.ErrConflict
	}
	return record.TargetReceipt{Record: r.saved, Replayed: true}, nil
}
func (r *walmartFlowRecords) SaveTarget(ctx context.Context, p record.TargetPrepared) (record.TargetReceipt, error) {
	_, key, hash, saved, e := p.Read(ctx)
	if e != nil {
		return record.TargetReceipt{}, e
	}
	r.key, r.hash, r.saved = key, hash, saved
	return record.TargetReceipt{Record: saved}, nil
}

func walmartFlowFixture(t *testing.T, full ...bool) (*UploadService, collection.Scope, *walmartFlowRecords, *walmartFlowMerchant, *uploadKernel) {
	t.Helper()
	uploader, scope, records, _, kernel, _, authority := uploadFixture(t)
	b := records.saved.Merchant
	b.Platform = storecenter.PlatformWalmart
	b.Site = "walmart-us"
	b.ApplicationType = ""
	b.ScopesHash = collection.Digest("walmart granted scopes")
	b.CredentialExpiresAt = time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	b.Walmart = storecenter.WalmartMerchantBinding{Environment: "sandbox", Market: "US", GrantID: uuid.NewString(), Generation: 1, GrantVersion: 1}
	source := records.saved.Source
	snapshot := catalog.PublishedSnapshot{Identity: catalog.SnapshotIdentity{TenantID: scope.OrganizationID, ProductKey: source.Source.ProductKey}, PublicationID: source.Source.PublicationID, Version: 1, Snapshot: catalog.ProductSnapshot{Title: "Original", Variants: []catalog.Variant{{SourceID: "original-sku", SKU: "human-sku"}}}}
	collections, e := collection.NewService(uploadCollections{}, authority, uploadUnusedSources{})
	require.NoError(t, e)
	sources := uploadSources{source: source}
	preparations, e := preparation.NewService(sources, collections, authority)
	require.NoError(t, e)
	selector, e := preparation.NewSourceSelector(preparations, collections, sources, uploadCatalog{snapshot})
	require.NoError(t, e)
	selector, e = selector.WithExecution(authority, collection.ExecutionOwnerAuthority{Authorization: authority})
	require.NoError(t, e)
	rules := marketplace.RuleSnapshot{Selection: marketplace.RuleSelection{FeedType: marketplace.MatchFeed, Version: "5-synthetic"}, Header: marketplace.FeedHeader{Locale: "en", Version: "5-synthetic"}, Schema: []byte(`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["MPItem"],"properties":{"MPItem":{"type":"array","minItems":1,"items":{"type":"object","required":["Item"]}}}}`)}
	if len(full) > 0 && full[0] {
		rules.Selection.FeedType = marketplace.FullFeed
		rules.Selection.ProductType = "Shirts"
		rules.Schema = []byte(`{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","required":["MPItem"],"properties":{"MPItem":{"type":"array","minItems":1,"items":{"type":"object","required":["Orderable","Visible"]}}}}`)
	}
	merchant := &walmartFlowMerchant{binding: b, rules: rules}
	stores := walmartFlowStores{merchant}
	ruleReader := WalmartRuleReader{Stores: stores}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: source.Source.ProductKey, TargetPlatform: "walmart", SourceSnapshotVersion: 1}, Assets: []asset.ApprovedAsset{}}
	if rules.Selection.FeedType == marketplace.FullFeed {
		inventory.Assets = []asset.ApprovedAsset{{ID: "approved", Role: asset.RoleMain, URL: "https://images.example.org/approved.jpg"}}
	}
	flowRecords := &walmartFlowRecords{uploadRecords: records}
	targets, e := record.NewTargetService(record.TargetDependencies{Sources: selector, Products: record.OriginalTargetProduct{}, Assets: uploadAssets{inventory}, Rules: uploader.dependencies.Rules, WalmartRules: ruleReader, WalmartImages: walmartFlowProbe{}, Records: flowRecords, Images: uploadProbe{}, Authorizer: authority})
	require.NoError(t, e)
	input := record.TargetInput{SourceID: source.ID, StoreID: b.StoreID, EffectiveVersion: 1, Walmart: &marketplace.DraftInput{Selection: rules.Selection, SKUs: []marketplace.SKUInput{{SourceSKUKey: "original-sku", Identifier: marketplace.Identifier{Type: "GTIN", ID: "12345678901234"}, Condition: "New", Price: "10", ShippingWeight: "1"}}}}
	if rules.Selection.FeedType == marketplace.FullFeed {
		input.Walmart.SKUs[0].Visible = map[string]any{"productName": "New Shirt", "shortDescription": "Accurate description", "mainImageUrl": inventory.Assets[0].URL}
	}
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	receipt, e := targets.Create(ctx, uuid.NewString(), input)
	require.NoError(t, e)
	require.True(t, record.PublicationReady(receipt.Record))
	require.Nil(t, receipt.Record.Temu)
	require.Equal(t, storecenter.PlatformWalmart, receipt.Record.Merchant.Platform)
	uploader.dependencies.Sources = selector
	uploader.dependencies.Assets = uploadAssets{inventory}
	uploader.dependencies.Records = flowRecords
	uploader.dependencies.WalmartRules = ruleReader
	uploader.dependencies.WalmartImages = walmartFlowProbe{}
	uploader.dependencies.WalmartStores = stores
	uploader.dependencies.WalmartReadStores = stores
	return uploader, scope, flowRecords, merchant, kernel
}

func TestWalmartSharedTargetAndUploadRetainExactFeedAndNeverRepeatUnknown(t *testing.T) {
	for _, lost := range []bool{false, true} {
		t.Run(map[bool]string{false: "accepted", true: "lost-response"}[lost], func(t *testing.T) {
			uploader, scope, records, merchant, _ := walmartFlowFixture(t)
			merchant.lost = lost
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			key := uuid.NewString()

			_, se := uploader.source(ctx, scope, records.saved)
			require.NoError(t, se, "source")
			require.NoError(t, uploader.walmartFacts(ctx, scope, records.saved), "facts")
			require.Equal(t, merchant.binding, records.saved.Merchant, "binding")
			result, e := uploader.Upload(ctx, scope, key, records.saved.ID)
			require.NoError(t, e)
			require.Len(t, merchant.submitted, 1)
			require.Equal(t, records.saved.Walmart.Payload, merchant.submitted[0])
			if lost {
				require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
				require.Nil(t, result.Walmart)
			} else {
				require.Equal(t, "accepted-feed", result.Walmart.FeedID)
				require.Contains(t, result.Message, "待查询")
			}
			_, e = uploader.Upload(ctx, scope, key, records.saved.ID)
			require.NoError(t, e)
			_, e = uploader.Upload(ctx, scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, e)
			require.Len(t, merchant.submitted, 1)
			var body map[string]any
			require.NoError(t, json.Unmarshal(merchant.submitted[0], &body))
			require.NotContains(t, body, "Visible")
		})
	}
}

func TestWalmartFinalAuthorityOrRecordDriftStopsDispatch(t *testing.T) {
	for _, change := range []string{"grant", "record", "wire"} {
		t.Run(change, func(t *testing.T) {
			uploader, scope, records, merchant, kernel := walmartFlowFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			kernel.onAcquire = func(submission.AcquireExecutionCommand) {
				switch change {
				case "grant":
					merchant.binding.Walmart.Generation++
				case "record":
					records.saved.Revision++
				case "wire":
					records.saved.Walmart.Payload = []byte(`{}`)
				}
			}
			result, e := uploader.Upload(ctx, scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, e)
			require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
			require.Empty(t, merchant.submitted)
		})
	}
}

type walmartFlowProbe struct{}

func (walmartFlowProbe) ProbeWalmart(_ context.Context, a asset.ApprovedAsset) (marketplace.ImageObservation, error) {
	return marketplace.ImageObservation{AssetID: a.ID, URL: a.URL, ContentHash: collection.Digest("image content"), Width: 600, Height: 600, Bytes: 123, MediaType: "image/jpeg"}, nil
}
func TestWalmartFullSetupUsesSharedTargetImagesAndExplicitProductType(t *testing.T) {
	uploader, scope, records, merchant, _ := walmartFlowFixture(t, true)
	result, e := uploader.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, e)
	require.NotNil(t, result.Walmart)
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(merchant.submitted[0], &body))
	require.Contains(t, string(body["MPItem"]), `"Visible":{"Shirts":`)
	require.Contains(t, string(body["MPItem"]), `"mainImageUrl":"https://images.example.org/approved.jpg"`)
	require.NotContains(t, string(body["MPItem"]), `"Item":`)
}

type walmartFlowResults struct {
	receipt    submission.OfficialReceipt
	records    []submission.WalmartResultRecord
	generation int64
}

func (r *walmartFlowResults) BeginWalmartRead(_ context.Context, _ collection.Scope, id string) (submission.WalmartResultRequest, error) {
	if id != r.receipt.ID {
		return submission.WalmartResultRequest{}, record.ErrNotFound
	}
	r.generation++
	return submission.WalmartResultRequest{Receipt: r.receipt, Generation: r.generation, TrackedSKUs: submission.WalmartExpectedSKUs(r.receipt.Walmart.SKUs)}, nil
}
func (r *walmartFlowResults) SaveWalmartRead(_ context.Context, _ collection.Scope, req submission.WalmartResultRequest, o marketplace.FeedObservation) (submission.WalmartResultRecord, error) {
	v := submission.WalmartResultRecord{ID: uuid.NewString(), ReceiptID: req.Receipt.ID, Generation: req.Generation, Latest: true, Observation: o}
	r.records = append([]submission.WalmartResultRecord{v}, r.records...)
	return v, nil
}
func (r *walmartFlowResults) ReadWalmartResults(context.Context, collection.Scope, submission.ExecutionTarget) ([]submission.WalmartResultRecord, error) {
	return r.records, nil
}
func TestWalmartPublicationSeparatesAcceptedIngestionAndNativePublishedStatus(t *testing.T) {
	uploader, scope, records, merchant, _ := walmartFlowFixture(t)
	ctx := authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: scope.OrganizationID, EffectiveOrganizationID: scope.OrganizationID, UserID: scope.ActorID, EffectiveMemberID: scope.MemberID, TokenExpiresAt: time.Now().Add(time.Minute)})
	accepted, e := uploader.Upload(ctx, scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, e)
	receipt, e := uploader.dependencies.Receipts.ReadOfficial(ctx, scope, accepted.AttemptID)
	require.NoError(t, e)
	results := &walmartFlowResults{receipt: receipt}
	app := Application{Sources: uploader.dependencies.Sources.(*preparation.SourceSelector), Authorization: &uploadAuth{scope: scope}, PublicationReceipts: uploader.dependencies.Receipts, TargetMerchants: &rulesFixture{binding: merchant.binding}, WalmartStatusStores: walmartFlowStores{merchant}, WalmartResults: results}
	sku := records.saved.Walmart.SKUs[0].SellerSKU
	merchant.feed = marketplace.FeedObservation{FeedID: receipt.Walmart.FeedID, FeedStatus: "PROCESSED", Received: 1, Succeeded: 1, Items: []marketplace.ItemResult{{SKU: sku, IngestionStatus: "SUCCESS"}}}
	merchant.products = map[string]marketplace.Product{sku: {SKU: sku, PublicationStatus: "UNPUBLISHED", LifecycleStatus: "ACTIVE"}}
	view, e := app.PublicationStatus(ctx, records.saved.Source.ID, merchant.binding.StoreID)
	require.NoError(t, e)
	require.Equal(t, "SUCCESS", view.WalmartResults[0].Observation.Items[0].IngestionStatus)
	require.Equal(t, "UNPUBLISHED", view.WalmartResults[0].Observation.Items[0].PublicationStatus)
	require.NotNil(t, view.WalmartResults[0].Observation.Items[0].PublicationObservedAt)
	merchant.products[sku] = marketplace.Product{SKU: sku, PublicationStatus: "PUBLISHED", LifecycleStatus: "ACTIVE"}
	view, e = app.PublicationStatus(ctx, records.saved.Source.ID, merchant.binding.StoreID, receipt.ID)
	require.NoError(t, e)
	require.Equal(t, "PUBLISHED", view.WalmartResults[0].Observation.Items[0].PublicationStatus)
	require.Len(t, merchant.submitted, 1, "result reads cannot dispatch another Feed")
	_, e = app.PublicationStatus(ctx, records.saved.Source.ID, merchant.binding.StoreID, uuid.NewString())
	require.ErrorIs(t, e, record.ErrForbidden)
}
