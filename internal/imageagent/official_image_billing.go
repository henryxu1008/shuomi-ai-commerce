package imageagent

import "task-processor/internal/aicapability"

// Image quotes consume the one official-price algorithm and original resource
// owner, with their own admitted image-usage bound and immutable source facts.
type ImageBilling = aicapability.OfficialModelBilling

func validImageBilling(b ImageBilling, model, version string, maximum int64) bool {
	n, err := b.ReservationMicropoints(model)
	return err == nil && maximum == n && b.OfficialTariff.PriceVersion == version
}
