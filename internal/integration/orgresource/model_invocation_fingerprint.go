package orgresourceadapter

import "task-processor/internal/aicapability"

// modelPointFactEncoding preserves the current ledger's exact JSON identity.
// These fixed zero slots are private to the ledger; they are not writable
// model-review results and cannot reconstruct retired review histories.
type modelPointFactEncoding struct {
	aicapability.InvocationRecord
	ReviewScore            float64
	ReviewNeedsHumanReview bool
	ReviewReasons          []string
}

func modelPointFactHash(fact aicapability.InvocationRecord) string {
	return modelPointHash(modelPointFactEncoding{InvocationRecord: fact})
}
