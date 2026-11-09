package preparation

import "task-processor/internal/product/collection"

type manualOriginalExclusion struct {
	scope                 Scope
	itemID, publicationID string
}

// WithManualOriginalExclusion configures a copy at server composition time.
// There is no request-controlled flag and no persisted second source owner.
func (s *Service) WithManualOriginalExclusion(scope Scope, itemID, publicationID string) (*Service, error) {
	if s == nil || scope.Validate() != nil || !collection.ValidID(itemID) || !collection.ValidID(publicationID) {
		return nil, ErrInvalid
	}
	configured := *s
	configured.manualOriginal = &manualOriginalExclusion{scope: scope, itemID: itemID, publicationID: publicationID}
	return &configured, nil
}

// CheckManualOriginal is evaluated for every selected item by the original
// transfer transaction, including whole-batch selections beyond the first page.
func (c TransferCommit) CheckManualOriginal(item collection.Item) error {
	bound := c.manualOriginal
	if bound != nil && c.scope == bound.scope && (item.ID == bound.itemID || item.Source.PublicationID == bound.publicationID) {
		return ErrForbidden
	}
	return nil
}
