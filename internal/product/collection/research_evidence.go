package collection

import (
	"context"
	"strings"

	"task-processor/internal/product/sourcing"
)

type researchEvidenceStore interface {
	ReadResearchEvidenceItem(context.Context, Scope, string) (Item, error)
}

// ReadResearchEvidenceItem checks the current private item and its live batch.
// Only a new selection supplies a revision. An already saved exact binding
// remains readable after a move when the current item and batch are accessible.
func (s *Service) ReadResearchEvidenceItem(ctx context.Context, scope Scope, in SelectionInput, original *Source) (Item, error) {
	current, err := s.authorize(ctx, PermissionRead)
	if err != nil || current != scope {
		return Item{}, ErrForbidden
	}
	store, ok := s.store.(researchEvidenceStore)
	if !ok {
		return Item{}, ErrUnavailable
	}
	if !ValidID(in.ItemID) || len(in.OriginalPublicationID) > 128 || strings.TrimSpace(in.OriginalPublicationID) == "" || in.OriginalVersion == 0 ||
		(in.ExpectedRevision <= 0 && original == nil) {
		return Item{}, ErrInvalid
	}
	item, err := store.ReadResearchEvidenceItem(ctx, scope, in.ItemID)
	if err != nil {
		return Item{}, err
	}
	if item.ID != in.ItemID || item.ArchivedAt != nil || item.Source.PublicationID != in.OriginalPublicationID ||
		item.Source.Version != in.OriginalVersion || (in.ExpectedRevision > 0 && item.Revision != in.ExpectedRevision) ||
		(original != nil && item.Source != *original) {
		return Item{}, ErrConflict
	}
	return item, nil
}

// ReadResearchSupplierAcquisition reuses the actor-bound published acquisition
// reader. Its existing live Sourcing permission is required for this read.
func (s *Service) ReadResearchSupplierAcquisition(ctx context.Context, scope Scope, in SelectionInput, original *Source) (Item, sourcing.PublishedAcquisition, error) {
	item, err := s.ReadResearchEvidenceItem(ctx, scope, in, original)
	if err != nil {
		return Item{}, sourcing.PublishedAcquisition{}, err
	}
	if item.Source.Kind != "acquisition" || !ValidID(item.Source.OperationID) || s.acquisitions == nil {
		return Item{}, sourcing.PublishedAcquisition{}, ErrNotFound
	}
	published, err := s.acquisitions.ReadPublished(ctx, item.Source.OperationID)
	if err != nil {
		return Item{}, sourcing.PublishedAcquisition{}, err
	}
	item, err = s.ReadResearchEvidenceItem(ctx, scope, in, &item.Source)
	if err != nil {
		return Item{}, sourcing.PublishedAcquisition{}, err
	}
	return item, published, nil
}
