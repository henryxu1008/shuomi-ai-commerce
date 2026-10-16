package einomodel

import "strings"

// DashScopeDeepSeekV41FlashNonThinkingPolicy fixes the request semantics of
// this exact provider/model pair. ModelProfile includes this version in its
// digest and quote; changing it requires a new quote and route admission.
const DashScopeDeepSeekV41FlashNonThinkingPolicy = "dashscope-deepseek-v4.1-flash-nonthinking-v1"

func validFixedAdapterPolicy(adapter AdapterKind, provider, modelID, policy string) bool {
	selectedModel := strings.EqualFold(provider, "DASHSCOPE") && modelID == "deepseek-v4.1-flash"
	if selectedModel || policy == DashScopeDeepSeekV41FlashNonThinkingPolicy {
		return selectedModel && provider == "DASHSCOPE" && adapter == AdapterOpenAICompatible && policy == DashScopeDeepSeekV41FlashNonThinkingPolicy
	}
	return true
}
