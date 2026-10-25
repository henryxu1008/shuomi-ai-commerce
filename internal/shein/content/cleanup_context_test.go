package content

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"sync"
	"task-processor/internal/shein/authorizedbrand"
	"testing"
)

func TestCleanupContextPreservesAuthorizedAndSourceBrandRules(t *testing.T) {
	service := NewSensitiveWordServiceInMemory(nil)
	ctx := &CleanupContext{Context: context.Background(), SourceBrand: "Sony", AuthorizedBrand: &authorizedbrand.Resolved{Enabled: true, Name: "Logitech", NameEn: "Logitech"}}
	got := service.SanitizeDisplayTextWithContext(ctx, "Logitech mouse by Sony for Office")
	if !strings.Contains(got, "Logitech") || strings.Contains(strings.ToLower(got), "sony") || !strings.Contains(got, "Office") {
		t.Fatalf("brand/case cleanup changed: %q", got)
	}
	if got := service.SanitizeDisplayTextWithContext(nil, "Simple Office mouse"); got != "Simple Office mouse" {
		t.Fatalf("nil context changed display text: %q", got)
	}
}

func TestInMemoryCleanupUsesExplicitCustomPhrasesWithoutChangingDefaults(t *testing.T) {
	service := NewSensitiveWordServiceInMemory(nil)
	ctx := &CleanupContext{Context: context.Background()}
	input := "Whimsy BPA Free Office curtain"
	if got := service.SanitizeDisplayTextWithContext(ctx, input); got != input {
		t.Fatalf("default cleanup = %q, want custom phrases retained before explicit configuration", got)
	}
	service.mutex.Lock()
	service.config.StaticWords["en"] = append(service.config.StaticWords["en"], "bpa free", "whimsy")
	service.mutex.Unlock()
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

func TestSensitiveWordMemoryContractAndDefensiveCopies(t *testing.T) {
	s := NewSensitiveWordServiceInMemory(nil)
	wantDefaults := map[string][]string{
		"en": {"amazon", "ebay", "alibaba", "aliexpress", "walmart", "target", "brand", "trademark", "copyright", "patent", "licensed"},
		"zh": {"亚马逊", "淘宝", "天猫", "京东", "拼多多", "品牌", "商标", "版权", "专利", "授权"},
		"ja": {"アマゾン", "楽天", "ヤフー", "ブランド", "商標", "著作権", "特許"},
		"ko": {"아마존", "쿠팡", "네이버", "브랜드", "상표", "저작권", "특허"},
	}
	if !reflect.DeepEqual(s.config.StaticWords, wantDefaults) || len(s.config.DynamicWords) != 0 {
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
	if len(wire) != 2 || wire["static_words"] == nil || wire["dynamic_words"] == nil {
		t.Fatalf("configuration JSON contract changed: %s", raw)
	}
	var decoded SensitiveWordConfig
	if err := json.Unmarshal(raw, &decoded); err != nil || !reflect.DeepEqual(decoded.StaticWords, s.config.StaticWords) || !reflect.DeepEqual(decoded.DynamicWords, s.config.DynamicWords) {
		t.Fatalf("word-map JSON round trip failed: %+v, %v", decoded, err)
	}
	s.mutex.Lock()
	s.config.DynamicWords["en"] = []string{"  OfficeForbidden ", "officeforbidden"}
	s.config.DynamicWords["ru"] = []string{"запрещено"}
	s.config.DynamicWords["fr"] = []string{" Interdit ", "interdit"}
	s.config.StaticWords["en"] = append(s.config.StaticWords["en"], "CustomPhrase", "customphrase")
	s.mutex.Unlock()
	words := s.getAllSensitiveWords()
	for _, want := range []string{"officeforbidden", "запрещено", "interdit", "customphrase"} {
		count := 0
		for _, word := range words {
			if word == want {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("normalized multilingual word %q appeared %d times in %v", want, count, words)
		}
	}
	words[0] = "changed"
	if slices.Contains(s.getAllSensitiveWords(), "changed") || s.config.StaticWords["en"][0] != "amazon" || s.config.DynamicWords["en"][0] != "  OfficeForbidden " {
		t.Fatal("query results must not expose mutable word storage")
	}
}

func TestSensitiveWordConcurrentCleanupReads(t *testing.T) {
	s := NewSensitiveWordServiceInMemory(nil)
	s.mutex.Lock()
	s.config.DynamicWords["en"] = []string{"ConcurrentForbidden"}
	s.mutex.Unlock()
	ctx := &CleanupContext{SourceBrand: "Sony", AuthorizedBrand: &authorizedbrand.Resolved{Enabled: true, Name: "Logitech", NameEn: "Logitech"}}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 5; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 100; i++ {
				if got := s.SanitizeDisplayTextWithContext(ctx, "Logitech Office Sony ConcurrentForbidden"); got != "Logitech Office" {
					t.Errorf("concurrent cleanup changed current brand/word rules: %q", got)
				}
				words := s.getAllSensitiveWords()
				if !slices.Contains(words, "concurrentforbidden") {
					t.Error("concurrent cleanup lost configured words")
				}
				words[0] = "private-copy-only"
				if slices.Contains(s.getAllSensitiveWords(), "private-copy-only") {
					t.Error("concurrent reads must return independent copies")
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if !reflect.DeepEqual(s.config.DynamicWords["en"], []string{"ConcurrentForbidden"}) {
		t.Fatalf("concurrent reads changed word storage: %v", s.config.DynamicWords)
	}
}
