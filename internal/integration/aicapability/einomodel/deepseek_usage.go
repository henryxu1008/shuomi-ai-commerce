package einomodel

import "task-processor/internal/aicapability"

// This strategy uses only its declared basic counters. Optional supplier cache
// or pricing metadata is neither required nor represented as observed zero.
func observeReferenceTextUsage(raw []byte) ObservedUsage {
	var envelope struct {
		Usage *struct {
			Prompt     *int64 `json:"prompt_tokens"`
			Completion *int64 `json:"completion_tokens"`
			Total      *int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if !decodeOfficialUsage(raw, &envelope) || envelope.Usage == nil {
		return ObservedUsage{}
	}
	u := envelope.Usage
	if !observedCounter(u.Prompt, 1<<30) || !observedCounter(u.Completion, 1<<30) || !observedCounter(u.Total, 2<<30) || *u.Prompt+*u.Completion != *u.Total {
		return ObservedUsage{}
	}
	return ObservedUsage{PromptTokens: int(*u.Prompt), CompletionTokens: int(*u.Completion), TotalTokens: int(*u.Total), Known: true,
		BillableUsage: &aicapability.BillableModelUsage{MappingVersion: aicapability.ReferenceTextTotalsV1, Complete: true, InputText: *u.Prompt, OutputText: *u.Completion}}
}

func observeDeepSeekUsage(raw []byte) ObservedUsage {
	var envelope struct {
		Model string `json:"model"`
	}
	if !decodeOfficialUsage(raw, &envelope) || envelope.Model != "deepseek-v4.1-flash" {
		return ObservedUsage{}
	}
	return observeReferenceTextUsage(raw)
}
