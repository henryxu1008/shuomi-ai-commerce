package goods

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
	"task-processor/internal/product/asset"
)

func draftPointer[T any](v T) *T { return &v }

func TestIncompleteTemuDraftRetainsExplicitFactsWithoutProducingSubmission(t *testing.T) {
	input := DraftInput{Mode: "local", Selection: model.RuleSelection{CatID: 34}, Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: "Explicit title", CatID: 34}, SKUs: []model.LocalSKU{{Quantity: draftPointer(int64(0)), Price: model.LocalSKUPrice{BasePrice: model.LocalMoney{Amount: "1.23", Currency: "EUR"}}}}}}
	snapshot := RuleSnapshot{Mode: "local", Selection: input.Selection, ObservedAt: time.Now(), SpecificationRevision: "revision-a"}
	result := Build(input, snapshot, asset.ApprovedAssetInventory{}, nil)
	require.False(t, result.ReadyForUpload)
	require.Empty(t, result.SubmissionPayload)
	require.NotEmpty(t, result.Issues)
	require.NotNil(t, result.Local.SKUs[0].Quantity)
	require.EqualValues(t, 0, *result.Local.SKUs[0].Quantity)
	require.Equal(t, "1.23", result.Local.SKUs[0].Price.BasePrice.Amount)
	input.Local.Basic.Name = "Changed"
	require.Equal(t, "Explicit title", result.Local.Basic.Name)
	input.Local.SKUs[0].Quantity = nil
	require.NotNil(t, result.Local.SKUs[0].Quantity)
}

func TestTemuDraftDoesNotOpenGeorgiaOrArmeniaFromCategoryOrNumericalFacts(t *testing.T) {
	catalog, err := rules.NewOfficialCatalog()
	require.NoError(t, err)
	for _, region := range []int64{75, 10} {
		target := catalog.LocalTarget("EU", region)
		input := DraftInput{Mode: "local", Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Name: "Explicit title", CatID: 34}}, Selection: model.RuleSelection{CatID: 34}}
		result := Build(input, RuleSnapshot{Mode: "local", Selection: input.Selection, SelectableLeaf: true, LocalTarget: &target, ObservedAt: time.Now(), SpecificationRevision: catalog.Revision()}, asset.ApprovedAssetInventory{}, nil)
		require.False(t, result.ReadyForUpload)
		require.Contains(t, issueCodes(result), "local_specifications_missing")
		require.Empty(t, result.SubmissionPayload)
	}
}

func TestTemuDraftRejectsMixedModeBodiesAndUnapprovedImageUrls(t *testing.T) {
	input := DraftInput{Mode: "local", Local: &model.LocalGoodsV2{Basic: model.LocalGoodsBasic{Gallery: &model.LocalGallery{CarouselImages: []string{"https://unapproved.example/image.jpg"}}}}, Supplier: &model.SupplierGoods{Name: "Mixed"}}
	result := Build(input, RuleSnapshot{Mode: "local"}, asset.ApprovedAssetInventory{}, nil)
	require.Contains(t, issueCodes(result), "mode_payload_mismatch")
	require.False(t, result.ReadyForUpload)
	input.Supplier = nil
	result = Build(input, RuleSnapshot{Mode: "local"}, asset.ApprovedAssetInventory{}, nil)
	require.Contains(t, issueCodes(result), "unapproved_image_url")
	require.False(t, result.ReadyForUpload)
}

func TestTemuSupplierDraftUsesModeSpecificPriceFieldsWithoutDefaultCurrency(t *testing.T) {
	input := DraftInput{Mode: "fully_managed", Supplier: &model.SupplierGoods{SKCs: []model.SupplierSKC{{SKUs: []model.SupplierSKU{{SupplierPrice: draftPointer(int64(1234))}}}}}}
	result := Build(input, RuleSnapshot{Mode: "fully_managed"}, asset.ApprovedAssetInventory{}, nil)
	require.Contains(t, issueCodes(result), "currency_invalid")
	require.Empty(t, result.Supplier.SKCs[0].SKUs[0].Currency)
	input.Mode = "semi_managed"
	input.Supplier.SemiManaged = &model.SupplierSemiManaged{SiteIDs: []int64{100}}
	result = Build(input, RuleSnapshot{Mode: "semi_managed"}, asset.ApprovedAssetInventory{}, nil)
	require.Contains(t, issueCodes(result), "mode_field_forbidden")
	require.False(t, result.ReadyForUpload)
}

func issueCodes(value Draft) []string {
	out := []string{}
	for _, issue := range value.Issues {
		out = append(out, issue.Code)
	}
	return out
}

func TestSupplierTitleHasNoInvented250CharacterOfficialLimit(t *testing.T) {
	name := strings.Repeat("A", 320)
	input := DraftInput{Mode: "fully_managed", Supplier: &model.SupplierGoods{Name: name, Cat1ID: draftPointer(int64(999))}}
	result := Build(input, RuleSnapshot{Mode: "fully_managed"}, asset.ApprovedAssetInventory{}, nil)
	require.NotContains(t, issueCodes(result), "title_invalid")
	require.Equal(t, name, result.Supplier.Name)
	require.False(t, result.ReadyForUpload)
}

func TestSupplierDraftChecksOfficialAttributeIDsAndExplicitPackagingFacts(t *testing.T) {
	input := DraftInput{Mode: "fully_managed", Supplier: &model.SupplierGoods{Name: "Explicit", Properties: []model.SupplierBasicProperty{{VID: draftPointer(int64(999)), PID: draftPointer(int64(1)), TemplatePID: draftPointer(int64(2)), RefPID: draftPointer(int64(3)), Name: "Material", Value: "Cotton"}}, SKCs: []model.SupplierSKC{{SKUs: []model.SupplierSKU{{Currency: "CNY", SupplierPrice: draftPointer(int64(10))}}}}}}
	template := model.SupplierAttributeTemplate{Properties: []model.SupplierProperty{{PID: draftPointer(int64(1)), TemplatePID: draftPointer(int64(2)), RefPID: draftPointer(int64(3)), Name: "Material", Required: draftPointer(true), IsSale: draftPointer(false), ControlType: draftPointer(int64(1)), ChooseMaxNum: draftPointer(int64(1)), Values: []model.SupplierPropertyValue{{VID: draftPointer(int64(9007199254740993)), Value: "Cotton"}}}}}
	result := Build(input, RuleSnapshot{Mode: "fully_managed", SupplierAttributes: &template}, asset.ApprovedAssetInventory{}, nil)
	require.Contains(t, issueCodes(result), "supplier_attribute_value_unverified")
	require.Contains(t, issueCodes(result), "supplier_packaging_missing")
	input.Supplier.Properties[0].VID = draftPointer(int64(9007199254740993))
	result = Build(input, RuleSnapshot{Mode: "fully_managed", SupplierAttributes: &template}, asset.ApprovedAssetInventory{}, nil)
	require.NotContains(t, issueCodes(result), "supplier_attribute_value_unverified")
	require.False(t, result.ReadyForUpload)
	require.Empty(t, result.SubmissionPayload)
}
