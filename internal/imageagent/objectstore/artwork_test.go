package objectstore

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
)

func TestArtworkPixelsRetainActualProvenanceThroughStoreAndRecovery(t *testing.T) {
	ctx := context.Background()
	objects := &fakeObjectStore{objects: map[string]fakeObject{}}
	store := newTestStore(t, objects)
	id := testIdentity()
	provenance := imageagent.ArtworkProvenance{Protocol: imageagent.ArtworkSchema, TenantID: id.TenantID, OwnerUserID: id.OwnerUserID, RunID: id.RunID, SlotID: id.SlotID, PlanRevision: id.PlanRevision, Attempt: id.Attempt, InputDigest: strings.Repeat("a", 64), IntentID: strings.Repeat("b", 64), Fingerprint: strings.Repeat("c", 64)}
	asset := validAsset(t, 3, 2)
	asset.SourceAssetID = ""
	asset.Artwork = &provenance
	asset.Operations = []string{"generate_artwork"}
	prepared, err := store.PrepareSlotArtifacts(PrepareSlotArtifactsInput{Identity: id, Assets: []ArtifactInput{asset}})
	require.NoError(t, err, "owned original artwork cannot require a fictional source image")
	require.Equal(t, &provenance, prepared.Manifest.Assets[0].Artwork)
	require.NoError(t, store.PreserveSlotArtifacts(ctx, id, prepared))
	require.NoError(t, store.EnsureStaged(ctx, prepared))
	recovered, err := store.RecoverSlotArtifacts(ctx, id, prepared.Manifest)
	require.NoError(t, err)
	require.Equal(t, &provenance, recovered.Manifest.Assets[0].Artwork)
	final, err := store.Finalize(ctx, recovered.Manifest)
	require.NoError(t, err)
	require.Equal(t, &provenance, final.Assets[0].Artwork)
	require.Empty(t, final.Assets[0].SourceAssetID)
	other := asset
	wrong := provenance
	wrong.OwnerUserID = "another-actor"
	other.Artwork = &wrong
	_, err = store.PrepareSlotArtifacts(PrepareSlotArtifactsInput{Identity: id, Assets: []ArtifactInput{other}})
	require.Error(t, err, "artifact storage must reject another creative execution scope")
}
