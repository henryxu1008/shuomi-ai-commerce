package research

import (
	"context"
	"task-processor/internal/product/collection"
)

// manualOriginalSource is supplied by server composition, never by an HTTP
// request. It grants one existing original manual use, not execution eligibility.
type manualOriginalSource interface {
	ManualOriginal(collection.Scope, string) (candidate, link string, selection Selection, enabled bool)
}

func (s *Service) manualOriginal(scope collection.Scope, project string) (string, string, Selection, bool) {
	if source, ok := s.Sources.(manualOriginalSource); ok {
		return source.ManualOriginal(scope, project)
	}
	return "", "", Selection{}, false
}

func sameManualOriginal(a, b Selection) bool {
	return a.ItemID == b.ItemID || a.PublicationID == b.PublicationID
}

// Called after original-key replay: committed receipts remain recoverable.
func (s *Service) authorizeManualCommand(ctx context.Context, scope collection.Scope, project string, c Command) error {
	candidate, link, selection, enabled := s.manualOriginal(scope, project)
	if !enabled {
		return nil
	}
	for _, in := range c.Selections {
		if sameManualOriginal(in, selection) {
			return ErrUnsupported
		}
	}
	if c.CandidateID == "" {
		return nil
	}
	current, err := s.Store.Load(ctx, scope, project, c.CandidateID)
	if err != nil {
		return err
	}
	protected := c.CandidateID == candidate
	for _, in := range current.Links {
		protected = protected || sameManualOriginal(in.Ref.Selection, selection)
	}
	if !protected {
		return nil
	}
	if current.ID != candidate || current.SelectedLinkID != link {
		return ErrUnsupported
	}
	bound := false
	for _, in := range current.Links {
		bound = bound || in.ID == link && !in.Removed && in.Ref.Selection == selection
	}
	if !bound {
		return ErrConflict
	}
	switch c.Action {
	case SaveAssessment, "review":
		return nil
	case "decide":
		if c.State == "REJECTED" {
			return nil
		}
	}
	return ErrUnsupported
}

func (s *Service) manualEvidenceAllowed(scope collection.Scope, project, candidate, link string, in Selection) bool {
	id, selected, selection, enabled := s.manualOriginal(scope, project)
	if !enabled || candidate != id && !sameManualOriginal(in, selection) {
		return true
	}
	return candidate == id && link == selected && in == selection
}
