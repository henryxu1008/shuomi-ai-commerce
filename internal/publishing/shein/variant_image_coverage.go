package shein

import sheinmarketpub "task-processor/internal/marketplace/shein/publishing"

// VariantImageCoverageStatus returns the current blocked variant image coverage message.
func VariantImageCoverageStatus(pkg *Package) (string, bool) {
	if pkg == nil {
		return "", false
	}
	return sheinmarketpub.VariantImageCoverageStatus(pkg.Metadata)
}
