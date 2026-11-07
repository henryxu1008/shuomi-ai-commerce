package aicapability

import (
	"encoding/hex"
	"strings"
)

const (
	SingleImageMiniModel         = "gpt-5.4-mini-2026-03-17"
	SingleImageMiniAdapterPolicy = "openai-mini-vision-v1"
	SingleImageMiniMetering      = "openai-mini-image-high-v1"
	SingleImagePromptTokenBound  = int64(3001)
	MaximumSingleImageBytes      = 768 << 10
	MaximumSingleImageDimension  = 8192
	MaximumSingleImagePixels     = 16777216
)

// SingleImageInput is finite content identity, not an image or an authority.
// The original source/asset owner authorizes SelectionDigest; the final guarded
// transport must verify the actual in-memory bytes against ContentSHA256.
type SingleImageInput struct {
	SelectionDigest string
	ContentSHA256   string
	MediaType       string
	ByteLength      int
	Width, Height   int
}

func VisionTextOperation(operation Operation) bool {
	return operation == OperationProductImageAnalysis || operation == OperationOwnProductIntake
}

func singleImageDigest(value string) bool {
	if len(value) != 64 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func (i SingleImageInput) Valid() bool {
	return singleImageDigest(i.SelectionDigest) && singleImageDigest(i.ContentSHA256) &&
		(i.MediaType == "image/jpeg" || i.MediaType == "image/png") &&
		i.ByteLength > 0 && i.ByteLength <= MaximumSingleImageBytes &&
		i.Width > 0 && i.Width <= MaximumSingleImageDimension &&
		i.Height > 0 && i.Height <= MaximumSingleImageDimension &&
		int64(i.Width)*int64(i.Height) <= MaximumSingleImagePixels
}

func SingleImageProfile(p ModelProfile) bool {
	return p.Validate() == nil && SingleImageProfileShape(p) &&
		(p.OfficialTariff.IsReferenceEstimate() || p.AdapterPolicyVersion == SingleImageMiniAdapterPolicy &&
			p.OfficialTariff.UsageVersion == "openai-text-v1" && p.OfficialTariff.Channel == "openai-direct")
}

func SingleImageProfileShape(p ModelProfile) bool {
	if p.ValidateShape() != nil || SingleImagePromptEstimate(p) <= 0 ||
		SingleImagePromptEstimate(p) >= p.MaximumPromptTokens {
		return false
	}
	switch p.AdapterPolicyVersion {
	case SingleImageMiniAdapterPolicy:
		return p.ProviderID == "openai" && p.AdapterKind == "openai-compatible" && p.ModelID == SingleImageMiniModel &&
			p.UsageMappingVersion == "openai-text-v1" &&
			(p.ImagePromptTokenEstimate == 0 || p.ImagePromptTokenEstimate == SingleImagePromptTokenBound)
	case EinoOpenAIImagePolicy:
		return p.AdapterKind == "openai-compatible" && p.UsageMappingVersion == "openai-text-v1"
	case EinoGeminiImagePolicy:
		return p.ProviderID == "google" && p.AdapterKind == "google-generate-content" && p.UsageMappingVersion == GeminiUsageMapping
	case EinoQwenImagePolicy:
		return p.ProviderID == "DASHSCOPE" && p.AdapterKind == "qwen-compatible" && p.UsageMappingVersion == QwenUsageMapping
	}
	return false
}
