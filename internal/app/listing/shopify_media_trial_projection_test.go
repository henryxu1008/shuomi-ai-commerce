package listingapp

import (
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
	selected := shopifyMediaSourceEvidence{inventory: inventory}
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
}
