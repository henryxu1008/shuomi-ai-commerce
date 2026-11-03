package imageagentapp

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/aicapability"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/asset"
	"task-processor/internal/product/catalog"
	"testing"
)

func TestAnalysisMaterialReaderRoutesOriginalOwnersWithoutURLFallback(t *testing.T) {
	_, facts, ctx, in, _ := analysisSourceFixture(t, false)
	public, owned, generated := 0, 0, 0
	r := AnalysisMaterialReader{Snapshots: facts, ReadPublic: func(_ context.Context, _ imageagent.AuthorizedAsset, max int64) ([]byte, error) {
		public++
		require.EqualValues(t, aicapability.MaximumSingleImageBytes, max)
		return facts.content, nil
	}, ReadOwned: func(context.Context, agent.Binding, string, string, int64) ([]byte, bool, error) {
		owned++
		return facts.content, true, nil
	}, ReadGenerated: func(context.Context, asset.SourceSelection, asset.ApprovedAsset, int64) ([]byte, error) {
		generated++
		return facts.content, nil
	}}
	img := catalog.IdentifyImages(facts.snapshot.Snapshot.Images)[0]
	raw, err := r.ReadAnalysisSource(ctx, in.Binding, img, aicapability.MaximumSingleImageBytes)
	require.NoError(t, err)
	require.Equal(t, facts.content, raw)
	require.Zero(t, public)
	require.Equal(t, 1, owned)
	img.URL = "https://injected.example/other.png"
	_, err = r.ReadAnalysisSource(ctx, in.Binding, img, aicapability.MaximumSingleImageBytes)
	require.Error(t, err)
	require.Equal(t, 1, owned)
	approved := facts.inventory.Assets[0]
	approved.GenerationEvidence = &asset.GenerationEvidence{ArtifactHash: "hash"}
	_, err = r.ReadAnalysisApproved(ctx, in.Binding, approved, aicapability.MaximumSingleImageBytes)
	require.NoError(t, err)
	require.Equal(t, 1, generated)
	require.Zero(t, public)
	r.ReadGenerated = nil
	_, err = r.ReadAnalysisApproved(ctx, in.Binding, approved, aicapability.MaximumSingleImageBytes)
	require.Error(t, err)
	require.Zero(t, public, "missing immutable reader cannot fall through to public URL")
}
