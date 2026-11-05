package draft

import (
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/marketplace/shein/workspace"
	"task-processor/internal/product/catalog/canonical"
)

func TestAssemblerBuildKeepsMissingVariantsVisibleInDraftAndReadiness(t *testing.T) {
	canonical := &canonical.Product{
		Title:       "Minimal Product",
		Description: "Product description",
		Attributes: map[string]canonical.Attribute{
			"color": {Value: "Black"},
			"size":  {Value: "One Size"},
			"price": {Value: "19.9"},
		},
		Images: []canonical.Image{{URL: "main.jpg"}},
	}

	pkg := buildLocalPackage(canonical, "US", "en", nil)
	require.Equal(t, "Minimal Product", pkg.SpuName)
	require.Empty(t, pkg.SkcList)
	require.NotNil(t, pkg.DraftPayload)
	require.Empty(t, pkg.DraftPayload.SKCList)
	require.Empty(t, pkg.DraftPayload.SupplierCode)
	require.NotNil(t, pkg.PreviewPayload)
	require.Empty(t, pkg.PreviewPayload.SKCList)
	require.Contains(t, workspace.BuildSubmitPayloadReadinessChecks(pkg, "publish"), workspace.BuildSubmitReadinessCheck(
		"variants", "规格结构", false, "当前还没有完整的 SKC/SKU 结构，提交前需要至少一个 SKC 和一个 SKU",
		[]string{"shein.skc_list", "shein.draft_payload.skc_list"}, "补充规格", false,
	))
}

func TestAssemblerUsesOnlySourceSKUPackageMeasurements(t *testing.T) {
	product := &canonical.Product{
		Title: "Measured product", Description: "Product description",
		Specifications: &canonical.ProductSpecs{
			Weight:     &canonical.Weight{Value: 800, Unit: "g"},
			Dimensions: &canonical.Dimensions{Length: 80, Width: 70, Height: 60, Unit: "cm"},
		},
		Variants: []canonical.Variant{
			{SKU: "missing"},
			{SKU: "measured", Weight: &canonical.Weight{Value: 120, Unit: "g"}, Dimensions: &canonical.Dimensions{Length: 10, Width: 9, Height: 8, Unit: "cm"}},
		},
	}
	pkg := buildLocalPackage(product, "US", "en", nil)
	skus := pkg.DraftPayload.SKCList[0].SKUList
	require.Len(t, skus, 2)
	require.Zero(t, skus[0].Weight)
	require.Empty(t, skus[0].Length)
	require.Empty(t, skus[0].Width)
	require.Empty(t, skus[0].Height)
	require.EqualValues(t, 120, skus[1].Weight)
	require.Equal(t, "10", skus[1].Length)
	require.Contains(t, pkg.ReviewNotes, "部分 SKU 缺少来源包装重量或尺寸，请核对原商品规格")
	check := workspace.BuildManualNotesReadinessCheck(pkg)
	require.False(t, check.OK)
	require.True(t, check.WarningOnly)
}

func TestAssemblerBuildMarks1688LLMOnlyFactsForReview(t *testing.T) {
	canonical := &canonical.Product{
		Title:         "1688 Product",
		Description:   "Scraped description",
		SellingPoints: []string{"LLM-only waterproof claim"},
		FieldTraces: map[string]canonical.FieldTrace{
			"title": {
				Sources: []canonical.Source{
					{Type: canonical.SourceProductURL, Detail: "https://detail.1688.com/offer/123.html"},
					{Type: canonical.SourceScrapedData, Detail: "scraped title"},
				},
			},
			"selling_points": {
				Sources: []canonical.Source{
					{Type: canonical.SourceProductURL, Detail: "https://detail.1688.com/offer/123.html"},
					{Type: canonical.SourceScrapedData, Detail: "normalized from product page: https://detail.1688.com/offer/123.html"},
					{Type: canonical.SourceLLM, Detail: "LLM-generated product normalization"},
				},
				NeedsReview: true,
			},
		},
		Images: []canonical.Image{{URL: "main.jpg"}},
	}

	pkg := buildLocalPackage(canonical, "US", "en", nil)

	if pkg == nil || pkg.Metadata == nil {
		t.Fatal("expected package metadata")
	}
	if pkg.Metadata["source_platform"] != "1688" {
		t.Fatalf("source_platform = %q, want 1688", pkg.Metadata["source_platform"])
	}
	if pkg.Metadata["source_fact_review_required"] != "true" {
		t.Fatalf("source_fact_review_required = %q, want true", pkg.Metadata["source_fact_review_required"])
	}
	if pkg.Metadata["source_fact_review_fields"] != "selling_points" {
		t.Fatalf("source_fact_review_fields = %q, want selling_points", pkg.Metadata["source_fact_review_fields"])
	}
}

func TestAssemblerBuildDoesNotPropagatePromptTextIntoSKCNames(t *testing.T) {
	canonical := &canonical.Product{
		Title: "Flannel non-slip floor mat",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Flannel non-slip floor mat - Please design an image that can be printed on my non-slip floor mat. The image should include suitable English text and graphics, and the graphics and text should have a 3D visual effect. Please ensure it does not infringe on copyright. 3000 pixels * 2"},
		},
		VariantDimensions: []canonical.ScrapedVariantDimension{{Name: "color", Values: []string{"Black"}}},
		Variants: []canonical.Variant{
			{SKU: "Flannel non-slip floor mat - Black", Attributes: map[string]canonical.Attribute{"color": {Value: "Black"}}},
		},
		Images: []canonical.Image{{URL: "main.jpg"}},
	}

	pkg := buildLocalPackage(canonical, "US", "en", nil)
	if pkg.ProductNameEn != "Flannel non-slip floor mat" {
		t.Fatalf("product title = %q, want sanitized title", pkg.ProductNameEn)
	}
	if len(pkg.SkcList) != 1 {
		t.Fatalf("skc count = %d, want 1", len(pkg.SkcList))
	}
	if pkg.SkcList[0].SkcName != "Flannel non-slip floor mat - Black" {
		t.Fatalf("skc title = %q, want sanitized short title", pkg.SkcList[0].SkcName)
	}
	if pkg.TitleDiagnostics == nil || !pkg.TitleDiagnostics.PromptContaminated {
		t.Fatalf("title diagnostics = %+v, want contamination recorded", pkg.TitleDiagnostics)
	}
}
