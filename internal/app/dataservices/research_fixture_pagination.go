package dataservicesapp

import (
	"context"
	"fmt"

	collectionstore "task-processor/internal/integration/persistence/product/collection"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

const (
	ResearchPaginationKeyword = "under sink organizer pagination"
	ResearchPaginationCount   = 50
)

type ResearchPaginationBatch struct {
	ID      string `json:"id"`
	BatchID string `json:"batchId"`
	Keyword string `json:"keyword"`
}

type researchPaginationManifest struct {
	batch         ResearchPaginationBatch
	sources       []researchFixtureSource
	byPublication map[string]researchFixtureSource
}

func (f *ResearchFixture) newPaginationManifest() *researchPaginationManifest {
	bundle := collection.StableID(f.bundleID(), "pagination-v1")
	originals := make([]ResearchFixtureOriginal, 0, ResearchPaginationCount)
	for n := 1; n <= ResearchPaginationCount; n++ {
		originals = append(originals, ResearchFixtureOriginal{
			ID: fmt.Sprintf("TPG%07d", n), ReferenceVersion: "1", Title: fmt.Sprintf("合成 · 分页收纳架 %02d", n),
			Site: "us", Width: fmt.Sprintf("%d–55 cm", 38+n%5), Installation: "no drilling", Pack: "1 piece",
			Price: float64(24+n%6) + 0.99, Currency: "USD",
		})
	}
	sources := f.fixtureSources(bundle, ResearchPaginationKeyword, originals)
	byPublication := make(map[string]researchFixtureSource, len(sources))
	for _, source := range sources {
		byPublication[source.command.PublicationID] = source
	}
	return &researchPaginationManifest{
		batch:   ResearchPaginationBatch{ID: bundle, BatchID: sources[0].batch.ID, Keyword: ResearchPaginationKeyword},
		sources: sources, byPublication: byPublication,
	}
}

func (f *ResearchFixture) PaginationEnabled() bool {
	return f != nil && f.pagination != nil
}

func (f *ResearchFixture) OwnsPagination(scope collection.Scope, project string) bool {
	return f.PaginationEnabled() && f.scope == scope && f.project == project
}

func (f *ResearchFixture) IsPaginationPublication(id string) bool {
	if !f.PaginationEnabled() {
		return false
	}
	_, ok := f.pagination.byPublication[id]
	return ok
}

// The fixed batch reader or writer owns this scope only for the operation
// that checked it. That operation checks current project and member rights
// again before returning a batch or committing a write.
type researchPaginationAuthority struct {
	scope sourcing.PublicationScope
}

func (a researchPaginationAuthority) Authorize(ctx context.Context) (sourcing.PublicationScope, error) {
	if ctx.Err() != nil {
		return sourcing.PublicationScope{}, collection.ErrForbidden
	}
	return a.scope, nil
}

func (f *ResearchFixture) ImportPagination(ctx context.Context, scope collection.Scope, project string) (ResearchPaginationBatch, error) {
	if !f.PaginationEnabled() {
		return ResearchPaginationBatch{}, collection.ErrForbidden
	}
	if err := f.check(ctx, scope, project, true); err != nil {
		return ResearchPaginationBatch{}, err
	}
	b := f.pagination.batch
	authority := researchPaginationAuthority{sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}}
	if _, err := f.importBatch(ctx, scope, project, b.ID, b.Keyword, f.pagination.sources, authority); err != nil {
		return ResearchPaginationBatch{}, err
	}
	return b, nil
}

func (f *ResearchFixture) ReadPagination(ctx context.Context, scope collection.Scope, project string) (ResearchPaginationBatch, error) {
	if !f.PaginationEnabled() {
		return ResearchPaginationBatch{}, collection.ErrForbidden
	}
	if err := f.check(ctx, scope, project, false); err != nil {
		return ResearchPaginationBatch{}, err
	}
	b := f.pagination.batch
	authority := researchPaginationAuthority{sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}}
	producer, collections, err := f.openReader(authority)
	if err != nil {
		return ResearchPaginationBatch{}, err
	}
	if _, err := f.read(ctx, producer, collections, b.ID, b.Keyword, f.pagination.sources); err != nil {
		return ResearchPaginationBatch{}, err
	}
	if err := f.check(ctx, scope, project, false); err != nil {
		return ResearchPaginationBatch{}, err
	}
	return b, nil
}

// ResearchPaginationPageReader is bound to the current page's evidence context.
type ResearchPaginationPageReader struct {
	fixture     *ResearchFixture
	ctx         context.Context
	producer    *sourcing.InternalProducer
	collections *collectionstore.Repository
}

func (f *ResearchFixture) BindPaginationPage(ctx context.Context, scope collection.Scope, project string) (*ResearchPaginationPageReader, error) {
	if !f.OwnsPagination(scope, project) {
		return nil, collection.ErrForbidden
	}
	if err := f.check(ctx, scope, project, false); err != nil {
		return nil, err
	}
	authority := researchPaginationAuthority{sourcing.PublicationScope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}}
	producer, collections, err := f.openReader(authority)
	if err != nil {
		return nil, err
	}
	return &ResearchPaginationPageReader{fixture: f, ctx: ctx, producer: producer, collections: collections}, nil
}

func (r *ResearchPaginationPageReader) ReadOriginal(in collection.SelectionInput) (ResearchPaginationBatch, ResearchFixtureItem, error) {
	source, ok := r.fixture.pagination.byPublication[in.OriginalPublicationID]
	if !ok {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, collection.ErrForbidden
	}
	// Verify the exact publication, Catalog snapshot and live member item
	// through the same owners used by the existing pagination reader.
	entry, err := r.fixture.readSource(r.ctx, r.producer, r.collections, source)
	if err != nil {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, err
	}
	if entry.Item.ID != in.ItemID || entry.Item.Revision != in.ExpectedRevision ||
		entry.Item.Source.PublicationID != in.OriginalPublicationID || entry.Item.Source.Version != in.OriginalVersion {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, collection.ErrConflict
	}
	return r.fixture.pagination.batch, entry, nil
}

func (r *ResearchPaginationPageReader) Reauthorize(ctx context.Context) error {
	return r.fixture.check(ctx, r.fixture.scope, r.fixture.project, false)
}

func (f *ResearchFixture) ReadPaginationOriginal(ctx context.Context, scope collection.Scope, project string, item collection.Item) (ResearchPaginationBatch, ResearchFixtureItem, error) {
	if !f.OwnsPagination(scope, project) || !f.IsPaginationPublication(item.Source.PublicationID) {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, collection.ErrForbidden
	}
	producer, collections, err := f.openReader(researchFixturePublicationAuthority{f, false})
	if err != nil {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, err
	}
	entry, err := f.readSource(ctx, producer, collections, f.pagination.byPublication[item.Source.PublicationID])
	if err != nil {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, err
	}
	if entry.Item.ID != item.ID || entry.Item.BatchID != item.BatchID || entry.Item.Source != item.Source || entry.Item.Revision != item.Revision {
		return ResearchPaginationBatch{}, ResearchFixtureItem{}, collection.ErrConflict
	}
	return f.pagination.batch, entry, nil
}
