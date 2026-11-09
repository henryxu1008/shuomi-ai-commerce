package target

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/collection"
	"testing"
)

func TestColorSplitMappingRequiresExplicitPositions(t *testing.T) {
	for _, raw := range []string{`{"skcIndex":0,"skuIndex":0}`, `{"sourceIndex":null,"skcIndex":0,"skuIndex":0}`, `{"sourceIndex":0,"skcIndex":0}`, `{"sourceIndex":0,"sourceIndex":1,"skcIndex":0,"skuIndex":0}`} {
		var value ColorSKUMapping
		require.Error(t, json.Unmarshal([]byte(raw), &value))
	}
	var image ColorSharedImage
	require.Error(t, json.Unmarshal([]byte(`{"colorValueIds":[34]}`), &image))
}

func colorDraftFixture() (goods.OfficialDraftInput, completiondraft.Content, ColorSplitSelection) {
	id := func(v int64) *int64 { return &v }
	zero := 0
	stockZero := int64(0)
	sku := func(code string, size int64, stock []model.ProductStock) model.ProductSKU {
		return model.ProductSKU{SupplierSKU: code, SaleAttributes: []model.AttributeValue{{AttributeID: 13, AttributeValueID: id(size)}}, Prices: []model.ProductPrice{{Currency: "USD", BasePrice: 12.5}}, Stock: stock}
	}
	base := goods.OfficialDraftInput{Product: model.PublishProduct{SKCs: []model.ProductSKC{
		{SaleAttribute: model.AttributeValue{AttributeID: 12, AttributeValueID: id(34)}, SKUs: []model.ProductSKU{sku("BLACK-M", 36, []model.ProductStock{{Quantity: 0}}), sku("BLACK-L", 37, []model.ProductStock{})}},
		{SaleAttribute: model.AttributeValue{AttributeID: 12, AttributeValueID: id(35)}, SKUs: []model.ProductSKU{sku("WHITE-M", 36, []model.ProductStock{})}},
	}}, Images: []goods.OfficialImageSlot{{Group: "spu", AssetID: "common", Sort: 1, Type: 1}, {Group: "sku", SKC: 1, SKU: 0, AssetID: "white-only", Sort: 1, Type: 1}}}
	candidate := completiondraft.Content{Structured: &completiondraft.StructuredContent{Variants: []completiondraft.CandidateVariant{{SourceIndex: &zero, Stock: &stockZero}}}}
	candidate.Structured.Variants = append(candidate.Structured.Variants, completiondraft.CandidateVariant{SourceIndex: func() *int { v := 1; return &v }()}, completiondraft.CandidateVariant{SourceIndex: func() *int { v := 2; return &v }()})
	selection := ColorSplitSelection{ColorAttributeID: 12, SKUs: []ColorSKUMapping{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 1, SKUIndex: 0}}, SharedImages: []ColorSharedImage{{SlotIndex: 0, ColorValueIDs: []int64{34, 35}}}}
	return base, candidate, selection
}

func TestColorSplitKeepsOnlySelectedMembershipAndImageOwnership(t *testing.T) {
	base, candidate, selection := colorDraftFixture()
	before := collection.Digest(base)
	groups, err := deriveColorDrafts(base, candidate, selection)
	require.NoError(t, err)
	require.Len(t, groups, 2)
	require.Equal(t, before, collection.Digest(base))
	require.Len(t, groups[0].Draft.Product.SKCs[0].SKUs, 2)
	require.Len(t, groups[1].Draft.Product.SKCs[0].SKUs, 1)
	require.Equal(t, 0, groups[0].Draft.Product.SKCs[0].SKUs[0].Stock[0].Quantity)
	require.Empty(t, groups[0].Draft.Product.SKCs[0].SKUs[1].Stock)
	require.Empty(t, groups[1].Draft.Product.SKCs[0].SKUs[0].Stock)
	require.Len(t, groups[0].Draft.Images, 1)
	require.Len(t, groups[1].Draft.Images, 2)
	require.Equal(t, 0, groups[1].Draft.Images[1].SKC)
	require.Equal(t, "white-only", groups[1].Draft.Images[1].AssetID)
	require.Equal(t, 12.5, groups[1].Draft.Product.SKCs[0].SKUs[0].Prices[0].BasePrice)
}
func TestColorSplitRejectsDuplicateOrMissingSourceRows(t *testing.T) {
	base, candidate, selection := colorDraftFixture()
	selection.SKUs[2] = selection.SKUs[0]
	_, err := deriveColorDrafts(base, candidate, selection)
	require.Error(t, err)
}
func TestColorIdentityIsStableAndSeparateFromStandard(t *testing.T) {
	scope := collection.Scope{OrganizationID: "org", ActorID: "actor", MemberID: "member"}
	source, store := uuid.NewString(), uuid.NewString()
	family := ColorFamilyIdentity(scope, source, store, 12)
	black := ColorTargetIdentity(scope, family, ColorPartitionIdentity(family, 12, 34))
	require.NotEqual(t, TargetIdentity(scope, source, store), black)
	require.NotEqual(t, black, ColorTargetIdentity(scope, family, ColorPartitionIdentity(family, 12, 35)))
	require.Equal(t, family, ColorFamilyIdentity(scope, source, store, 12))
}

func TestColorEditRejectsTheOtherPartitionsExclusiveImage(t *testing.T) {
	base, candidate, selection := colorDraftFixture()
	groups, err := deriveColorDrafts(base, candidate, selection)
	require.NoError(t, err)
	family := ColorSplitReceipt{Records: []TargetRecord{}}
	for i, g := range groups {
		family.Records = append(family.Records, TargetRecord{TargetID: []string{"black", "white"}[i], Input: TargetInput{Draft: g.Draft, ColorPartition: &ColorPartition{Members: g.Members, ColorAttributeID: 12, ColorValueID: g.ColorValueID}}})
	}
	input := family.Records[0].Input
	input.TargetID = "black"
	input.ExpectedRevision = 1
	require.NoError(t, ValidateColorEdit(family, input, TargetRecord{Input: TargetInput{Draft: base}}))
	input.Draft.Images = append(input.Draft.Images, goods.OfficialImageSlot{Group: "sku", AssetID: "white-only", Type: 1, Sort: 1})
	require.Error(t, ValidateColorEdit(family, input, TargetRecord{Input: TargetInput{Draft: base}}))
}

func TestColorSplitRejectsUnmatchedImagePosition(t *testing.T) {
	base, candidate, selection := colorDraftFixture()
	base.Images = append(base.Images, goods.OfficialImageSlot{Group: "sku", SKC: 8, AssetID: "wrong", Type: 1, Sort: 1})
	_, err := deriveColorDrafts(base, candidate, selection)
	require.Error(t, err)
}

func TestColorEditKeepsParentSKUAndOmittedColorImageOwnership(t *testing.T) {
	base, candidate, selection := colorDraftFixture()
	base.Images = append(base.Images, goods.OfficialImageSlot{Group: "sku", SKC: 0, SKU: 0, AssetID: "black-m-only", Type: 1, Sort: 1}, goods.OfficialImageSlot{Group: "skc", SKC: 1, AssetID: "white-omitted-swatch", Type: 6, Sort: 1})
	groups, err := deriveColorDrafts(base, candidate, selection)
	require.NoError(t, err)
	family := ColorSplitReceipt{}
	for i, g := range groups {
		family.Records = append(family.Records, TargetRecord{TargetID: []string{"black", "white"}[i], Input: TargetInput{Draft: g.Draft, ColorPartition: &ColorPartition{Members: g.Members, ColorAttributeID: 12, ColorValueID: g.ColorValueID}}})
	}
	for _, slot := range []goods.OfficialImageSlot{{Group: "sku", SKU: 1, AssetID: "black-m-only", Type: 1, Sort: 2}, {Group: "spu", AssetID: "white-omitted-swatch", Type: 1, Sort: 2}} {
		in := family.Records[0].Input
		in.TargetID = "black"
		in.ExpectedRevision = 1
		in.Draft.Images = append(append([]goods.OfficialImageSlot{}, in.Draft.Images...), slot)
		require.Error(t, ValidateColorEdit(family, in, TargetRecord{Input: TargetInput{Draft: base}}))
	}
}
