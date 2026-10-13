package submissionpersistence_test

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	record "task-processor/internal/listing/record/target"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
)

func seedTemuSupplierRecord(t *testing.T, db *gorm.DB, original record.TargetRecord) record.TargetRecord {
	t.Helper()
	p := func(v int64) *int64 { return &v }
	saved := original
	saved.ID = uuid.NewString()
	saved.Merchant.StoreID = uuid.NewString()
	saved.Merchant.MerchantMode, saved.Merchant.Site, saved.Merchant.IdentityPartition, saved.Merchant.RegionID = storecenter.MerchantFullyManaged, "temu-supplier", storecenter.PartitionCN, 0
	saved.Merchant.ApplicationRevision = "v1:temu:CN:manual_token:6:r1"
	saved.Merchant.SupplierIdentityHash = collection.Digest("supplier-pa-uniqueId")
	scope := collection.Scope{OrganizationID: saved.Merchant.OrganizationID, ActorID: "actor-a", MemberID: "member-a"}
	saved.TargetID = record.TargetIdentity(scope, saved.Source.ID, saved.Merchant.StoreID)
	in := temugoods.DraftInput{Mode: "fully_managed", Selection: model.RuleSelection{ParentCatID: 12, CatID: 34, SupplierCategoryPath: [10]int64{12, 34}}, Images: []temugoods.ImageSlot{{Field: "material", Usage: 5, AssetID: original.TemuImageObservations[0].AssetID}, {Field: "gallery", Usage: 2, AssetID: original.TemuImageObservations[0].AssetID}, {Field: "thumb", Usage: 3, AssetID: original.TemuImageObservations[0].AssetID}}, Supplier: &model.SupplierGoods{Name: "Explicit", Cat1ID: p(12), Cat2ID: p(34), Cat3ID: p(0), Cat4ID: p(0), Cat5ID: p(0), Cat6ID: p(0), Cat7ID: p(0), Cat8ID: p(0), Cat9ID: p(0), Cat10ID: p(0), SKCs: []model.SupplierSKC{{MainSpecs: []model.SupplierSpecification{{ID: p(0), ParentID: p(0)}}, SKUs: []model.SupplierSKU{{Currency: "CNY", SupplierPrice: p(100), ExternalCode: "sku-a", Packaging: &model.SupplierSKUPackaging{Weight: &model.SupplierWeight{ValueMG: p(1000)}, Volume: &model.SupplierVolume{LengthMM: p(100), WidthMM: p(100), HeightMM: p(100)}, Sensitive: &model.SupplierSensitive{IsSensitive: p(0)}, SensitiveLimits: &model.SupplierSensitiveLimits{}}}}}}}}
	snapshot := *original.TemuRuleSnapshot
	snapshot.Mode, snapshot.Selection = in.Mode, in.Selection
	snapshot.LocalTarget, snapshot.LocalFieldRules, snapshot.LocalAttributes, snapshot.LocalVariations = nil, nil, nil, nil
	nameSource, err := rules.SupplierNameSource()
	require.NoError(t, err)
	snapshot.SpecificationSources = append(snapshot.SpecificationSources, nameSource)
	snapshot.SupplierAttributes = &model.SupplierAttributeTemplate{Properties: []model.SupplierProperty{}}
	snapshot.SupplierSizeRequirement = &model.SupplierSizeRequirement{CatID: 34, Observed: true}
	snapshot.SupplierCategoryPath = []model.SupplierCategory{{CatID: 12, ParentCatID: p(0), CatType: p(777), Level: p(1), Leaf: boolPtr(false), Hidden: boolPtr(false), HiddenType: p(0)}, {CatID: 34, ParentCatID: p(12), CatType: p(777), Level: p(2), Leaf: boolPtr(true), Hidden: boolPtr(false), HiddenType: p(0)}}
	observations := []temugoods.ImageObservation{}
	for _, usage := range []int64{5, 2, 3} {
		o := original.TemuImageObservations[0]
		o.Usage = usage
		observations = append(observations, o)
	}
	inventory := asset.ApprovedAssetInventory{Scope: asset.InventoryScope{TenantID: scope.OrganizationID, ProductKey: saved.Source.Source.ProductKey, TargetPlatform: "temu", SourceSnapshotVersion: saved.EffectiveVersion}, Assets: []asset.ApprovedAsset{{ID: observations[0].AssetID, URL: observations[0].SourceURL}}}
	draft := temugoods.Build(in, snapshot, inventory, observations)
	require.True(t, draft.ReadyForUpload, draft.Issues)
	saved.Input = record.TargetInput{SourceID: saved.Source.ID, StoreID: saved.Merchant.StoreID, EffectiveVersion: saved.EffectiveVersion, Temu: &in}
	saved.Temu, saved.TemuRuleSnapshot, saved.TemuImageObservations = &draft, &snapshot, observations
	saved.RulesHash = temugoods.RuleHash(snapshot)
	raw, err := json.Marshal(saved)
	require.NoError(t, err)
	require.NoError(t, db.Exec("INSERT INTO listing_target_records(organization_id,actor_id,member_id,id,target_id,source_id,store_id,site,revision,input_hash,body_hash,body_json,created_at) VALUES(?,?,?,?,?,?,?,?,1,?,?,?::jsonb,?)", scope.OrganizationID, scope.ActorID, scope.MemberID, saved.ID, saved.TargetID, saved.Source.ID, saved.Merchant.StoreID, saved.Merchant.Site, collection.Digest(saved.Input), collection.Digest(saved), string(raw), saved.CreatedAt).Error)
	require.NoError(t, db.Exec("INSERT INTO listing_preparation_targets(organization_id,actor_id,member_id,id,source_id,store_id,site,current_record_id,revision) VALUES(?,?,?,?,?,?,?,?,1)", scope.OrganizationID, scope.ActorID, scope.MemberID, saved.TargetID, saved.Source.ID, saved.Merchant.StoreID, saved.Merchant.Site, saved.ID).Error)
	return saved
}
