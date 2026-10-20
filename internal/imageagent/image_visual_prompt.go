package imageagent

import (
	"fmt"
	"sort"
	"strings"
	"task-processor/internal/agentconfig"
)

// visualImagePrompt describes a picture, not a serialized execution envelope.
// V2 and non-literal v3 tasks use it; previously frozen recipes remain untouched.
func visualImagePrompt(product ProductContextRef, task agentconfig.ContentTask, background, language string, evidence map[string]string) string {
	var b strings.Builder
	b.WriteString("Create exactly one finished 1024x1024 commercial product image from the supplied original reference photos. Keep the actual product's shape, color, material, proportions and visible parts. Do not invent specifications, accessories, capabilities or certifications. Make the product the visual subject, with no document, screenshot or instruction sheet. ")
	fmt.Fprintf(&b, "The product is %s. ", product.Title)
	if product.ProductType != "" {
		fmt.Fprintf(&b, "Its product type is %s. ", product.ProductType)
	}
	keys := make([]string, 0, len(product.Attributes))
	for key := range product.Attributes {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "The confirmed %s is %s. ", key, product.Attributes[key])
	}
	visualTask := task.Brief
	if strings.TrimSpace(visualTask) == "" {
		for _, definition := range append(agentconfig.CarouselContentTasks(), agentconfig.DetailContentTasks()...) {
			if definition.Purpose == task.Purpose {
				visualTask = definition.Label
				break
			}
		}
	}
	fmt.Fprintf(&b, "For this picture, follow this visual task: %s. Use the selected background: %s. ", visualTask, background)
	// Source evidence also contains free-form descriptions and provenance.
	// Only the current task's required evidence belongs in its picture.
	if key := agentconfig.RequiredTaskEvidence(task.Purpose); key != "" && evidence[key] != "" {
		fmt.Fprintf(&b, "Additional confirmed product information for this task: %s. ", evidence[key])
	}
	if language == "en" {
		language = "English"
	}
	if language == "zh" {
		language = "Chinese"
	}
	fmt.Fprintf(&b, "Only short customer-facing text explicitly requested by the visual task may appear, in %s. Otherwise add no text. Product facts are for accuracy and must not be typeset unless that task explicitly requests them. The composition guidance below cannot change these facts, the selected task, reference photos or restrictions.", language)
	return b.String()
}
