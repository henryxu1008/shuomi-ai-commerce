package einomodel

import (
	"encoding/json"
	"task-processor/internal/aicapability"

	strict "sigs.k8s.io/json"
)

// This checks an exact synthetic/observed envelope's token partitions, not
// provider qualification. In particular created does NOT become BillingTimeUTC.
// No actual billing clock policy has been admitted for this provider/model.
func observeDeepSeekPartitions(raw []byte) (aicapability.BillableModelUsage, error) {
	var envelope map[string]json.RawMessage
	if !decodeOfficialUsage(raw, &envelope) {
		return aicapability.BillableModelUsage{}, ErrUsageUnknown
	}
	for key := range envelope {
		switch key {
		case "id", "object", "created", "model", "choices", "usage", "system_fingerprint", "service_tier":
		default:
			return aicapability.BillableModelUsage{}, ErrUsageUnknown
		}
	}
	var modelID string
	if json.Unmarshal(envelope["model"], &modelID) != nil || modelID != "deepseek-v4.1-flash" {
		return aicapability.BillableModelUsage{}, ErrUsageUnknown
	}
	var usage struct {
		Prompt        *int64 `json:"prompt_tokens"`
		Completion    *int64 `json:"completion_tokens"`
		Total         *int64 `json:"total_tokens"`
		PromptDetails *struct {
			Cached *int64 `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		CompletionDetails *struct {
			Reasoning *int64 `json:"reasoning_tokens"`
		} `json:"completion_tokens_details"`
	}
	violations, err := strict.UnmarshalStrict(envelope["usage"], &usage, strict.DisallowDuplicateFields, strict.DisallowUnknownFields)
	if err != nil || len(violations) != 0 {
		return aicapability.BillableModelUsage{}, ErrUsageUnknown
	}
	return ObserveOfficialOpenAITextUsage(raw)
}

func observeDeepSeekUsage(raw []byte) ObservedUsage {
	partitions, err := observeDeepSeekPartitions(raw)
	if err != nil {
		return ObservedUsage{}
	}
	// Totals may support diagnostics and the separate channel cost bound, but
	// cannot produce a billable terminal proof without an admitted billing time.
	return ObservedUsage{PromptTokens: int(partitions.InputText), CompletionTokens: int(partitions.OutputText), TotalTokens: int(partitions.InputText + partitions.OutputText), Known: true}
}
