package publishing

import "strings"

const (
	// VariantImageCoverageStatusKey stores the variant image coverage status in package metadata.
	VariantImageCoverageStatusKey = "variant_image_coverage_status"
	// VariantImageCoverageMessageKey stores the variant image coverage warning in package metadata.
	VariantImageCoverageMessageKey = "variant_image_coverage_message"
)

const (
	variantImageCoverageStatusBlocked = "blocked"
)

// VariantImageCoverageStatus returns the current blocked variant image coverage message.
func VariantImageCoverageStatus(metadata map[string]string) (string, bool) {
	if metadata == nil {
		return "", false
	}
	if strings.TrimSpace(metadata[VariantImageCoverageStatusKey]) != variantImageCoverageStatusBlocked {
		return "", false
	}
	message := strings.TrimSpace(metadata[VariantImageCoverageMessageKey])
	if message == "" {
		message = "变体图片覆盖不完整，请为每个颜色规格补齐独立商品图后再提交"
	}
	return message, true
}
