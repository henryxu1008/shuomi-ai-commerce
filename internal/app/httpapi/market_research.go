package httpapi

import (
	"context"
	"errors"
	"gorm.io/gorm"
	pc "task-processor/internal/aiworkbench/projectcenter"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/authidentity"
	"task-processor/internal/authz"
	"task-processor/internal/core/config"
	"task-processor/internal/dataservice"
	store "task-processor/internal/integration/persistence/product/research"
	kernel "task-processor/internal/kernel/module"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	rh "task-processor/internal/product/research/httpapi"
)

func WithMarketResearch(db *gorm.DB) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.marketResearchDB = db; o.marketResearchConfigs++ }
}

func WithMarketResearchSupplierReview() CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.marketResearchSupplierReview = true }
}

type researchModule struct {
	handler  *rh.Handler
	service  *research.Service
	projects researchProjects
	trial    *marketResearchSHEINTrial
	fixture  *dataservicesapp.ResearchFixture
}

func (researchModule) Name() string                { return rh.ModuleName }
func (researchModule) Enabled(*config.Config) bool { return true }
func (m researchModule) Register(r *kernel.Registry) error {
	r.AddRoutes(rh.Routes(m.handler)...)
	if m.trial != nil {
		r.AddRoutes(marketResearchSHEINTrialRoutes(m.trial)...)
	}
	if m.fixture != nil {
		r.AddRoutes(marketResearchFixtureRoutes(m, researchFixturePath)...)
		if m.fixture.PaginationEnabled() {
			r.AddRoutes(marketResearchFixtureRoutes(m, researchPaginationFixturePath)...)
		}
	}
	return nil
}

type researchProjects struct {
	service *pc.Service
	auth    *projectSources
}

func (a researchProjects) Authorize(ctx context.Context, s collection.Scope, project string, write bool) error {
	return a.authorize(ctx, s, project, write, write)
}
func (a researchProjects) AuthorizeAnalysis(ctx context.Context, s collection.Scope, project string, active bool) error {
	return a.authorize(ctx, s, project, false, active)
}

// Account reconciliation needs fresh Market permission for the caller without
// inheriting the original submitter's project or Collection authority.
func (a researchProjects) AuthorizeDataMarket(ctx context.Context, s collection.Scope) error {
	fresh, id, err := a.auth.fresh(ctx, pc.Scope{OrganizationID: s.OrganizationID, ActorID: s.ActorID})
	if err != nil || id.EffectiveMemberID != s.MemberID || !authz.AllowedOrganization(fresh, a.auth.policy, id.UserID, id.TenantID, id.Roles, dataservice.PermissionMarket) {
		return research.ErrForbidden
	}
	return nil
}

func (a researchProjects) authorize(ctx context.Context, s collection.Scope, project string, write, active bool) error {
	scope := pc.Scope{OrganizationID: s.OrganizationID, ActorID: s.ActorID}
	fresh, id, err := a.auth.fresh(ctx, scope)
	if err != nil || id.EffectiveMemberID != s.MemberID {
		return research.ErrForbidden
	}
	if err = a.auth.authorize(fresh, scope, write); err != nil {
		return research.ErrForbidden
	}
	if !authz.AllowedOrganization(fresh, a.auth.policy, id.UserID, id.TenantID, id.Roles, collection.PermissionRead) {
		return research.ErrForbidden
	}
	return a.requireResearchProject(fresh, scope, project, active)
}

func (a researchProjects) requireResearchProject(ctx context.Context, scope pc.Scope, project string, active bool) error {
	view, err := a.service.Get(ctx, scope, project)
	if err != nil {
		return research.ErrNotFound
	}
	if view.Kind != "PRODUCT_RESEARCH" {
		return research.ErrNotFound
	}
	if active && view.Archived {
		return research.ErrArchived
	}
	return nil
}

type researchSources struct {
	collections    *collection.Service
	data           *dataservicesapp.Module
	fixture        *dataservicesapp.ResearchFixture
	manualOriginal *dataservicesapp.ResearchManualOriginalReader
}

func (a researchSources) Observe(ctx context.Context, scope collection.Scope, project string, in research.Selection) (research.Observation, error) {
	selection := collection.SelectionInput{ItemID: in.ItemID, ExpectedRevision: in.Revision, OriginalPublicationID: in.PublicationID, OriginalVersion: in.Version}
	proof, err := a.collections.Select(ctx, selection)
	if err != nil {
		return research.Observation{}, research.ErrNotFound
	}
	actual, err := proof.Scope(ctx)
	if err != nil || actual != scope {
		return research.Observation{}, research.ErrForbidden
	}
	detail, err := proof.Read(ctx)
	if err != nil || detail.Item.ArchivedAt != nil {
		return research.Observation{}, research.ErrNotFound
	}
	principal := dataacquisition.Principal{Scope: scope}
	var observation research.Observation
	switch {
	case a.manualOriginal != nil && collection.IsResearchFixtureProduct(detail.Item.Source.ProductKey):
		var entry dataservicesapp.ResearchFixtureItem
		entry, err = a.readManualOriginal(ctx, scope, project, in)
		if err == nil && (entry.Item.Source != detail.Item.Source || entry.Item.ID != detail.Item.ID || entry.Item.BatchID != detail.Item.BatchID) {
			err = research.ErrConflict
		}
		if err == nil {
			observation = fixtureAmazonObservation(a.manualOriginal.BundleID(), researchManualOriginalKeyword, 5, entry, in)
		}
	case collection.IsResearchFixtureProduct(detail.Item.Source.ProductKey):
		observation, err = a.observeFixture(ctx, scope, project, in, detail.Item)
	case detail.Item.Source.Kind == "amazon_data":
		observation, err = a.observeAmazon(ctx, principal, in, detail.Item)
	case detail.Item.Source.Kind == collection.SourceSHEINUSData:
		observation, err = a.observeSHEINUS(ctx, principal, in, detail.Item)
	default:
		return research.Observation{}, research.ErrNotFound
	}
	if err != nil {
		return research.Observation{}, err
	}
	// Revalidate the member reference after provenance materialization.
	if _, err = a.collections.Select(ctx, selection); err != nil {
		return research.Observation{}, research.ErrNotFound
	}
	return observation, nil
}

func (a researchSources) observeAmazon(ctx context.Context, principal dataacquisition.Principal, in research.Selection, item collection.Item) (research.Observation, error) {
	original, err := a.data.Provenance(ctx, principal, item.Source)
	if err != nil {
		return research.Observation{}, researchProvenanceFailure(err)
	}
	e := original.Evidence
	site, err := dataacquisition.ResolveSite(e.Site)
	if err != nil {
		return research.Observation{}, research.ErrUnavailable
	}
	v := research.Observation{Query: research.QuerySummary{Mode: original.Query.Mode, Keyword: original.Query.Keyword, CategoryNode: original.Query.CategoryNode, ASINCount: len(original.Query.ASINs), Limit: original.Query.Limit, Fields: append([]string{}, original.Query.Fields...)}, Item: item, Ref: research.EvidenceRef{Selection: in, Source: original.Source, JobID: original.JobID, Checksum: original.Checksum}, BatchID: item.BatchID, Site: e.Site, ASIN: e.ASIN, Title: e.Title, SourceURL: "https://" + site.Domain + "/dp/" + e.ASIN, CapturedAt: e.CapturedAt, ParserVersion: e.ParserVersion, Currency: e.Currency, Missing: append([]string{}, e.Missing...)}
	if e.Availability == "available" && e.Price > 0 {
		price := e.Price
		v.Price = &price
		v.PriceBasis = "displayed_listing_offer"
	}
	return v, nil
}

func (a researchSources) observeSHEINUS(ctx context.Context, principal dataacquisition.Principal, in research.Selection, item collection.Item) (research.Observation, error) {
	original, err := a.data.ReadSHEINUSProvenance(ctx, principal, item.Source)
	if err != nil {
		if errors.Is(err, dataacquisition.ErrConflict) {
			return research.Observation{}, research.ErrConflict
		}
		return research.Observation{}, researchProvenanceFailure(err)
	}
	e := original.Observation
	v := research.Observation{
		Item: item, BatchID: item.BatchID, SourceURL: e.SourceURL, ParserVersion: e.ParserVersion, Missing: append([]string{}, e.Missing...),
		Ref: research.EvidenceRef{Selection: in, Source: original.Source, JobID: original.JobID, Checksum: original.Checksum,
			QueryRef: original.QueryRef, ObservationVersion: original.ObservationVersion, ObservationEncodingVersion: original.ObservationEncodingVersion},
		Query: research.QuerySummary{Mode: original.QuerySummary.Mode, CategoryRef: original.QuerySummary.CategoryRef, Keyword: original.QuerySummary.Keyword, Fields: []string{}},
		SHEIN: &research.SHEINCard{Market: e.Market, NativeGoodsID: e.NativeGoodsID, Price: (*research.SHEINCardPrice)(e.Price)},
	}
	if e.Title != nil {
		v.Title = *e.Title
	}
	if e.CapturedAt != nil {
		v.CapturedAt = *e.CapturedAt
	}
	return v, nil
}

func composeMarketResearch(ctx context.Context, db *gorm.DB, project projectModule, collections *collection.Service, data *dataservicesapp.Module, supplierReview bool) (kernel.Module, error) {
	if db == nil || project.handler == nil || collections == nil || data == nil {
		return nil, errors.New("market research requires current project, collection and data owners")
	}
	if err := store.VerifyRuntimePermissions(ctx, db); err != nil {
		return nil, err
	}
	repo, err := store.New(ctx, db)
	if err != nil {
		return nil, err
	}
	auth, ok := project.handler.Service.Reader.(*projectSources)
	if !ok {
		return nil, research.ErrUnavailable
	}
	projects := researchProjects{project.handler.Service, auth}
	sources := &researchSources{collections: collections, data: data}
	service := research.New(repo, projects, sources)
	if supplierReview && store.VerifySupplierReviewSchema(ctx, db) == nil {
		service.SupplierSources = sources
		service.SupplierExpiry = supplierEvidenceExpiry
	}
	if supplierReview && repo.AssessmentsReady() {
		service.SupplierSources = sources
		service.AssessmentExpiry = supplierEvidenceExpiry
	}
	h := &rh.Handler{Service: service, Bind: func(ctx context.Context, header string) (context.Context, collection.Scope, error) {
		ctx, err := (productReviewCapabilityBinder{}).Bind(ctx, header)
		if err != nil {
			return nil, collection.Scope{}, err
		}
		id, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
		if !ok {
			return nil, collection.Scope{}, research.ErrForbidden
		}
		return ctx, collection.Scope{OrganizationID: id.TenantID, ActorID: id.UserID, MemberID: id.EffectiveMemberID}, nil
	}}
	return researchModule{handler: h, service: service, projects: projects}, nil
}

// A saved Collection reference without a Data Services item is a different
// producer (for example plugin capture), not an ambiguous research write.
func researchProvenanceFailure(err error) error {
	if errors.Is(err, dataacquisition.ErrNotFound) {
		return research.ErrUnsupported
	}
	if errors.Is(err, dataacquisition.ErrForbidden) {
		return research.ErrForbidden
	}
	return research.ErrUnavailable
}
