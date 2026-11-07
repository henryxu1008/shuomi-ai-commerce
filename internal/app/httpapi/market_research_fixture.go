package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	pc "task-processor/internal/aiworkbench/projectcenter"
	dataservicesapp "task-processor/internal/app/dataservices"
	"task-processor/internal/authz"
	"task-processor/internal/httproute"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

type ResearchFixtureBinding struct {
	Scope             collection.Scope
	Project           string
	PaginationEnabled bool
}

func WithResearchFixture(binding ResearchFixtureBinding) CurrentApplicationOption {
	return func(o *currentApplicationOptions) { o.researchFixture = &binding }
}

const researchFixturePath = "/api/v1/workbench/research/projects/:project_id/acceptance-fixture"
const researchPaginationFixturePath = researchFixturePath + "/pagination"

func (m *researchModule) enableFixture(ctx context.Context, db *gorm.DB, binding ResearchFixtureBinding) error {
	fixture, err := dataservicesapp.NewResearchFixture(ctx, db, binding.Scope, binding.Project,
		m.projects.authorizeFixture, binding.PaginationEnabled)
	if err != nil {
		return err
	}
	sources, ok := m.service.Sources.(*researchSources)
	if !ok {
		return research.ErrUnavailable
	}
	m.fixture = fixture
	sources.fixture = fixture
	return nil
}

func (a researchProjects) authorizeFixture(ctx context.Context, scope collection.Scope, project string, write bool) error {
	projectScope := pc.Scope{OrganizationID: scope.OrganizationID, ActorID: scope.ActorID}
	if write {
		if err := a.authorize(ctx, scope, project, true, true); err != nil {
			return collection.ErrForbidden
		}
	} else if err := a.requireResearchProject(ctx, projectScope, project, true); err != nil {
		// ProjectCenter.Get already checks live ProjectCenter read permission.
		return collection.ErrForbidden
	}

	fresh, id, err := a.auth.fresh(ctx, projectScope)
	if err != nil || id.EffectiveMemberID != scope.MemberID {
		return collection.ErrForbidden
	}
	permission := collection.PermissionRead
	if write {
		permission = collection.PermissionManage
	}
	if !authz.AllowedOrganization(fresh, a.auth.policy, id.UserID, id.TenantID, id.Roles, permission) ||
		!authz.AllowedOrganization(fresh, a.auth.policy, id.UserID, id.TenantID, id.Roles, authz.PermissionProductSourcingWrite) {
		return collection.ErrForbidden
	}
	return nil
}

func marketResearchFixtureRoutes(m researchModule, path string) []httproute.Descriptor {
	routes := make([]httproute.Descriptor, 0, 2)
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		write := method == http.MethodPost
		permission := "workbench.project.read"
		if write {
			permission = "workbench.project.manage"
		}
		routes = append(routes, httproute.Descriptor{
			Method: method, Path: path, Module: m.Name(), Permission: permission,
			AuthPolicy: httproute.AuthPolicyCurrentIdentity, OrganizationAccessPolicy: httproute.OrganizationAccessPolicyLiveWrite,
			RequestTimeout: 20 * time.Second, RejectUnreadRequestBody: true,
			Handler: func(c *gin.Context) { m.serveFixture(c, write, path == researchPaginationFixturePath) },
		})
	}
	return routes
}

func validMarketResearchFixtureDescriptor(d httproute.Descriptor) bool {
	for _, expected := range marketResearchFixtureRoutes(researchModule{}, d.Path) {
		if d.Method == expected.Method && d.Path == expected.Path && d.Module == expected.Module &&
			d.Permission == expected.Permission && d.AuthPolicy == expected.AuthPolicy &&
			d.OrganizationAccessPolicy == expected.OrganizationAccessPolicy && d.OrganizationTargetResolver == nil &&
			d.RequestTimeout == expected.RequestTimeout && d.RejectUnreadRequestBody && d.Handler != nil {
			return true
		}
	}
	return false
}

func (m researchModule) serveFixture(c *gin.Context, write, pagination bool) {
	c.Header("Cache-Control", "private, no-store")
	req := c.Request
	project := c.Param("project_id")
	if !collection.ValidID(project) || req.URL.RawQuery != "" || req.URL.ForceQuery || len(req.TransferEncoding) != 0 || req.Header.Get("Content-Encoding") != "" ||
		write && req.ContentLength != 2 || !write && req.ContentLength != 0 {
		httproute.RejectUnreadRequestBody(c)
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST"})
		return
	}
	if write {
		raw, err := io.ReadAll(io.LimitReader(req.Body, 3))
		if err != nil || !bytes.Equal(raw, []byte("{}")) {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST"})
			return
		}
	}
	ctx, scope, err := m.handler.Bind(req.Context(), c.GetHeader("Authorization"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN"})
		return
	}
	var out any
	switch {
	case pagination && write:
		out, err = m.fixture.ImportPagination(ctx, scope, project)
	case pagination:
		out, err = m.fixture.ReadPagination(ctx, scope, project)
	case write:
		out, err = m.fixture.Import(ctx, scope, project)
	default:
		out, err = m.fixture.Read(ctx, scope, project)
	}
	switch {
	case errors.Is(err, collection.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN"})
	case errors.Is(err, collection.ErrNotFound):
		c.JSON(http.StatusOK, gin.H{"ready": false})
	case errors.Is(err, collection.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"code": "REVISION_CONFLICT"})
	case err != nil:
		c.JSON(http.StatusServiceUnavailable, gin.H{"code": "DEPENDENCY_UNAVAILABLE"})
	default:
		c.JSON(http.StatusOK, gin.H{"ready": true, "batch": out})
	}
}

func (a researchSources) CanReadPageEvidenceConcurrently(scope collection.Scope, project string, selection research.Selection) bool {
	return a.fixture.OwnsPagination(scope, project) && a.fixture.IsPaginationPublication(selection.PublicationID)
}

func (a researchSources) observeFixture(ctx context.Context, scope collection.Scope, project string, selection research.Selection, item collection.Item) (research.Observation, error) {
	if a.fixture.IsPaginationPublication(item.Source.PublicationID) {
		batch, entry, err := a.fixture.ReadPaginationOriginal(ctx, scope, project, item)
		if err != nil {
			return research.Observation{}, supplierEvidenceFailure(err)
		}
		return fixtureAmazonObservation(batch.ID, batch.Keyword, dataservicesapp.ResearchPaginationCount, entry, selection), nil
	}
	batch, err := a.fixture.Read(ctx, scope, project)
	if err != nil {
		return research.Observation{}, supplierEvidenceFailure(err)
	}
	entry, err := fixtureAmazonEntry(batch, item)
	if err != nil {
		return research.Observation{}, err
	}
	return fixtureAmazonObservation(batch.ID, batch.Keyword, 5, entry, selection), nil
}

func fixtureAmazonObservation(bundleID, keyword string, limit int, entry dataservicesapp.ResearchFixtureItem, selection research.Selection) research.Observation {
	original := entry.Original
	return research.Observation{
		Query: research.QuerySummary{Mode: "keyword", Keyword: keyword, Limit: limit, Fields: []string{"attributes", "price", "currency"}},
		Item:  entry.Item, Ref: fixtureAmazonRef(bundleID, entry, selection),
		BatchID: entry.Item.BatchID, Site: original.Site, ASIN: original.ID, Title: original.Title, Currency: original.Currency, Price: &original.Price,
		PriceBasis: "synthetic_listing_offer", ParserVersion: "research_acceptance_fixture/v1", FixtureVersion: original.ReferenceVersion, Missing: []string{},
	}
}

func fixtureAmazonEntry(batch dataservicesapp.ResearchFixtureBatch, item collection.Item) (dataservicesapp.ResearchFixtureItem, error) {
	for _, entry := range batch.Amazon {
		if entry.Item.Source == item.Source && entry.Item.ID == item.ID && entry.Item.BatchID == item.BatchID && entry.Item.Revision == item.Revision {
			return entry, nil
		}
	}
	return dataservicesapp.ResearchFixtureItem{}, research.ErrConflict
}

func fixtureAmazonRef(bundleID string, entry dataservicesapp.ResearchFixtureItem, selection research.Selection) research.EvidenceRef {
	return research.EvidenceRef{Selection: selection, Source: entry.Item.Source, JobID: bundleID, Checksum: entry.Checksum}
}

func (a researchSources) fixtureMarket(ctx context.Context, scope collection.Scope, project string, ref research.EvidenceRef, item collection.Item) (research.SupplierMarket, error) {
	batch, err := a.fixture.Read(ctx, scope, project)
	if err != nil {
		return research.SupplierMarket{}, supplierEvidenceFailure(err)
	}
	entry, err := fixtureAmazonEntry(batch, item)
	if err != nil || fixtureAmazonRef(batch.ID, entry, ref.Selection) != ref {
		return research.SupplierMarket{}, research.ErrConflict
	}
	o := entry.Original
	return research.SupplierMarket{Ref: ref, Revision: item.Revision, ASIN: o.ID, Title: o.Title, FixtureVersion: o.ReferenceVersion,
		Attributes: map[string]string{"width": o.Width, "installation": o.Installation, "pack": o.Pack}}, nil
}

func (a researchSources) fixtureOffer(ctx context.Context, scope collection.Scope, project string, in collection.SelectionInput, bound *research.SupplierOriginal) (research.SupplierOffer, error) {
	batch, err := a.fixture.Read(ctx, scope, project)
	if err != nil {
		return research.SupplierOffer{}, supplierEvidenceFailure(err)
	}
	entry := batch.Supplier
	var expected *collection.Source
	if bound != nil {
		expected = &bound.Source
	}
	item, err := a.collections.ReadResearchEvidenceItem(ctx, scope, in, expected)
	if err != nil || item.ID != entry.Item.ID || item.Source != entry.Item.Source || item.Revision != entry.Item.Revision {
		return research.SupplierOffer{}, research.ErrConflict
	}
	o := entry.Original
	original := research.SupplierOriginal{ItemID: item.ID, Source: item.Source, OfferID: o.ID,
		CatalogPublicationID: entry.Receipt.CatalogPublicationID, EnvelopeHash: entry.Receipt.EnvelopeHash, RawChecksum: entry.Checksum}
	if bound != nil && *bound != original {
		return research.SupplierOffer{}, research.ErrConflict
	}
	attrs := map[string]string{"width": o.Width}
	return research.SupplierOffer{Original: original, Revision: item.Revision, Title: o.Title, Attributes: attrs, FixtureVersion: o.ReferenceVersion,
		Variants: []research.SupplierVariant{{ID: o.SKU, Label: o.SKU, Attributes: attrs}}}, nil
}
