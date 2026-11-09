package httpapi

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/authidentity"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/dataacquisition"
	"task-processor/internal/product/research"
	"task-processor/internal/product/sourcing"
)

func supplierEvidenceExpiry(ctx context.Context) time.Time {
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.TokenExpiresAt.IsZero() {
		return time.Time{}
	}
	expires := time.Now().Add(5 * time.Second)
	if identity.TokenExpiresAt.Before(expires) {
		expires = identity.TokenExpiresAt
	}
	if limit, ok := ctx.Deadline(); ok && limit.Before(expires) {
		expires = limit
	}
	return expires
}

func (a researchSources) ReadSupplierMarket(ctx context.Context, scope collection.Scope, project string, ref research.EvidenceRef) (research.SupplierMarket, error) {
	if ref.Source.Kind != "amazon_data" {
		return research.SupplierMarket{}, research.ErrUnsupported
	}
	in := collection.SelectionInput{ItemID: ref.Selection.ItemID, OriginalPublicationID: ref.Selection.PublicationID, OriginalVersion: ref.Selection.Version}
	item, err := a.collections.ReadResearchEvidenceItem(ctx, scope, in, &ref.Source)
	if err != nil {
		return research.SupplierMarket{}, supplierEvidenceFailure(err)
	}
	if collection.IsResearchFixtureProduct(item.Source.ProductKey) {
		if a.manualOriginal != nil {
			return a.manualMarket(ctx, scope, project, ref, item)
		}
		return a.fixtureMarket(ctx, scope, project, ref, item)
	}
	original, err := a.data.Provenance(ctx, dataacquisition.Principal{Scope: scope}, item.Source)
	if err != nil {
		return research.SupplierMarket{}, researchProvenanceFailure(err)
	}
	if original.Source != ref.Source || original.JobID != ref.JobID || original.Checksum != ref.Checksum || original.Evidence.Site != "us" {
		return research.SupplierMarket{}, research.ErrConflict
	}
	item, err = a.collections.ReadResearchEvidenceItem(ctx, scope, in, &ref.Source)
	if err != nil {
		return research.SupplierMarket{}, supplierEvidenceFailure(err)
	}
	return research.SupplierMarket{Ref: ref, Revision: item.Revision, ASIN: original.Evidence.ASIN, Title: original.Evidence.Title, Attributes: supplierSpecAttributes(original.Evidence.Attributes)}, nil
}

func (a researchSources) ReadSupplierOffer(ctx context.Context, scope collection.Scope, project string, sel research.Selection, bound *research.SupplierOriginal) (research.SupplierOffer, error) {
	in := collection.SelectionInput{ItemID: sel.ItemID, ExpectedRevision: sel.Revision, OriginalPublicationID: sel.PublicationID, OriginalVersion: sel.Version}
	var original *collection.Source
	if bound != nil {
		if bound.ItemID != sel.ItemID {
			return research.SupplierOffer{}, research.ErrConflict
		}
		in.ExpectedRevision = 0
		original = &bound.Source
	}
	if _, fixture := collection.ResearchFixtureSupplierOperation(sel.PublicationID); fixture {
		return a.fixtureOffer(ctx, scope, project, in, bound)
	}
	item, published, err := a.collections.ReadResearchSupplierAcquisition(ctx, scope, in, original)
	if err != nil {
		return research.SupplierOffer{}, supplierEvidenceFailure(err)
	}
	op, p := published.Result.Operation, published.Result.Publication
	if p == nil || op.Command == nil || op.State != sourcing.AcquisitionPublished || op.ID != item.Source.OperationID ||
		op.Scope.OrganizationID != scope.OrganizationID || op.Scope.ActorID != scope.ActorID {
		return research.SupplierOffer{}, research.ErrConflict
	}
	r, e := p.Receipt, p.Envelope
	snapshot := published.Snapshot
	channel := e.RawReference.Metadata["channel"]
	canonical, err := sourcing.Canonical1688Source(e.Identity.SourceURL)
	// The acquisition reader has verified the original command and receipt.
	// Research consumes that identity without constructing a new publication.
	if err != nil || op.Command.ProductKey != r.ProductKey || op.Command.PublicationID != r.PublicationID ||
		canonical != op.Source || canonical.OfferID != e.Identity.SourceID || e.Identity.SourcePlatform != "1688" ||
		e.Identity.SourceType != sourcing.SourceTypeCrawler || e.Trace.SourceRunID != "acquisition:"+op.ID ||
		e.RawReference.ReferenceType != "acquisition_evidence_v1" || e.RawReference.ReferenceID != canonical.OfferID ||
		(channel != sourcing.AcquisitionChannelPublicHTTP && channel != sourcing.AcquisitionChannelPublicBrowser) ||
		r.Producer != (sourcing.ProducerDescriptor{Kind: sourcing.AcquisitionProducerKind, Version: "v1"}) ||
		r.OrganizationID != scope.OrganizationID || r.ActorID != scope.ActorID || r.PublicationID != item.Source.PublicationID ||
		r.CatalogVersion != item.Source.Version || r.ProductKey != item.Source.ProductKey || r.PublishedAt.IsZero() ||
		r.EnvelopeHash == "" || !strings.HasPrefix(e.RawReference.Checksum, "sha256:") || len(e.RawReference.Checksum) != 71 ||
		snapshot.Identity.TenantID != scope.OrganizationID || snapshot.Identity.ProductKey != r.ProductKey ||
		snapshot.PublicationID != r.CatalogPublicationID || snapshot.Version != r.CatalogVersion {
		return research.SupplierOffer{}, research.ErrConflict
	}
	value := research.SupplierOffer{Revision: item.Revision, Title: e.ProductCandidate.Title,
		Original:   research.SupplierOriginal{ItemID: item.ID, Source: item.Source, OfferID: canonical.OfferID, CatalogPublicationID: r.CatalogPublicationID, EnvelopeHash: r.EnvelopeHash, RawChecksum: e.RawReference.Checksum},
		Attributes: supplierSpecAttributes(e.ProductCandidate.Attributes), Variants: []research.SupplierVariant{}}
	if bound != nil && value.Original != *bound {
		return research.SupplierOffer{}, research.ErrConflict
	}
	seen := make(map[string]bool, len(e.ProductCandidate.Variants))
	for _, v := range e.ProductCandidate.Variants {
		if v.SourceID == "" || len(v.SourceID) > 128 {
			continue
		}
		if seen[v.SourceID] {
			return research.SupplierOffer{}, research.ErrConflict
		}
		seen[v.SourceID] = true
		label := v.Title
		if label == "" {
			label = v.SKU
		}
		if label == "" || len(label) > 512 {
			label = v.SourceID
		}
		value.Variants = append(value.Variants, research.SupplierVariant{ID: v.SourceID, Label: label, Attributes: supplierSpecAttributes(v.Attributes)})
	}
	return value, nil
}

func supplierEvidenceFailure(err error) error {
	switch {
	case errors.Is(err, collection.ErrConflict):
		return research.ErrConflict
	case errors.Is(err, collection.ErrForbidden), errors.Is(err, sourcing.ErrPublicationForbidden):
		return research.ErrForbidden
	case errors.Is(err, collection.ErrNotFound), errors.Is(err, sourcing.ErrAcquisitionUnknown):
		return research.ErrNotFound
	default:
		return research.ErrUnavailable
	}
}

func supplierSpecAttributes(in map[string]string) map[string]string {
	out := make(map[string]string)
	for key, value := range in {
		if len(key) == 0 || len(key) > 128 || len(value) > 512 || !utf8.ValidString(key) || !utf8.ValidString(value) ||
			strings.ContainsRune(key, 0) || strings.ContainsRune(value, 0) || strings.HasPrefix(strings.ToLower(key), "source.") ||
			strings.TrimSpace(value) == "" {
			continue
		}
		out[key] = value
	}
	return out
}
