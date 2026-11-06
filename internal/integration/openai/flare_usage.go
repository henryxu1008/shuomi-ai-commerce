package openai

import (
	"encoding/json"
	"errors"

	strict "sigs.k8s.io/json"
)

var ErrFlareUsageUnknown = errors.New("Flare image usage is unknown")

// ObserveFlareImageUsage maps explicit basic counters under the approved
// reference policy. Complete does not assert complete supplier billing detail.
func ObserveFlareImageUsage(raw []byte) (FlareImageUsage, error) {
	var body struct {
		Usage *struct {
			Input        *int64 `json:"input_tokens"`
			Output       *int64 `json:"output_tokens"`
			Total        *int64 `json:"total_tokens"`
			InputDetails *struct {
				Text  *int64 `json:"text_tokens"`
				Image *int64 `json:"image_tokens"`
			} `json:"input_tokens_details"`
			OutputDetails map[string]json.RawMessage `json:"output_tokens_details"`
		} `json:"usage"`
	}
	unknown := func() (FlareImageUsage, error) {
		return FlareImageUsage{}, ErrFlareUsageUnknown
	}
	if len(raw) == 0 || len(raw) > 48<<20 {
		return unknown()
	}
	violations, err := strict.UnmarshalStrict(raw, &body, strict.DisallowDuplicateFields)
	if err != nil || len(violations) != 0 || body.Usage == nil {
		return unknown()
	}
	u := body.Usage
	if !flareCounter(u.Input, 1<<30) || !flareCounter(u.Output, 1<<30) || !flareCounter(u.Total, 2<<30) || *u.Input+*u.Output != *u.Total {
		return unknown()
	}
	result := FlareImageUsage{InputTokens: *u.Input, OutputTokens: *u.Output, TotalTokens: *u.Total}
	if d := u.InputDetails; d != nil {
		if d.Text != nil {
			result.InputTextKnown, result.InputTextTokens = true, *d.Text
		}
		if d.Image != nil {
			result.InputImageKnown, result.InputImageTokens = true, *d.Image
		}
	}
	for field, raw := range u.OutputDetails {
		var count *int64
		if json.Unmarshal(raw, &count) != nil || !flareCounter(count, 1<<30) || field == "image_tokens" && *count != *u.Output || field != "image_tokens" && *count != 0 {
			return unknown()
		}
	}
	if result.Validate() != nil {
		return unknown()
	}
	return result, nil
}

func flareCounter(n *int64, limit int64) bool { return n != nil && *n >= 0 && *n <= limit }
