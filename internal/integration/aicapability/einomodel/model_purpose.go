package einomodel

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"task-processor/internal/aicapability"
)

// ComponentExecutionBinding is compiled from installed components, never
// supplied by a request. The shared profile fingerprint freezes it together
// with the exact model, connection, usage mapper and operation contracts.
func ComponentExecutionBinding(p aicapability.ModelProfile) string {
	component := ""
	switch AdapterKind(p.AdapterKind) {
	case AdapterOpenAICompatible:
		component = "eino-openai@v0.1.13/acl-openai@v0.1.17"
	case AdapterQwenCompatible:
		component = "eino-qwen@v0.1.9/acl-openai@v0.1.17"
	case AdapterClaudeNative:
		component = "eino-claude@v0.1.26"
	case AdapterGoogleGenerateContent:
		component = "eino-gemini@v0.1.36/genai@v1.72.0"
	case AdapterGoogleInteractions:
		component = "genai-interactions@v1.72.0"
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{component, p.AdapterPolicyVersion, p.ThinkingMode,
		p.UsageMappingVersion, p.PromptVersion, p.OutputSchemaVersion}, "\x00")))
	return hex.EncodeToString(sum[:])
}

// ReferenceMappingReady enumerates actual implemented protocol mappers.
// Components, provider names and SDK integer zero values confer no billing
// qualification on their own.
func ReferenceMappingReady(p aicapability.ModelProfile) bool {
	switch AdapterKind(p.AdapterKind) {
	case AdapterOpenAICompatible:
		return true
	case AdapterQwenCompatible:
		return p.UsageMappingVersion == aicapability.QwenUsageMapping &&
			(p.AdapterPolicyVersion == aicapability.EinoQwenTextPolicy || p.AdapterPolicyVersion == aicapability.EinoQwenImagePolicy)
	case AdapterClaudeNative:
		return p.UsageMappingVersion == aicapability.ClaudeUsageMapping && p.AdapterPolicyVersion == aicapability.EinoClaudeTextPolicy
	case AdapterGoogleGenerateContent:
		return p.UsageMappingVersion == aicapability.GeminiUsageMapping &&
			(p.AdapterPolicyVersion == aicapability.EinoGeminiTextPolicy || p.AdapterPolicyVersion == aicapability.EinoGeminiImagePolicy)
	}
	return false
}
