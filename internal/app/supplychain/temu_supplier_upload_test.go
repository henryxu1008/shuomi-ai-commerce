package supplychainapp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"task-processor/internal/listing/submission"
	temugoods "task-processor/internal/marketplace/temu/goods"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/collection"
	"task-processor/internal/storecenter"
	"testing"
)

type supplierUploadProbe struct {
	raw           []byte
	artifactDrift bool
	afterArtifact func()
	readError     error
}

func (p *supplierUploadProbe) ProbeTemu(_ context.Context, a asset.ApprovedAsset, cat, usage int64, _ string) (temugoods.ImageObservation, error) {
	hash := sha256.Sum256(p.raw)
	return temugoods.ImageObservation{AssetID: a.ID, SourceURL: a.URL, CatID: cat, Usage: usage, ContentHash: hex.EncodeToString(hash[:]), Width: 900, Height: 900, Bytes: int64(len(p.raw)), MediaType: "image/png"}, nil
}
func (p *supplierUploadProbe) ReadTemuImage(context.Context, temugoods.ImageObservation, string) ([]byte, error) {
	if p.readError != nil {
		return nil, p.readError
	}
	return append([]byte(nil), p.raw...), nil
}

func TestSupplierSourceDownloadFailureDoesNotClaimExternalImageEffect(t *testing.T) {
	s, scope, records, kernel, _, auth, probe := supplierUploadFixture(t, storecenter.MerchantSemiManaged)
	merchant := &nativeUploadMerchant{binding: records.saved.Merchant}
	s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
	key := uuid.NewString()
	probe.readError = errors.New("source timeout before provider dispatch")
	_, err := s.Upload(context.Background(), scope, key, records.saved.ID)
	require.Error(t, err)
	require.Empty(t, kernel.attempts, "source reads cannot claim an external-effect fence")
	require.Zero(t, merchant.imageCalls)
	require.Zero(t, merchant.productCalls)
	probe.readError = nil
	result, err := s.Upload(context.Background(), scope, key, records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionSucceeded, result.Status)
	require.Equal(t, 1, merchant.imageCalls)
	require.Equal(t, 1, merchant.productCalls)
}
func (p *supplierUploadProbe) ProbeTemuArtifact(_ context.Context, url string) (temugoods.ImageArtifact, error) {
	hash := collection.Digest("reencoded PNG")
	if p.artifactDrift {
		hash = collection.Digest("changed PNG")
	}
	if p.afterArtifact != nil {
		p.afterArtifact()
	}
	return temugoods.ImageArtifact{URL: url, ContentHash: hash, Width: 900, Height: 900, Bytes: 1234, MediaType: "image/png"}, nil
}
func (m *nativeUploadMerchant) UploadSupplierImage(_ context.Context, in model.SupplierImageUpload) (model.SupplierImageUploaded, string, error) {
	m.imageCalls++
	raw, err := base64.StdEncoding.DecodeString(in.Image)
	if err != nil || len(raw) == 0 {
		return model.SupplierImageUploaded{}, "", submission.ErrExecutionInvalid
	}
	return model.SupplierImageUploaded{URL: "https://images.example.org/temu.png"}, collection.Digest("image response"), nil
}
func (m *nativeUploadMerchant) CreateSupplierGoods(_ context.Context, in model.SupplierGoods) (temugoods.CreatedGoods, error) {
	m.productCalls++
	if m.fail {
		return temugoods.CreatedGoods{}, submission.ErrExecutionOutcomeUnknown
	}
	result := model.SupplierGoodsCreated{ProductID: uploadPointer(int64(9007199254740993))}
	for index, skc := range in.SKCs {
		id := int64(index + 100)
		result.SKCs = append(result.SKCs, model.SupplierCreatedSKC{ID: &id})
		for _, sku := range skc.SKUs {
			result.SKUs = append(result.SKUs, model.SupplierCreatedSKU{ID: uploadPointer(int64(1000 + len(result.SKUs))), SKCID: &id, ExternalCode: sku.ExternalCode})
		}
	}
	return temugoods.CreatedGoods{Supplier: &result, ResponseHash: collection.Digest("created supplier")}, nil
}
func supplierUploadFixture(t *testing.T, mode storecenter.OfficialMerchantMode) (*UploadService, collection.Scope, *uploadRecords, *uploadKernel, *uploadOfficial, *uploadAuth, *supplierUploadProbe) {
	t.Helper()
	s, scope, records, kernel, official, auth, _ := nativeUploadFixture(t)
	b := records.saved.Merchant
	b.MerchantMode, b.IdentityPartition, b.Site, b.RegionID = mode, storecenter.PartitionCN, "temu-supplier", 0
	b.ApplicationRevision = "v1:temu:CN:manual_token:6:r1"
	p := uploadPointer[int64]
	in := temugoods.DraftInput{Mode: string(mode), Selection: model.RuleSelection{ParentCatID: 12, CatID: 34, SupplierCategoryPath: [10]int64{12, 34}}, Images: []temugoods.ImageSlot{{Field: "material", AssetID: "asset-a", Usage: 5}, {Field: "gallery", AssetID: "asset-a", Usage: 2}, {Field: "thumb", AssetID: "asset-a", Usage: 3}}, Supplier: &model.SupplierGoods{Name: "Explicit title", Cat1ID: p(12), Cat2ID: p(34), Cat3ID: p(0), Cat4ID: p(0), Cat5ID: p(0), Cat6ID: p(0), Cat7ID: p(0), Cat8ID: p(0), Cat9ID: p(0), Cat10ID: p(0), SKCs: []model.SupplierSKC{{MainSpecs: []model.SupplierSpecification{{ID: p(0), ParentID: p(0)}}, SKUs: []model.SupplierSKU{{ExternalCode: "sku-a", Currency: "CNY", SupplierPrice: p(100), Packaging: &model.SupplierSKUPackaging{Weight: &model.SupplierWeight{ValueMG: p(1000)}, Volume: &model.SupplierVolume{LengthMM: p(100), WidthMM: p(100), HeightMM: p(100)}, SensitiveLimits: &model.SupplierSensitiveLimits{}, Sensitive: &model.SupplierSensitive{IsSensitive: p(0)}}}}}}}}
	if mode == storecenter.MerchantSemiManaged {
		in.Supplier.SemiManaged = &model.SupplierSemiManaged{SiteIDs: []int64{100, 101}}
		sku := &in.Supplier.SKCs[0].SKUs[0]
		sku.SupplierPrice = nil
		sku.SitePrices = []model.SupplierSitePrice{{SiteID: p(100), Price: p(100)}, {SiteID: p(101), Price: p(200)}}
		sku.Stock = &model.SupplierStock{Warehouses: []model.SupplierWarehouseStock{{WarehouseID: "warehouse-a", Target: p(0)}}}
		in.Supplier.Shipment = &model.SupplierShipment{FreightTemplateID: "freight-a", LimitSecond: p(86400)}
	}
	snapshot := *records.saved.TemuRuleSnapshot
	snapshot.Mode, snapshot.Selection = string(mode), in.Selection
	snapshot.LocalTarget, snapshot.LocalAttributes, snapshot.LocalVariations, snapshot.LocalFieldRules = nil, nil, nil, nil
	nameSource, err := rules.SupplierNameSource()
	require.NoError(t, err)
	snapshot.SpecificationSources = append(snapshot.SpecificationSources, nameSource)
	snapshot.SupplierAttributes = &model.SupplierAttributeTemplate{InputMaxSpecNum: p(0), Properties: []model.SupplierProperty{}}
	snapshot.SupplierSizeRequirement = &model.SupplierSizeRequirement{CatID: 34, Observed: true}
	snapshot.SupplierCategoryPath = []model.SupplierCategory{{CatID: 12, ParentCatID: p(0), CatType: p(777), Level: p(1), Leaf: uploadPointer(false), Hidden: uploadPointer(false), HiddenType: p(0)}, {CatID: 34, ParentCatID: p(12), CatType: p(777), Level: p(2), Leaf: uploadPointer(true), Hidden: uploadPointer(false), HiddenType: p(0)}}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 900, 900))))
	probe := &supplierUploadProbe{raw: buf.Bytes()}
	inventory := s.dependencies.Assets.(uploadAssets).inventory
	observed := []temugoods.ImageObservation{}
	for _, usage := range []int64{5, 2, 3} {
		o, e := probe.ProbeTemu(context.Background(), inventory.Assets[0], 34, usage, string(mode))
		require.NoError(t, e)
		observed = append(observed, o)
	}
	draft := temugoods.Build(in, snapshot, inventory, observed)
	require.True(t, draft.ReadyForUpload, draft.Issues)
	records.saved.Merchant, records.saved.Input.Temu, records.saved.TemuRuleSnapshot, records.saved.TemuImageObservations, records.saved.Temu = b, &in, &snapshot, observed, &draft
	records.saved.RulesHash = temugoods.RuleHash(snapshot)
	s.dependencies.TemuImages = probe
	s.dependencies.TemuRules = nativeUploadRules{b, snapshot}
	return s, scope, records, kernel, official, auth, probe
}
func TestSupplierActualConsumerSharesImageFenceAndRetainsProductOutcome(t *testing.T) {
	for _, mode := range []storecenter.OfficialMerchantMode{storecenter.MerchantFullyManaged, storecenter.MerchantSemiManaged} {
		t.Run(string(mode), func(t *testing.T) {
			s, scope, records, _, _, auth, _ := supplierUploadFixture(t, mode)
			merchant := &nativeUploadMerchant{binding: records.saved.Merchant}
			s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
			key := uuid.NewString()
			r, err := s.Upload(context.Background(), scope, key, records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, submission.ExecutionSucceeded, r.Status)
			require.NotNil(t, r.TemuProduct.Supplier)
			require.Equal(t, 1, merchant.imageCalls, "three positions of identical original image/bizType share one PA effect")
			require.Equal(t, 1, merchant.productCalls)
			_, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
			require.NoError(t, err)
			require.Equal(t, 1, merchant.imageCalls)
			require.Equal(t, 1, merchant.productCalls)
		})
	}
}
func TestSupplierArtifactDriftAndUnknownNeverAllowAnotherSend(t *testing.T) {
	s, scope, records, _, _, auth, probe := supplierUploadFixture(t, storecenter.MerchantSemiManaged)
	merchant := &nativeUploadMerchant{binding: records.saved.Merchant, fail: true}
	s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
	r, err := s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, r.Status)
	r, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.NoError(t, err)
	require.Equal(t, submission.ExecutionOutcomeUnknown, r.Status)
	require.Equal(t, 1, merchant.productCalls)
	require.Equal(t, 1, merchant.imageCalls)
	s, scope, records, _, _, auth, probe = supplierUploadFixture(t, storecenter.MerchantFullyManaged)
	merchant = &nativeUploadMerchant{binding: records.saved.Merchant}
	s.dependencies.TemuStores = nativeUploadStore{merchant, auth}
	probe.afterArtifact = func() { probe.artifactDrift = true }
	_, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.Error(t, err)
	require.Equal(t, 1, merchant.imageCalls)
	require.Zero(t, merchant.productCalls)
	_, err = s.Upload(context.Background(), scope, uuid.NewString(), records.saved.ID)
	require.Error(t, err)
	require.Equal(t, 1, merchant.imageCalls)
	require.Zero(t, merchant.productCalls)
}
