package supplychainapp

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	"task-processor/internal/listing/submission"
	goods "task-processor/internal/marketplace/shein/goods"
	marketplace "task-processor/internal/marketplace/tiktokshop"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type tiktokUploadMerchant struct {
	binding    storecenter.ProductMerchantBinding
	calls      []string
	loseGlobal bool
}

func (m *tiktokUploadMerchant) Binding() storecenter.ProductMerchantBinding { return m.binding }
func (m *tiktokUploadMerchant) UploadImage(context.Context, string, string, []byte) (marketplace.UploadedImage, error) {
	m.calls = append(m.calls, "image")
	return marketplace.UploadedImage{URI: "official-uri", URL: "https://images.example.org/official.jpg", Width: 600, Height: 600, UseCase: "MAIN_IMAGE", ResponseHash: collection.Digest("image response")}, nil
}
func (m *tiktokUploadMerchant) CreateGlobal(_ context.Context, input marketplace.CreateGlobalRequest) (marketplace.CreatedGlobalProduct, error) {
	m.calls = append(m.calls, "global")
	if m.loseGlobal {
		return marketplace.CreatedGlobalProduct{}, errors.New("response lost")
	}
	if input.SKUs[0].Price.Amount != "8.00" {
		return marketplace.CreatedGlobalProduct{}, errors.New("global base price changed")
	}
	return marketplace.CreatedGlobalProduct{ID: "global-1", SKUs: []marketplace.CreatedSKU{{ID: "global-sku-1", SellerSKU: "SKU1"}}, ResponseHash: collection.Digest("global response")}, nil
}
func (m *tiktokUploadMerchant) CreateLocal(context.Context, marketplace.CreateLocalRequest) (marketplace.CreatedLocalProduct, error) {
	m.calls = append(m.calls, "local")
	return marketplace.CreatedLocalProduct{ID: "local-1", SKUs: []marketplace.CreatedSKU{{ID: "local-sku-1", SellerSKU: "SKU1"}}, ResponseHash: collection.Digest("local response")}, nil
}
func (m *tiktokUploadMerchant) PublishUS(_ context.Context, id string, input marketplace.PublishGlobalRequest, expected marketplace.GlobalPublishExpectation) (marketplace.GlobalPublishResult, error) {
	m.calls = append(m.calls, "publish")
	if id != "global-1" || len(input.Targets) != 1 || input.Targets[0].Region != "US" || input.Targets[0].SKUs[0].RelatedGlobalSKUID != "global-sku-1" || input.Targets[0].SKUs[0].Price.SalePrice != "10.00" || expected.ShopID != "shop-1" {
		return marketplace.GlobalPublishResult{}, errors.New("wrong US correlation")
	}
	return marketplace.GlobalPublishResult{State: marketplace.RemoteDraft, ShopID: "shop-1", ProductID: "local-1", SKUs: []marketplace.PublishedSKU{{LocalSKUID: "local-sku-1", GlobalSKUID: "global-sku-1"}}, FailureReason: "official_validation_failed_saved_as_draft", Fingerprint: collection.Digest("publication response")}, nil
}

type tiktokUploadStore struct{ merchant *tiktokUploadMerchant }

func (s tiktokUploadStore) TikTokExecutionMerchant(_ context.Context, _ collection.Scope, _ string, _ string, expected *storecenter.ProductMerchantBinding) (TikTokExecutionMerchant, error) {
	if expected != nil && *expected != s.merchant.binding {
		return nil, record.ErrConflict
	}
	return s.merchant, nil
}

type tiktokUploadRules struct {
	binding storecenter.ProductMerchantBinding
	rules   marketplace.RuleSnapshot
}

func (r tiktokUploadRules) ReadTikTokTargetRules(context.Context, collection.Scope, string, marketplace.RuleSelection) (storecenter.ProductMerchantBinding, marketplace.RuleSnapshot, error) {
	return r.binding, r.rules, nil
}

type tiktokUploadProbe struct {
	observation marketplace.ImageObservation
	drift       bool
}

func (p *tiktokUploadProbe) ProbeTikTok(_ context.Context, approved asset.ApprovedAsset, _ string) (marketplace.ImageObservation, error) {
	v := p.observation
	v.AssetID, v.SourceURL = approved.ID, approved.URL
	if p.drift {
		v.ContentHash = collection.Digest("changed image")
	}
	return v, nil
}
func (p *tiktokUploadProbe) ReadTikTokContent(_ context.Context, approved asset.ApprovedAsset, _ string) (marketplace.ImageObservation, []byte, error) {
	v := p.observation
	v.AssetID, v.SourceURL = approved.ID, approved.URL
	if p.drift {
		v.ContentHash = collection.Digest("changed image")
	}
	return v, []byte("approved content"), nil
}

func TestTikTokReusesOneCanonicalImageForEquivalentApprovedAssets(t *testing.T) {
	s, scope, records, merchant, _, _ := tiktokUploadFixture(t, marketplace.GlobalPublishing)
	inventory := s.dependencies.Assets.(uploadAssets).inventory
	second := inventory.Assets[0]
	second.ID = "approved-second"
	second.URL = "https://images.example.org/second.jpg"
	inventory.Assets = append(inventory.Assets, second)
	s.dependencies.Assets = uploadAssets{inventory}
	records.saved.InventoryHash = collection.Digest(inventory)
	observed := records.saved.TikTokImageObservations[0]
	observed.AssetID = second.ID
	observed.SourceURL = second.URL
	records.saved.TikTokImageObservations = append(records.saved.TikTokImageObservations, observed)
	records.saved.Input.TikTok.MainImages = append(records.saved.Input.TikTok.MainImages, second.ID)
	draft := marketplace.BuildDraft(*records.saved.Input.TikTok, *records.saved.TikTokRuleSnapshot, inventory, records.saved.TikTokImageObservations)
	require.True(t, draft.ReadyForUpload, "%v", draft.Issues)
	records.saved.TikTok = &draft
	result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, result.Status)
	require.Equal(t, []string{"image", "global", "publish"}, merchant.calls)
}
func tiktokUploadFixture(t *testing.T, method marketplace.ListingMethod) (*UploadService, collection.Scope, *uploadRecords, *tiktokUploadMerchant, *uploadKernel, *tiktokUploadProbe) {
	s, scope, records, _, kernel, _, _ := uploadFixture(t)
	b := records.saved.Merchant
	b.Platform = storecenter.PlatformTikTokShop
	b.Site = "tiktokshop-us"
	b.ApplicationType = ""
	b.IdentityPartition = ""
	b.MerchantMode = ""
	b.ScopesHash = collection.Digest("granted scopes")
	b.CredentialExpiresAt = time.Now().UTC().Add(time.Hour)
	b.TikTok = storecenter.TikTokMerchantBinding{GrantID: uuid.NewString(), ShopID: "shop-1", GrantGeneration: 1, GrantRevision: 1, BindingRevision: 1}
	yes, no := true, false
	rules := marketplace.RuleSnapshot{Selection: marketplace.RuleSelection{Method: method, CategoryID: "cat"}, Listing: marketplace.ListingRules{Methods: []marketplace.ListingMethod{method}, Fingerprint: collection.Digest("listing rules")}, Category: marketplace.Category{ID: "cat", Leaf: &yes, Permissions: []string{"AVAILABLE"}}, Rules: marketplace.CategoryRules{SizeChart: &marketplace.SizeChartRule{Required: &no, Supported: &no}, PackageDimension: &marketplace.RequiredRule{Required: &no}}, Warehouses: []marketplace.Warehouse{{ID: "us-wh", Type: "SALES_WAREHOUSE", EffectStatus: "ENABLED", SubType: "CB_DIRECT_SHIPPING_WAREHOUSE"}}, GlobalWarehouses: []marketplace.GlobalWarehouse{{ID: "global-wh", Ownership: "SELLER"}}}
	input := marketplace.DraftInput{Selection: rules.Selection, Title: "A test product", Description: "Human approved product description", SaveMode: "LISTING", MainImages: []string{"approved"}, PackageWeight: marketplace.Weight{Value: "0.500", Unit: "KILOGRAM"}, SKUs: []marketplace.SKUInput{{SellerSKU: "SKU1", SalePrice: "10.00", GlobalBasePrice: "8.00", Inventory: []marketplace.InventoryInput{{WarehouseID: "us-wh", Quantity: 5}}, GlobalInventory: []marketplace.GlobalInventoryInput{{WarehouseID: "global-wh", Quantity: 5}}}}}
	if method == marketplace.LocalReplication {
		input.SaveMode = "AS_DRAFT"
		input.SKUs[0].GlobalInventory = nil
		input.SKUs[0].GlobalBasePrice = ""
		rules.GlobalWarehouses = nil
	}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: "product-a", TargetPlatform: "tiktokshop", SourceSnapshotVersion: 1}, Assets: []asset.ApprovedAsset{{ID: "approved", Role: asset.RoleMain, URL: "https://images.example.org/approved.jpg"}}}
	observed := marketplace.ImageObservation{AssetID: "approved", UseCase: "MAIN_IMAGE", SourceURL: inventory.Assets[0].URL, ContentHash: collection.Digest("approved content"), Width: 600, Height: 600, Bytes: 123, MediaType: "image/jpeg"}
	result := marketplace.BuildDraft(input, rules, inventory, []marketplace.ImageObservation{observed})
	require.True(t, result.ReadyForUpload, "%v", result.Issues)
	records.saved.Input = record.TargetInput{SourceID: records.saved.Source.ID, StoreID: b.StoreID, EffectiveVersion: 1, TikTok: &input}
	records.saved.Merchant = b
	records.saved.TikTok = &result
	records.saved.TikTokRuleSnapshot = &rules
	records.saved.TikTokImageObservations = []marketplace.ImageObservation{observed}
	records.saved.Temu = nil
	records.saved.Result = goods.OfficialDraft{}
	records.saved.ImageObservations = nil
	records.saved.ProofObservations = nil
	records.saved.RulesHash = marketplace.RuleHash(rules)
	records.saved.InventoryHash = collection.Digest(inventory)
	merchant := &tiktokUploadMerchant{binding: b}
	probe := &tiktokUploadProbe{observation: observed}
	s.dependencies.TikTokStores = tiktokUploadStore{merchant}
	s.dependencies.TikTokRules = tiktokUploadRules{b, rules}
	s.dependencies.TikTokImages = probe
	s.dependencies.Assets = uploadAssets{inventory}
	return s, scope, records, merchant, kernel, probe
}
func TestTikTokGlobalDraftRetainsAllStagesAndNeverCreatesAgain(t *testing.T) {
	s, scope, records, merchant, _, _ := tiktokUploadFixture(t, marketplace.GlobalPublishing)
	key := uuid.NewString()
	result, err := s.Upload(context.Background(), scope, key, records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, result.Status)
	require.NotNil(t, result.TikTok)
	require.Equal(t, marketplace.RemoteDraft, result.TikTok.Publication.State)
	require.Equal(t, "local-1", result.TikTok.Publication.ProductID)
	require.Equal(t, []string{"image", "global", "publish"}, merchant.calls)
	_, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Len(t, merchant.calls, 3)
}
func TestTikTokUnknownGlobalCreateStopsUSPublishAndNewIntentReplay(t *testing.T) {
	s, scope, records, merchant, _, _ := tiktokUploadFixture(t, marketplace.GlobalPublishing)
	merchant.loseGlobal = true
	result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
	require.Equal(t, []string{"image", "global"}, merchant.calls)
	result, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
	require.Len(t, merchant.calls, 2)
}
func TestTikTokLocalDraftAndActualSendImageDrift(t *testing.T) {
	s, scope, records, merchant, kernel, probe := tiktokUploadFixture(t, marketplace.LocalReplication)
	kernel.onAcquire = func(submission.AcquireExecutionCommand) { probe.drift = true }
	result, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, result.Status)
	require.Empty(t, merchant.calls)
	s, scope, records, merchant, _, _ = tiktokUploadFixture(t, marketplace.LocalReplication)
	result, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, "AS_DRAFT", result.TikTok.LocalSaveMode)
	require.Equal(t, []string{"image", "local"}, merchant.calls)
}
