package einomodel

import "task-processor/internal/aicapability"

func matchesResponseModel(actual string, cfg GuardConfig) bool {
	return actual != "" && (actual == cfg.ModelID || cfg.ResponseModelID != "" && actual == cfg.ResponseModelID)
}

// Native observations use present raw fields before SDK normalization. The
// reference strategy prices aggregate input/output only; it does not claim
// precise cache pricing or complete manufacturer billing subdivisions.
func observeNativeReferenceUsage(raw []byte, cfg GuardConfig) ObservedUsage {
	var envelope struct {
		Model        string `json:"model"`
		ModelVersion string `json:"modelVersion"`
		Usage        *struct {
			Input      *int64 `json:"input_tokens"`
			Output     *int64 `json:"output_tokens"`
			CacheRead  *int64 `json:"cache_read_input_tokens"`
			CacheWrite *int64 `json:"cache_creation_input_tokens"`
		} `json:"usage"`
		Metadata *struct {
			Prompt     *int64 `json:"promptTokenCount"`
			Candidates *int64 `json:"candidatesTokenCount"`
			Thoughts   *int64 `json:"thoughtsTokenCount"`
			Total      *int64 `json:"totalTokenCount"`
			ToolPrompt *int64 `json:"toolUsePromptTokenCount"`
		} `json:"usageMetadata"`
	}
	if !decodeOfficialUsage(raw, &envelope) {
		return ObservedUsage{}
	}
	var input, output int64
	switch cfg.Adapter {
	case AdapterQwenCompatible, AdapterOpenAICompatible:
		if !matchesResponseModel(envelope.Model, cfg) {
			return ObservedUsage{}
		}
		return observeReferenceTextUsage(raw)
	case AdapterClaudeNative:
		u := envelope.Usage
		if !matchesResponseModel(envelope.Model, cfg) || u == nil || !observedCounter(u.Input, 1<<30) ||
			!observedCounter(u.Output, 1<<30) || !observedCounter(u.CacheRead, 1<<30) || !observedCounter(u.CacheWrite, 1<<30) {
			return ObservedUsage{}
		}
		// Anthropic's ordinary input count excludes both cache counters.
		// Explicit parts are required to derive the reference input total,
		// including for a complete zero-usage proof.
		input, output = *u.Input+*u.CacheRead+*u.CacheWrite, *u.Output
	case AdapterGoogleGenerateContent:
		u := envelope.Metadata
		if !matchesResponseModel(envelope.ModelVersion, cfg) || u == nil || !observedCounter(u.Prompt, 1<<30) ||
			!observedCounter(u.Candidates, 1<<30) || !observedCounter(u.Total, 2<<30) ||
			*u.Total < *u.Prompt+*u.Candidates || u.ToolPrompt != nil && *u.ToolPrompt != 0 {
			return ObservedUsage{}
		}
		input, output = *u.Prompt, *u.Total-*u.Prompt
		// Absent thought count is not interpreted as zero: output is derived
		// from explicit total minus input, so it includes any hidden thoughts.
		if u.Thoughts != nil && (!observedCounter(u.Thoughts, 1<<30) || *u.Candidates+*u.Thoughts != output) {
			return ObservedUsage{}
		}
	default:
		return ObservedUsage{}
	}
	if input > 1<<30 || output > 1<<30 {
		return ObservedUsage{}
	}
	return ObservedUsage{PromptTokens: int(input), CompletionTokens: int(output), TotalTokens: int(input + output), Known: true,
		BillableUsage: &aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceTextTotalsV1, Complete: true, InputText: input, OutputText: output}}
}
