package publishing

import (
	"strings"

	sheinproduct "task-processor/internal/shein/api/product"
)

// FinalSubmitImagesRequireSKC reports whether final submit image readiness must
// include SKC/swatch evidence for the action.
func FinalSubmitImagesRequireSKC(action string) bool {
	return FinalReviewRequired(action)
}

// FinalSubmitImageReadinessInput is the distilled image state for final submit readiness.
type FinalSubmitImageReadinessInput struct {
	HasFinalDraft bool
	HasMainImage  bool
	HasGallery    bool
	HasSKCImage   bool
	HasSwatchRole bool
	RequiresSKC   bool
}

// SubmitImageDraftInput is the neutral image draft shape used by submit image policies.
type SubmitImageDraftInput struct {
	MainImage string
	WhiteBg   string
	Gallery   []string
	Source    []string
}

// FinalSubmitImagesReady reports whether final submit images satisfy action-specific readiness.
func FinalSubmitImagesReady(action string, input FinalSubmitImageReadinessInput) (bool, string) {
	if !input.HasFinalDraft {
		return false, "缺少最终图片草稿，无法确认主图和图库"
	}
	if !input.HasMainImage {
		return false, "最终确认页还没有设置主图"
	}
	if !input.HasGallery {
		return false, "最终图库为空，提交前至少需要一张图库图片"
	}
	if !input.RequiresSKC && !FinalSubmitImagesRequireSKC(action) {
		return true, "草稿保存图片已具备主图和图库；色块图、SKC 图和尺寸图会在正式发布前严格校验"
	}
	if !input.HasSKCImage {
		return false, "缺少 SKC/色块图，提交前需要为每个颜色规格准备可提交图片"
	}
	if !input.HasSwatchRole {
		return false, "缺少色块图标记，请在 SHEIN data images 中标记一张色块图"
	}
	return true, "最终图片已具备主图、图库和可用的色块/SKC 图；尺寸图未选择时不阻断提交"
}

// SubmitImageDraftHasImage reports whether an image draft contains any image URL.
func SubmitImageDraftHasImage(input SubmitImageDraftInput) bool {
	if SubmitImageURLsHaveImage(input.MainImage, input.WhiteBg) {
		return true
	}
	return SubmitImageURLSliceHasImage(input.Gallery) || SubmitImageURLSliceHasImage(input.Source)
}

// SubmitImageURLSliceHasImage reports whether any URL in a slice is non-empty.
func SubmitImageURLSliceHasImage(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return true
		}
	}
	return false
}

// SubmitImageURLsHaveImage reports whether any URL is non-empty.
func SubmitImageURLsHaveImage(values ...string) bool {
	return SubmitImageURLSliceHasImage(values)
}

// ProductImageInfoHasImage reports whether SHEIN product image info contains any image URL.
func ProductImageInfoHasImage(info *sheinproduct.ImageInfo) bool {
	if info == nil {
		return false
	}
	urls := make([]string, 0, len(info.ImageInfoList))
	for _, image := range info.ImageInfoList {
		urls = append(urls, image.ImageURL)
	}
	return SubmitImageURLSliceHasImage(urls)
}

// UniqueNonEmptyImageURLs returns trimmed unique image URLs preserving order.
func UniqueNonEmptyImageURLs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
