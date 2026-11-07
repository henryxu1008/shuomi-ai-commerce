package research

import (
	"context"

	"golang.org/x/sync/errgroup"

	"task-processor/internal/product/collection"
)

type concurrentPageSources interface {
	CanReadPageEvidenceConcurrently(collection.Scope, string, Selection) bool
}

const pageEvidenceWorkers = 4

// preparePageEvidence overlaps only reads admitted by the source adapter. Other
// originals are read first, so an unusually slow fixture batch cannot crowd them out.
func (s *Service) preparePageEvidence(ctx context.Context, scope collection.Scope, project string, rows []Candidate) map[Selection]*Observation {
	source, ok := s.Sources.(concurrentPageSources)
	if !ok {
		return nil
	}
	results := make(map[Selection]*Observation)
	var sequential, concurrent []Selection
	add := func(selection Selection) {
		if _, seen := results[selection]; seen {
			return
		}
		results[selection] = nil
		if source.CanReadPageEvidenceConcurrently(scope, project, selection) {
			concurrent = append(concurrent, selection)
		} else {
			sequential = append(sequential, selection)
		}
	}
	for _, candidate := range rows {
		for _, link := range candidate.Links {
			if !link.Removed {
				add(link.Ref.Selection)
			}
		}
		for _, decision := range candidate.History {
			add(decision.Ref.Selection)
		}
	}
	if len(concurrent) == 0 {
		return nil
	}
	for _, selection := range sequential {
		if ctx.Err() != nil {
			break
		}
		results[selection] = s.observePageSelection(ctx, scope, project, selection)
	}

	observed := make([]*Observation, len(concurrent))
	var workers errgroup.Group
	workers.SetLimit(pageEvidenceWorkers)
	for i, selection := range concurrent {
		if ctx.Err() != nil {
			break
		}
		workers.Go(func() error {
			if ctx.Err() == nil {
				observed[i] = s.observePageSelection(ctx, scope, project, selection)
			}
			return nil
		})
	}
	_ = workers.Wait()
	for i, selection := range concurrent {
		results[selection] = observed[i]
	}
	return results
}

func (s *Service) observePageSelection(ctx context.Context, scope collection.Scope, project string, selection Selection) *Observation {
	observation, err := s.Sources.Observe(ctx, scope, project, selection)
	if err != nil || !ValidObservation(observation, selection) {
		return nil
	}
	return &observation
}
