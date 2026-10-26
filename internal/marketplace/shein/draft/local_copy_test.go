package draft

import (
	"context"
	"strings"
	"testing"

	"task-processor/internal/product/catalog/canonical"
)

func TestBuildSheinListingCopyKeepsStructuredEnglishTitle(t *testing.T) {
	canonical := &canonical.Product{
		Title: "Flannel non-slip floor mat",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Flannel Non-slip Floor Mat"},
		},
	}

	copy := buildLocalCopy(nil, canonical, canonical.Title, nil)
	if copy.Title != "Flannel Non-slip Floor Mat" {
		t.Fatalf("title = %q, want clean structured title", copy.Title)
	}
	if copy.TitleDiagnostics == nil || copy.TitleDiagnostics.PromptContaminated {
		t.Fatalf("title diagnostics = %+v, want uncontaminated", copy.TitleDiagnostics)
	}
}

func TestBuildSheinListingCopySanitizesPromptLikeTitleWithRules(t *testing.T) {
	canonical := &canonical.Product{
		Title: "Flannel non-slip floor mat",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Flannel non-slip floor mat - Please design an image that can be printed on my non-slip floor mat. The image should include suitable English text and graphics, and the graphics and text should have a 3D visual effect. Please ensure it does not infringe on copyright. 3000 pixels * 2"},
		},
	}

	copy := buildLocalCopy(nil, canonical, canonical.Title, nil)
	if copy.Title != "Flannel non-slip floor mat" {
		t.Fatalf("title = %q, want sanitized base title", copy.Title)
	}
	if copy.SKCTitleBase != "Flannel non-slip floor mat" {
		t.Fatalf("skc base title = %q, want sanitized short title", copy.SKCTitleBase)
	}
	if copy.TitleDiagnostics == nil || !copy.TitleDiagnostics.PromptContaminated || copy.TitleDiagnostics.Source != "prompt_extracted_rule" {
		t.Fatalf("title diagnostics = %+v, want prompt_extracted_rule contamination", copy.TitleDiagnostics)
	}
}

func TestBuildSheinListingCopyDoesNotFallbackWhenPromptTitleNeedsLLMButLLMUnavailable(t *testing.T) {
	canonical := &canonical.Product{
		Title: "Flannel non-slip floor mat",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Please design an image for my floor mat with floral artwork and inspirational text, 3000 pixels"},
			"material":             {Value: "polyester"},
		},
	}

	copy := buildLocalCopy(nil, canonical, canonical.Title, nil)
	if copy.Title != "" {
		t.Fatalf("title = %q, want empty when prompt-like title cannot be safely resolved", copy.Title)
	}
	if copy.SKCTitleBase != "" {
		t.Fatalf("skc base title = %q, want empty when prompt-like title cannot be safely resolved", copy.SKCTitleBase)
	}
	if copy.TitleDiagnostics == nil {
		t.Fatal("title diagnostics = nil, want prompt contamination diagnostics")
	}
	if copy.TitleDiagnostics.Source != "unresolved_prompt_title" {
		t.Fatalf("title diagnostics source = %q, want unresolved_prompt_title", copy.TitleDiagnostics.Source)
	}
	if !copy.TitleDiagnostics.PromptContaminated {
		t.Fatalf("title diagnostics = %+v, want prompt contamination preserved", copy.TitleDiagnostics)
	}
	if !strings.Contains(copy.TitleDiagnostics.ResolutionNote, "llm title extraction unavailable") {
		t.Fatalf("resolution note = %q, want llm extraction unavailability note", copy.TitleDiagnostics.ResolutionNote)
	}
}

func TestBuildSheinListingCopyKeepsLongStructuredTitleWithoutLLMAddition(t *testing.T) {
	canonical := &canonical.Product{
		Title: "Flannel non-slip floor mat",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Flannel Non-slip Floor Mat"},
			"picture_request":      {Value: "Please design a floral print with inspirational words, 3000 pixels"},
		},
	}

	copy := buildLocalCopy(nil, canonical, canonical.Title, nil)
	if copy.Title != "Flannel Non-slip Floor Mat" {
		t.Fatalf("title = %q, want original long structured title", copy.Title)
	}
}

func TestBuildSheinListingCopyCleansSensitiveWords(t *testing.T) {
	canonical := &canonical.Product{
		Title:       "Amazon Trademark Vase",
		Description: "Amazon Trademark vase for home decor.",
		Attributes: map[string]canonical.Attribute{
			"product_english_name": {Value: "Amazon Trademark Vase"},
		},
	}

	copy := buildLocalCopy(context.Background(), canonical, canonical.Title, nil)

	assertNoSensitivePhrase(t, copy.Title, "title")
	assertNoSensitivePhrase(t, copy.Description, "description")
	assertNoSensitivePhrase(t, copy.SKCTitleBase, "skc base")
}

func TestBuildSheinListingCopyDoesNotFallbackToSyntheticCustomTitle(t *testing.T) {
	canonical := &canonical.Product{
		Title: "门帘",
		Attributes: map[string]canonical.Attribute{
			"material":        {Value: "polyester"},
			"picture_request": {Value: "帮我设计一个印在门帘上的图案，图案要有英文跟图案，元素多样，图片有3d视觉效果，摇滚风格，2277 × 4500px"},
		},
	}

	copy := buildLocalCopy(nil, canonical, canonical.Title, nil)
	if copy.Title != "" {
		t.Fatalf("title = %q, want empty instead of synthetic custom fallback", copy.Title)
	}
	if copy.SKCTitleBase != "" {
		t.Fatalf("skc base title = %q, want empty instead of synthetic custom fallback", copy.SKCTitleBase)
	}
}

func assertNoSensitivePhrase(t *testing.T, value string, field string) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Fatalf("%s is empty after cleanup", field)
	}
	normalized := strings.ToLower(value)
	for _, phrase := range []string{"amazon", "trademark"} {
		if strings.Contains(normalized, phrase) {
			t.Fatalf("%s = %q, want %q removed", field, value, phrase)
		}
	}
}
