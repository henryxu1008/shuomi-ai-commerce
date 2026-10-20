package imageagent

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"task-processor/internal/agentconfig"
)

func usesLiteralDetailCopy(set *ImageSetPlan, recipe *ImageSlotRecipe) bool {
	return set != nil && set.Text == nil && set.Method != nil && set.Method.PolicyVersion == FrozenLiteralCopyPolicyVersion && recipe != nil && recipe.Purpose == "key_benefits"
}

func validDetailLabel(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || utf8.RuneCountInString(value) > 32 || !agentconfig.ValidSetText(value, 128) {
		return false
	}
	if strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) >= 0 || strings.ContainsAny(value, "<>`\"'\\[]{}") {
		return false
	}
	lower := strings.ToLower(value)
	return !strings.Contains(lower, "://") && !strings.Contains(lower, "www.") && !strings.Contains(lower, "data:")
}

// Whole facts only: no translation, truncation, source notes or model copywriting.
func detailVisibleText(product ProductContextRef) []string {
	var labels []string
	if validDetailLabel(product.Title) {
		labels = append(labels, product.Title)
	}
	keys := make([]string, 0, len(product.Attributes))
	for key := range product.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(labels) == 2 {
			break
		}
		switch strings.ToLower(key) {
		case "产品尺寸", "尺寸", "规格", "规格尺寸", "材质", "颜色", "容量", "size", "dimensions", "material", "color", "capacity":
			if label := key + "：" + product.Attributes[key]; product.Attributes[key] != "" && validDetailLabel(label) {
				labels = append(labels, label)
			}
		}
	}
	return labels
}

func literalDetailPrompt(product ProductContextRef, brief string, recipe *ImageSlotRecipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Create exactly one finished 1024x1024 commercial product detail photograph from the supplied original reference photos. The product is %s. Preserve its actual shape, color, material, proportions and visible parts. Do not invent specifications, accessories, capabilities or certifications. Selected visual direction: %s. Use the selected background: %s. ", product.Title, brief, recipe.Background)
	b.WriteString("Composition: one large accurate product as the visual focus, surrounded by generous clean whitespace. No table, grid, parameter sheet, information panels, comparison chart, document or instruction page. ")
	if len(recipe.VisibleText) == 0 {
		b.WriteString("Make a clean photographic close-up with no text, letters, numbers, headings or labels. ")
	} else {
		labels, _ := json.Marshal(recipe.VisibleText)
		fmt.Fprintf(&b, "The complete and exclusive visible text list is %s. Place only these literal strings once each as small, clearly readable labels in the whitespace, without covering the product. Copy every character exactly, including its original language and units; do not translate, rewrite, expand or add any other text, heading, numbers or symbols. ", labels)
	}
	b.WriteString("These composition and exclusive text rules take priority over any generic text or layout suggestion in the selected visual direction or the method guidance below. Product facts, instructions and method guidance are never visible copy. Output the finished product image only.")
	return b.String()
}

func validateLiteralDetailRecipe(set *ImageSetPlan, slot Slot, product ProductContextRef) error {
	if set != nil && set.Text != nil {
		return validateReviewedImageText(set, slot, &product)
	}
	if !usesLiteralDetailCopy(set, slot.Recipe) {
		return nil
	}
	if !slices.Equal(slot.Recipe.VisibleText, detailVisibleText(product)) || slot.Recipe.Prompt != literalDetailPrompt(product, slot.Brief, slot.Recipe) {
		return fmt.Errorf("%w: detail copy does not match frozen product facts", ErrRevisionConflict)
	}
	return nil
}
