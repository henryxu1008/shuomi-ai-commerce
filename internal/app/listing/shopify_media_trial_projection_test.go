package listingapp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	wire "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/shopify"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/listing/submission"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/asset"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/catalog"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/product/collection"
	"github.com/stretchr/testify/require"
)

func TestShopifyTrialPreviewAndPlanUseTheSameEvidence(t *testing.T) {
	operation := submission.ShopifyMediaOperation{ID: uuid.NewString(),
		Owner:   collection.Scope{OrganizationID: "fixture-org", ActorID: "fixture-actor", MemberID: "fixture-member"},
		ShopKey: uuid.NewString(), Generation: 1, SourceKey: "fixture-coat", ProductID: "gid://shopify/Product/44"}
	input := ShopifyMediaPreviewRequest{SourceID: uuid.NewString(), StoreID: uuid.NewString(), Product: "44"}
	inventory := asset.ImageSetInventory{Head: asset.ImageInventoryHead{ActionID: "approval", PayloadHash: collection.Digest("selected")}, ApprovalActionID: "approval"}
	oldImage := wire.DraftGalleryEvidence{ID: "gid://shopify/MediaImage/40", Type: "IMAGE", Status: "READY",
		DraftMediaFingerprint: wire.DraftMediaFingerprint{Size: 11, Hash: sha256.Sum256([]byte("before trial"))}}
	remote := wire.DraftMediaEvidence{ID: operation.ProductID, Title: "Six-variant Shopify trial",
		Gallery: []wire.DraftGalleryEvidence{oldImage}}
	var local []catalog.Variant
	selected := shopifyMediaSourceEvidence{inventory: inventory, selection: asset.SourceSelectionRequest{
		OriginalPublicationID: "trial-original", OriginalSnapshotVersion: 1,
	}}
	for i, color := range []string{"Black", "Gray", "White"} {
		originalID := "source-" + color
		mainID := "main-" + color
		selected.source.Images = append(selected.source.Images, asset.SourceImage{ID: originalID, URL: originalID})
		selected.images = append(selected.images, ShopifyMediaPreviewImage{ID: mainID, Color: color,
			Role: asset.RoleMain, State: wire.DraftMediaUnverified, sourceID: originalID, uploadable: true})
		selected.positions = append(selected.positions, i)
		selected.proofs = append(selected.proofs, wire.DraftMediaFingerprint{Size: int64(len(mainID)), Hash: sha256.Sum256([]byte(mainID))})
		for j, size := range []string{"S", "M"} {
			sku := color + "-" + size
			local = append(local, catalog.Variant{SKU: sku, Attributes: []catalog.Attribute{{Name: "Color", Value: color}, {Name: "Size", Value: size}},
				Images: []catalog.Image{{URL: originalID}}})
			remote.Variants = append(remote.Variants, wire.DraftMediaVariant{ID: fmt.Sprintf("gid://shopify/ProductVariant/%d", 100+2*i+j), SKU: sku,
				Options: []wire.DraftMediaOption{{Name: "Color", Value: color}, {Name: "Size", Value: size}}})
		}
	}
	preview, plan, err := projectShopifyMediaEvidence(input, operation, selected, local, 1, remote,
		"https://fixture.myshopify.com/admin/products/44", time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.True(t, plan.Valid())
	require.Equal(t, 1, preview.GalleryCount)
	require.Len(t, preview.Variants, 6)
	require.Len(t, plan.Covers, 6)
	require.Len(t, plan.Images, 3)
	require.Equal(t, []string{oldImage.ID}, plan.GalleryIDs)
	for i, v := range preview.Variants {
		require.Equal(t, "different", v.State)
		require.Equal(t, fmt.Sprintf("%x", selected.proofs[i/2].Hash), plan.Covers[i].ImageHash)
		require.Equal(t, remote.Variants[i].ID, plan.Covers[i].VariantID)
	}

	latest := remote
	latest.Gallery = slices.Clone(remote.Gallery)
	latest.Gallery[0].Hash = sha256.Sum256([]byte("replaced old media"))
	_, changed, err := projectShopifyMediaEvidence(input, operation, selected, local, 1, latest, preview.AdminURL, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.NotEqual(t, plan.PreviewHash(), changed.PreviewHash())
	latest.Gallery[0].Hash = [32]byte{}
	latest.Gallery[0].Size = 0
	view, blocked, err := projectShopifyMediaEvidence(input, operation, selected, local, 1, latest, preview.AdminURL, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.False(t, blocked.Valid())
	require.Equal(t, wire.DraftMediaUnverified, view.Images[0].State)

	latest = remote
	latest.Variants = slices.Clone(remote.Variants)
	slices.Reverse(latest.Variants)
	latest.Variants[0].Options = slices.Clone(latest.Variants[0].Options)
	slices.Reverse(latest.Variants[0].Options)
	_, reordered, err := projectShopifyMediaEvidence(input, operation, selected, local, 1, latest, preview.AdminURL, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.Equal(t, plan.PreviewHash(), reordered.PreviewHash())

	latest = remote
	latest.Variants = slices.Clone(remote.Variants)
	latest.Variants[0].MediaIDs = []string{oldImage.ID}
	_, blocked, err = projectShopifyMediaEvidence(input, operation, selected, local, 1, latest, preview.AdminURL, time.Now().Add(time.Minute))
	require.NoError(t, err)
	require.False(t, blocked.Valid(), "unproven replacement of an old cover must stay unavailable")
	t.Run("the provider only uses invocation-local evidence", func(t *testing.T) {
		testShopifyMediaTrialProvider(t, plan, selected, remote)
	})
}

type mediaTrialWriterFixture struct {
	images                         []*wire.StagedDraftMedia
	appended                       func()
	uploads, appends, associations int
}

func (w *mediaTrialWriterFixture) Prepare(_ context.Context, digests [][32]byte) ([]*wire.StagedDraftMedia, error) {
	w.images = make([]*wire.StagedDraftMedia, len(digests))
	for i := range digests {
		w.images[i] = &wire.StagedDraftMedia{}
	}
	return w.images, nil
}
func (w *mediaTrialWriterFixture) Upload(_ context.Context, _ *wire.StagedDraftMedia, _ []byte) error {
	w.uploads++
	return nil
}
func (w *mediaTrialWriterFixture) Append(_ context.Context, _ []*wire.StagedDraftMedia) ([]string, error) {
	w.appends++
	w.appended()
	return []string{"gid://shopify/MediaImage/501", "gid://shopify/MediaImage/502", "gid://shopify/MediaImage/503"}, nil
}
func (w *mediaTrialWriterFixture) Associate(context.Context, []wire.DraftMediaAssociation) error {
	w.associations++
	return wire.ErrDraftMediaRejected
}

func testShopifyMediaTrialProvider(t *testing.T, plan submission.ShopifyMediaPlan, selected shopifyMediaSourceEvidence, remote wire.DraftMediaEvidence) {
	t.Helper()
	ctx := context.Background()
	evidence := shopifyTrialDispatchEvidence{source: selected, remote: remote}
	writer := &mediaTrialWriterFixture{appended: func() {
		for i, approved := range selected.proofs {
			evidence.remote.Gallery = append(evidence.remote.Gallery, wire.DraftGalleryEvidence{
				ID: fmt.Sprintf("gid://shopify/MediaImage/%d", 501+i), Type: "IMAGE", Status: "READY", DraftMediaFingerprint: approved,
			})
		}
	}}
	files := map[string][]byte{}
	for i, id := range plan.Images {
		files[id] = []byte(selected.images[i].ID)
	}
	var galleryReads, writerLeases int
	provider := &shopifyMediaTrialProvider{operation: plan.Operation(),
		read: func(_ context.Context, _ submission.ShopifyMediaPlan, gallery bool) (shopifyTrialDispatchEvidence, error) {
			if gallery {
				galleryReads++
				return evidence, nil
			}
			return shopifyTrialDispatchEvidence{source: evidence.source}, nil
		},
		image: func(_ context.Context, _ shopifyMediaSourceEvidence, hash string) ([]byte, error) {
			return files[hash], nil
		},
		writer: func(context.Context) (shopifyMediaTrialWriter, error) {
			writerLeases++
			return writer, nil
		},
	}
	prepare := submission.ShopifyMediaRequest{Plan: plan, Phase: submission.ShopifyMediaPrepare}
	_, err := provider.Send(ctx, prepare)
	require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
	require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaPrepare))
	prepared, err := provider.Send(ctx, prepare)
	require.NoError(t, err)
	require.True(t, prepared.OriginalResponse)
	require.Len(t, prepared.Staged, 3)
	for _, key := range prepared.Staged {
		_, err := uuid.Parse(key)
		require.NoError(t, err, "the execution engine receives only opaque generated handles")
	}

	bad := submission.ShopifyMediaRequest{Plan: plan, Phase: submission.ShopifyMediaUpload, Index: 0, Staged: []string{prepared.Staged[1]}}
	require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaUpload))
	_, err = provider.Send(ctx, bad)
	require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
	bad.Staged = []string{prepared.Staged[0]}
	originalBytes := files[plan.Images[0]]
	files[plan.Images[0]] = []byte("different approved file bytes")
	require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaUpload))
	_, err = provider.Send(ctx, bad)
	require.ErrorIs(t, err, submission.ErrExecutionEvidenceRequired)
	files[plan.Images[0]] = originalBytes
	require.Zero(t, writer.uploads)
	require.Equal(t, 1, writerLeases, "bad upload data must be refused before acquiring a short-lived writer")
	for i, key := range prepared.Staged {
		require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaUpload))
		_, err = provider.Send(ctx, submission.ShopifyMediaRequest{Plan: plan, Phase: submission.ShopifyMediaUpload, Index: i, Staged: []string{key}})
		require.NoError(t, err)
	}
	require.Equal(t, 1, galleryReads, "uploading the signed files does not fetch all remote originals again")
	require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaAppend))
	appended, err := provider.Send(ctx, submission.ShopifyMediaRequest{Plan: plan, Phase: submission.ShopifyMediaAppend, Staged: prepared.Staged})
	require.NoError(t, err)
	require.Equal(t, 1, writer.appends)

	view, err := provider.Read(ctx, plan)
	require.NoError(t, err)
	require.Len(t, view.Gallery, 4)
	originalVariant := evidence.remote.Variants[0]
	evidence.remote.Variants = slices.Clone(evidence.remote.Variants)
	evidence.remote.Variants[0].SKU = "changed after append"
	require.ErrorIs(t, provider.Verify(ctx, plan, submission.ShopifyMediaAssociate), submission.ErrExecutionEvidenceRequired)
	evidence.remote.Variants[0] = originalVariant
	extraGallery := evidence.remote.Gallery
	extra := extraGallery[1]
	extra.ID = "gid://shopify/MediaImage/701"
	evidence.remote.Gallery = append(slices.Clone(extraGallery), extra)
	require.ErrorIs(t, provider.Verify(ctx, plan, submission.ShopifyMediaAssociate), submission.ErrExecutionEvidenceRequired, "new remote media invalidates the association readback")
	evidence.remote.Gallery = extraGallery
	require.Zero(t, writer.associations)
	require.NoError(t, provider.Verify(ctx, plan, submission.ShopifyMediaAssociate))
	reply, err := provider.Send(ctx, submission.ShopifyMediaRequest{Plan: plan, Phase: submission.ShopifyMediaAssociate,
		Associations: []submission.ShopifyMediaAssociation{{VariantID: plan.Covers[0].VariantID, MediaID: appended.Candidates[0]}}})
	require.NoError(t, err)
	require.True(t, reply.OriginalResponse)
	require.True(t, reply.Rejected)
	require.Equal(t, 1, writer.associations)
}
