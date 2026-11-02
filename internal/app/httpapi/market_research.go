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

type researchModule struct {
	handler *rh.Handler
	service *research.Service
}

func (researchModule) Name() string                { return rh.ModuleName }
func (researchModule) Enabled(*config.Config) bool { return true }
func (m researchModule) Register(r *kernel.Registry) error {
	r.AddRoutes(rh.Routes(m.handler)...)
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
	view, err := a.service.Get(fresh, scope, project)
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
	collections *collection.Service
	data        *dataservicesapp.Module
}

func (a researchSources) Observe(ctx context.Context, scope collection.Scope, in research.Selection) (research.Observation, error) {
	proof, err := a.collections.Select(ctx, collection.SelectionInput{ItemID: in.ItemID, ExpectedRevision: in.Revision, OriginalPublicationID: in.PublicationID, OriginalVersion: in.Version})
	if err != nil {
		return research.Observation{}, research.ErrNotFound
	}
	actual, err := proof.Scope(ctx)
	if err != nil || actual != scope {
		return research.Observation{}, research.ErrForbidden
	}
	detail, err := proof.Read(ctx)
	if err != nil || detail.Item.ArchivedAt != nil || detail.Item.Source.Kind != "amazon_data" {
		return research.Observation{}, research.ErrNotFound
	}
	original, err := a.data.Provenance(ctx, dataacquisition.Principal{Scope: scope}, detail.Item.Source)
	if err != nil {
		return research.Observation{}, researchProvenanceFailure(err)
	}
	// Revalidate the member reference after provenance materialization.
	if _, err = a.collections.Select(ctx, collection.SelectionInput{ItemID: in.ItemID, ExpectedRevision: in.Revision, OriginalPublicationID: in.PublicationID, OriginalVersion: in.Version}); err != nil {
		return research.Observation{}, research.ErrNotFound
	}
	e := original.Evidence
	site, err := dataacquisition.ResolveSite(e.Site)
	if err != nil {
		return research.Observation{}, research.ErrUnavailable
	}
	v := research.Observation{Query: research.QuerySummary{Mode: original.Query.Mode, Keyword: original.Query.Keyword, CategoryNode: original.Query.CategoryNode, ASINCount: len(original.Query.ASINs), Limit: original.Query.Limit, Fields: append([]string{}, original.Query.Fields...)}, Item: detail.Item, Ref: research.EvidenceRef{Selection: in, Source: original.Source, JobID: original.JobID, Checksum: original.Checksum}, BatchID: detail.Item.BatchID, Site: e.Site, ASIN: e.ASIN, Title: e.Title, SourceURL: "https://" + site.Domain + "/dp/" + e.ASIN, CapturedAt: e.CapturedAt, ParserVersion: e.ParserVersion, Currency: e.Currency, Missing: append([]string{}, e.Missing...)}
	if e.Availability == "available" && e.Price > 0 {
		price := e.Price
		v.Price = &price
		v.PriceBasis = "displayed_listing_offer"
	}
	return v, nil
}
func composeMarketResearch(ctx context.Context, db *gorm.DB, project projectModule, collections *collection.Service, data *dataservicesapp.Module) (kernel.Module, error) {
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
	service := research.New(repo, researchProjects{project.handler.Service, auth}, researchSources{collections, data})
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
	return researchModule{handler: h, service: service}, nil
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
