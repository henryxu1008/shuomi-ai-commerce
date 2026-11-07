package collection

import "strings"

const ResearchFixtureProductPrefix = "research-fixture-v1-"
const ResearchFixtureSupplierPublicationPrefix = "source-run:research-acceptance:"

func ResearchFixtureSupplierOperation(publicationID string) (string, bool) {
	id, ok := strings.CutPrefix(publicationID, ResearchFixtureSupplierPublicationPrefix)
	return id, ok && ValidID(id)
}

// IsResearchFixtureProduct identifies synthetic sources even when fixture loading is disabled.
func IsResearchFixtureProduct(key string) bool {
	return strings.HasPrefix(key, ResearchFixtureProductPrefix)
}
