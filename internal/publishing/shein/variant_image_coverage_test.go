package shein

import (
	sheinpolicy "task-processor/internal/marketplace/shein/publishing"
	"testing"
)

func TestVariantImageCoverageReadsCanonicalMetadata(t *testing.T) {
	t.Parallel()

	pkg := &Package{}
	pkg.Metadata = map[string]string{sheinpolicy.VariantImageCoverageMessageKey: "needs images", sheinpolicy.VariantImageCoverageStatusKey: "blocked"}
	message, blocked := VariantImageCoverageStatus(pkg)
	if !blocked || message != "needs images" {
		t.Fatalf("VariantImageCoverageStatus() = %q, %v; want blocked message", message, blocked)
	}

	pkg.Metadata = nil
	if message, blocked := VariantImageCoverageStatus(pkg); message != "" || blocked {
		t.Fatalf("cleared metadata status = %q, %v", message, blocked)
	}
}
