package productsourcing

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"task-processor/internal/imageagent"
	"task-processor/internal/product/collection"
	"task-processor/internal/product/sourcing"
	"testing"
)

func TestArtworkEnvelopeKeepsStableRunOriginAndHasNoInventedCommercialFacts(t *testing.T) {
	p := imageagent.RunProjection{Run: imageagent.Run{ScopeProtocol: imageagent.OrganizationScopeProtocol, ID: "run", TenantID: "org", UserID: "actor", MemberID: "member", BusinessTaskID: "creative"}, Plan: imageagent.Plan{Revision: 1, Artwork: &imageagent.ArtworkPlan{Theme: "原创图案", Requirements: "动物插画"}}}
	ref := imageagent.PublishedAssetRef{SHA256: "original-image-hash", Width: 1024, Height: 1024, ContentType: "image/png", Artwork: &imageagent.ArtworkProvenance{SlotID: "artwork", InputDigest: "input", IntentID: "intent", Fingerprint: "fingerprint"}}
	origin := artworkOriginFromRun(p, ref)
	require.Empty(t, origin.Identity.TraceID, "a save request trace must not change the original durable artwork identity")
	id := collection.StableID("artwork", "original-image-hash")
	envelope, err := artworkEnvelope(id, p.Plan.Artwork, origin, ref, "https://cdn.example/artwork.png")
	require.NoError(t, err)
	require.Equal(t, "agent_generated", envelope.Identity.SourceType)
	require.Equal(t, "pod_artwork", envelope.Identity.SourcePlatform)
	require.Empty(t, envelope.ProductCandidate.Variants)
	snapshot, err := sourcing.ToSnapshot(envelope)
	require.NoError(t, err)
	require.Empty(t, snapshot.Variants)
	var stored ArtworkOrigin
	require.NoError(t, json.Unmarshal([]byte(envelope.RawReference.Metadata[artworkOriginKey]), &stored))
	require.Equal(t, origin, stored)
	require.Equal(t, 1024, snapshot.Images[0].Width)
}
