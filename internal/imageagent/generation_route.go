package imageagent

// Only complete, explicitly admitted routes may enter an immutable quote or intent.
func validGenerationRoute(provider, model, protocol string) bool {
	return provider == "grsai" && model == "gpt-image-2.5" && protocol == "grsai-json-sync-v1" ||
		provider == "bailian" && model == "qwen-image-3.0" && protocol == "dashscope-image-sync-v1"
}

func validateGenerationRouteInput(m GenerationProviderMetadata, input SlotExecutionInput) error {
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
