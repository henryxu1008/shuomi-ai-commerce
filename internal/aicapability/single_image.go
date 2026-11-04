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
	Detail          string
	MeteringVersion string
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
		int64(i.Width)*int64(i.Height) <= MaximumSingleImagePixels &&
		i.Detail == "high" && i.MeteringVersion == SingleImageMiniMetering
}

func SingleImageProfile(p ModelProfile) bool {
	return p.Validate() == nil && SingleImageProfileShape(p) && p.OfficialTariff.UsageVersion == "openai-text-v1" && p.OfficialTariff.Channel == "openai-direct"
}

func SingleImageProfileShape(p ModelProfile) bool {
	return p.ValidateShape() == nil && p.ProviderID == "openai" && p.AdapterKind == "openai-compatible" &&
		p.ModelID == SingleImageMiniModel && p.AdapterPolicyVersion == SingleImageMiniAdapterPolicy &&
		p.UsageMappingVersion == "openai-text-v1"
}
