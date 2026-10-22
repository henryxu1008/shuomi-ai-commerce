package content

import (
	"context"
	"encoding/json"
	"strings"
	sheinproduct "task-processor/internal/shein/api/product"
	"task-processor/internal/shein/authorizedbrand"
	"testing"
)

func TestCleanupContextPreservesAuthorizedAndSourceBrandRules(t *testing.T) {
	service := NewSensitiveWordServiceInMemory(nil)
	defer service.Close()
	ctx := &CleanupContext{Context: context.Background(), SourceBrand: "Sony", AuthorizedBrand: &authorizedbrand.Resolved{Enabled: true, Name: "Logitech", NameEn: "Logitech"}}
	got := service.SanitizeDisplayTextWithContext(ctx, "Logitech mouse by Sony for Office")
	if !strings.Contains(got, "Logitech") || strings.Contains(strings.ToLower(got), "sony") || !strings.Contains(got, "Office") {
		t.Fatalf("brand/case cleanup changed: %q", got)
	}
	if err := service.ProcessProductData(nil); err != nil {
		t.Fatalf("nil payload: %v", err)
	}
	if err := service.ProcessProductData(&CleanupContext{}); err != nil {
		t.Fatalf("empty payload: %v", err)
	}
	if got := service.SanitizeDisplayTextWithContext(nil, "Simple Office mouse"); got != "Simple Office mouse" {
		t.Fatalf("nil context changed display text: %q", got)
	}
}

func TestCleanupValidationResultPreservesLanguageAndSKCJSON(t *testing.T) {
	raw := []byte(`{"form":"f","form_name":"name","module":"m","messages":["包含敏感词：[OuterForbidden]"],"other_language_message_map":{"en":["包含敏感词：[LanguageForbidden]"]},"skc_error_message_map":{"blue":{"messages":["包含敏感词：[SKCForbidden]"],"otherLanguageMessageMap":{"fr":["包含敏感词：[SKCLanguageForbidden]"]}}}}`)
	var result ValidationResult
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Form != "f" || result.FormName != "name" || result.Module != "m" {
		t.Fatalf("validation metadata lost: %+v", result)
	}
	service := NewSensitiveWordServiceInMemory(nil)
	defer service.Close()
	found := service.extractSensitiveWordsFromValidation([]ValidationResult{result})
	for _, word := range []string{"OuterForbidden", "LanguageForbidden", "SKCForbidden", "SKCLanguageForbidden"} {
		ok := false
		for _, got := range found {
			if got == strings.ToLower(word) {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("missing %q in %v", word, found)
		}
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"other_language_message_map"`) || !strings.Contains(string(encoded), `"otherLanguageMessageMap"`) {
		t.Fatalf("wire keys changed: %s", encoded)
	}
	payload := &sheinproduct.Product{MultiLanguageNameList: []sheinproduct.LanguageContent{{Language: "en", Name: "OuterForbidden Office"}}}
	if !service.HandleValidationErrors(&CleanupContext{ProductData: payload}, []ValidationResult{result}) {
		t.Fatal("validation cleanup did not apply")
	}
	if strings.Contains(payload.MultiLanguageNameList[0].Name, "OuterForbidden") {
		t.Fatalf("payload not cleaned: %+v", payload)
	}
}

func TestInMemoryCleanupUsesExplicitCustomPhrasesWithoutChangingDefaults(t *testing.T) {
	service := NewSensitiveWordServiceInMemory(nil)
	defer service.Close()
	ctx := &CleanupContext{Context: context.Background()}
	input := "Whimsy BPA Free Office curtain"
	if got := service.SanitizeDisplayTextWithContext(ctx, input); got != input {
		t.Fatalf("default cleanup = %q, want custom phrases retained before explicit configuration", got)
	}
	service.AddStaticSensitiveWordsByLanguage("en", []string{"bpa free", "whimsy"})
	got := service.SanitizeDisplayTextWithContext(ctx, input)
	for _, phrase := range []string{"bpa free", "whimsy"} {
		if strings.Contains(strings.ToLower(got), phrase) {
			t.Fatalf("explicit custom cleanup = %q, want %q removed", got, phrase)
		}
	}
	if !strings.Contains(got, "Office curtain") {
		t.Fatalf("custom cleanup lost ordinary display content: %q", got)
	}
}
