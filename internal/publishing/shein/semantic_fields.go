package shein

import sheinproduct "task-processor/internal/shein/api/product"

// SetPreviewPayload updates the canonical preview payload.
func SetPreviewPayload(pkg *Package, preview *sheinproduct.Product) *Package {
	if pkg == nil {
		return nil
	}
	pkg.PreviewPayload = preview
	return pkg
}
