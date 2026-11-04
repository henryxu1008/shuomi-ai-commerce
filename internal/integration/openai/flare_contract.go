package openai

// These are transport observations and input limits, not tariffs or balances.
// Application composition maps them to the existing governed billing owner.
const FlareImageModel = "gpt-image-2.5-flare-2026-09-08"

type FlareImageInputLimits struct {
	Size, Quality                                 string
	AllowAutoSize                                 bool
	MaximumPromptBytes, MaximumReferenceImages    int
	MaximumReferenceBytes, MaximumReferencePixels int64
}

func (e FlareImageInputLimits) Validate() error {
	if e.Size != "1024x1024" || e.Quality != "auto" || !e.AllowAutoSize || e.MaximumPromptBytes < 1 || e.MaximumPromptBytes > 64<<10 || e.MaximumReferenceImages < 1 || e.MaximumReferenceImages > 16 || e.MaximumReferenceBytes < 1 || e.MaximumReferenceBytes > 64<<20 || e.MaximumReferencePixels < 1 || e.MaximumReferencePixels > 1<<30 {
		return ErrFlareUsageUnknown
	}
	return nil
}

type FlareImageUsage struct {
	InputTokens, OutputTokens, TotalTokens int64
	InputTextTokens, InputImageTokens      int64
	InputTextKnown, InputImageKnown        bool
}

func (u FlareImageUsage) Validate() error {
	if u.InputTokens < 0 || u.InputTokens > 1<<30 || u.OutputTokens < 0 || u.OutputTokens > 1<<30 || u.TotalTokens != u.InputTokens+u.OutputTokens ||
		u.InputTextTokens < 0 || u.InputTextTokens > u.InputTokens || u.InputImageTokens < 0 || u.InputImageTokens > u.InputTokens ||
		!u.InputTextKnown && u.InputTextTokens != 0 || !u.InputImageKnown && u.InputImageTokens != 0 || u.InputTextKnown && u.InputImageKnown && u.InputTextTokens+u.InputImageTokens != u.InputTokens {
		return ErrFlareUsageUnknown
	}
	return nil
}

// A cost projection supplied by the existing quote owner. The adapter never
// computes a tariff, reserves resources or treats this as vendor hard evidence.
type SourceImageReferenceQuote struct {
	PriceVersion, ProfileDigest string
	MaximumCostMicros           int64
}
