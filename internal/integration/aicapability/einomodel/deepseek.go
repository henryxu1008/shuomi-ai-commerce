package einomodel

import (
	"strings"
	"task-processor/internal/aicapability"
)

// DashScopeDeepSeekV41FlashNonThinkingPolicy fixes the request semantics of
// this exact provider/model pair. ModelProfile includes this version in its
// digest and quote; changing it requires a new quote and route admission.
const DashScopeDeepSeekV41FlashNonThinkingPolicy = "dashscope-deepseek-v4.1-flash-nonthinking-v1"

func validFixedAdapterPolicy(adapter AdapterKind, provider, modelID, policy string) bool {
	switch policy {
	case TitlePilotOpenAIAdapterPolicy, OpenAIMiniTextAdapterPolicy:
		return adapter == AdapterOpenAICompatible && provider == "openai" && modelID == aicapability.SingleImageMiniModel
	case aicapability.EinoGeminiTextPolicy, aicapability.EinoGeminiImagePolicy:
		return adapter == AdapterGoogleGenerateContent && provider == "google" && validGeminiModelID(modelID)
	case aicapability.EinoQwenTextPolicy, aicapability.EinoQwenImagePolicy:
		return adapter == AdapterQwenCompatible && provider == "DASHSCOPE"
	case aicapability.EinoClaudeTextPolicy:
		return adapter == AdapterClaudeNative && provider == "anthropic"
	case aicapability.EinoOpenAIImagePolicy:
		return adapter == AdapterOpenAICompatible
	}
	if adapter == AdapterGoogleGenerateContent || adapter == AdapterQwenCompatible || strings.HasPrefix(policy, "eino-") {
		return false
	}
	if policy == aicapability.SingleImageMiniAdapterPolicy {
		return adapter == AdapterOpenAICompatible && provider == "openai" && modelID == aicapability.SingleImageMiniModel
	}
	selectedModel := strings.EqualFold(provider, "DASHSCOPE") && modelID == "deepseek-v4.1-flash"
	if selectedModel || policy == DashScopeDeepSeekV41FlashNonThinkingPolicy {
		return selectedModel && provider == "DASHSCOPE" && adapter == AdapterOpenAICompatible && policy == DashScopeDeepSeekV41FlashNonThinkingPolicy
	}
	return true
}
