package researchpersistence

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"task-processor/internal/product/collection"
	"task-processor/internal/product/research"
)

type supplierSourceFixture struct {
	amazon                    research.SupplierMarket
	offer                     research.SupplierOffer
	currentRevision           int64
	marketDenied, offerDenied bool
}

func (s *supplierSourceFixture) ReadSupplierMarket(_ context.Context, _ collection.Scope, _ string, ref research.EvidenceRef) (research.SupplierMarket, error) {
	if s.marketDenied || ref != s.amazon.Ref {
		return research.SupplierMarket{}, research.ErrForbidden
	}
	return s.amazon, nil
}

func (s *supplierSourceFixture) ReadSupplierOffer(_ context.Context, _ collection.Scope, _ string, in research.Selection, original *research.SupplierOriginal) (research.SupplierOffer, error) {
	if s.offerDenied {
		return research.SupplierOffer{}, research.ErrForbidden
	}
	if original == nil && in.Revision != s.currentRevision || original != nil && *original != s.offer.Original {
		return research.SupplierOffer{}, research.ErrConflict
	}
	result := s.offer
	result.Revision = s.currentRevision
	return result, nil
}

func verifySupplierReview(t *testing.T, ctx context.Context, svc *research.Service, scope collection.Scope, project string, current research.CandidateView) {
	t.Helper()
	sel := research.Selection{ItemID: uuid.NewString(), Revision: 2, PublicationID: "source-run:acquisition:" + uuid.NewString(), Version: 3}
	ref := current.Observation.Ref
	source := &supplierSourceFixture{
		amazon:          research.SupplierMarket{Ref: ref, Revision: 1, ASIN: current.Observation.ASIN, Title: "Amazon item", Attributes: map[string]string{"pack": "1 piece"}},
		offer:           research.SupplierOffer{Original: research.SupplierOriginal{ItemID: sel.ItemID, Source: collection.Source{Kind: "acquisition", ProductKey: "offer", PublicationID: sel.PublicationID, Version: sel.Version, OperationID: uuid.NewString()}, OfferID: "12345", CatalogPublicationID: sel.PublicationID, EnvelopeHash: "test-envelope", RawChecksum: "test-raw"}, Title: "1688 item", Variants: []research.SupplierVariant{{ID: "blue", Label: "Blue", Attributes: map[string]string{"pack": "2 pieces"}}}},
		currentRevision: 2,
	}
	cmd := research.Command{Action: "record_supplier_review", CandidateID: current.ID, ExpectedRevision: current.Revision, LinkID: current.SelectedLinkID, EvidenceDigest: current.EvidenceDigest,
		Supplier: &research.SupplierInput{Selection: sel, SKUID: "blue", Quantity: 20, Unit: "piece", Status: "CONFIRMED", Reason: "包装差异已看过", Comparisons: []research.SupplierComparison{{MarketKey: "pack", SupplierKey: "pack", SupplierLevel: "sku", Result: "DIFFERENT"}}}}
	key := uuid.NewString()
	_, err := svc.Execute(ctx, scope, project, key, cmd)
	require.ErrorIs(t, err, research.ErrUnsupported, "the feature defaults closed")
	svc.SupplierSources = source
	svc.SupplierExpiry = func(ctx context.Context) time.Time { return time.Now().Add(5 * time.Second) }

	preview, err := svc.SupplierPreview(ctx, scope, project, current.ID, sel)
	require.NoError(t, err)
	require.Equal(t, "blue", preview.Offer.Variants[0].ID)
	var wg sync.WaitGroup
	var receipts [2]research.Receipt
	var failures [2]error
	for i := range receipts {
		wg.Go(func() { receipts[i], failures[i] = svc.Execute(ctx, scope, project, key, cmd) })
	}
	wg.Wait()
	require.NoError(t, failures[0])
	require.NoError(t, failures[1])
	first := receipts[0]
	require.Equal(t, current.Revision+1, first.Revisions[current.ID])
	require.Equal(t, first.Revisions, receipts[1].Revisions)
	require.NotEqual(t, first.Replayed, receipts[1].Replayed)
	altered := cmd
	input := *cmd.Supplier
	input.Quantity++
	altered.Supplier = &input
	_, err = svc.Execute(ctx, scope, project, key, altered)
	require.ErrorIs(t, err, research.ErrConflict)
	replay, err := svc.Execute(ctx, scope, project, key, cmd)
	require.NoError(t, err)
	require.True(t, replay.Replayed)
	require.Equal(t, first.Revisions, replay.Revisions)
	_, err = svc.Execute(ctx, scope, project, uuid.NewString(), cmd)
	require.ErrorIs(t, err, research.ErrConflict)
	source.currentRevision++
	_, err = svc.SupplierPreview(ctx, scope, project, current.ID, sel)
	require.ErrorIs(t, err, research.ErrConflict)
	page, err := svc.SupplierReviews(ctx, scope, project, current.ID, 0)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Equal(t, 20, page.Items[0].Detail.Quantity)
	require.Equal(t, "2 pieces", page.Items[0].Detail.Comparisons[0].SupplierValue)
	require.True(t, page.Items[0].Current)
	for _, deny := range []*bool{&source.marketDenied, &source.offerDenied} {
		*deny = true
		page, err = svc.SupplierReviews(ctx, scope, project, current.ID, 0)
		require.NoError(t, err)
		require.Len(t, page.Items, 1)
		require.Nil(t, page.Items[0].Detail)
		raw, err := json.Marshal(page)
		require.NoError(t, err)
		require.NotContains(t, string(raw), "包装差异")
		require.NotContains(t, string(raw), "2 pieces")
		require.NotContains(t, string(raw), "blue")
		receipt, err := svc.Lookup(ctx, scope, project, key)
		require.NoError(t, err)
		require.Equal(t, first.Revisions, receipt.Revisions)
		*deny = false
	}
	source.offer.Original.Source.Version++
	page, err = svc.SupplierReviews(ctx, scope, project, current.ID, 0)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.Nil(t, page.Items[0].Detail, "a new original cannot substitute for the saved evidence")
}
