package research

import (
	"context"

	"golang.org/x/sync/errgroup"

	"task-processor/internal/product/collection"
)

// PageEvidenceReader reads admitted sources under the current page's read budget.
type PageEvidenceReader interface {
	Observe(Selection) (Observation, error)
	Reauthorize(context.Context) error
}

type concurrentPageSources interface {
	CanReadPageEvidenceConcurrently(collection.Scope, string, Selection) bool
	BindPageEvidence(context.Context, collection.Scope, string) (PageEvidenceReader, error)
}

const pageEvidenceWorkers = 4

// preparePageEvidence overlaps only reads admitted by the source adapter. Other
// originals are read first, so an unusually slow fixture batch cannot crowd them out.
func (s *Service) preparePageEvidence(ctx, sourceCtx context.Context, scope collection.Scope, project string, rows []Candidate) map[Selection]*Observation {
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
		if sourceCtx.Err() != nil {
			return results
		}
		results[selection] = s.observePageSelection(sourceCtx, scope, project, selection)
	}
	if sourceCtx.Err() != nil {
		return results
	}
	reader, err := source.BindPageEvidence(sourceCtx, scope, project)
	if err != nil {
		return results
	}

	observed := make([]*Observation, len(concurrent))
	var workers errgroup.Group
	workers.SetLimit(pageEvidenceWorkers)
	for i, selection := range concurrent {
		if sourceCtx.Err() != nil {
			break
		}
		workers.Go(func() error {
			if sourceCtx.Err() == nil {
				observation, err := reader.Observe(selection)
				if err == nil && ValidObservation(observation, selection) {
					observed[i] = &observation
				}
			}
			return nil
		})
	}
	_ = workers.Wait()
	// A slow row can exhaust the source budget. Recheck permission with the
	// live request so completed originals can still be returned safely.
	if reader.Reauthorize(ctx) != nil {
		return results
	}
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
