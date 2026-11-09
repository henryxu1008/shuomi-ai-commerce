package goods

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	model "task-processor/internal/marketplace/shein/model"
	"task-processor/internal/product/asset"
	"testing"
)

func TestSparseVariantSetHasHonestSharedPublishBlocker(t *testing.T) {
	v := func(id int64) model.ProductSKU {
		return model.ProductSKU{SaleAttributes: []model.AttributeValue{{AttributeID: 13, AttributeValueID: &id}}}
	}
	b := officialBuild{result: OfficialDraft{Product: model.PublishProduct{SKCs: []model.ProductSKC{{SaleAttribute: model.AttributeValue{AttributeID: 12}, SKUs: []model.ProductSKU{v(36), v(37)}}, {SaleAttribute: model.AttributeValue{AttributeID: 12}, SKUs: []model.ProductSKU{v(36)}}}}}}
	b.validateVariantMatrix()
	require.Contains(t, b.result.Issues, OfficialIssue{Field: "skc_list.1.sku_list", Code: "variant_set_unconfirmed", Message: "各颜色规格集合不同，平台发布资格待确认；可保留当前草稿"})
}

func TestSparseSetAlonePreventsWireEvenWithCompleteFieldsAndRemoteImages(t *testing.T) {
	input, rules, inventory := officialFixture()
	rules.Attributes.MainAttributeStatus = rulePointer(2)
	rules.Attributes.Attributes[0].Options = append(rules.Attributes.Attributes[0].Options, model.AttributeOption{ID: 35, Show: rulePointer(1)})
	rules.Attributes.Attributes = append(rules.Attributes.Attributes, model.Attribute{ID: 13, Type: rulePointer(1), Show: rulePointer(1), MainLabel: rulePointer(0), Mode: rulePointer(2), Status: rulePointer(3), Options: []model.AttributeOption{{ID: 36, Show: rulePointer(1)}, {ID: 37, Show: rulePointer(1)}}})
	raw, _ := json.Marshal(input.Product.SKCs[0])
	var second model.ProductSKC
	require.NoError(t, json.Unmarshal(raw, &second))
	second.SupplierCode = "second-color"
	second.SaleAttribute.AttributeValueID = rulePointer(int64(35))
	input.Product.SKCs = append(input.Product.SKCs, second)
	oldImages := append([]OfficialImageSlot{}, input.Images...)
	input.Images = nil
	inventory.Assets = append(inventory.Assets, asset.ApprovedAsset{ID: "swatch", URL: "https://images.example.org/swatch.jpg", Width: 80, Height: 80})
	for i := range input.Product.SKCs {
		first := input.Product.SKCs[i].SKUs[0]
		input.Product.SKCs[i].SKUs = nil
		for j := 0; j < 2; j++ {
			sku := first
			sku.SupplierSKU = fmt.Sprintf("SKU-%d-%d", i, j)
			sku.SaleAttributes = []model.AttributeValue{{AttributeID: 13, AttributeValueID: rulePointer(int64(36 + j))}}
			input.Product.SKCs[i].SKUs = append(input.Product.SKCs[i].SKUs, sku)
		}
		for _, image := range oldImages {
			image.SKC = i
			input.Images = append(input.Images, image)
		}
		input.Images = append(input.Images, OfficialImageSlot{Group: "skc", SKC: i, AssetID: "swatch", Type: 6, Sort: 4})
	}
	observe := []OfficialImageObservation{}
	for i, a := range inventory.Assets {
		typ := []int{1, 2, 5, 6}[i]
		observe = append(observe, OfficialImageObservation{AssetID: a.ID, SourceURL: a.URL, Width: a.Width, Height: a.Height, Type: typ, RemoteURL: "https://img.shein.com/" + a.ID + ".jpg", ResponseHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContentHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Bytes: 1000, MediaType: "image/jpeg"})
	}
	linked := func() {
		rules.Linked = nil
		for _, g := range LinkedRuleGroups(input.Product) {
			rules.Linked = append(rules.Linked, model.LinkedRules{GroupID: g.ID, Attributes: []model.LinkedAttributeRule{}})
		}
	}
	linked()
	complete := BuildOfficial(input, rules, inventory, observe)
	require.Empty(t, complete.Issues)
	require.NotEmpty(t, complete.SubmissionPayload)
	input.Product.SKCs[1].SKUs = input.Product.SKCs[1].SKUs[:1]
	linked()
	result := AssessOfficial(input, rules, inventory, observe, nil).Native()
	require.Len(t, result.Issues, 1)
	require.Equal(t, "variant_set_unconfirmed", result.Issues[0].Code)
	require.False(t, result.ReadyForUpload)
	require.Empty(t, result.SubmissionPayload)
	diagnostic, err := AssessOfficial(input, rules, inventory, observe, nil).Diagnostic(officialRequest(t))
	require.NoError(t, err)
	require.NotEmpty(t, diagnostic.OfflineChecks.Blockers)
}

func TestVariantDimensionsAreDefiniteErrorsNotUnconfirmedSets(t *testing.T) {
	for _, main := range []bool{false, true} {
		t.Run(fmt.Sprint(main), func(t *testing.T) {
			id := int64(36)
			attribute := int64(13)
			mainID := int64(12)
			if main {
				mainID = 99
			} else {
				attribute = 99
			}
			b := officialBuild{result: OfficialDraft{Product: model.PublishProduct{SKCs: []model.ProductSKC{{SaleAttribute: model.AttributeValue{AttributeID: 12}, SKUs: []model.ProductSKU{{SaleAttributes: []model.AttributeValue{{AttributeID: 13, AttributeValueID: &id}}}}}, {SaleAttribute: model.AttributeValue{AttributeID: mainID}, SKUs: []model.ProductSKU{{SaleAttributes: []model.AttributeValue{{AttributeID: attribute, AttributeValueID: &id}}}}}}}}}
			b.validateVariantMatrix()
			found := false
			for _, issue := range b.result.Issues {
				if issue.Code == "invalid" {
					found = true
				}
			}
			require.True(t, found, "dimension mismatch is a definite error")
		})
	}
}
