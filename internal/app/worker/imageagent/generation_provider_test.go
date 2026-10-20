package imageagentworker

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"task-processor/internal/agent"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	openai "task-processor/internal/integration/openai"
	"task-processor/internal/integration/skillimage"
	"task-processor/internal/skill"
)

type officialImageUseStore struct{ material skill.Package }

func (s officialImageUseStore) AcquireUseForTarget(context.Context, skill.Principal, agent.SkillVersionRef, string) (skill.Package, func(), error) {
	return s.material, func() {}, nil
}

type officialImageUseAuthority struct{}

func (officialImageUseAuthority) AuthorizeExecution(context.Context, imageagent.ExecutionIdentity) error {
	return nil
}

type qwenFactoryTransport func(*http.Request) (*http.Response, error)

func (f qwenFactoryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGenerationProviderFactoryUsesBoundQwenAdapterAndOriginalObservation(t *testing.T) {
	for _, model := range []string{"qwen-image-3.0", "qwen-image-3.0-pro"} {
		t.Run(model, func(t *testing.T) { testGenerationProviderFactoryUsesBoundQwenAdapter(t, model) })
	}
}

func TestGenerationProviderRefusalSurvivesCapabilityAndToolBoundaries(t *testing.T) {
	testGenerationProviderFactoryUsesBoundQwenAdapter(t, "qwen-image-3.0-pro", true)
}

func testGenerationProviderFactoryUsesBoundQwenAdapter(t *testing.T, model string, refused ...bool) {
	var pngBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 1024, 1024))))
	dispatches, observed, downloads := 0, 0, 0
	expectedObservations := 1
	expectedPrompt := "approved exact product and method"
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "org-image-v1", Config: &openai.ClientConfig{APIKey: "fixture-only", APIStyle: "dashscope", Model: model, BaseURL: "https://workspace.cn-beijing.maas.aliyuncs.com", Timeout: time.Second}}}
	factory := generationProviderFactory{resolver: resolver, price: config.ImageAgentGenerationConfig{PriceVersion: "trial-price", PointsPerImage: 1}, profile: synchronousEditProfile{}, providerHTTPClient: &http.Client{Transport: qwenFactoryTransport(func(r *http.Request) (*http.Response, error) {
		dispatches++
		require.Equal(t, "/api/v1/services/aigc/multimodal-generation/generation", r.URL.Path)
		var payload map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, model, payload["model"])
		messages := payload["input"].(map[string]any)["messages"].([]any)
		content := messages[0].(map[string]any)["content"].([]any)
		require.Equal(t, expectedPrompt, content[len(content)-1].(map[string]any)["text"], "prompt bytes must remain exact")
		if len(refused) > 0 && refused[0] {
			return &http.Response{StatusCode: 403, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"code":"AllocationQuota.FreeTierOnly","message":"quota exhausted","request_id":"070f62f5-244a-9703-85c5-a5f0c0d039e0"}`))}, nil
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"request_id":"qwen-receipt","output":{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[{"image":"https://output.example/result.png"}]}}]},"usage":{"input_image_count":1,"output_image_count":1,"output_width":1024,"output_height":1024}}`))}, nil
	})}, generatedFetcher: func(context.Context, string) ([]byte, error) {
		downloads++
		require.Equal(t, expectedObservations, observed, "durable success precedes output materialization")
		return pngBytes.Bytes(), nil
	}}
	prepared, err := factory.prepare(context.Background(), func(_ context.Context, proof imageagent.GenerationSuccess) error {
		observed++
		require.Equal(t, "qwen-receipt", proof.ResponseID)
		require.False(t, proof.UsageKnown)
		return nil
	})
	require.NoError(t, err)
	require.Zero(t, dispatches)
	sum := sha256.Sum256(pngBytes.Bytes())
	input := imageagent.SlotExecutionInput{RunID: "run-1", IdempotencyKey: "attempt", TenantID: "org-1", UserID: "user-1", PlanRevision: 1, Attempt: 1, TargetPlatform: "product", ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "slot"}, AssetCatalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/image.png"}}}, ProductContext: imageagent.ProductContextRef{ProductID: "product-1", Title: "Product"}, SourceBytes: pngBytes.Bytes(), SourceDigest: hex.EncodeToString(sum[:])}
	hash := strings.Repeat("a", 64)
	input.ImagePolicyContext = nil
	input.SourceReferences = [][]byte{pngBytes.Bytes()}
	input.ImageSet = &imageagent.ImageSetPlan{Schema: imageagent.ImageSetSchema, Source: imageagent.ImageSourceBinding{ContextKind: imageagent.ImageSourceSupply, ProductID: "product-1", OperationID: "source-1", OriginalPublicationID: "publication-1", OriginalVersion: 1, EffectiveVersion: 1, CatalogHash: "catalog-v2:" + hash}, Target: imageagent.ImageTarget{Platform: "product"}, Configuration: agent.ConfigurationSnapshotRef{Kind: "agent-configuration-v1", ID: "9e7afaa9-a9f9-48ba-a11a-b5bb377f08e9", Digest: hash}, ConfigurationEpoch: "1", ParametersDigest: hash, InputDigest: hash, MaxPoints: 1}
	input.Slot.Recipe = &imageagent.ImageSlotRecipe{Purpose: "product_identity", Background: "white", Language: "en", Placement: imageagent.ImagePlacement{Group: "carousel", Order: 1}, PromptVersion: imageagent.ImageSetSchema, Prompt: "approved exact product and method", References: []imageagent.ImageSourceObservation{{AssetID: "source-1", SHA256: hex.EncodeToString(sum[:]), MediaType: "image/png", Bytes: int64(len(pngBytes.Bytes())), Width: 1024, Height: 1024}}, Quote: imageagent.ImageGenerationQuote{Provider: prepared.Metadata.Provider, Model: prepared.Metadata.Model, Protocol: prepared.Metadata.Protocol, Resolution: "1024x1024", Quality: "auto", PriceVersion: prepared.Metadata.PriceVersion, Points: prepared.Metadata.Points, RouteReference: prepared.Metadata.RouteReference, CredentialReference: prepared.Metadata.CredentialReference, ConfigurationVersion: prepared.Metadata.ConfigurationVersion}}
	input.ImageSet.QuoteDigest, err = imageagent.ImageSetQuoteDigest(imageagent.Plan{Set: input.ImageSet, Slots: []imageagent.Slot{input.Slot}})
	require.NoError(t, err)
	quote, err := prepared.Executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	require.Equal(t, "bailian", quote.Operations[0].Provider)
	require.Equal(t, model, quote.Operations[0].Model)
	result, err := prepared.Executor.GenerateQuotedSlot(context.Background(), input, quote)
	if len(refused) > 0 && refused[0] {
		require.Error(t, err)
		proof, known := prepared.Rejection(err)
		require.True(t, known, "capability/tool wrapping must retain typed refusal evidence")
		require.Equal(t, "bailian_free_tier", proof.Kind)
		require.Equal(t, "070f62f5-244a-9703-85c5-a5f0c0d039e0", proof.RequestID)
		require.Zero(t, observed)
		require.Zero(t, downloads)
		require.Equal(t, 1, dispatches)
		return
	}
	require.NoError(t, err)
	require.Len(t, result.Assets, 1)
	require.Equal(t, 1, dispatches)
	require.Equal(t, 1, observed)
	require.Equal(t, 1, downloads)
	require.NoError(t, factory.revalidate(context.Background(), prepared.Metadata))
	resolver.resolved.Config.Model = "qwen-image-3.0"
	if model == "qwen-image-3.0" {
		resolver.resolved.Config.Model = "qwen-image-3.0-pro"
	}
	require.ErrorIs(t, factory.revalidate(context.Background(), prepared.Metadata), imageagent.ErrRevisionConflict)
	resolver.resolved.Config.Model = model
	packages, err := skill.OfficialPackages()
	require.NoError(t, err)
	imageMethods := 0
	for _, official := range packages {
		if !strings.HasPrefix(official.Key, "official-method-v1:image-") {
			continue
		}
		imageMethods++
		t.Run(official.Key, func(t *testing.T) {
			input.ImageSet.Method = &imageagent.ImageMethodBinding{PolicyVersion: imageagent.FrozenVisualBriefPolicyVersion, Skill: agent.SkillVersionRef{Kind: "skill-version-v1", ID: "2bb819bd-af13-4e68-95e0-1cc86130fb1f", OwnerKind: "PLATFORM", OwnerID: "official", Version: official.Version, Digest: official.Package.Digest, PolicyVersion: "skill-method-v1"}}
			adapter, err := skillimage.New(officialImageUseStore{official.Package}, officialImageUseAuthority{})
			require.NoError(t, err)
			id := imageagent.ExecutionIdentity{ScopeProtocol: imageagent.OrganizationScopeProtocol, RunID: "run-1", TenantID: "org-1", UserID: "user-1", MemberID: "member-1", BusinessTaskID: "product-1"}
			brief, release, err := imageagent.AcquireImageMethod(context.Background(), adapter, id, input.ImageSet.Method)
			require.NoError(t, err)
			defer release()
			input.ImageMethodText = brief
			expectedPrompt, err = imageagent.ComposeImageMethodPrompt(input.Slot.Recipe.Prompt, input.ImageMethodText)
			require.NoError(t, err)
			require.Contains(t, expectedPrompt, brief)
			for _, forbidden := range []string{"SKILL.md", "Method file:", "BEGIN UNTRUSTED METHOD", "listingkit-method-v1", "仅用于说明"} {
				require.NotContains(t, expectedPrompt, forbidden)
			}
			expectedObservations = observed + 1
			before := dispatches
			methodQuote, err := prepared.Executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
			require.NoError(t, err)
			input.ImageSet.Method.PolicyVersion = imageagent.HistoricalImageMethodPolicyVersion
			before = dispatches
			_, err = prepared.Executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
			require.Error(t, err)
			_, err = prepared.Executor.GenerateQuotedSlot(context.Background(), input, methodQuote)
			require.Error(t, err)
			require.Equal(t, before, dispatches, "historical policy cannot grant a provider POST")
			input.ImageSet.Method.PolicyVersion = imageagent.FrozenVisualBriefPolicyVersion
			_, err = prepared.Executor.GenerateQuotedSlot(context.Background(), input, methodQuote)
			require.Equal(t, before+1, dispatches, "each official image method must reach exactly one provider POST")
			require.NoError(t, err)
		})
	}
	require.Equal(t, 3, imageMethods)
	resolver.resolved.Config.Model = "qwen-mt-image-2.0"
	require.Error(t, factory.revalidate(context.Background(), prepared.Metadata))
}

type generationConfigTestResolver struct{ resolved *openai.ResolvedClientConfig }

func (r *generationConfigTestResolver) ResolveClientConfig(context.Context, string, *openai.ClientConfig) (*openai.ResolvedClientConfig, error) {
	return r.resolved, nil
}

func TestGenerationProviderFactoryUsesPinnedGRSAIProtocol(t *testing.T) {
	var pngBytes bytes.Buffer
	require.NoError(t, png.Encode(&pngBytes, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	dispatches, observed, downloads := 0, 0, 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dispatches++
		require.Equal(t, "/v1/api/generate", r.URL.Path)
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "gpt-image-2.5", body["model"])
		require.Equal(t, "json", body["replyType"])
		_, _ = w.Write([]byte(`{"id":"response-1","status":"succeeded","results":[{"url":"https://output.example/image.png"}]}`))
	}))
	defer server.Close()
	resolver := &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{CacheKey: "db:1:1:image_gpt_image_2", Config: &openai.ClientConfig{APIKey: "controlled-only", APIStyle: "grsai", Model: "gpt-image-2.5", BaseURL: server.URL, Timeout: time.Second}}}
	factory := generationProviderFactory{resolver: resolver, price: config.ImageAgentGenerationConfig{PriceVersion: "test-price", PointsPerImage: 17}, profile: synchronousEditProfile{}, providerHTTPClient: server.Client(), generatedFetcher: func(context.Context, string) ([]byte, error) {
		downloads++
		require.Equal(t, 1, observed)
		return pngBytes.Bytes(), nil
	}}
	prepared, err := factory.prepare(context.Background(), func(_ context.Context, proof imageagent.GenerationSuccess) error {
		observed++
		require.Equal(t, "response-1", proof.ResponseID)
		require.False(t, proof.UsageKnown)
		return nil
	})
	require.NoError(t, err)
	require.Zero(t, dispatches)
	require.EqualValues(t, 17, prepared.Metadata.Points)
	sum := sha256.Sum256(pngBytes.Bytes())
	input := imageagent.SlotExecutionInput{RunID: "run-1", IdempotencyKey: "attempt-key", TenantID: "org-1", UserID: "user-1", PlanRevision: 1, Attempt: 1, TargetPlatform: "product", ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}, Slot: imageagent.Slot{ID: "main-1", Role: imageagent.SlotRoleMain, SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "slot-key"}, AssetCatalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/image.png"}}}, ProductContext: imageagent.ProductContextRef{ProductID: "product-1", Title: "Product"}, SourceBytes: pngBytes.Bytes(), SourceDigest: hex.EncodeToString(sum[:])}
	quote, err := prepared.Executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	require.Len(t, quote.Operations, 1)
	require.Equal(t, "grsai", quote.Operations[0].Provider)
	result, err := prepared.Executor.GenerateQuotedSlot(context.Background(), input, quote)
	require.NoError(t, err)
	require.Len(t, result.Assets, 1)
	require.Equal(t, 1, dispatches)
	require.Equal(t, 1, downloads)
	require.NoError(t, factory.revalidate(context.Background(), prepared.Metadata))
	resolver.resolved.CacheKey = "db:1:2:image_gpt_image_2"
	require.ErrorIs(t, factory.revalidate(context.Background(), prepared.Metadata), imageagent.ErrRevisionConflict)
	require.Equal(t, 1, dispatches, "route checks never submit")
}

func TestGenerationProviderFactoryMissingPriceOrWrongRouteCannotBecomeAvailable(t *testing.T) {
	for _, mode := range []string{"no_price", "empty_version", "negative_price", "wrong_model", "wrong_style", "no_credential", "plain_http", "query_secret", "wrong_path", "unbounded_timeout"} {
		t.Run(mode, func(t *testing.T) {
			cfg := &openai.ClientConfig{APIKey: "controlled-only", Model: "gpt-image-2.5", APIStyle: "grsai", BaseURL: "https://provider.example/v1", Timeout: time.Minute}
			price := config.ImageAgentGenerationConfig{PriceVersion: "p1", PointsPerImage: 17}
			switch mode {
			case "no_price":
				price.PointsPerImage = 0
			case "empty_version":
				price.PriceVersion = ""
			case "negative_price":
				price.PointsPerImage = -1
			case "wrong_model":
				cfg.Model = "nano-banana-fast"
			case "wrong_style":
				cfg.APIStyle = "openai-compatible"
			case "no_credential":
				cfg.APIKey = ""
			case "plain_http":
				cfg.BaseURL = "http://provider.example"
			case "query_secret":
				cfg.BaseURL = "https://provider.example/?key=must-not-log"
			case "wrong_path":
				cfg.BaseURL = "https://provider.example/v1/images/edits"
			case "unbounded_timeout":
				cfg.Timeout = 0
			}
			factory := generationProviderFactory{resolver: &generationConfigTestResolver{resolved: &openai.ResolvedClientConfig{Config: cfg, CacheKey: "db:1:1:profile"}}, price: price, profile: synchronousEditProfile{}}
			_, err := factory.prepare(context.Background(), func(context.Context, imageagent.GenerationSuccess) error { return nil })
			require.ErrorIs(t, err, imageagent.ErrBudgetQuoteUnavailable)
			require.NotContains(t, err.Error(), "must-not-log")
			require.NotContains(t, err.Error(), "controlled-only")
		})
	}
}
