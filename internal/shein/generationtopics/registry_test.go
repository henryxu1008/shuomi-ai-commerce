package generationtopics

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTopicResolutionKeepsKnownDirectivesAndIgnoresUnknownKeys(t *testing.T) {
	definitions, unknown := ResolveSheinTopicKeys([]string{" baby ", "CHILDREN", "rock", "children", "ROCK"})
	if len(definitions) != 2 || definitions[0].Key != "children" || definitions[1].Key != "baby" {
		t.Fatalf("definitions = %+v, want deduplicated topics in priority order", definitions)
	}
	if !reflect.DeepEqual(unknown, []string{"rock"}) {
		t.Fatalf("unknown = %v, want one normalized rock key", unknown)
	}
	summary := AssembleSheinPolicySummary(definitions, SheinGenerationPolicyMaxDirectives, SheinGenerationPolicyMaxChars)
	for _, directive := range []string{
		"Do not mention children, babies, or age-specific users.",
		"Do not mention babies, newborns, or infant-specific usage.",
	} {
		if !strings.Contains(summary, directive) {
			t.Fatalf("summary = %q, missing %q", summary, directive)
		}
	}
	if strings.Contains(summary, "rock") {
		t.Fatalf("unknown topic entered summary: %q", summary)
	}
	lexicons := CollectLexiconsFromDefinitions(definitions)
	for language, words := range map[string][]string{"en": {"kids", "baby"}, "zh": {"儿童", "婴儿"}} {
		for _, word := range words {
			if !slices.Contains(lexicons[language], word) {
				t.Fatalf("%s lexicon = %v, missing %q", language, lexicons[language], word)
			}
		}
	}
}

func TestTopicOverlayKeepsDefaultsAndDeduplicatesExplicitDirectivesAndLexicons(t *testing.T) {
	definitions := ResolveSheinTopicDefinitionsWithOverlay([]string{"children", "rock"}, func(key string) (DefinitionOverlay, error) {
		if key != "children" {
			t.Fatalf("unknown key reached overlay: %s", key)
		}
		return DefinitionOverlay{
			Enabled:           true,
			PromptDirectives:  []string{"Avoid toddler-focused positioning.", " avoid TODDLER-focused positioning. "},
			LexiconByLanguage: map[string][]string{" EN ": {"toddler", "Toddler", " kids "}, "zh": {"幼童", "幼童"}},
		}, nil
	})
	summary := AssembleSheinPolicySummary(definitions, SheinGenerationPolicyMaxDirectives, SheinGenerationPolicyMaxChars)
	if !strings.Contains(summary, "Do not mention children, babies, or age-specific users.") || strings.Count(summary, "Avoid toddler-focused positioning.") != 1 {
		t.Fatalf("summary = %q, want default plus one explicit override directive", summary)
	}
	lexicons := CollectLexiconsFromDefinitions(definitions)
	if !reflect.DeepEqual(lexicons["en"], []string{"child", "children", "kid", "kids", "toddler"}) {
		t.Fatalf("english lexicon = %v, want defaults plus one toddler", lexicons["en"])
	}
	if !slices.Contains(lexicons["zh"], "儿童") || !slices.Contains(lexicons["zh"], "幼童") || len(lexicons["zh"]) != 4 {
		t.Fatalf("chinese lexicon = %v, want default and deduplicated override words", lexicons["zh"])
	}
}

func TestPolicySummaryPreservesDirectiveAndCharacterCaps(t *testing.T) {
	definitions, _ := ResolveSheinTopicKeys([]string{"children", "baby", "food", "meals", "knives"})
	summary := AssembleSheinPolicySummary(definitions, SheinGenerationPolicyMaxDirectives, SheinGenerationPolicyMaxChars)
	if len(strings.Split(summary, "\n")) != SheinGenerationPolicyMaxDirectives || utf8.RuneCountInString(summary) > SheinGenerationPolicyMaxChars {
		t.Fatalf("summary exceeds current caps: %q", summary)
	}
	for _, definition := range definitions {
		if !strings.Contains(summary, definition.PromptDirectives[0]) {
			t.Fatalf("summary lost required topic directive: %s", definition.Key)
		}
	}
	if got := AssembleSheinPolicySummary(definitions, 1, SheinGenerationPolicyMaxChars); got != definitions[0].PromptDirectives[0] {
		t.Fatalf("one-directive cap = %q", got)
	}
	if got := AssembleSheinPolicySummary(definitions, SheinGenerationPolicyMaxDirectives, 1); got != "" {
		t.Fatalf("too-small character cap = %q, want empty", got)
	}
}
