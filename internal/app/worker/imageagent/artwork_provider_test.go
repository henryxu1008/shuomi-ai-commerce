package imageagentworker

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	"task-processor/internal/agentconfig"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	openai "task-processor/internal/integration/openai"
)

func TestArtworkProviderProducesActualNewPixelsAfterDurableObservation(t *testing.T) {
	var pixels bytes.Buffer
	require.NoError(t, png.Encode(&pixels, image.NewNRGBA(image.Rect(0, 0, 1024, 1024))))
	submits, observations, downloads := 0, 0, 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		submits++
		var body map[string]json.RawMessage
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body)) {
			w.WriteHeader(400)
			return
		}
		assert.NotContains(t, body, "images")
		assert.JSONEq(t, `"Create an original forest animal print."`, string(body["prompt"]))
		_, _ = w.Write([]byte(`{"id":"artwork-response","status":"succeeded","results":[{"url":"https://output.example/artwork.png"}]}`))
	}))
	defer server.Close()
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:1:1:image_gpt_image_2", Config: &openai.ClientConfig{APIKey: "controlled-only", APIStyle: "grsai", Model: "gpt-image-2.5", BaseURL: server.URL, Timeout: time.Second}}}
	factory := generationProviderFactory{resolver: resolver, price: config.ImageAgentGenerationConfig{PriceVersion: "test-price", PointsPerImage: 17}, profile: synchronousEditProfile{}, providerHTTPClient: server.Client(), generatedFetcher: func(context.Context, string) ([]byte, error) {
		downloads++
		require.Equal(t, 1, observations, "provider outcome must be persisted before downloading pixels")
		return pixels.Bytes(), nil
	}}
	prepared, err := factory.prepare(context.Background(), func(_ context.Context, proof imageagent.GenerationSuccess) error {
		observations++
		require.Equal(t, "artwork-response", proof.ResponseID)
		return nil
	})
	require.NoError(t, err)
	m := prepared.Metadata
	a := &imageagent.ArtworkPlan{Schema: imageagent.ArtworkSchema, Theme: "原创森林动物", PromptVersion: imageagent.ArtworkSchema, Prompt: "Create an original forest animal print.", Configuration: agent.ConfigurationSnapshotRef{Kind: agentconfig.SnapshotKind, ID: "00000000-0000-4000-8000-000000000001", Digest: strings.Repeat("a", 64)}, ModeSnapshotRef: agent.ModeSnapshotRef{Kind: agent.ExecutionModeSnapshotKind, ID: "00000000-0000-4000-8000-000000000002", Digest: strings.Repeat("b", 64)}, ConfigurationEpoch: "1", InputDigest: strings.Repeat("c", 64), ParametersDigest: strings.Repeat("d", 64), MaxPoints: m.Points, Quote: imageagent.ImageGenerationQuote{Provider: "grsai", Model: "gpt-image-2.5", Protocol: "grsai-json-sync-v1", Resolution: "1024x1024", Quality: "auto", Points: m.Points, PriceVersion: m.PriceVersion, RouteReference: m.RouteReference, CredentialReference: m.CredentialReference, ConfigurationVersion: m.ConfigurationVersion}}
	a.QuoteDigest = imageagent.ArtworkQuoteDigest(a.Quote)
	catalog, err := imageagent.NormalizeAssetCatalog(imageagent.AssetCatalog{})
	require.NoError(t, err)
	in := imageagent.SlotExecutionInput{Artwork: a, AssetCatalog: catalog, RunID: "artwork-run", TenantID: "org-1", UserID: "actor-1", PlanRevision: 1, Attempt: 1, IdempotencyKey: "artwork-attempt", TargetPlatform: "sds", Slot: imageagent.Slot{ID: "artwork", Role: imageagent.SlotRoleMain, Brief: a.Theme, IdempotencyKey: "artwork-slot", Status: imageagent.SlotStatusPending}}
	quote, err := prepared.Executor.QuoteSlot(context.Background(), in, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	require.Equal(t, "generate_artwork", quote.Operations[0].Name)
	require.Zero(t, submits, "quoting is not provider dispatch")
	output, err := prepared.Executor.GenerateQuotedSlot(context.Background(), in, quote)
	require.NoError(t, err)
	require.Equal(t, 1, submits)
	require.Equal(t, 1, downloads)
	require.Empty(t, output.SourceAssetID)
	require.Len(t, output.Assets, 1)
	require.Equal(t, pixels.Bytes(), output.Assets[0].Bytes)
	require.Equal(t, 1024, output.Assets[0].Width)
	require.Equal(t, "image/png", output.Assets[0].ContentType)
	require.Empty(t, output.Assets[0].URL, "private provider locators cannot become public image assets")
}
