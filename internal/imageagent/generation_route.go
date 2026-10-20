package imageagent

// Only complete, explicitly admitted routes may enter an immutable quote or intent.
func validGenerationRoute(provider, model, protocol string) bool {
	return GenerationRouteQuality(provider, model, protocol) != ""
}

// GenerationRouteQuality is the fixed quality of an admitted exact route.
// The existing quote and intent fields freeze this value without changing
// historical metadata serialization or allowing a client-selected quality.
func GenerationRouteQuality(provider, model, protocol string) string {
	if provider == "grsai" && protocol == "grsai-json-sync-v1" {
		switch model {
		case "gpt-image-2-vip":
			return "medium"
		case "gpt-image-2.5":
			return "auto"
		}
	}
	if provider == "bailian" && protocol == "dashscope-image-sync-v1" && (model == "qwen-image-3.0" || model == "qwen-image-3.0-pro") {
		return "auto"
	}
	return ""
}

func validateGenerationRouteInput(m GenerationProviderMetadata, input SlotExecutionInput) error {
	if input.ImageSet != nil {
		if err := ValidateImageMethodDispatch(input.ImageSet.Method); err != nil {
			return err
		}
	}
	if m.Provider != "bailian" {
		return nil
	}
	var prompt string
	if input.Artwork != nil {
		prompt = input.Artwork.Prompt
	} else if input.Slot.Recipe != nil {
		prompt = input.Slot.Recipe.Prompt
		if input.ImageMethodText != "" {
			var err error
			prompt, err = ComposeImageMethodPrompt(prompt, input.ImageMethodText)
			if err != nil {
				return err
			}
		}
		refs := input.Slot.Recipe.References
		if len(refs) < 1 || len(refs) > 3 {
			return ErrValidation
		}
		var total int64
		for _, ref := range refs {
			total += ref.Bytes
			if ref.Bytes <= 0 || ref.Bytes > 10<<20 || total > 16<<20 || ref.Width < 1 || ref.Height < 1 || ref.Width > 8000 || ref.Height > 8000 || ref.Width > 10*ref.Height || ref.Height > 10*ref.Width || ref.MediaType != "image/png" && ref.MediaType != "image/jpeg" {
				return ErrValidation
			}
		}
	}
	if len(prompt) > 16<<10 {
		return ErrValidation
	}
	return nil
}
