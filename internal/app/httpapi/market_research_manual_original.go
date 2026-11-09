package httpapi

import (
	"context"
	"errors"
	"net"
	"net/url"

	"gorm.io/gorm"
	dataservicesapp "task-processor/internal/app/dataservices"
	listingapp "task-processor/internal/app/listing"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

const researchManualOriginalKeyword = "under sink organizer"

func WithResearchManualOriginal(binding dataservicesapp.ResearchManualOriginalBinding) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.researchManualOriginal = &binding }
}

func (o currentApplicationOptions) validateResearchManualOriginal(issuer string) error {
	if o.researchManualOriginal == nil {
		return nil
	}
	u, err := url.Parse(issuer)
	local := false
	if err == nil {
		ip := net.ParseIP(u.Hostname())
		local = u.Hostname() == "localhost" || ip != nil && ip.IsLoopback()
	}
	if err != nil || !local || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" ||
		o.researchManualOriginal.Validate() != nil || o.researchFixture != nil || o.marketResearchConfigs != 1 ||
		!o.marketResearchSupplierReview || o.productAcquisitionDB == nil || o.productCollections != 1 || o.dataServices == nil {
		return errors.New("manual research original requires one exact local binding and current owners; full fixture mode must remain disabled")
	}
	return nil
}

func (m *researchModule) enableManualOriginal(ctx context.Context, db *gorm.DB, b dataservicesapp.ResearchManualOriginalBinding) error {
	sources, ok := m.service.Sources.(*researchSources)
	if !ok || sources.fixture != nil || m.fixture != nil {
		return research.ErrUnavailable
	}
	reader, err := dataservicesapp.NewResearchManualOriginalReader(ctx, db, b, m.projects.authorizeFixture)
	if err != nil {
		return err
	}
	sources.manualOriginal = reader
	return nil
}

func (a researchSources) ManualOriginal(scope collection.Scope, project string) (string, string, research.Selection, bool) {
	b, ok := a.manualOriginal.Binding(scope, project)
	return b.CandidateID, b.LinkID, research.Selection{ItemID: b.ItemID, Revision: b.ItemRevision, PublicationID: b.PublicationID, Version: b.CatalogVersion}, ok
}

func (a researchSources) readManualOriginal(ctx context.Context, scope collection.Scope, project string, in research.Selection) (dataservicesapp.ResearchFixtureItem, error) {
	entry, err := a.manualOriginal.Read(ctx, scope, project, collection.SelectionInput{ItemID: in.ItemID, ExpectedRevision: in.Revision, OriginalPublicationID: in.PublicationID, OriginalVersion: in.Version})
	if err != nil {
		return dataservicesapp.ResearchFixtureItem{}, supplierEvidenceFailure(err)
	}
	return entry, nil
}

func (a researchSources) manualMarket(ctx context.Context, scope collection.Scope, project string, ref research.EvidenceRef, item collection.Item) (research.SupplierMarket, error) {
	entry, err := a.readManualOriginal(ctx, scope, project, ref.Selection)
	if err != nil {
		return research.SupplierMarket{}, err
	}
	if entry.Item.ID != item.ID || entry.Item.Source != item.Source || entry.Item.Revision != item.Revision || fixtureAmazonRef(a.manualOriginal.BundleID(), entry, ref.Selection) != ref {
		return research.SupplierMarket{}, research.ErrConflict
	}
	return fixtureMarketProjection(ref, entry), nil
}

func (o currentApplicationOptions) restrictManualOriginalTransfers(app *listingapp.Application) error {
	b := o.researchManualOriginal
	if b == nil {
		return nil
	}
	if b.Validate() != nil {
		return research.ErrInvalid
	}
	if app == nil || app.Preparations == nil {
		return research.ErrUnavailable
	}
	configured, err := app.Preparations.WithManualOriginalExclusion(b.Scope, b.ItemID, b.PublicationID)
	if err != nil {
		return err
	}
	app.Preparations = configured
	return nil
}
