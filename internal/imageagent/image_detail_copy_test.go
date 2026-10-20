package imageagent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDetailCopySelectsWholeFactsDeterministically(t *testing.T) {
	p := ProductContextRef{Title: "真实商品", Attributes: map[string]string{"颜色": "深灰色", "产品尺寸": "33×38×33厘米", "internal_note": "secret", "Material": "Fabric"}}
	require.Equal(t, []string{"真实商品", "Material：Fabric"}, detailVisibleText(p))
	p.Title = strings.Repeat("长", 33)
	require.Equal(t, []string{"Material：Fabric", "产品尺寸：33×38×33厘米"}, detailVisibleText(p), "skip whole overlong facts, never truncate")
	p.Attributes = map[string]string{"internal_note": "private"}
	require.Empty(t, detailVisibleText(p))
	r := &ImageSlotRecipe{Background: "white"}
	require.Contains(t, literalDetailPrompt(p, "detail", r), "no text, letters, numbers")
	for _, invalid := range []string{"", " padded", "one\ntwo", "one\ttwo", "https://example.com", "WWW.example.com", "data:image/png", "<b>bold</b>", "[instruction]", "quoted\"copy", "hidden\u202ecopy", strings.Repeat("界", 33)} {
		require.False(t, validDetailLabel(invalid), "%q", invalid)
	}
	require.True(t, validDetailLabel(strings.Repeat("界", 32)))
	require.True(t, validDetailLabel(strings.Repeat("🟢", 32)), "count Unicode code points rather than UTF-16 code units")
}

func TestDetailRecipeCopyAndPromptMustMatchFrozenFacts(t *testing.T) {
	p := ProductContextRef{Title: "真实商品"}
	r := &ImageSlotRecipe{Purpose: "key_benefits", Background: "white", VisibleText: detailVisibleText(p)}
	slot := Slot{Brief: "details", Recipe: r}
	set := &ImageSetPlan{Method: &ImageMethodBinding{PolicyVersion: ImageMethodPolicyVersion}}
	r.Prompt = literalDetailPrompt(p, slot.Brief, r)
	require.NoError(t, validateLiteralDetailRecipe(set, slot, p))
	clone := CloneImageSlotRecipe(r)
	clone.VisibleText[0] = "invented claim"
	require.Equal(t, []string{"真实商品"}, r.VisibleText)
	slot.Recipe = clone
	require.ErrorIs(t, validateLiteralDetailRecipe(set, slot, p), ErrRevisionConflict)
	slot.Recipe = r
	p.Title = "changed later"
	require.ErrorIs(t, validateLiteralDetailRecipe(set, slot, p), ErrRevisionConflict)
	p.Title = "真实商品"
	r.Prompt += " add a table"
	require.ErrorIs(t, validateLiteralDetailRecipe(set, slot, p), ErrRevisionConflict)
	set.Method.PolicyVersion = FrozenVisualBriefPolicyVersion
	require.NoError(t, validateLiteralDetailRecipe(set, slot, p), "never rebuild historical v2 prompts")
}

func TestFrozenV2RecipeBytesRetainTheOriginalShape(t *testing.T) {
	const original = `{"Purpose":"key_benefits","Background":"white","Language":"zh","Placement":{"Group":"detail","Order":1},"PromptVersion":"product-image-set-v1","Prompt":"original literal v2 prompt","References":null,"Quote":{"Provider":"","Model":"","Protocol":"","Resolution":"","Quality":"","PriceVersion":"","Points":0,"RouteReference":"","CredentialReference":"","ConfigurationVersion":""}}`
	var recipe ImageSlotRecipe
	require.NoError(t, json.Unmarshal([]byte(original), &recipe))
	raw, err := json.Marshal(CloneImageSlotRecipe(&recipe))
	require.NoError(t, err)
	require.Equal(t, original, string(raw), "omitempty must preserve old recipe JSON and all hashes consuming it")
}
