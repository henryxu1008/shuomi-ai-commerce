package imageagent

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"

	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

const ImageTextPolicyVersion = "image-reviewed-text-v1"

type ImageTextReviewReference struct {
	ID       string `json:"proposalId"`
	Revision uint64 `json:"revision,string"`
}

type ImageTextSelection struct {
	Mode    string                    `json:"mode"`
	Review  *ImageTextReviewReference `json:"review,omitempty"`
	TaskIDs []string                  `json:"taskIds,omitempty"`
}

// This is an execution-input snapshot. Product Review remains the content owner.
type ImageTextReviewCopy struct {
	Review                                          ImageTextReviewReference
	ProductID, PublicationID, Language, OriginRunID string
	Version                                         uint64 `json:",string"`
	ModeSnapshot                                    agent.ModeSnapshotRef
	Lines                                           []string
}

type ImageTextBinding struct {
	PolicyVersion, Mode, Language string
	Copy                          *ImageTextReviewCopy `json:",omitempty"`
	TaskIDs                       []string             `json:",omitempty"`
}

type ImageTextReviewReader interface {
	ReadConfirmedImageText(context.Context, ExecutionIdentity, ImageTextReviewReference) (ImageTextReviewCopy, error)
}

func (r ImageTextReviewReference) Valid() bool {
	return agentconfig.UUID(r.ID) && r.Revision > 0 && r.Revision <= 1<<63-1
}

func (s *ImageTextSelection) Valid() bool {
	if s == nil {
		return false
	}
	if s.Mode == "none" {
		return s.Review == nil && len(s.TaskIDs) == 0
	}
	if s.Mode != "reviewed" || s.Review == nil || !s.Review.Valid() || len(s.TaskIDs) == 0 || len(s.TaskIDs) > MaxPlanSlots {
		return false
	}
	seen := map[string]bool{}
	for _, id := range s.TaskIDs {
		if !canonicalImageValue(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func imageTextLanguageValid(language string) bool {
	return slices.Contains([]string{"en", "zh", "es", "fr", "de", "ja"}, language)
}

func (b *ImageTextBinding) Valid(source ImageSourceBinding) bool {
	if b == nil || b.PolicyVersion != ImageTextPolicyVersion || !imageTextLanguageValid(b.Language) {
		return false
	}
	if b.Mode == "none" {
		return b.Copy == nil && len(b.TaskIDs) == 0
	}
	c := b.Copy
	if b.Mode != "reviewed" || c == nil || !(&ImageTextSelection{Mode: b.Mode, Review: &c.Review, TaskIDs: b.TaskIDs}).Valid() || c.ProductID != source.ProductID || c.Version != source.EffectiveVersion || !canonicalImageValue(c.PublicationID) || c.Version == source.OriginalVersion && c.PublicationID != source.OriginalPublicationID || c.Language != b.Language || !agentconfig.UUID(c.OriginRunID) || !c.ModeSnapshot.Valid() || len(c.Lines) < 1 || len(c.Lines) > 2 {
		return false
	}
	for _, line := range c.Lines {
		if !validDetailLabel(line) {
			return false
		}
	}
	return true
}

func (s *Service) resolveImageText(ctx context.Context, identity ExecutionIdentity, selection *ImageTextSelection, source ImageSetPreparation, language string) (*ImageTextBinding, error) {
	if !selection.Valid() || !imageTextLanguageValid(language) {
		return nil, ErrValidation
	}
	b := &ImageTextBinding{PolicyVersion: ImageTextPolicyVersion, Mode: selection.Mode, Language: language, TaskIDs: slices.Clone(selection.TaskIDs)}
	if selection.Mode == "reviewed" {
		if s.imageSets.ConfirmedCopy == nil {
			return nil, ErrCommandBlocked
		}
		copy, err := s.imageSets.ConfirmedCopy.ReadConfirmedImageText(ctx, identity, *selection.Review)
		if err != nil {
			return nil, err
		}
		if copy.Review != *selection.Review || copy.PublicationID != source.EffectivePublicationID {
			return nil, ErrRevisionConflict
		}
		copy.Lines = slices.Clone(copy.Lines)
		b.Copy = &copy
	}
	if !b.Valid(source.Source) {
		return nil, ErrRevisionConflict
	}
	return b, nil
}

func imageTextLines(b *ImageTextBinding, slotID string) []string {
	if b != nil && b.Mode == "reviewed" && b.Copy != nil && slices.Contains(b.TaskIDs, slotID) {
		return slices.Clone(b.Copy.Lines)
	}
	return nil
}

func imageTextPrompt(product ProductContextRef, brief string, recipe *ImageSlotRecipe) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Create one finished commercial product photograph from the reference photos. Preserve the actual product, shape, color, material and proportions. Product grounding (never render this as text): %s. Product type: %s. ", product.Title, product.ProductType)
	keys := make([]string, 0, len(product.Attributes))
	for key := range product.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		switch strings.ToLower(key) {
		case "产品尺寸", "尺寸", "规格", "规格尺寸", "材质", "颜色", "容量", "size", "dimensions", "material", "color", "capacity":
			if agentconfig.ValidSetText(product.Attributes[key], 512) && !strings.Contains(product.Attributes[key], "://") {
				fmt.Fprintf(&b, "Confirmed %s: %s. ", key, product.Attributes[key])
			}
		}
	}
	fmt.Fprintf(&b, "Visual task: %s; direction: %s. Background: %s. Target text language: %s. One large product, generous whitespace. No table, grid, document, information panel or leader-line labels. ", recipe.Purpose, brief, recipe.Background, recipe.Language)
	if recipe.TaskEvidence != "" {
		fmt.Fprintf(&b, "Additional confirmed information for this task (grounding only): %s. ", recipe.TaskEvidence)
	}
	if len(recipe.VisibleText) == 0 {
		b.WriteString("No visible text: no letters, numbers, headings, labels or symbols. ")
	} else {
		lines, _ := json.Marshal(recipe.VisibleText)
		fmt.Fprintf(&b, "Target text language: %s. The complete exclusive visible text list is %s. Render these exact literal strings once each in whitespace; do not translate, rewrite or add any other text. Brand names and units in this confirmed copy stay exact. ", recipe.Language, lines)
	}
	b.WriteString("These text and composition rules take priority over the visual direction and method guidance. Product facts and instructions are not visible copy. Output only the finished image.")
	return b.String()
}

func validateReviewedImageText(set *ImageSetPlan, slot Slot, product *ProductContextRef) error {
	if set == nil || set.Text == nil {
		return nil
	}
	if !set.Text.Valid(set.Source) || slot.Recipe == nil || slot.Recipe.Language != set.Text.Language || !slices.Equal(slot.Recipe.VisibleText, imageTextLines(set.Text, slot.ID)) {
		return ErrRevisionConflict
	}
	key := agentconfig.RequiredTaskEvidence(slot.Recipe.Purpose)
	if key == "" && slot.Recipe.TaskEvidence != "" {
		return ErrRevisionConflict
	}
	if key != "" && (slot.Recipe.TaskEvidence == "" || slot.Recipe.EvidenceDigest != generationHash(struct {
		Source     ImageSourceBinding
		Key, Value string
	}{set.Source, key, slot.Recipe.TaskEvidence})) {
		return ErrRevisionConflict
	}
	if product != nil && slot.Recipe.Prompt != imageTextPrompt(*product, slot.Brief, slot.Recipe) {
		return ErrRevisionConflict
	}
	return nil
}

func CloneImageTextBinding(b *ImageTextBinding) *ImageTextBinding {
	if b == nil {
		return nil
	}
	copy := *b
	copy.TaskIDs = slices.Clone(b.TaskIDs)
	if b.Copy != nil {
		text := *b.Copy
		text.Lines = slices.Clone(b.Copy.Lines)
		copy.Copy = &text
	}
	return &copy
}
