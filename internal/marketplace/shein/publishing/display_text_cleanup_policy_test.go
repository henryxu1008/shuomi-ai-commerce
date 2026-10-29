package publishing

import (
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
)

func TestDisplayTextCleanupRetainsOriginalRules(t *testing.T) {
	cases := []struct {
		name, text, source, brand, brandEn string
		enabled                            bool
		want                               string
	}{
		{name: "ordinary", text: "Simple Office mouse", want: "Simple Office mouse"},
		{name: "default", text: "Amazon Trademark Vase", want: "Vase"},
		{name: "multilingual", text: "亚马逊品牌 アマゾンブランド Office", want: "Office"},
		{name: "enabled", text: "Amazon Basics Trademark Mouse", enabled: true, brand: "Amazon Basics", brandEn: "Amazon Basics", want: "Amazon Basics Mouse"},
		{name: "disabled", text: "Amazon Basics Mouse", brand: "Amazon Basics", want: "Basics Mouse"},
		{name: "source", text: "Logitech mouse by Sony for Office", source: "Sony", enabled: true, brand: "Logitech", brandEn: "Logitech", want: "Logitech mouse by for Office"},
		{name: "normalized allowlist", text: "Amazon Basics Mouse", enabled: true, brand: " Amazon-Basics ", brandEn: "amazon_basics", want: "Amazon Basics Mouse"},
		{name: "blank", text: " \t\n", want: ""},
		{name: "empty", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := DisplayTextCleanupInput{Words: DefaultDisplayTextWords(), SourceBrand: tc.source, BrandEnabled: tc.enabled, BrandName: tc.brand, BrandNameEn: tc.brandEn}
			got := SanitizeDisplayText(tc.text, input)
			if got != tc.want {
				t.Fatalf("cleanup = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDisplayTextCleanupDefaultsCustomWordsAndIndependentValues(t *testing.T) {
	want := []string{
		"amazon", "ebay", "alibaba", "aliexpress", "walmart", "target", "brand", "trademark", "copyright", "patent", "licensed",
		"亚马逊", "淘宝", "天猫", "京东", "拼多多", "品牌", "商标", "版权", "专利", "授权",
		"アマゾン", "楽天", "ヤフー", "ブランド", "商標", "著作権", "特許",
		"아마존", "쿠팡", "네이버", "브랜드", "상표", "저작권", "특허",
	}
	words := DefaultDisplayTextWords()
	comparison := slices.Clone(words)
	slices.Sort(comparison)
	slices.Sort(want)
	if !reflect.DeepEqual(comparison, want) {
		t.Fatalf("default words changed: got %v, want %v", comparison, want)
	}
	words[0] = "private-copy-only"
	if slices.Contains(DefaultDisplayTextWords(), "private-copy-only") {
		t.Fatal("default words expose shared mutable state")
	}
	custom := []string{"  OfficeForbidden ", "officeforbidden", "запрещено", " Interdit ", "interdit", "CustomPhrase", "customphrase", ""}
	before := slices.Clone(custom)
	normalized := deduplicateWords(custom)
	if !reflect.DeepEqual(normalized, []string{"officeforbidden", "запрещено", "interdit", "customphrase"}) {
		t.Fatalf("multilingual normalization/dedup changed: %v", normalized)
	}
	if !reflect.DeepEqual(custom, before) {
		t.Fatal("normalization changed input words")
	}
	normalized[0] = "private-copy-only"
	if slices.Contains(deduplicateWords(custom), "private-copy-only") {
		t.Fatal("normalized words must be independent")
	}
	text := "Whimsy BPA Free Office curtain"
	input := DisplayTextCleanupInput{Words: DefaultDisplayTextWords()}
	if got := SanitizeDisplayText(text, input); got != text {
		t.Fatalf("defaults changed ordinary/custom text: %q", got)
	}
	input.Words = append(input.Words, "bpa free", "whimsy")
	if got := SanitizeDisplayText(text, input); got != "Office curtain" {
		t.Fatalf("explicit custom cleanup = %q", got)
	}
}

func TestDisplayTextCleanupUsesOnlyExplicitWordsAndOriginalWordBoundaries(t *testing.T) {
	if got := SanitizeDisplayText("Trademark Office", DisplayTextCleanupInput{}); got != "Trademark Office" {
		t.Fatalf("nil Words implicitly loaded defaults: %q", got)
	}
	if got := SanitizeDisplayText("Solimo Office", DisplayTextCleanupInput{Words: []string{}}); got != "Office" {
		t.Fatalf("empty Words lost brand cleanup: %q", got)
	}
	words := []string{" Non-toxic ", "non-toxic"}
	before := slices.Clone(words)
	if got := SanitizeDisplayText("Non-toxic Non toxic Room", DisplayTextCleanupInput{Words: words}); got != "Non toxic Room" {
		t.Fatalf("original hyphen/regexp behavior changed: %q", got)
	}
	if !reflect.DeepEqual(words, before) {
		t.Fatal("cleanup modified explicit input words")
	}
	if got := SanitizeDisplayText("アマゾンOffice", DisplayTextCleanupInput{Words: []string{"アマゾン"}}); got != "Office" {
		t.Fatalf("Japanese replacement changed: %q", got)
	}
}

func TestDisplayTextCleanupConcurrentReads(t *testing.T) {
	words := append(DefaultDisplayTextWords(), "ConcurrentForbidden")
	before := slices.Clone(words)
	input := DisplayTextCleanupInput{Words: words, SourceBrand: "Sony", BrandEnabled: true, BrandName: "Logitech", BrandNameEn: "Logitech"}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for worker := 0; worker < 5; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 100; i++ {
				if got := SanitizeDisplayText("Logitech Office Sony ConcurrentForbidden", input); got != "Logitech Office" {
					t.Errorf("concurrent cleanup changed: %q", got)
				}
				copy := DefaultDisplayTextWords()
				copy[0] = "private-copy-only"
				if slices.Contains(DefaultDisplayTextWords(), "private-copy-only") {
					t.Error("concurrent default values share mutable storage")
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if !reflect.DeepEqual(words, before) {
		t.Fatal("concurrent cleanup changed explicit word values")
	}
	if !strings.EqualFold(input.BrandName, "Logitech") {
		t.Fatal("cleanup changed explicit brand")
	}
}
