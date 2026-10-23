package content

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"
	sheinproduct "task-processor/internal/shein/api/product"
	"task-processor/internal/shein/authorizedbrand"
	"testing"
	"time"
)

func TestCleanupContextPreservesAuthorizedAndSourceBrandRules(t *testing.T) {
	service := NewSensitiveWordServiceInMemory(nil)
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

func TestSensitiveWordMutationsCompleteMetadataSynchronously(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*SensitiveWordService)
		touch  bool
	}{
		{"auto dynamic", func(s *SensitiveWordService) { s.AddDynamicSensitiveWords([]string{"OfficeForbidden"}) }, true},
		{"auto static", func(s *SensitiveWordService) { s.AddStaticSensitiveWords([]string{"OfficeForbidden"}) }, true},
		{"auto duplicate", func(s *SensitiveWordService) { s.AddStaticSensitiveWords([]string{"Existing"}) }, false},
		{"auto empty", func(s *SensitiveWordService) { s.AddDynamicSensitiveWords(nil) }, false},
		{"explicit static duplicate", func(s *SensitiveWordService) { s.AddStaticSensitiveWordsByLanguage("en", []string{"Existing"}) }, true},
		{"explicit dynamic", func(s *SensitiveWordService) { s.AddDynamicSensitiveWordsByLanguage("fr", []string{"OfficeForbidden"}) }, true},
		{"explicit empty", func(s *SensitiveWordService) { s.AddDynamicSensitiveWordsByLanguage("en", nil) }, false},
		{"clear empty", func(s *SensitiveWordService) { s.ClearDynamicSensitiveWords() }, true},
		{"validation duplicate", func(s *SensitiveWordService) {
			if !s.HandleValidationErrors(nil, []ValidationResult{{Messages: []string{"包含敏感词：[Existing]"}}}) {
				t.Error("validation cleanup must still request retry")
			}
		}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Isolate the in-memory mutation contract from background lifecycle and IO.
			before := time.Unix(1, 0).UTC()
			s := &SensitiveWordService{
				config: &SensitiveWordConfig{StaticWords: map[string][]string{"en": {"existing"}}, DynamicWords: map[string][]string{}, LastUpdated: before, Version: "before"},
				logger: componentLogger(nil, "shein.sensitive_word"),
			}
			started := time.Now()
			tc.mutate(s)
			finished := time.Now()
			s.mutex.RLock()
			updated, version := s.config.LastUpdated, s.config.Version
			s.mutex.RUnlock()
			if tc.touch {
				if updated.Before(started) || updated.After(finished) || version != "1.0" {
					t.Fatalf("mutation returned before metadata was updated: %v / %q", updated, version)
				}
			} else if !updated.Equal(before) || version != "before" {
				t.Fatalf("no-op mutation changed metadata: %v / %q", updated, version)
			}
		})
	}
}

func TestSensitiveWordMemoryContractAndDefensiveCopies(t *testing.T) {
	s := NewSensitiveWordServiceInMemory(nil)
	wantDefaults := map[string][]string{
		"en": {"amazon", "ebay", "alibaba", "aliexpress", "walmart", "target", "brand", "trademark", "copyright", "patent", "licensed"},
		"zh": {"亚马逊", "淘宝", "天猫", "京东", "拼多多", "品牌", "商标", "版权", "专利", "授权"},
		"ja": {"アマゾン", "楽天", "ヤフー", "ブランド", "商標", "著作権", "特許"},
		"ko": {"아마존", "쿠팡", "네이버", "브랜드", "상표", "저작권", "특허"},
	}
	if !reflect.DeepEqual(s.config.StaticWords, wantDefaults) || len(s.config.DynamicWords) != 0 || s.config.Version != "1.0" || s.config.LastUpdated.IsZero() {
		t.Fatalf("default in-memory contract changed: %+v", s.config)
	}
	raw, err := json.Marshal(s.config)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if len(wire) != 4 || wire["static_words"] == nil || wire["dynamic_words"] == nil || wire["last_updated"] == nil || wire["version"] == nil {
		t.Fatalf("configuration JSON contract changed: %s", raw)
	}
	var decoded SensitiveWordConfig
	if err := json.Unmarshal(raw, &decoded); err != nil || !decoded.LastUpdated.Equal(s.config.LastUpdated) || decoded.Version != "1.0" {
		t.Fatalf("metadata JSON round trip failed: %+v, %v", decoded, err)
	}
	s.AddDynamicSensitiveWords([]string{"  OfficeForbidden ", "officeforbidden", "запрещено"})
	s.AddDynamicSensitiveWordsByLanguage("fr", []string{" Interdit ", "interdit"})
	s.AddStaticSensitiveWordsByLanguage("en", []string{"CustomPhrase", "customphrase"})
	static, dynamic := s.GetSensitiveWordsByLanguage("en")
	if !slices.Contains(static, "customphrase") || !reflect.DeepEqual(dynamic, []string{"officeforbidden"}) {
		t.Fatalf("language/normalization contract changed: %v / %v", static, dynamic)
	}
	_, russian := s.GetSensitiveWordsByLanguage("ru")
	_, french := s.GetSensitiveWordsByLanguage("fr")
	if !reflect.DeepEqual(russian, []string{"запрещено"}) || !reflect.DeepEqual(french, []string{"interdit"}) {
		t.Fatalf("language routing changed: %v / %v", russian, french)
	}
	static[0], dynamic[0] = "changed", "changed"
	allStatic, allDynamic := s.GetStaticSensitiveWords(), s.GetDynamicSensitiveWords()
	allStatic[0], allDynamic[0] = "changed", "changed"
	static, dynamic = s.GetSensitiveWordsByLanguage("en")
	if static[0] != "amazon" || dynamic[0] != "officeforbidden" || slices.Contains(s.GetStaticSensitiveWords(), "changed") || slices.Contains(s.GetDynamicSensitiveWords(), "changed") {
		t.Fatal("query results must not expose mutable word storage")
	}
	s.ClearDynamicSensitiveWords()
	if len(s.GetDynamicSensitiveWords()) != 0 || !slices.Contains(s.GetStaticSensitiveWords(), "customphrase") {
		t.Fatal("clear must affect only dynamic words")
	}
}

func TestSensitiveWordConcurrentAddGetClear(t *testing.T) {
	s := NewSensitiveWordServiceInMemory(nil)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 5; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			<-start
			for i := 0; i < 100; i++ {
				switch worker {
				case 0:
					s.AddDynamicSensitiveWords([]string{"ConcurrentForbidden"})
				case 1:
					s.AddDynamicSensitiveWordsByLanguage("en", []string{"ExplicitForbidden"})
				case 2:
					s.ClearDynamicSensitiveWords()
				case 3:
					s.AddStaticSensitiveWords([]string{"StaticForbidden"})
					s.AddStaticSensitiveWordsByLanguage("fr", []string{"Interdit"})
				case 4:
					s.GetSensitiveWordsByLanguage("en")
					s.GetDynamicSensitiveWords()
					s.GetStaticSensitiveWords()
					s.mutex.RLock()
					updated, version := s.config.LastUpdated, s.config.Version
					s.mutex.RUnlock()
					if updated.IsZero() || version != "1.0" {
						t.Error("concurrent read observed invalid metadata")
					}
				}
			}
		}(worker)
	}
	close(start)
	wg.Wait()
	s.ClearDynamicSensitiveWords()
	s.AddDynamicSensitiveWords([]string{"FinalForbidden"})
	if got := s.GetDynamicSensitiveWords(); !reflect.DeepEqual(got, []string{"finalforbidden"}) {
		t.Fatalf("final update lost after concurrent clear: %v", got)
	}
}
