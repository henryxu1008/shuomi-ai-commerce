package completiondraft

import (
	"context"
	"strconv"

	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

// Revisions are JSON numbers under the existing browser safe-integer contract.
const MaxHistoryRevision int64 = 1<<53 - 1

func ParseRevision(value string) (int64, error) {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 1 || n > MaxHistoryRevision || strconv.FormatInt(n, 10) != value {
		return 0, preparation.ErrInvalid
	}
	return n, nil
}

type HistoryQuery struct {
	Before int64
	Limit  int
}

func (q HistoryQuery) Normalize() (HistoryQuery, error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 100 || q.Before < 0 || q.Before > MaxHistoryRevision {
		return HistoryQuery{}, preparation.ErrInvalid
	}
	return q, nil
}

type HistoryPage struct {
	Source          preparation.SourceItem `json:"source"`
	DraftID         string                 `json:"draftId"`
	CurrentRevision int64                  `json:"currentRevision"`
	Items           []Record               `json:"items"`
	NextBefore      int64                  `json:"nextBefore,omitempty"`
}

func historyRecordMatches(r Record, source preparation.SourceItem, draft string) bool {
	return r.Source == source && r.DraftID == draft && collection.ValidID(r.ID) && r.Revision > 0 && r.Revision <= MaxHistoryRevision && r.Status == StatusPending && !r.CreatedAt.IsZero() && (SaveInput{SourceID: source.ID, Content: r.Content}).Validate() == nil
}

func (s *Service) ListHistory(ctx context.Context, sourceID string, q HistoryQuery) (HistoryPage, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return HistoryPage{}, err
	}
	q, err = q.Normalize()
	if err != nil || !collection.ValidID(sourceID) {
		return HistoryPage{}, preparation.ErrInvalid
	}
	source, err := s.source(ctx, scope, sourceID)
	if err != nil {
		return HistoryPage{}, err
	}
	page, err := s.repo.ListHistory(ctx, scope, sourceID, q)
	if err != nil {
		return HistoryPage{}, err
	}
	draft := collection.StableID(scope.OrganizationID, scope.ActorID, sourceID, "completion-draft")
	if page.Source != source || page.DraftID != draft || page.CurrentRevision < 0 || page.CurrentRevision > MaxHistoryRevision || len(page.Items) > q.Limit || page.Items == nil {
		return HistoryPage{}, preparation.ErrUnavailable
	}
	previous := int64(0)
	for _, r := range page.Items {
		if !historyRecordMatches(r, source, draft) || r.Revision > page.CurrentRevision || (q.Before > 0 && r.Revision >= q.Before) || (previous > 0 && r.Revision >= previous) {
			return HistoryPage{}, preparation.ErrUnavailable
		}
		previous = r.Revision
	}
	if page.NextBefore != 0 && (len(page.Items) == 0 || page.NextBefore != previous || previous <= 1) {
		return HistoryPage{}, preparation.ErrUnavailable
	}
	if page.CurrentRevision == 0 && (len(page.Items) > 0 || page.NextBefore != 0) || page.CurrentRevision > 0 && q.Before == 0 && len(page.Items) == 0 {
		return HistoryPage{}, preparation.ErrUnavailable
	}
	if err = s.validateRead(ctx, scope, Record{Source: source}); err != nil {
		return HistoryPage{}, err
	}
	return page, nil
}

func (s *Service) ReadRevision(ctx context.Context, sourceID string, revision int64) (Record, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return Record{}, err
	}
	if !collection.ValidID(sourceID) || revision < 1 || revision > MaxHistoryRevision {
		return Record{}, preparation.ErrInvalid
	}
	source, err := s.source(ctx, scope, sourceID)
	if err != nil {
		return Record{}, err
	}
	r, err := s.repo.ReadRevision(ctx, scope, sourceID, revision)
	if err != nil {
		return Record{}, err
	}
	if r.Revision != revision || !historyRecordMatches(r, source, collection.StableID(scope.OrganizationID, scope.ActorID, sourceID, "completion-draft")) {
		return Record{}, preparation.ErrUnavailable
	}
	if err = s.validateRead(ctx, scope, r); err != nil {
		return Record{}, err
	}
	return r, nil
}
