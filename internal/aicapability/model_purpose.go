package aicapability

// These finite purpose contracts bind the installed component and its input
// semantics. A trusted model definition selects one; a name or provider label
// never grants a purpose. No SDK options are accepted from users/model output.
const (
	EinoGeminiTextPolicy  = "eino-gemini-v0.1.36-text-v1"
	EinoGeminiImagePolicy = "eino-gemini-v0.1.36-single-image-v1"
	EinoQwenTextPolicy    = "eino-qwen-v0.1.9-text-v1"
	EinoQwenImagePolicy   = "eino-qwen-v0.1.9-single-image-v1"
	EinoClaudeTextPolicy  = "eino-claude-v0.1.26-text-v1"
	EinoOpenAIImagePolicy = "eino-openai-v0.1.13-single-image-v1"
	GeminiUsageMapping    = "gemini-generate-content-usage-v1"
	QwenUsageMapping      = "qwen-text-v1"
	ClaudeUsageMapping    = "claude-text-v1"
)

func ValidGeminiThinkingMode(mode string) bool {
	return mode == "budget-zero" || mode == "minimal" || mode == "low"
}

func SingleImagePurpose(policy string) bool {
	switch policy {
	case SingleImageMiniAdapterPolicy, EinoGeminiImagePolicy, EinoQwenImagePolicy, EinoOpenAIImagePolicy:
		return true
	}
	return false
}

// SingleImagePromptEstimate is a frozen reference input estimate, not a
// manufacturer's hard bound. The Mini policy retains its admitted estimate;
// other purposes require an explicit bounded estimate in the trusted profile.
func SingleImagePromptEstimate(p ModelProfile) int64 {
	if p.AdapterPolicyVersion == SingleImageMiniAdapterPolicy {
		return SingleImagePromptTokenBound
	}
	return p.ImagePromptTokenEstimate
}
