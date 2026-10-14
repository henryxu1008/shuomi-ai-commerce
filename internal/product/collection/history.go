package collection

import (
	"context"
	"math"
	"strconv"
	"task-processor/internal/product/catalog"
	"task-processor/internal/product/sourcing"
	"time"
)

type OwnVersion struct {
	Version       uint64    `json:"version,string"`
	PublicationID string    `json:"publicationId"`
	SavedAt       time.Time `json:"savedAt"`
	Title         string    `json:"title"`
}
type OwnHistoryPage struct {
	Item       Item         `json:"item"`
	Versions   []OwnVersion `json:"versions"`
	NextBefore string       `json:"nextBefore,omitempty"`
}
type OwnVersionDetail struct {
	Item    Item                    `json:"item"`
	Version OwnVersion              `json:"version"`
	Product catalog.ProductSnapshot `json:"product"`
}

func (s *Service) WithOwnHistory(reader sourcing.OwnHistoryReadStore) *Service {
	s.history = reader
	return s
}
func ParseHistoryVersion(value string) (uint64, error) {
	v, err := strconv.ParseUint(value, 10, 63)
	if err != nil || v == 0 || strconv.FormatUint(v, 10) != value {
		return 0, ErrInvalid
	}
	return v, nil
}
func (s *Service) historyItem(ctx context.Context, scope Scope, id string) (Item, error) {
	item, err := s.store.ReadItem(ctx, scope, id)
	if err != nil {
		return Item{}, err
	}
	if item.ID != id || item.Source.Kind != "own" {
		return Item{}, ErrNotFound
	}
	if _, err = s.store.ReadBatch(ctx, scope, item.BatchID); err != nil {
		return Item{}, err
	}
	if item.Source.ProductKey == "" || !ValidID(item.Source.PublicationID) || item.Source.Version == 0 || item.Source.Version > math.MaxInt64 {
		return Item{}, ErrUnavailable
	}
	return item, nil
}
func (s *Service) finishHistory(ctx context.Context, scope Scope, item Item) (Item, error) {
	current, err := s.authorize(ctx, PermissionRead)
	if err != nil || current != scope {
		return Item{}, ErrForbidden
	}
	latest, err := s.historyItem(ctx, scope, item.ID)
	if err != nil {
		return Item{}, err
	}
	if latest.Source.ProductKey != item.Source.ProductKey || latest.Source.Version < item.Source.Version {
		return Item{}, ErrUnavailable
	}
	return latest, ctx.Err()
}

type historyReadAuthority struct{ scope Scope }

func (a historyReadAuthority) Authorize(ctx context.Context) (sourcing.PublicationScope, error) {
	if ctx.Err() != nil || a.scope.Validate() != nil {
		return sourcing.PublicationScope{}, ErrForbidden
	}
	return sourcing.PublicationScope{OrganizationID: a.scope.OrganizationID, ActorID: a.scope.ActorID}, nil
}
func (s *Service) historyVersion(ctx context.Context, scope Scope, item Item, ref sourcing.OwnHistoryReference) (OwnVersionDetail, error) {
	if ref.Version == 0 || ref.Version > item.Source.Version || !ValidID(ref.PublicationID) {
		return OwnVersionDetail{}, ErrUnavailable
	}
	reader, err := sourcing.NewInternalReader(historyReadAuthority{scope}, s.history)
	if err != nil {
		return OwnVersionDetail{}, ErrUnavailable
	}
	published, err := reader.Read(ctx, ref.PublicationID)
	if err != nil {
		return OwnVersionDetail{}, ErrUnavailable
	}
	r, e := published.Receipt, published.Envelope
	if r.OrganizationID != scope.OrganizationID || r.ActorID != scope.ActorID || r.ProductKey != item.Source.ProductKey || r.CatalogVersion != ref.Version || r.PublicationID != ref.PublicationID || r.Producer != (sourcing.ProducerDescriptor{Kind: "own_product", Version: "v1"}) || r.PublishedAt.IsZero() || e.Identity.SourceType != "user_input" || e.Identity.SourcePlatform != "own_product" || e.Identity.SourceID != ref.PublicationID || e.RawReference.ReferenceType != "user_input" || e.RawReference.ReferenceID != ref.PublicationID {
		return OwnVersionDetail{}, ErrUnavailable
	}
	if r.ExpectedBaseVersion != nil {
		if *r.ExpectedBaseVersion != 0 || r.ProductKey != "own-"+r.PublicationID || len(e.RawReference.Metadata) != 0 {
			return OwnVersionDetail{}, ErrUnavailable
		}
	} else {
		parent, err := ParseHistoryVersion(e.RawReference.Metadata["previous_catalog_version"])
		if err != nil || parent >= r.CatalogVersion || !ValidID(e.RawReference.Metadata["previous_publication_id"]) || e.RawReference.Metadata["previous_publication_id"] == r.PublicationID {
			return OwnVersionDetail{}, ErrUnavailable
		}
	}
	return OwnVersionDetail{Item: item, Version: OwnVersion{Version: r.CatalogVersion, PublicationID: r.PublicationID, SavedAt: r.PublishedAt, Title: published.Snapshot.Title}, Product: published.Snapshot}, nil
}
func (s *Service) beginHistory(ctx context.Context, id string) (Scope, Item, error) {
	scope, err := s.authorize(ctx, PermissionRead)
	if err != nil {
		return Scope{}, Item{}, err
	}
	if !ValidID(id) {
		return Scope{}, Item{}, ErrInvalid
	}
	item, err := s.historyItem(ctx, scope, id)
	if err != nil {
		return scope, item, err
	}
	if s.history == nil {
		return scope, item, ErrUnavailable
	}
	_, err = s.historyVersion(ctx, scope, item, sourcing.OwnHistoryReference{PublicationID: item.Source.PublicationID, Version: item.Source.Version})
	return scope, item, err
}
func (s *Service) ListOwnHistory(ctx context.Context, id string, before uint64, limit int) (OwnHistoryPage, error) {
	if ctx == nil || s == nil {
		return OwnHistoryPage{}, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	if before > math.MaxInt64 || limit < 1 || limit > 20 {
		return OwnHistoryPage{}, ErrInvalid
	}
	scope, item, err := s.beginHistory(ctx, id)
	if err != nil {
		return OwnHistoryPage{}, err
	}
	refs, err := s.history.ListOwnVersions(ctx, sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, item.Source.ProductKey, item.Source.Version, sourcing.OwnHistoryQuery{Before: before, Limit: limit})
	if err != nil {
		return OwnHistoryPage{}, ErrUnavailable
	}
	if len(refs) > limit+1 {
		return OwnHistoryPage{}, ErrUnavailable
	}
	if before == 0 && (len(refs) == 0 || refs[0].PublicationID != item.Source.PublicationID || refs[0].Version != item.Source.Version) {
		return OwnHistoryPage{}, ErrUnavailable
	}
	result := OwnHistoryPage{Item: item, Versions: make([]OwnVersion, 0, limit)}
	previous := uint64(0)
	for i, ref := range refs {
		if i > 0 && ref.Version >= previous || before != 0 && ref.Version >= before {
			return OwnHistoryPage{}, ErrUnavailable
		}
		previous = ref.Version
		detail, err := s.historyVersion(ctx, scope, item, ref)
		if err != nil {
			return OwnHistoryPage{}, err
		}
		if i < limit {
			result.Versions = append(result.Versions, detail.Version)
		} else {
			result.NextBefore = strconv.FormatUint(result.Versions[len(result.Versions)-1].Version, 10)
		}
	}
	result.Item, err = s.finishHistory(ctx, scope, item)
	if err != nil {
		return OwnHistoryPage{}, err
	}
	return result, nil
}
func (s *Service) ReadOwnVersion(ctx context.Context, id string, version uint64) (OwnVersionDetail, error) {
	if ctx == nil || s == nil {
		return OwnVersionDetail{}, ErrForbidden
	}
	ctx, cancel := context.WithTimeout(ctx, Timeout)
	defer cancel()
	if version == 0 || version > math.MaxInt64 {
		return OwnVersionDetail{}, ErrInvalid
	}
	scope, item, err := s.beginHistory(ctx, id)
	if err != nil {
		return OwnVersionDetail{}, err
	}
	if version > item.Source.Version {
		return OwnVersionDetail{}, ErrNotFound
	}
	refs, err := s.history.ListOwnVersions(ctx, sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}, item.Source.ProductKey, item.Source.Version, sourcing.OwnHistoryQuery{Version: version, Limit: 1})
	if err != nil {
		return OwnVersionDetail{}, ErrUnavailable
	}
	if len(refs) == 0 {
		return OwnVersionDetail{}, ErrNotFound
	}
	if len(refs) != 1 || refs[0].Version != version {
		return OwnVersionDetail{}, ErrUnavailable
	}
	result, err := s.historyVersion(ctx, scope, item, refs[0])
	if err != nil {
		return OwnVersionDetail{}, err
	}
	result.Item, err = s.finishHistory(ctx, scope, item)
	return result, err
}
