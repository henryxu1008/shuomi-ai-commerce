package dataservicesapp

import (
	"context"
	"errors"

	"gorm.io/gorm"
	collectionstore "task-processor/internal/integration/persistence/product/collection"
	sourcestore "task-processor/internal/integration/persistence/product/sourcing"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
)

const researchFixtureKeyword = "under sink organizer"

var researchFixtureProducer = sourcing.ProducerDescriptor{Kind: "research_acceptance_fixture", Version: "v1"}

type ResearchFixtureOriginal struct {
	ID               string  `json:"id"`
	ReferenceVersion string  `json:"referenceVersion"`
	Title            string  `json:"title"`
	Site             string  `json:"site"`
	Width            string  `json:"width"`
	Installation     string  `json:"installation,omitempty"`
	Pack             string  `json:"pack,omitempty"`
	Price            float64 `json:"price,omitempty"`
	Currency         string  `json:"currency,omitempty"`
	SKU              string  `json:"sku,omitempty"`
}

type ResearchFixtureItem struct {
	Item     collection.Item             `json:"item"`
	Original ResearchFixtureOriginal     `json:"original"`
	Checksum string                      `json:"checksum"`
	Receipt  sourcing.PublicationReceipt `json:"-"`
}

type ResearchFixtureBatch struct {
	ID       string                `json:"id"`
	Keyword  string                `json:"keyword"`
	Amazon   []ResearchFixtureItem `json:"amazon"`
	Supplier ResearchFixtureItem   `json:"supplier"`
}

type ResearchFixtureAuthorizer func(context.Context, collection.Scope, string, bool) error

type ResearchFixture struct {
	db          *gorm.DB
	collections *collectionstore.Repository
	scope       collection.Scope
	project     string
	authorize   ResearchFixtureAuthorizer
	pagination  *researchPaginationManifest
}

func NewResearchFixture(ctx context.Context, db *gorm.DB, scope collection.Scope, project string, authorize ResearchFixtureAuthorizer, pagination bool) (*ResearchFixture, error) {
	if ctx == nil || db == nil || scope.Validate() != nil || !collection.ValidID(project) || authorize == nil {
		return nil, collection.ErrInvalid
	}
	collections, err := fixtureCollectionRepository(ctx, db)
	if err != nil {
		return nil, err
	}
	fixture := &ResearchFixture{db: db, collections: collections, scope: scope, project: project, authorize: authorize}
	if pagination {
		fixture.pagination = fixture.newPaginationManifest()
	}
	return fixture, nil
}

func (f *ResearchFixture) check(ctx context.Context, scope collection.Scope, project string, write bool) error {
	if f == nil || ctx == nil || ctx.Err() != nil || scope != f.scope || project != f.project {
		return collection.ErrForbidden
	}
	return f.authorize(ctx, scope, project, write)
}

func (f *ResearchFixture) bundleID() string {
	return collection.StableID(researchFixtureProducer.Kind, researchFixtureProducer.Version, f.scope.OrganizationID, f.scope.ActorID, f.scope.MemberID, f.project)
}

type researchFixtureSource struct {
	original ResearchFixtureOriginal
	command  sourcing.PublicationCommand
	batch    collection.PublicationBatch
}

func (f *ResearchFixture) sources() []researchFixtureSource {
	originals := []ResearchFixtureOriginal{
		{ID: "B000123456", ReferenceVersion: "3", Title: "合成 · Adjustable under sink organizer", Site: "us", Width: "38–55 cm", Installation: "no drilling", Pack: "1 piece", Price: 29.99, Currency: "USD"},
		{ID: "B000123457", ReferenceVersion: "1", Title: "合成 · Fixed under sink organizer", Site: "us", Width: "42 cm", Installation: "requires drilling", Pack: "1 piece", Price: 24.99, Currency: "USD"},
		{ID: "981645030344", ReferenceVersion: "2", Title: "合成 · Fixture supplier original", Site: "1688", Width: "40–55 cm", SKU: "sku-adjustable-white-01"},
	}
	return f.fixtureSources(f.bundleID(), researchFixtureKeyword, originals)
}

func (f *ResearchFixture) fixtureSources(bundle, keyword string, originals []ResearchFixtureOriginal) []researchFixtureSource {
	zero := uint64(0)
	out := make([]researchFixtureSource, 0, len(originals))
	for _, original := range originals {
		id := collection.StableID(bundle, original.ID)
		publication := id
		platform, kind, batchKey, batchTitle, variant := "amazon", "amazon_data", bundle, "合成 · Amazon US · "+keyword, original.ID
		attrs := map[string]string{"width": original.Width, "installation": original.Installation, "pack": original.Pack}
		if original.Site == "1688" {
			platform, kind, batchKey, batchTitle, variant = "1688", "acquisition", id, "合成 · 1688 货源", original.SKU
			publication = collection.ResearchFixtureSupplierPublicationPrefix + id
			attrs = map[string]string{"width": original.Width}
		}
		envelope := sourcing.SourceEnvelope{
			Identity: sourcing.SourceIdentity{
				SourceType: sourcing.SourceTypeManualImport, SourcePlatform: platform, SourceID: original.ID, SourceVersion: original.ReferenceVersion,
			},
			RawReference: sourcing.RawSourceReference{
				ReferenceType: researchFixtureProducer.Kind, ReferenceID: id,
				Checksum: "sha256:" + collection.Digest(original),
				Metadata: map[string]string{"bundle": bundle, "member": f.scope.MemberID, "project": f.project, "keyword": keyword},
			},
			ProductCandidate: sourcing.ProductCandidate{Title: original.Title, Attributes: attrs,
				Variants: []sourcing.ProductVariantCandidate{{SourceID: variant, SKU: variant, Attributes: attrs, Currency: original.Currency, Price: original.Price}}},
			Trace: sourcing.SourceTrace{SourceRunID: researchFixtureProducer.Kind + ":" + id},
		}
		out = append(out, researchFixtureSource{original: original,
			command: sourcing.PublicationCommand{PublicationID: publication, Producer: researchFixtureProducer,
				ProductKey: collection.ResearchFixtureProductPrefix + id, ExpectedBaseVersion: &zero, Envelope: envelope},
			batch: collection.PublicationBatch{ID: collection.StableID(f.scope.OrganizationID, f.scope.ActorID, "data-batch", kind, batchKey),
				Name: batchTitle, Kind: kind, OperationID: batchKey}})
	}
	return out
}

type researchFixturePublicationAuthority struct {
	fixture *ResearchFixture
	write   bool
}

func (a researchFixturePublicationAuthority) Authorize(ctx context.Context) (sourcing.PublicationScope, error) {
	f := a.fixture
	if err := f.check(ctx, f.scope, f.project, a.write); err != nil {
		return sourcing.PublicationScope{}, err
	}
	return sourcing.PublicationScope{OrganizationID: f.scope.OrganizationID, ActorID: f.scope.ActorID}, nil
}

func (f *ResearchFixture) Import(ctx context.Context, scope collection.Scope, project string) (ResearchFixtureBatch, error) {
	if err := f.check(ctx, scope, project, true); err != nil {
		return ResearchFixtureBatch{}, err
	}
	return f.importBatch(ctx, scope, project, f.bundleID(), researchFixtureKeyword, f.sources(), researchFixturePublicationAuthority{f, true})
}

func (f *ResearchFixture) importBatch(ctx context.Context, scope collection.Scope, project, bundle, keyword string, sources []researchFixtureSource, authority sourcing.PublicationAuthorizer) (ResearchFixtureBatch, error) {
	var batch ResearchFixtureBatch
	err := f.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?,0))", bundle).Error; err != nil {
			return collection.ErrUnavailable
		}
		if err := f.check(ctx, scope, project, true); err != nil {
			return err
		}
		writer, err := sourcestore.NewTransactionWriter(tx, newCatalogBridge)
		if err != nil {
			return err
		}
		producer, err := sourcing.NewInternalProducer(authority, writer, researchFixtureProducer)
		if err != nil {
			return err
		}
		collections, err := fixtureCollectionRepository(ctx, tx)
		if err != nil {
			return err
		}
		batch, err = f.read(ctx, producer, collections, bundle, keyword, sources)
		if err == nil {
			return f.check(ctx, scope, project, true)
		}
		if !errors.Is(err, collection.ErrNotFound) {
			return err
		}
		for _, source := range sources {
			receipt, err := producer.Publish(ctx, source.command)
			if err != nil {
				return err
			}
			ref := source.reference(receipt)
			if _, err = collectionstore.AppendResearchFixture(ctx, tx, scope, source.batch, ref, receipt.PublishedAt); err != nil {
				return err
			}
		}
		batch, err = f.read(ctx, producer, collections, bundle, keyword, sources)
		if err != nil {
			return err
		}
		return f.check(ctx, scope, project, true)
	})
	if err != nil {
		return ResearchFixtureBatch{}, err
	}
	return batch, nil
}

func (f *ResearchFixture) Read(ctx context.Context, scope collection.Scope, project string) (ResearchFixtureBatch, error) {
	if err := f.check(ctx, scope, project, false); err != nil {
		return ResearchFixtureBatch{}, err
	}
	producer, collections, err := f.openReader(researchFixturePublicationAuthority{f, false})
	if err != nil {
		return ResearchFixtureBatch{}, err
	}
	return f.read(ctx, producer, collections, f.bundleID(), researchFixtureKeyword, f.sources())
}

func (f *ResearchFixture) openReader(authority sourcing.PublicationAuthorizer) (*sourcing.InternalProducer, *collectionstore.Repository, error) {
	store, err := sourcestore.NewRepository(f.db, newCatalogBridge)
	if err != nil {
		return nil, nil, err
	}
	producer, err := sourcing.NewInternalProducer(authority, store, researchFixtureProducer)
	if err != nil {
		return nil, nil, err
	}
	return producer, f.collections, nil
}

func fixtureCollectionRepository(ctx context.Context, db *gorm.DB) (*collectionstore.Repository, error) {
	return collectionstore.NewRepository(ctx, db, func(*gorm.DB) (collectionstore.OwnPublisher, error) {
		return nil, collection.ErrUnavailable
	})
}

func (s researchFixtureSource) reference(receipt sourcing.PublicationReceipt) collection.Source {
	operation := receipt.PublicationID
	if id, ok := collection.ResearchFixtureSupplierOperation(receipt.PublicationID); ok {
		operation = id
	}
	return collection.Source{ProductKey: receipt.ProductKey, PublicationID: receipt.PublicationID,
		OperationID: operation, Version: receipt.CatalogVersion, Kind: s.batch.Kind}
}

func (f *ResearchFixture) read(ctx context.Context, producer *sourcing.InternalProducer, collections *collectionstore.Repository, bundle, keyword string, sources []researchFixtureSource) (ResearchFixtureBatch, error) {
	batch := ResearchFixtureBatch{ID: bundle, Keyword: keyword}
	found := 0
	for _, source := range sources {
		entry, err := f.readSource(ctx, producer, collections, source)
		if errors.Is(err, collection.ErrNotFound) {
			continue
		}
		if err != nil {
			return ResearchFixtureBatch{}, err
		}
		found++
		if source.batch.Kind == "amazon_data" {
			batch.Amazon = append(batch.Amazon, entry)
		} else {
			batch.Supplier = entry
		}
	}
	if found == 0 {
		return ResearchFixtureBatch{}, collection.ErrNotFound
	}
	if found != len(sources) {
		return ResearchFixtureBatch{}, collection.ErrConflict
	}
	return batch, nil
}

func (f *ResearchFixture) readSource(ctx context.Context, producer *sourcing.InternalProducer, collections *collectionstore.Repository, source researchFixtureSource) (ResearchFixtureItem, error) {
	itemID := collection.StableID(f.scope.OrganizationID, f.scope.ActorID, source.command.PublicationID, "item")
	receipt, err := producer.Verify(ctx, source.command)
	if errors.Is(err, collection.ErrForbidden) {
		return ResearchFixtureItem{}, err
	}
	item, itemErr := collections.ReadResearchEvidenceItem(ctx, f.scope, itemID)
	if errors.Is(err, sourcing.ErrSourcePublicationNotFound) && errors.Is(itemErr, collection.ErrNotFound) {
		return ResearchFixtureItem{}, collection.ErrNotFound
	}
	if err != nil || itemErr != nil || item.Source != source.reference(receipt) || item.BatchID != source.batch.ID {
		return ResearchFixtureItem{}, collection.ErrConflict
	}
	return ResearchFixtureItem{Item: item, Original: source.original, Checksum: source.command.Envelope.RawReference.Checksum, Receipt: receipt}, nil
}
