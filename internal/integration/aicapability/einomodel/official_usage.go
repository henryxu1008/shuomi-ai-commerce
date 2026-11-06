package einomodel

import (
	"task-processor/internal/aicapability"

	strict "sigs.k8s.io/json"
)

func decodeOfficialUsage(raw []byte, target any) bool {
	// Existing transports enforce their own smaller limits before calling here.
	// Parsing never authorizes a larger request, response or provider expense.
	if len(raw) == 0 || len(raw) > 16<<20 {
		return false
	}
	violations, err := strict.UnmarshalStrict(raw, target, strict.DisallowDuplicateFields)
	return err == nil && len(violations) == 0
}

func observedCounter(n *int64, max int64) bool { return n != nil && *n >= 0 && *n <= max }

// ObserveOfficialOpenAITextUsage is for an admitted Chat Completions billing
// profile with cached-input pricing. It preserves presence; absent cache counts
// cannot silently become zero or full-price inputs. Mini image inputs and
// reasoning outputs are already contained in the prompt/completion totals.
func ObserveOfficialOpenAITextUsage(raw []byte) (aicapability.BillableModelUsage, error) {
	var envelope struct {
		Usage *struct {
			Prompt        *int64 `json:"prompt_tokens"`
			Completion    *int64 `json:"completion_tokens"`
			Total         *int64 `json:"total_tokens"`
			PromptDetails *struct {
				Cached *int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
			CompletionDetails *struct {
				Reasoning *int64 `json:"reasoning_tokens"`
			} `json:"completion_tokens_details"`
		} `json:"usage"`
	}
	unknown := func() (aicapability.BillableModelUsage, error) {
		return aicapability.BillableModelUsage{}, ErrUsageUnknown
	}
	if !decodeOfficialUsage(raw, &envelope) || envelope.Usage == nil {
		return unknown()
	}
	u := envelope.Usage
	if !observedCounter(u.Prompt, 1<<30) || !observedCounter(u.Completion, 1<<30) || !observedCounter(u.Total, 2<<30) || *u.Prompt+*u.Completion != *u.Total || u.PromptDetails == nil || !observedCounter(u.PromptDetails.Cached, *u.Prompt) {
		return unknown()
	}
	if u.CompletionDetails != nil && u.CompletionDetails.Reasoning != nil && !observedCounter(u.CompletionDetails.Reasoning, *u.Completion) {
		return unknown()
	}
	return aicapability.BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: *u.Prompt, CachedInputText: *u.PromptDetails.Cached, OutputText: *u.Completion}, nil
}

// ObserveOfficialOpenAIImageUsage is for the separately admitted direct Images
// image-only protocol. It does not admit a model or compute a pre-send bound.
// Direct Images inputs have no cached-input discount under this protocol.
func ObserveOfficialOpenAIImageUsage(raw []byte) (aicapability.BillableModelUsage, error) {
	var envelope struct {
		Usage *struct {
			Input        *int64 `json:"input_tokens"`
			Output       *int64 `json:"output_tokens"`
			Total        *int64 `json:"total_tokens"`
			InputDetails *struct {
				Text  *int64 `json:"text_tokens"`
				Image *int64 `json:"image_tokens"`
			} `json:"input_tokens_details"`
			OutputDetails *struct {
				Text  *int64 `json:"text_tokens"`
				Image *int64 `json:"image_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	}
	unknown := func() (aicapability.BillableModelUsage, error) {
		return aicapability.BillableModelUsage{}, ErrUsageUnknown
	}
	if !decodeOfficialUsage(raw, &envelope) || envelope.Usage == nil {
		return unknown()
	}
	u := envelope.Usage
	if !observedCounter(u.Input, 1<<30) || !observedCounter(u.Output, 1<<30) || !observedCounter(u.Total, 2<<30) || *u.Input+*u.Output != *u.Total || u.InputDetails == nil || !observedCounter(u.InputDetails.Text, *u.Input) || !observedCounter(u.InputDetails.Image, *u.Input) || *u.InputDetails.Text+*u.InputDetails.Image != *u.Input {
		return unknown()
	}
	if d := u.OutputDetails; d != nil {
		if d.Text != nil && *d.Text != 0 || d.Image != nil && *d.Image != *u.Output {
			return unknown()
		}
	}
	return aicapability.BillableModelUsage{MappingVersion: "openai-images-direct-v1", Complete: true, InputText: *u.InputDetails.Text, InputImage: *u.InputDetails.Image, OutputImage: *u.Output}, nil
}
