// Package completiondraft owns actor-private, unbound candidate text. It never
// changes the canonical Product or grants official platform readiness.
package completiondraft

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/authidentity"
	"task-processor/internal/listing/preparation"
	"task-processor/internal/product/collection"
)

const StatusPending = "pending_completion"
const MaxPayloadBytes = 64 << 10

type Content struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}
type SaveInput struct {
	SourceID         string  `json:"sourceId"`
	ExpectedRevision int64   `json:"expectedRevision"`
	Content          Content `json:"content"`
}

func (in SaveInput) Validate() error {
	if !collection.ValidID(in.SourceID) || in.ExpectedRevision < 0 || in.ExpectedRevision == math.MaxInt64 || len(in.Content.Title) > 1024 || len(in.Content.Description) > 32768 || !utf8.ValidString(in.Content.Title) || !utf8.ValidString(in.Content.Description) || strings.ContainsRune(in.Content.Title, 0) || strings.ContainsRune(in.Content.Description, 0) {
		return preparation.ErrInvalid
	}
	raw, err := json.Marshal(in)
	if err != nil || len(raw) > MaxPayloadBytes {
		return preparation.ErrInvalid
	}
	return nil
}
func InputHash(in SaveInput) string { b, _ := json.Marshal(in); return collection.Digest(string(b)) }

type Record struct {
	ID        string                 `json:"id"`
	DraftID   string                 `json:"draftId"`
	Revision  int64                  `json:"revision"`
	Source    preparation.SourceItem `json:"source"`
	Content   Content                `json:"content"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"createdAt"`
}
type Receipt struct {
	Record   Record `json:"record"`
	Replayed bool   `json:"replayed"`
}
type ItemPage struct {
	CollectionItemID string `json:"collectionItemId"`
	collection.Page[Record]
}
type CollectionOwner interface {
	preparation.CollectionOwnerAuthority
	ReadItem(context.Context, string) (collection.ItemDetail, error)
}
type Repository interface {
	Save(context.Context, Prepared) (Receipt, error)
	Read(context.Context, collection.Scope, string) (Record, error)
	ReadCommand(context.Context, collection.Scope, string) (Receipt, error)
	List(context.Context, collection.Scope, string, collection.Query) (collection.Page[Record], error)
	ListForCollectionItem(context.Context, collection.Scope, string, collection.Query) (collection.Page[Record], error)
	ListHistory(context.Context, collection.Scope, string, HistoryQuery) (HistoryPage, error)
	ReadRevision(context.Context, collection.Scope, string, int64) (Record, error)
}
type Service struct {
	sources      *preparation.SourceSelector
	owners       CollectionOwner
	preparations *preparation.Service
	auth         preparation.Authorizer
	repo         Repository
}

func NewService(sources *preparation.SourceSelector, owners CollectionOwner, preparations *preparation.Service, auth preparation.Authorizer, repo Repository) (*Service, error) {
	if sources == nil || owners == nil || preparations == nil || auth == nil || repo == nil {
		return nil, preparation.ErrUnavailable
	}
	return &Service{sources, owners, preparations, auth, repo}, nil
}
func (s *Service) authorize(ctx context.Context, purpose string) (collection.Scope, error) {
	if s == nil || ctx == nil || ctx.Err() != nil {
		return collection.Scope{}, preparation.ErrForbidden
	}
	id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || !time.Now().Before(id.TokenExpiresAt) {
		return collection.Scope{}, preparation.ErrForbidden
	}
	scope, err := s.auth.Authorize(ctx, purpose)
	if err != nil {
		return collection.Scope{}, err
	}
	if scope.Validate() != nil || id.TenantID != scope.OrganizationID || id.EffectiveOrganizationID != scope.OrganizationID || id.UserID != scope.ActorID || id.EffectiveMemberID != scope.MemberID {
		return collection.Scope{}, preparation.ErrForbidden
	}
	owner, err := s.owners.AuthorizeOwner(ctx)
	if err != nil {
		return collection.Scope{}, err
	}
	current, err := owner.Scope(ctx)
	if err != nil || current != scope || ctx.Err() != nil || !time.Now().Before(id.TokenExpiresAt) {
		return collection.Scope{}, preparation.ErrForbidden
	}
	return scope, nil
}
func (s *Service) source(ctx context.Context, scope collection.Scope, id string) (preparation.SourceItem, error) {
	selected, err := s.sources.Select(ctx, id)
	if err != nil {
		return preparation.SourceItem{}, err
	}
	current, source, _, err := selected.Read(ctx)
	if err != nil || current != scope {
		return preparation.SourceItem{}, preparation.ErrForbidden
	}
	return source, nil
}

// Prepared is minted only after current owner and exact original source checks.
// Read remints the exact source proof before entering persistence. Commit checks
// current authority and the source reloaded on the repository's own transaction.
type Prepared struct {
	scope     collection.Scope
	key, hash string
	input     SaveInput
	source    preparation.SourceItem
	refresh   func(context.Context) error
	authority func(context.Context) error
}

func (p Prepared) Read(ctx context.Context) (collection.Scope, string, string, SaveInput, preparation.SourceItem, error) {
	if p.refresh == nil || ctx == nil || ctx.Err() != nil {
		return collection.Scope{}, "", "", SaveInput{}, preparation.SourceItem{}, preparation.ErrForbidden
	}
	if err := p.refresh(ctx); err != nil {
		return collection.Scope{}, "", "", SaveInput{}, preparation.SourceItem{}, err
	}
	return p.scope, p.key, p.hash, p.input, p.source, nil
}

// ValidateCommit does not query Product through a second pool connection. The
// adapter must supply its actor/member-qualified source read on the active tx.
func (p Prepared) ValidateCommit(ctx context.Context, retained preparation.SourceItem) error {
	if p.authority == nil || ctx == nil || ctx.Err() != nil || retained != p.source {
		return preparation.ErrForbidden
	}
	return p.authority(ctx)
}
func (s *Service) Save(ctx context.Context, key string, in SaveInput) (Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionManage)
	if err != nil {
		return Receipt{}, err
	}
	if !collection.ValidID(key) || in.Validate() != nil {
		return Receipt{}, preparation.ErrInvalid
	}
	source, err := s.source(ctx, scope, in.SourceID)
	if err != nil {
		return Receipt{}, err
	}
	authority := func(ctx context.Context) error {
		for _, purpose := range []string{preparation.PermissionManage, preparation.PermissionRead} {
			current, err := s.authorize(ctx, purpose)
			if err != nil {
				return err
			}
			if current != scope {
				return preparation.ErrForbidden
			}
		}
		return nil
	}
	p := Prepared{scope: scope, key: key, hash: InputHash(in), input: in, source: source, authority: authority, refresh: func(ctx context.Context) error {
		if err := authority(ctx); err != nil {
			return err
		}
		selected, err := s.source(ctx, scope, in.SourceID)
		if err != nil {
			return err
		}
		if selected != source {
			return preparation.ErrForbidden
		}
		return nil
	}}
	return s.repo.Save(ctx, p)
}
func (s *Service) validateRead(ctx context.Context, scope collection.Scope, r Record) error {
	current, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return err
	}
	if current != scope {
		return preparation.ErrForbidden
	}
	source, err := s.source(ctx, scope, r.Source.ID)
	if err != nil {
		return err
	}
	if source != r.Source {
		return preparation.ErrForbidden
	}
	return nil
}
func (s *Service) Read(ctx context.Context, sourceID string) (Record, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return Record{}, err
	}
	if !collection.ValidID(sourceID) {
		return Record{}, preparation.ErrInvalid
	}
	if _, err = s.source(ctx, scope, sourceID); err != nil {
		return Record{}, err
	}
	r, err := s.repo.Read(ctx, scope, sourceID)
	if err != nil {
		return Record{}, err
	}
	if err = s.validateRead(ctx, scope, r); err != nil {
		return Record{}, err
	}
	return r, nil
}
func (s *Service) ReadCommand(ctx context.Context, key string) (Receipt, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return Receipt{}, err
	}
	if !collection.ValidID(key) {
		return Receipt{}, preparation.ErrInvalid
	}
	r, err := s.repo.ReadCommand(ctx, scope, key)
	if err != nil {
		return Receipt{}, err
	}
	if err = s.validateRead(ctx, scope, r.Record); err != nil {
		return Receipt{}, err
	}
	return r, nil
}
func (s *Service) List(ctx context.Context, preparationID string, q collection.Query) (collection.Page[Record], error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return collection.Page[Record]{}, err
	}
	if q.Validate() != nil || !collection.ValidID(preparationID) {
		return collection.Page[Record]{}, preparation.ErrInvalid
	}
	if _, err = s.preparations.Read(ctx, preparationID); err != nil {
		return collection.Page[Record]{}, err
	}
	page, err := s.repo.List(ctx, scope, preparationID, q)
	if err != nil {
		return collection.Page[Record]{}, err
	}
	for _, r := range page.Items {
		if r.Source.PreparationID != preparationID || !collection.ValidID(r.Source.ID) || ctx.Err() != nil {
			return collection.Page[Record]{}, preparation.ErrUnavailable
		}
	}
	// The repository validates each immutable record against its exact qualified
	// retained source on this page read. Recheck live page authority once before
	// returning; do not multiply uncached IAM calls by the requested page size.
	current, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return collection.Page[Record]{}, err
	}
	if current != scope {
		return collection.Page[Record]{}, preparation.ErrForbidden
	}
	return page, nil
}

// ListForCollectionItem reads saved heads across preparations. Current item
// authority gates discovery; each retained source keeps its original version.
func (s *Service) ListForCollectionItem(ctx context.Context, itemID string, q collection.Query) (ItemPage, error) {
	ctx, cancel := context.WithTimeout(ctx, preparation.Timeout)
	defer cancel()
	scope, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return ItemPage{}, err
	}
	if !collection.ValidID(itemID) || q.Validate() != nil {
		return ItemPage{}, preparation.ErrInvalid
	}
	readItem := func() error {
		detail, err := s.owners.ReadItem(ctx, itemID)
		if err != nil {
			return err
		}
		if detail.Item.ID != itemID || detail.Item.Source.Kind != "own" || detail.Item.ArchivedAt != nil {
			return preparation.ErrNotFound
		}
		return nil
	}
	if err = readItem(); err != nil {
		return ItemPage{}, err
	}
	page, err := s.repo.ListForCollectionItem(ctx, scope, itemID, q)
	if err != nil {
		return ItemPage{}, err
	}
	for _, r := range page.Items {
		if r.Source.CollectionItemID != itemID || r.Source.Source.Kind != "own" || !collection.ValidID(r.Source.ID) || ctx.Err() != nil {
			return ItemPage{}, preparation.ErrUnavailable
		}
	}
	if err = readItem(); err != nil {
		return ItemPage{}, err
	}
	current, err := s.authorize(ctx, preparation.PermissionRead)
	if err != nil {
		return ItemPage{}, err
	}
	if current != scope {
		return ItemPage{}, preparation.ErrForbidden
	}
	return ItemPage{CollectionItemID: itemID, Page: page}, nil
}
