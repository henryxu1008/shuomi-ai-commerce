package research

import (
	"context"
	"sort"
	"task-processor/internal/product/collection"
)

type AnalysisChoice struct {
	CandidateID      string    `json:"candidateId"`
	ExpectedRevision int64     `json:"expectedRevision"`
	LinkID           string    `json:"linkId"`
	EvidenceDigest   string    `json:"evidenceDigest"`
	Selection        Selection `json:"selection"`
}
type AnalysisObservation struct {
	Choice      AnalysisChoice
	Observation Observation
}

// InspectSelection reads only exact selected observations, never a page or a
// current Catalog substitute. No result is returned if any choice fails.
func (s *Service) InspectSelection(ctx context.Context, scope collection.Scope, project string, choices []AnalysisChoice) ([]AnalysisObservation, error) {
	return s.analysisSelection(ctx, scope, project, choices, true)
}
func (s *Service) AuthorizeAnalysisHistory(ctx context.Context, scope collection.Scope, project string, choices []AnalysisChoice) error {
	_, err := s.analysisSelection(ctx, scope, project, choices, false)
	return err
}
func (s *Service) analysisSelection(ctx context.Context, scope collection.Scope, project string, choices []AnalysisChoice, current bool) ([]AnalysisObservation, error) {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return nil, err
	}
	if len(choices) < 2 || len(choices) > 5 {
		return nil, ErrInvalid
	}
	if err := s.Projects.AuthorizeAnalysis(ctx, scope, project, current); err != nil {
		return nil, err
	}
	ordered := append([]AnalysisChoice(nil), choices...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Selection.ItemID < ordered[j].Selection.ItemID })
	seen := map[string]bool{}
	out := make([]AnalysisObservation, 0, len(ordered))
	for index, in := range ordered {
		if !collection.ValidID(in.CandidateID) || in.ExpectedRevision < 1 || !collection.ValidID(in.LinkID) || len(in.EvidenceDigest) != 64 || !ValidSelection(in.Selection) || seen[in.CandidateID] || (index > 0 && in.Selection.ItemID == ordered[index-1].Selection.ItemID) {
			return nil, ErrInvalid
		}
		seen[in.CandidateID] = true
		c, err := s.Store.Load(ctx, scope, project, in.CandidateID)
		if err != nil {
			return nil, err
		}
		if c.ProjectID != project {
			return nil, ErrNotFound
		}
		if current && (c.Revision != in.ExpectedRevision || c.SelectedLinkID != in.LinkID) {
			return nil, ErrConflict
		}
		var ref *EvidenceRef
		for _, link := range c.Links {
			if link.ID == in.LinkID && !link.Removed {
				r := link.Ref
				ref = &r
				break
			}
		}
		if ref == nil {
			return nil, ErrNotFound
		}
		if ref.Selection != in.Selection || ref.Digest() != in.EvidenceDigest {
			return nil, ErrConflict
		}
		if ref.Source.Kind != "amazon_data" {
			return nil, ErrUnsupported
		}
		v, err := s.Sources.Observe(ctx, scope, in.Selection)
		if err != nil {
			return nil, err
		}
		if !ValidObservation(v, in.Selection) || v.Ref != *ref {
			return nil, ErrConflict
		}
		out = append(out, AnalysisObservation{Choice: in, Observation: v})
	}
	if err := s.Projects.AuthorizeAnalysis(ctx, scope, project, current); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Service) AuthorizeAnalysisProject(ctx context.Context, scope collection.Scope, project string, active bool) error {
	if err := s.authorize(ctx, scope, project, false); err != nil {
		return err
	}
	return s.Projects.AuthorizeAnalysis(ctx, scope, project, active)
}
