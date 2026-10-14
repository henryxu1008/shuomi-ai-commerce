package collection

import (
	"encoding/json"
	"math"
	"strconv"
	"task-processor/internal/product/sourcing"
)

// A new immutable user declaration records its exact original-source basis.
func OwnRevisionEnvelope(operation string, product OwnProduct, publication string, version uint64) (sourcing.SourceEnvelope, error) {
	if !ValidID(publication) || version == 0 || version > math.MaxInt64 {
		return sourcing.SourceEnvelope{}, ErrInvalid
	}
	envelope, err := OwnEnvelope(operation, product)
	if err != nil {
		return sourcing.SourceEnvelope{}, err
	}
	envelope.RawReference.Metadata = map[string]string{"previous_publication_id": publication, "previous_catalog_version": strconv.FormatUint(version, 10)}
	items := 2 + len(product.Attributes) + len(product.Images) + len(product.Variants)
	for _, variant := range product.Variants {
		items += len(variant.Attributes) + len(variant.Images)
	}
	if items > sourcing.MaxSourceEnvelopeAggregateItems {
		return sourcing.SourceEnvelope{}, ErrInvalid
	}
	// Normalize again: the lineage metadata belongs to the existing Source budget.
	normalized, err := sourcing.NormalizePublicationEnvelope(envelope)
	if err != nil {
		return sourcing.SourceEnvelope{}, ErrInvalid
	}
	encoded, err := json.Marshal(normalized)
	if err != nil || len(encoded) > sourcing.MaxEncodedEnvelopeBytes {
		return sourcing.SourceEnvelope{}, ErrInvalid
	}
	return normalized, nil
}
