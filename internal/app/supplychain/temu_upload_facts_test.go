package supplychainapp

import (
	"context"
	"github.com/stretchr/testify/require"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
	"time"
)

type nativeUploadRules struct {
	binding  storecenter.ProductMerchantBinding
	snapshot temugoods.RuleSnapshot
}

func (r nativeUploadRules) ReadTemuTargetRules(context.Context, collection.Scope, string, model.RuleSelection) (storecenter.ProductMerchantBinding, temugoods.RuleSnapshot, error) {
	return r.binding, r.snapshot, nil
}

type nativeUploadProbe struct {
	drift      bool
	reads      int
	afterProbe func()
}

func TestTemuCurrentFactsRemintOriginalSourceAfterLongImageObservation(t *testing.T) {
	s, scope, records, _, _, _, probe := nativeUploadFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	probe.afterProbe = func() { probe.afterProbe = nil; time.Sleep(5100 * time.Millisecond) }
	_, err := s.temuFacts(ctx, scope, records.saved)
	require.NoError(t, err)
}

func (p *nativeUploadProbe) ProbeTemu(_ context.Context, a asset.ApprovedAsset, cat, usage int64, _ string) (temugoods.ImageObservation, error) {
	p.reads++
	if p.afterProbe != nil {
		p.afterProbe()
	}
	hash := collection.Digest("exact PNG")
	if p.drift {
		hash = collection.Digest("changed PNG")
	}
	return temugoods.ImageObservation{AssetID: a.ID, SourceURL: a.URL, CatID: cat, Usage: usage, ContentHash: hash, Width: 900, Height: 900, Bytes: 1234, MediaType: "image/png"}, nil
}

func nativeUploadFixture(t *testing.T) (*UploadService, collection.Scope, *uploadRecords, *uploadKernel, *uploadOfficial, *uploadAuth, *nativeUploadProbe) {
	t.Helper()
	s, scope, records, _, kernel, official, auth := uploadFixture(t)
	saved := records.saved
	p := func(v int64) *int64 { return &v }
	b := saved.Merchant
	b.Platform = storecenter.PlatformTemu
	b.Site = "temu-local"
	b.IdentityPartition = storecenter.PartitionUS
	b.MerchantMode = storecenter.MerchantLocal
	b.RegionID = 211
	b.ApplicationType = ""
	b.ApplicationRevision = "v1:temu:US:manual_token:1:r1"
	b.VerificationRevision = "official-v1"
	b.ScopesHash = collection.Digest("scopes")
	b.CredentialExpiresAt = time.Now().UTC().Add(time.Hour)
	a := asset.ApprovedAsset{ID: "asset-a", URL: "https://images.example.org/a.png", Role: asset.RoleGallery, Width: 900, Height: 900}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: saved.Source.Source.ProductKey, TargetPlatform: "temu", SourceSnapshotVersion: saved.EffectiveVersion}, Assets: []asset.ApprovedAsset{a}}
	input := temugoods.DraftInput{Mode: "local", Selection: model.RuleSelection{CatID: 34, CostTemplateID: "freight-a"}, PackageUnits: rules.LocalPackageUnits{Weight: "lb", Dimension: "in"}, Images: []temugoods.ImageSlot{{Field: "sku", AssetID: a.ID, Usage: 3}}, Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: "Explicit title", CatID: 34}, ServicePromise: model.LocalServicePromise{ShipmentLimitDay: p(1), CostTemplateID: "freight-a"}, SKUs: []model.LocalSKU{{ExternalSKUID: "sku-a", Quantity: p(0), Price: model.LocalSKUPrice{BasePrice: model.LocalMoney{Amount: "2.00", Currency: "USD"}}, PackageInfo: model.LocalPackageInfo{Weight: "0.1", Length: "0.1", Width: "0.1", Height: "0.1"}, Specifications: []model.LocalSpecificationValue{{ParentSpecID: p(10), Name: "Blue"}}}}}}
	catalog, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	target := catalog.LocalTarget("US", 211)
	required := true
	snapshot := temugoods.RuleSnapshot{Mode: "local", Selection: input.Selection, LocalTarget: &target, SpecificationRevision: catalog.Revision(), SelectableLeaf: true, ObservedAt: time.Now().UTC(), FreightTemplates: []model.LocalFreightTemplate{{ID: "freight-a"}}, LocalAttributes: &model.LocalAttributeTemplate{CatID: p(34), Attributes: []model.LocalAttribute{}}, LocalFieldRules: &model.LocalCategoryRules{ExpandCatType: p(1)}, LocalVariations: &model.LocalVariationTemplate{CatID: p(34), Variations: []model.LocalVariation{{ParentSpecID: p(10), Name: "Color", Type: p(1), Required: &required}}}}
	probe := &nativeUploadProbe{}
	observed, err := probe.ProbeTemu(context.Background(), a, 34, 3, "local")
	require.NoError(t, err)
	draft := temugoods.Build(input, snapshot, inventory, []temugoods.ImageObservation{observed})
	require.True(t, draft.ReadyForUpload, draft.Issues)
	saved.Merchant = b
	saved.Input.Draft = record.TargetInput{}.Draft
	saved.Input.Temu = &input
	saved.Result = record.TargetRecord{}.Result
	saved.ImageObservations = nil
	saved.ProofObservations = nil
	saved.Temu = &draft
	saved.TemuRuleSnapshot = &snapshot
	saved.TemuImageObservations = []temugoods.ImageObservation{observed}
	saved.RulesHash = temugoods.RuleHash(snapshot)
	saved.InventoryHash = collection.Digest(inventory)
	records.saved = saved
	s.dependencies.Assets = uploadAssets{inventory}
	s.dependencies.TemuRules = nativeUploadRules{b, snapshot}
	s.dependencies.TemuImages = probe
	require.True(t, record.PublicationReady(saved))
	return s, scope, records, kernel, official, auth, probe
}

func TestTemuCurrentFactsRejectAssetScopeApprovalContentAndStoreDriftBeforeSend(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	s, scope, records, _, _, auth, probe := nativeUploadFixture(t)
	saved := records.saved
	b, snapshot, input := saved.Merchant, *saved.TemuRuleSnapshot, *saved.Input.Temu
	inventory, err := s.dependencies.Assets.GetApprovedInventory(ctx, asset.InventoryScope{})
	require.NoError(t, err)
	proof, err := s.source(ctx, scope, saved)
	require.NoError(t, err)
	product, err := s.dependencies.Products.ReadEffectiveTargetProduct(ctx, proof, saved.EffectiveVersion, saved.ApplyReceiptID)
	require.NoError(t, err)
	require.Equal(t, saved.ProductHash, collection.Digest(product.Snapshot))
	require.Equal(t, collection.Digest(saved.TemuImageObservations), collection.Digest(record.ProbeTemuTargetImages(context.Background(), probe, input, inventory)))
	_, err = s.temuFacts(ctx, scope, saved)
	require.NoError(t, err)
	changed := inventory
	changed.Scope.TargetPlatform = "shein"
	s.dependencies.Assets = uploadAssets{changed}
	_, err = s.temuFacts(ctx, scope, saved)
	require.ErrorIs(t, err, record.ErrNotReady)
	changed = inventory
	changed.Assets = append([]asset.ApprovedAsset(nil), inventory.Assets...)
	changed.Assets[0].ID = "replacement"
	s.dependencies.Assets = uploadAssets{changed}
	_, err = s.temuFacts(ctx, scope, saved)
	require.ErrorIs(t, err, record.ErrNotReady)
	s.dependencies.Assets = uploadAssets{inventory}
	probe.drift = true
	_, err = s.temuFacts(ctx, scope, saved)
	require.ErrorIs(t, err, record.ErrConflict)
	probe.drift = false
	changedBinding := b
	changedBinding.ConnectionRevision++
	s.dependencies.TemuRules = nativeUploadRules{changedBinding, snapshot}
	_, err = s.temuFacts(ctx, scope, saved)
	require.ErrorIs(t, err, record.ErrConflict)
	s.dependencies.TemuRules = nativeUploadRules{b, snapshot}
	auth.denied = true
	_, err = s.temuFacts(ctx, scope, saved)
	require.ErrorIs(t, err, record.ErrForbidden)
}
