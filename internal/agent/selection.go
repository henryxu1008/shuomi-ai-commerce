package agent

import (
	"strconv"

	"github.com/google/uuid"
)

const MaxSelectionProducts = 5
const SelectionContextKind = "product-selection-v1"

// SelectionProductRef carries exact existing Collection/Catalog identities.
// It grants no read authority; the consumer reauthorizes every candidate.
type SelectionProductRef struct {
	ItemID, ItemRevision, ProductKey, CatalogVersion, PublicationID string
}

func (r SelectionProductRef) Valid() bool {
	return selectionUUID(r.ItemID) && selectionVersion(r.ItemRevision) &&
		ValidID(r.ProductKey) && selectionVersion(r.CatalogVersion) && ValidID(r.PublicationID)
}

func selectionVersion(value string) bool {
	v, err := strconv.ParseUint(value, 10, 63)
	return err == nil && v > 0 && strconv.FormatUint(v, 10) == value
}

func selectionUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

func (b Binding) validSelection() bool {
	if !selectionUUID(b.ContextID) || b.TargetPlatform != "product" ||
		b.ProductKey != "" || b.CatalogVersion != "" || b.PublicationID != "" {
		return false
	}
	count, previous, ended := 0, "", false
	products := map[string]bool{}
	for _, ref := range b.Selection {
		if ref == (SelectionProductRef{}) {
			ended = true
			continue
		}
		if ended || !ref.Valid() || ref.ItemID <= previous || products[ref.ProductKey] {
			return false
		}
		products[ref.ProductKey] = true
		previous, count = ref.ItemID, count+1
	}
	return count >= 2
}
