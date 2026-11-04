package aicapability

const ReferenceImagesTotalsV1 = "reference-images-totals-v1"
const FlareImageModel = "gpt-image-2.5-flare-2026-09-08"
const FlareImageProtocol = "openai-images-sync-v1"
const ReferenceImageOperationScope = "product-image-effects-v1"

// ReferenceImageEstimate is a platform-maintained estimate within the original
// image price/intent JSON. It is not a supplier bound or an observed usage fact.
// Value fields preserve canonical equality between API, worker and stored facts.
type ReferenceImageEstimate struct {
	InputTokens            int64  `json:"inputTokens"`
	OutputTokens           int64  `json:"outputTokens"`
	OperationScope         string `json:"operationScope"`
	Size                   string `json:"size"`
	Quality                string `json:"quality"`
	AllowAutoSize          bool   `json:"allowAutoSize"`
	MaximumPromptBytes     int    `json:"maximumPromptBytes"`
	MaximumReferenceImages int    `json:"maximumReferenceImages"`
	MaximumReferenceBytes  int64  `json:"maximumReferenceBytes"`
	MaximumReferencePixels int64  `json:"maximumReferencePixels"`
}

func (e ReferenceImageEstimate) Validate() error {
	if e.InputTokens <= 0 || e.OutputTokens <= 0 || e.InputTokens > 1<<30 || e.OutputTokens > 1<<30 ||
		e.OperationScope != ReferenceImageOperationScope || e.Size != "1024x1024" || e.Quality != "auto" || !e.AllowAutoSize ||
		e.MaximumPromptBytes < 1 || e.MaximumPromptBytes > 64<<10 || e.MaximumReferenceImages < 1 || e.MaximumReferenceImages > 16 ||
		e.MaximumReferenceBytes < 1 || e.MaximumReferenceBytes > 64<<20 || e.MaximumReferencePixels < 1 || e.MaximumReferencePixels > 1<<30 {
		return ErrOfficialModelPrice
	}
	return nil
}

func (p OfficialModelTariff) IsReferenceImageEstimate() bool {
	return p.UsageVersion == ReferenceImagesTotalsV1
}

// ReferenceImageCNYMicros is a budget/display projection of the same settled
// micropoints, never a second charge or a supplier-invoice assertion.
func ReferenceImageCNYMicros(points int64) int64 {
	result := points / 100
	if points%100 != 0 {
		result++
	}
	return result
}

// Known records explicit presence of all three counters, including zero.
// Missing fine partitions remain missing and are never relabelled image tokens.
type ReferenceImageTotals struct {
	Known           bool  `json:"known"`
	Input           int64 `json:"input"`
	Output          int64 `json:"output"`
	Total           int64 `json:"total"`
	InputTextKnown  bool  `json:"inputTextKnown,omitempty"`
	InputImageKnown bool  `json:"inputImageKnown,omitempty"`
}

func (u BillableModelUsage) validateReferenceImages() error {
	t := u.ReferenceImageTotals
	if !u.Complete || !t.Known || t.Input < 0 || t.Input > 1<<30 || t.Output < 0 || t.Output > 1<<30 || t.Total != t.Input+t.Output ||
		u.InputText < 0 || u.InputText > t.Input || u.InputImage < 0 || u.InputImage > t.Input ||
		!t.InputTextKnown && u.InputText != 0 || !t.InputImageKnown && u.InputImage != 0 ||
		t.InputTextKnown && t.InputImageKnown && u.InputText+u.InputImage != t.Input ||
		u.OutputImage != t.Output || u.OutputText != 0 || u.CachedInputText != 0 || u.CachedInputImage != 0 || u.OutputImageItems != 0 {
		return ErrOfficialModelPrice
	}
	return nil
}
