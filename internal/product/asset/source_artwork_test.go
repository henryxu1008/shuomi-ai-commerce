package asset_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/asset/assettest"
	"testing"
)

func TestGeneratedSourceApprovalCannotBypassItsNewWriteGuard(t *testing.T) {
	selection := &sourceSelectionFixture{selection: asset.SourceSelection{GeneratedArtwork: true, TenantID: "org", ActorID: "actor", MemberID: "member", ItemID: "item", ProductKey: "product", OriginalPublicationID: "publication", OriginalSnapshotVersion: 1, EffectiveCatalogVersion: 1, TargetPlatform: "sds", Images: []asset.SourceImage{{ID: "image", URL: "https://cdn.example/artwork.png", ReferenceHash: asset.ReferenceHash("image", "https://cdn.example/artwork.png"), Width: 1024, Height: 1024}}}}
	repo := assettest.NewMemoryRepository()
	service, err := asset.NewSourceApprovalService(selection, repo, repo)
	require.NoError(t, err)
	cmd := asset.SourceApprovalCommand{ActionID: "approve", Selection: asset.SourceSelectionRequest{ItemID: "item", OriginalPublicationID: "publication", OriginalSnapshotVersion: 1, EffectiveCatalogVersion: 1, TargetPlatform: "sds"}, Images: []asset.SourceImageChoice{{ID: "image", Role: asset.RoleDesign}}}
	_, err = service.Approve(context.Background(), cmd)
	require.ErrorIs(t, err, asset.ErrSourceApprovalForbidden, "a generated source requires an explicitly injected fresh gate")
	allow := true
	calls := 0
	service.WithSourceApprovalCommitGuard(func(context.Context, asset.SourceSelection, asset.ApprovalCommit) error {
		calls++
		if !allow {
			return asset.ErrSourceApprovalForbidden
		}
		return nil
	})
	first, err := service.Approve(context.Background(), cmd)
	require.NoError(t, err)
	allow = false
	replay, err := service.Approve(context.Background(), cmd)
	require.NoError(t, err)
	require.Equal(t, first, replay)
	require.Equal(t, 1, calls)
	cmd.ActionID = "another-action"
	_, err = service.Approve(context.Background(), cmd)
	require.ErrorIs(t, err, asset.ErrSourceApprovalForbidden)
}
