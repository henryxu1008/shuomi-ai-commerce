package rules_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/temu/model"
	"task-processor/internal/marketplace/temu/rules"
)

func variationPointer[T any](v T) *T { return &v }
func variationFixture() (model.LocalVariationTemplate, []model.LocalSKU) {
	return model.LocalVariationTemplate{CatID: variationPointer(int64(34)), Variations: []model.LocalVariation{
			{ParentSpecID: variationPointer(int64(10)), Required: variationPointer(false), Type: variationPointer(int64(0)), Values: []model.LocalVariationValue{{SpecID: variationPointer(int64(9007199254740993)), Name: "Blue"}}},
			{ParentSpecID: variationPointer(int64(11)), Required: variationPointer(true), Type: variationPointer(int64(1))},
		}}, []model.LocalSKU{{Specifications: []model.LocalSpecificationValue{
			{SpecID: variationPointer(int64(9007199254740993))},
			{ParentSpecID: variationPointer(int64(11)), Name: "Small"},
		}}, {Specifications: []model.LocalSpecificationValue{
			{ParentSpecID: variationPointer(int64(11)), Name: "Large"},
			{ParentSpecID: variationPointer(int64(10)), SpecID: variationPointer(int64(9007199254740993)), Name: "Blue"},
		}}}
}

func TestLocalVariationsUseCurrentPresetCustomAndRequiredRules(t *testing.T) {
	template, skus := variationFixture()
	before, err := json.Marshal(skus)
	require.NoError(t, err)
	check := rules.CheckLocalVariations(34, template, skus)
	require.Empty(t, check.Issues)
	after, err := json.Marshal(skus)
	require.NoError(t, err)
	require.Equal(t, before, after, "checking must not fill parent IDs or rename supplied values")
	raw, err := json.Marshal(check)
	require.NoError(t, err)
	require.NotContains(t, string(raw), "ready")

	for _, tc := range []struct {
		name, field, code string
		change            func(*model.LocalVariationTemplate, *[]model.LocalSKU)
	}{
		{"wrong category", "variations", "category_mismatch", func(t *model.LocalVariationTemplate, _ *[]model.LocalSKU) { t.CatID = variationPointer(int64(35)) }},
		{"missing current template", "variations", "template_unverified", func(t *model.LocalVariationTemplate, _ *[]model.LocalSKU) { t.Variations = nil }},
		{"missing required flag", "variations.0", "requirement_unverified", func(t *model.LocalVariationTemplate, _ *[]model.LocalSKU) { t.Variations[0].Required = nil }},
		{"unknown selected type", "skuList.0.specDetails.0", "variation_type_unverified", func(t *model.LocalVariationTemplate, _ *[]model.LocalSKU) {
			t.Variations[0].Type = variationPointer(int64(9))
		}},
		{"no preset to custom fallback", "skuList.0.specDetails.0", "custom_not_allowed", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications[0] = model.LocalSpecificationValue{ParentSpecID: variationPointer(int64(10)), Name: "Blue"}
		}},
		{"foreign preset", "skuList.0.specDetails.0", "preset_not_in_template", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications[0] = model.LocalSpecificationValue{ParentSpecID: variationPointer(int64(10)), SpecID: variationPointer(int64(55))}
		}},
		{"preset name cannot override ID", "skuList.0.specDetails.0", "preset_name_mismatch", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) { (*s)[0].Specifications[0].Name = "Red" }},
		{"custom requires parent", "skuList.0.specDetails.1", "parent_unverified", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications[1].ParentSpecID = nil
		}},
		{"empty custom value", "skuList.0.specDetails.1", "custom_value_missing", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) { (*s)[0].Specifications[1].Name = " " }},
		{"required parent absent", "skuList.0.specDetails", "required_parent_missing", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications = (*s)[0].Specifications[:1]
		}},
		{"duplicate parent", "skuList.0.specDetails.1", "parent_duplicated", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications[1] = (*s)[0].Specifications[0]
		}},
		{"duplicate SKU regardless of order", "skuList.1.specDetails", "sku_variation_duplicated", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) { (*s)[1].Specifications[0].Name = "Small" }},
		{"too many parents", "skuList.0.specDetails", "parent_count_invalid", func(_ *model.LocalVariationTemplate, s *[]model.LocalSKU) {
			(*s)[0].Specifications = append((*s)[0].Specifications, model.LocalSpecificationValue{ParentSpecID: variationPointer(int64(12)), Name: "Other"})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rule, inputs := variationFixture()
			tc.change(&rule, &inputs)
			require.Contains(t, rules.CheckLocalVariations(34, rule, inputs).Issues, rules.LocalVariationIssue{Field: tc.field, Code: tc.code})
		})
	}
}

func TestLocalVariationsRequireOneParentEvenWhenAllOptional(t *testing.T) {
	template, _ := variationFixture()
	template.Variations = template.Variations[:1]
	skus := []model.LocalSKU{{}}
	require.Contains(t, rules.CheckLocalVariations(34, template, skus).Issues, rules.LocalVariationIssue{Field: "skuList.0.specDetails", Code: "parent_count_invalid"})
	skus[0].Specifications = []model.LocalSpecificationValue{{SpecID: variationPointer(int64(9007199254740993))}}
	require.Empty(t, rules.CheckLocalVariations(34, template, skus).Issues)
	template.Variations[0].Type = variationPointer(int64(2))
	skus[0].Specifications = []model.LocalSpecificationValue{{ParentSpecID: variationPointer(int64(10)), Name: "Custom blue"}}
	require.Empty(t, rules.CheckLocalVariations(34, template, skus).Issues)
	for _, inputs := range [][]model.LocalSKU{nil, make([]model.LocalSKU, 501)} {
		require.Contains(t, rules.CheckLocalVariations(34, template, inputs).Issues, rules.LocalVariationIssue{Field: "skuList", Code: "sku_count_invalid"})
	}
	skus[0].Specifications[0].Name = strings.Repeat("x", (1<<20)+1)
	inputs := []model.LocalSKU{skus[0], skus[0]}
	require.Contains(t, rules.CheckLocalVariations(34, template, inputs).Issues, rules.LocalVariationIssue{Field: "skuList", Code: "variation_input_too_large"})
}
