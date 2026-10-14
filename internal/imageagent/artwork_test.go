package imageagent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
)

func artworkContractInput(t *testing.T) SlotExecutionInput {
	t.Helper()
	catalog, err := NormalizeAssetCatalog(AssetCatalog{})
	require.NoError(t, err)
	artwork := &ArtworkPlan{
		Schema: ArtworkSchema, Theme: "原创森林动物图案", Requirements: "独立印花，不含商品照片",
		PromptVersion: ArtworkSchema, Prompt: "Create original flat forest animal artwork for printing.",
		Configuration:      agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "00000000-0000-4000-8000-000000000001", Digest: strings.Repeat("a", 64)},
		ModeSnapshotRef:    agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: "00000000-0000-4000-8000-000000000002", Digest: strings.Repeat("b", 64)},
		ConfigurationEpoch: "1", ParametersDigest: strings.Repeat("c", 64), InputDigest: strings.Repeat("d", 64), MaxPoints: 12,
		Quote: ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: 12, PriceVersion: "price-1", RouteReference: "route-1", CredentialReference: "credential-1", ConfigurationVersion: "config-1"},
	}
	artwork.QuoteDigest = ArtworkQuoteDigest(artwork.Quote)
	return SlotExecutionInput{TenantID: "org-1", UserID: "actor-1", RunID: "run-1", TargetPlatform: "sds", PlanRevision: 1, Attempt: 1, IdempotencyKey: "artwork-attempt-1", AssetCatalog: catalog, Artwork: artwork,
		Slot: Slot{ID: "artwork", Role: SlotRoleMain, Brief: artwork.Theme, IdempotencyKey: "artwork-slot-1", Status: SlotStatusPending}}
}

func TestArtworkRequiresExplicitSourceFreeProtocol(t *testing.T) {
	input := artworkContractInput(t)
	plan := Plan{Revision: 1, IdempotencyKey: "artwork-plan-1", CreatedBy: input.UserID, Artwork: input.Artwork, Slots: []Slot{input.Slot}}
	require.NoError(t, ValidateInitialSubmittedPlan(plan), "from-zero artwork must not need a fabricated product or original image")
	require.NoError(t, ValidateArtworkExecution(input))
	legacy := plan
	legacy.Artwork = nil
	require.Error(t, ValidateSubmittedPlan(legacy), "ordinary image plans still require originals")
	for name, change := range map[string]func(*SlotExecutionInput){
		"source asset": func(i *SlotExecutionInput) { i.Slot.SourceAssetIDs = []string{"placeholder"} },
		"image set":    func(i *SlotExecutionInput) { i.ImageSet = &ImageSetPlan{Schema: ImageSetSchema} },
		"recipe": func(i *SlotExecutionInput) {
			i.Slot.Recipe = &ImageSlotRecipe{References: []ImageSourceObservation{{AssetID: "placeholder"}}}
		},
		"fake product":   func(i *SlotExecutionInput) { i.ProductContext.ProductID = "fake-product" },
		"fake catalog":   func(i *SlotExecutionInput) { i.AssetCatalog.ProductContext.ProductID = "fake-product" },
		"changed quote":  func(i *SlotExecutionInput) { i.Artwork.Quote.Points++ },
		"wrong protocol": func(i *SlotExecutionInput) { i.Artwork.Schema = ImageSetSchema },
	} {
		t.Run(name, func(t *testing.T) {
			bad := artworkContractInput(t)
			change(&bad)
			require.Error(t, ValidateArtworkExecution(bad))
		})
	}
}

func TestArtworkProvenanceAndGenerationBindActualInput(t *testing.T) {
	input := artworkContractInput(t)
	digest, err := ArtworkGenerationInputDigest(input)
	require.NoError(t, err)
	intent := generationTestIntent()
	intent.Identity.SlotID = input.Slot.ID
	intent.CatalogHash, intent.SourceDigest = input.AssetCatalog.Manifest.Hash, digest
	intent.InputProtocol, intent.InputDigest, intent.PromptVersion = ArtworkSchema, digest, ArtworkSchema
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	provenance, err := NewArtworkProvenance(input, fact)
	require.NoError(t, err)
	require.NoError(t, provenance.ValidateForInput(input))
	wrongRoute := intent
	wrongRoute.CredentialReference = "other-credential"
	wrongFact, err := NewGenerationFact(wrongRoute)
	require.NoError(t, err)
	_, err = NewArtworkProvenance(input, wrongFact)
	require.Error(t, err, "output cannot adopt another credential under the same creative digest")
	changed := input
	changed.Artwork = CloneArtworkPlan(input.Artwork)
	changed.Artwork.Requirements = "新的意图"
	newDigest, err := ArtworkGenerationInputDigest(changed)
	require.NoError(t, err)
	require.NotEqual(t, digest, newDigest)
	require.Error(t, provenance.ValidateForInput(changed), "a different creative intent cannot consume the original output")
	intent.SourceDigest = strings.Repeat("f", 64)
	_, err = NewGenerationFact(intent)
	require.Error(t, err, "creative source digest cannot pretend to identify a photograph")
	intent.SourceDigest, intent.InputProtocol = digest, ImageSetSchema
	_, err = NewGenerationFact(intent)
	require.Error(t, err, "artwork prompt cannot cross into the product image protocol")
}

func TestArtworkManifestAndCandidateKeepCreativeProvenance(t *testing.T) {
	input := artworkContractInput(t)
	digest, err := ArtworkGenerationInputDigest(input)
	require.NoError(t, err)
	intent := generationTestIntent()
	intent.Identity.SlotID = input.Slot.ID
	intent.CatalogHash, intent.SourceDigest, intent.InputProtocol, intent.InputDigest, intent.PromptVersion = input.AssetCatalog.Manifest.Hash, digest, ArtworkSchema, digest, ArtworkSchema
	fact, err := NewGenerationFact(intent)
	require.NoError(t, err)
	p, err := NewArtworkProvenance(input, fact)
	require.NoError(t, err)
	owner, err := ArtifactOwnerKey(input.UserID)
	require.NoError(t, err)
	ref := PublishedAssetRef{ObjectKey: "image-agent/public/org-1/" + owner + "/run-1/1/artwork/1/0-" + strings.Repeat("e", 64) + ".png", SHA256: strings.Repeat("e", 64), SizeBytes: 120, Width: 1024, Height: 1024, ContentType: "image/png", Operations: []string{"generate_artwork"}, Artwork: &p}
	require.NoError(t, ValidatePublishedAssetRefForSlot(input, ref, 0))
	other := input
	other.Artwork = nil
	require.Error(t, ValidatePublishedAssetRefForSlot(other, ref, 0), "ordinary image execution cannot consume source-free output")
	manifest, err := NormalizeFinalManifest(FinalManifest{Assets: []PublishedAssetRef{ref}})
	require.NoError(t, err, "actual creative output needs no fabricated SourceAssetID")
	result, err := NewSlotEffectV3PublishedResult(SlotExecutionResult{SlotID: "artwork", Attempt: 1, Candidates: []AssetCandidate{{AssetID: "artwork-result", DurableAsset: DurableAssetIdentity{ObjectKey: ref.ObjectKey, SHA256: ref.SHA256}, Width: ref.Width, Height: ref.Height, Operations: ref.Operations, Artwork: &p}}})
	require.NoError(t, err)
	fingerprint, err := SlotEffectV3PublishedResultFingerprint(result)
	require.NoError(t, err)
	require.NoError(t, ValidateSlotEffectV3Completion(result, manifest, fingerprint))
	projection := SlotProjection{Slot: input.Slot, Attempt: 1, Candidates: []AssetCandidate{{AssetID: "artwork-result", DurableAsset: DurableAssetIdentity{ObjectKey: ref.ObjectKey, SHA256: ref.SHA256}, Width: ref.Width, Height: ref.Height, Operations: ref.Operations, Artwork: &p}}}
	raw, err := json.Marshal(projection)
	require.NoError(t, err)
	var restored SlotProjection
	require.NoError(t, json.Unmarshal(raw, &restored))
	require.Equal(t, &p, restored.Candidates[0].Artwork, "persisted projections must retain actual creative provenance for adoption and recovery")
	changed := *manifest.Assets[0].Artwork
	changed.InputDigest = strings.Repeat("f", 64)
	manifest.Assets[0].Artwork = &changed
	require.Error(t, ValidateSlotEffectV3Completion(result, manifest, fingerprint), "creative provenance must survive staging to candidate persistence")
	ref.SourceAssetID = "fake-original"
	_, err = NormalizeFinalManifest(FinalManifest{Assets: []PublishedAssetRef{ref}})
	require.Error(t, err, "creative and original-image provenance must be mutually exclusive")
	ref.SourceAssetID, ref.Artwork = "", nil
	_, err = NormalizeFinalManifest(FinalManifest{Assets: []PublishedAssetRef{ref}})
	require.Error(t, err, "an unbound empty-source image cannot become a persisted candidate")
}
