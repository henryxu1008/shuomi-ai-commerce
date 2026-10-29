package draft

import (
	"testing"

	"task-processor/internal/product/catalog/canonical"
)

func TestAssemblerBuildCreatesDefaultSKCWhenCanonicalVariantsMissing(t *testing.T) {
	canonical := &canonical.Product{
		Title:       "Minimal Product",
		Description: "fallback variant",
		Attributes: map[string]canonical.Attribute{
			"color": {Value: "Black"},
			"size":  {Value: "One Size"},
			"price": {Value: "19.9"},
		},
		Images: []canonical.Image{{URL: "main.jpg"}},
	}

	pkg := buildLocalPackage(canonical, "US", "en", nil)
	if pkg == nil || pkg.DraftPayload == nil {
		t.Fatalf("expected package with request draft")
	}
	if len(pkg.DraftPayload.SKCList) != 1 {
		t.Fatalf("request draft skc count = %d, want 1", len(pkg.DraftPayload.SKCList))
	}
	if len(pkg.DraftPayload.SKCList[0].SKUList) != 1 {
		t.Fatalf("request draft sku count = %d, want 1", len(pkg.DraftPayload.SKCList[0].SKUList))
	}
	if pkg.DraftPayload.SKCList[0].SKUList[0].SupplierSKU != "DEFAULT-001" {
		t.Fatalf("default supplier sku = %q, want DEFAULT-001", pkg.DraftPayload.SKCList[0].SKUList[0].SupplierSKU)
	}
	if pkg.PreviewPayload == nil || len(pkg.PreviewPayload.SKCList) != 1 {
		t.Fatalf("preview product skc count = %d, want 1", len(pkg.PreviewPayload.SKCList))
	}
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
