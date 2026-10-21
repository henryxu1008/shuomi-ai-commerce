package sourcing

import (
	"errors"
	"strings"
	"task-processor/internal/product/description"
)

// ErrSourceIdentityRequired means an envelope lacks a strict source identity.
var ErrSourceIdentityRequired = errors.New("source identity is required")

// Normalize returns a provider-neutral envelope with normalized identity and
// warning codes while preserving its raw evidence, lineage, and source metadata.
func Normalize(in SourceEnvelope) (SourceEnvelope, error) {
	out := in.Normalize()
	if !out.Identity.Valid() {
		return SourceEnvelope{}, ErrSourceIdentityRequired
	}
	images := []string{}
	for _, image := range out.AssetCandidates {
		images = append(images, image.URL)
	}
	for _, v := range out.ProductCandidate.Variants {
		for _, image := range v.Images {
			images = append(images, image.URL)
		}
	}
	if err := description.Validate(out.ProductCandidate.Description, out.ProductCandidate.DescriptionFormat, images); err != nil {
		return SourceEnvelope{}, err
	}
	return out, nil
}

// NormalizePublicationEnvelope enforces the current source-neutral identity
// contract at the durable publication boundary. The four retained encoding
// slots must be zero after their original whitespace normalization; they never
// supply identity values.
func NormalizePublicationEnvelope(in SourceEnvelope) (SourceEnvelope, error) {
	identity := in.Identity
	if strings.TrimSpace(identity.SourceType) == "" ||
		strings.TrimSpace(identity.SourcePlatform) == "" ||
		strings.TrimSpace(identity.SourceID) == "" ||
		strings.TrimSpace(identity.Platform) != "" ||
		strings.TrimSpace(identity.Region) != "" ||
		strings.TrimSpace(identity.ProductID) != "" ||
		identity.StoreID != 0 {
		return SourceEnvelope{}, ErrInvalidSourcePublication
	}

	out, err := Normalize(in)
	if err != nil {
		return SourceEnvelope{}, err
	}
	// Preserve the existing zero-slot encoding for empty and whitespace input.
	// No retained slot participates in source identity derivation.
	out.Identity.Platform = ""
	out.Identity.Region = ""
	out.Identity.ProductID = ""
	out.Identity.StoreID = 0
	return out, nil
}
