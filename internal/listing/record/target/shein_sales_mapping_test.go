package target

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/listing/completiondraft"
	"task-processor/internal/marketplace/shein/goods"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/collection"
	"testing"
)

func salesFixture() (goods.OfficialDraftInput, catalog.ProductSnapshot, completiondraft.Content, SheinSalesSelection, goods.OfficialRuleSnapshot, asset.ApprovedAssetInventory) {
	p := func(v int) *int { return &v }
	id := func(v int64) *int64 { return &v }
	source := catalog.ProductSnapshot{Variants: []catalog.Variant{
		{SourceID: "black-m", SKU: "BLACK-M", Attributes: []catalog.Attribute{{Name: "颜色", Value: "黑"}, {Name: "尺码", Value: "M"}}},
		{SourceID: "black-l", SKU: "BLACK-L", Attributes: []catalog.Attribute{{Name: "颜色", Value: "黑"}, {Name: "尺码", Value: "L"}}},
		{SourceID: "white-m", SKU: "WHITE-M", Attributes: []catalog.Attribute{{Name: "颜色", Value: "白"}, {Name: "尺码", Value: "M"}}},
	}}
	base := goods.OfficialDraftInput{Product: model.PublishProduct{CategoryID: 123, ProductTypeID: 456, Names: []model.LanguageContent{{Language: "en", Name: "KEEP"}}, SKCs: []model.ProductSKC{{SKUs: []model.ProductSKU{
		{SupplierSKU: "BLACK-M", Stock: []model.ProductStock{{Quantity: 0, WarehouseID: "warehouse"}}, Prices: []model.ProductPrice{{BasePrice: 12.5, Currency: "USD", SubSite: "shein-us"}}},
		{SupplierSKU: "BLACK-L"}, {SupplierSKU: "WHITE-M"},
	}}}}, Images: []goods.OfficialImageSlot{{Group: "spu", AssetID: "common", Type: 1, Sort: 1}, {Group: "sku", SKU: 2, AssetID: "white-only", Type: 1, Sort: 1}}}
	candidate := completiondraft.Content{Structured: &completiondraft.StructuredContent{Variants: []completiondraft.CandidateVariant{{SourceIndex: p(0), Stock: id(0)}, {SourceIndex: p(1)}, {SourceIndex: p(2)}}}}
	selection := SheinSalesSelection{Dimensions: []SheinSalesDimension{{SourceName: "颜色", AttributeID: 12, Main: true}, {SourceName: "尺码", AttributeID: 13}}, Values: []SheinSalesValue{{SourceName: "颜色", SourceValue: "黑", OptionID: 34}, {SourceName: "颜色", SourceValue: "白", OptionID: 35}, {SourceName: "尺码", SourceValue: "M", OptionID: 36}, {SourceName: "尺码", SourceValue: "L", OptionID: 37}}, Rows: []SheinSalesRow{{SourceIndex: 0, SKCIndex: 0, SKUIndex: 0}, {SourceIndex: 1, SKCIndex: 0, SKUIndex: 1}, {SourceIndex: 2, SKCIndex: 0, SKUIndex: 2}}, Images: []SheinSalesImage{{SlotIndex: 0}, {SlotIndex: 1}}}
	rules := goods.OfficialRuleSnapshot{ApplicationMode: model.ModeSelfOperated, Attributes: model.AttributeTemplate{ProductTypeID: 456, MainAttributeStatus: p(3), Attributes: []model.Attribute{
		{ID: 12, Name: "Color", Type: p(1), MainLabel: p(1), Show: p(1), Status: p(3), Mode: p(2), Options: []model.AttributeOption{{ID: 34, Show: p(1)}, {ID: 35, Show: p(1)}}},
		{ID: 13, Name: "Size", Type: p(1), MainLabel: p(0), Show: p(1), Status: p(3), Mode: p(2), Options: []model.AttributeOption{{ID: 36, Show: p(1)}, {ID: 37, Show: p(1)}}},
	}}}
	inventory := asset.ApprovedAssetInventory{Assets: []asset.ApprovedAsset{{ID: "common"}, {ID: "white-only"}}}
	return base, source, candidate, selection, rules, inventory
}

func TestSalesMappingKeepsThreeRowsZeroNullPricesAndImageOwnership(t *testing.T) {
	base, source, candidate, sel, rules, inventory := salesFixture()
	before := collection.Digest([]any{base, source, candidate, inventory})
	result, err := deriveSheinSales(base, source, candidate, sel, rules, inventory)
	require.NoError(t, err)
	require.Len(t, result.Product.SKCs, 2)
	require.Len(t, result.Product.SKCs[0].SKUs, 2)
	require.Len(t, result.Product.SKCs[1].SKUs, 1)
	require.Equal(t, "BLACK-M", result.Product.SKCs[0].SKUs[0].SupplierSKU)
	require.Equal(t, "WHITE-M", result.Product.SKCs[1].SKUs[0].SupplierSKU)
	require.Equal(t, 0, result.Product.SKCs[0].SKUs[0].Stock[0].Quantity)
	require.Empty(t, result.Product.SKCs[0].SKUs[1].Stock)
	require.Empty(t, result.Product.SKCs[1].SKUs[0].Stock)
	require.Equal(t, base.Product.SKCs[0].SKUs[0].Prices, result.Product.SKCs[0].SKUs[0].Prices)
	require.Equal(t, goods.OfficialImageSlot{Group: "sku", SKC: 1, SKU: 0, AssetID: "white-only", Type: 1, Sort: 1}, result.Images[1])
	require.Equal(t, base.Images[0], result.Images[0])
	require.Equal(t, before, collection.Digest([]any{base, source, candidate, inventory}))
}
func TestSalesMappingRejectsAmbiguousRowsValuesAndProtectedFields(t *testing.T) {
	for _, name := range []string{"duplicate row", "missing row", "wrong sku", "missing value", "collision", "stock", "null stock", "skc metadata", "skc image", "placement", "variant approval", "size reference", "sample reference"} {
		t.Run(name, func(t *testing.T) {
			b, s, c, sel, r, inv := salesFixture()
			switch name {
			case "duplicate row":
				sel.Rows[2] = sel.Rows[0]
			case "missing row":
				sel.Rows = sel.Rows[:2]
			case "wrong sku":
				b.Product.SKCs[0].SKUs[2].SupplierSKU = "BLACK-M"
			case "missing value":
				sel.Values = sel.Values[:3]
			case "collision":
				sel.Values[3].OptionID = 36
			case "stock":
				b.Product.SKCs[0].SKUs[0].Stock[0].Quantity = 1
			case "null stock":
				b.Product.SKCs[0].SKUs[1].Stock = []model.ProductStock{{Quantity: 0}}
			case "skc metadata":
				b.Product.SKCs[0].SupplierCode = "cannot-clone"
			case "skc image":
				b.Images[1].Group = "skc"
				b.Images[1].SKU = 0
			case "placement":
				inv.Assets[1].OfficialPlacement = &asset.ImageOfficialPlacement{Group: "sku", SKU: 2, Type: 1, Sort: 1, Site: "shein-us"}
			case "variant approval":
				inv.Assets[1].Presentation = &asset.ImagePresentation{Group: "variant", VariantID: "black-m"}
			case "size reference":
				b.Product.SizeAttributes = []model.SizeAttribute{{AttributeID: 99, RelatedSaleAttributeID: importPtr(int64(77)), RelatedSaleAttributeValueID: importPtr(int64(88))}}
			case "sample reference":
				b.Product.Sample = &model.SampleInfo{Spec: model.SampleSpec{Main: model.SampleMainSpec{AttributeID: 12, ValueID: 999}}}
			}
			_, err := deriveSheinSales(b, s, c, sel, r, inv)
			require.Error(t, err)
			var field *SheinSalesFieldError
			require.ErrorAs(t, err, &field)
			require.NotEmpty(t, field.Field)
		})
	}
}
func TestSalesSelectionRejectsMissingZeroPositions(t *testing.T) {
	for _, raw := range []string{`{"skcIndex":0,"skuIndex":0}`, `{"sourceIndex":null,"skcIndex":0,"skuIndex":0}`, `{"sourceIndex":0,"skcIndex":0,"skuIndex":0,"skuIndex":1}`} {
		var row SheinSalesRow
		require.Error(t, json.Unmarshal([]byte(raw), &row))
	}
	var slot SheinSalesImage
	require.Error(t, json.Unmarshal([]byte(`{}`), &slot))
}
