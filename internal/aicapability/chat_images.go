package aicapability

const (
	ChatImageAdapterPolicy = "openai-mini-chat-vision-v1"
	ChatImagePromptVersion = "ai-workbench-chat-images-v1"
	ChatImageAnswerVersion = "ai-workbench-chat-image-answer-v1"
	MaximumChatImages      = 4
	MaximumChatImageBytes  = 2 << 20
	MaximumChatImageWire   = 3 << 20
)

// ValidChatImages applies the group limit without relaxing any single image limit.
// Position matters: the caller must preserve the slice in the quote and on the wire.
func ValidChatImages(images []SingleImageInput) bool {
	if len(images) == 0 || len(images) > MaximumChatImages {
		return false
	}
	total := 0
	for _, image := range images {
		if !image.Valid() {
			return false
		}
		total += image.ByteLength
	}
	return total <= MaximumChatImageBytes
}

func ChatImageProfileShape(p ModelProfile) bool {
	return p.ValidateShape() == nil && p.AdapterPolicyVersion == ChatImageAdapterPolicy &&
		p.ProviderID == "openai" && p.AdapterKind == "openai-compatible" && p.ModelID == SingleImageMiniModel &&
		p.BillingMode != ModelUsageOnly && p.UsageMappingVersion == "openai-text-v1" &&
		(p.ImagePromptTokenEstimate == 0 || p.ImagePromptTokenEstimate == SingleImagePromptTokenBound) &&
		p.MaximumPromptTokens > SingleImagePromptTokenBound
}

func ChatImageProfile(p ModelProfile) bool {
	return ChatImageProfileShape(p) && p.Validate() == nil &&
		p.OfficialTariff.UsageVersion == "openai-text-v1" && p.OfficialTariff.Channel == "openai-direct"
}
