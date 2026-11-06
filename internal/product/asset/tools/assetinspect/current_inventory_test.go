package assetinspect

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/product/asset"
	"testing"
)

func TestCurrentApprovedOriginsPassOriginalToolSchema(t *testing.T) {
	for _, origin := range []string{"source", "generated"} {
		t.Run(origin, func(t *testing.T) {
			published, inventory := projectionFixture()
			image := &inventory.Assets[0]
			image.Presentation = &asset.ImagePresentation{Group: "carousel", Order: 1}
			image.SelectionReceipt = &asset.SelectionReceipt{ActionID: "original-selection", AssetID: "original-asset"}
			if origin == "source" {
				image.SelectionReceipt = nil
				image.RunID = ""
				image.PlanRevision = 0
				image.SlotID = ""
				image.Attempt = 0
				image.SourceAssetID = "source-1"
				image.SourceApproval = &asset.SourceApprovalProvenance{ActorID: "actor", MemberID: "member", OriginalPublicationID: published.PublicationID, OriginalSnapshotVersion: published.Version, ReferenceHash: asset.ReferenceHash(image.SourceAssetID, image.URL)}
			} else {
				image.GenerationEvidence = &asset.GenerationEvidence{IntentID: "intent", Fingerprint: "fingerprint", SettlementProofDigest: "proof", ArtifactHash: "hash"}
			}
			invoker, err := NewInvoker(&snapshotFixture{value: published}, &inventoryFixture{value: inventory}, &freshPrincipalFixture{}, testAgent(), testDependencies())
			require.NoError(t, err)
			result, err := invoker.Invoke(context.Background(), testMetadata(), Input{ProductKey: published.Identity.ProductKey, CatalogVersion: "9007199254740993", TargetPlatform: inventory.Scope.TargetPlatform})
			require.NoError(t, err)
			var output Output
			require.NoError(t, json.Unmarshal(result.Output, &output))
			require.Equal(t, inventory.Assets, output.ApprovedAssets)
		})
	}
}
