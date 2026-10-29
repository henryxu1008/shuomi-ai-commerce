package publishing

import "testing"

func TestVariantImageCoverageMetadataRoundTrip(t *testing.T) {
	t.Parallel()
	metadata := map[string]string{VariantImageCoverageStatusKey: "blocked", VariantImageCoverageMessageKey: "needs images"}
	message, blocked := VariantImageCoverageStatus(metadata)
	if !blocked || message != "needs images" {
		t.Fatalf("VariantImageCoverageStatus() = %q, %v; want blocked message", message, blocked)
	}
	for _, metadata := range []map[string]string{nil, {}, {VariantImageCoverageStatusKey: "ready", VariantImageCoverageMessageKey: "ignored"}} {
		if message, blocked := VariantImageCoverageStatus(metadata); blocked || message != "" {
			t.Fatalf("nonblocked metadata = %q, %v", message, blocked)
		}
	}
	message, blocked = VariantImageCoverageStatus(map[string]string{VariantImageCoverageStatusKey: " blocked ", VariantImageCoverageMessageKey: "  "})
	if !blocked || message != "变体图片覆盖不完整，请为每个颜色规格补齐独立商品图后再提交" {
		t.Fatalf("fallback = %q, %v", message, blocked)
	}
	message, blocked = VariantImageCoverageStatus(map[string]string{VariantImageCoverageStatusKey: "blocked", VariantImageCoverageMessageKey: " needs images "})
	if !blocked || message != "needs images" {
		t.Fatalf("trimmed message = %q, %v", message, blocked)
	}
}
