package dataservicesapp

import (
	"context"
	"encoding/hex"

	"gorm.io/gorm"
	"task-processor/internal/product/collection"
)

// ResearchManualOriginalBinding is an operator-frozen identity, not a request
// parameter or a new source owner. Only the existing B000123457 template is read.
type ResearchManualOriginalBinding struct {
	Scope          collection.Scope `json:"scope"`
	Project        string           `json:"project"`
	CandidateID    string           `json:"candidateId"`
	LinkID         string           `json:"linkId"`
	ItemID         string           `json:"itemId"`
	ItemRevision   int64            `json:"itemRevision"`
	PublicationID  string           `json:"publicationId"`
	CatalogVersion uint64           `json:"catalogVersion"`
	Checksum       string           `json:"checksum"`
	EnvelopeHash   string           `json:"envelopeHash"`
	SnapshotHash   string           `json:"snapshotHash"`
}

func (b ResearchManualOriginalBinding) Validate() error {
	if b.Scope.Validate() != nil || !collection.ValidID(b.Project) || !collection.ValidID(b.CandidateID) || !collection.ValidID(b.LinkID) ||
		!collection.ValidID(b.ItemID) || !collection.ValidID(b.PublicationID) || b.ItemRevision != 1 || b.CatalogVersion != 1 {
		return collection.ErrInvalid
	}
	// Use the original producer's deterministic command; never accept a different
	// product merely because its key has the fixture prefix.
	f := &ResearchFixture{scope: b.Scope, project: b.Project}
	source := f.sources()[1]
	if b.PublicationID != source.command.PublicationID || b.ItemID != collection.StableID(b.Scope.OrganizationID, b.Scope.ActorID, b.PublicationID, "item") || b.Checksum != source.command.Envelope.RawReference.Checksum {
		return collection.ErrInvalid
	}
	for _, h := range []string{b.EnvelopeHash, b.SnapshotHash} {
		decoded, err := hex.DecodeString(h)
		if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != h {
			return collection.ErrInvalid
		}
	}
	return nil
}

// ResearchManualOriginalReader deliberately exports no import/publish method.
// It reuses the current producer's Verify and Collection proof without injecting
// the full fixture object into Research or registering fixture HTTP routes.
type ResearchManualOriginalReader struct {
	fixture *ResearchFixture
	binding ResearchManualOriginalBinding
}

func NewResearchManualOriginalReader(ctx context.Context, db *gorm.DB, b ResearchManualOriginalBinding, authorize ResearchFixtureAuthorizer) (*ResearchManualOriginalReader, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}
	f, err := NewResearchFixture(ctx, db, b.Scope, b.Project, authorize, false)
	if err != nil {
		return nil, err
	}
	return &ResearchManualOriginalReader{fixture: f, binding: b}, nil
}

func (r *ResearchManualOriginalReader) Binding(scope collection.Scope, project string) (ResearchManualOriginalBinding, bool) {
	if r == nil || scope != r.binding.Scope || project != r.binding.Project {
		return ResearchManualOriginalBinding{}, false
	}
	return r.binding, true
}

func (r *ResearchManualOriginalReader) Read(ctx context.Context, scope collection.Scope, project string, in collection.SelectionInput) (ResearchFixtureItem, error) {
	b, ok := r.Binding(scope, project)
	if !ok {
		return ResearchFixtureItem{}, collection.ErrForbidden
	}
	if in.ItemID != b.ItemID || in.ExpectedRevision != b.ItemRevision || in.OriginalPublicationID != b.PublicationID || in.OriginalVersion != b.CatalogVersion {
		return ResearchFixtureItem{}, collection.ErrConflict
	}
	f := r.fixture
	if err := f.check(ctx, scope, project, false); err != nil {
		return ResearchFixtureItem{}, err
	}
	producer, collections, err := f.openReader(researchFixturePublicationAuthority{f, false})
	if err != nil {
		return ResearchFixtureItem{}, err
	}
	entry, err := f.readSource(ctx, producer, collections, f.sources()[1])
	if err != nil {
		return ResearchFixtureItem{}, err
	}
	if entry.Item.ID != b.ItemID || entry.Item.Revision != b.ItemRevision || entry.Item.ArchivedAt != nil || entry.Checksum != b.Checksum ||
		entry.Receipt.EnvelopeHash != b.EnvelopeHash || entry.Receipt.SnapshotHash != b.SnapshotHash || entry.Receipt.CatalogVersion != b.CatalogVersion {
		return ResearchFixtureItem{}, collection.ErrConflict
	}
	if err = f.check(ctx, scope, project, false); err != nil {
		return ResearchFixtureItem{}, err
	}
	// Membership/project checks can involve another system. Re-read the private
	// reference after them so a move/archive/version change during that check
	// cannot make the just-materialized original appear current.
	current, err := collections.ReadResearchEvidenceItem(ctx, scope, b.ItemID)
	if err != nil {
		return ResearchFixtureItem{}, err
	}
	if current.ID != entry.Item.ID || current.Revision != entry.Item.Revision || current.Source != entry.Item.Source || current.BatchID != entry.Item.BatchID || current.ArchivedAt != nil {
		return ResearchFixtureItem{}, collection.ErrConflict
	}
	return entry, nil
}

func (r *ResearchManualOriginalReader) BundleID() string { return r.fixture.bundleID() }
