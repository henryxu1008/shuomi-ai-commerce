package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"task-processor/internal/imageagent"
	imagepolicy "task-processor/internal/marketplace/imagepolicy"
	productimage "task-processor/internal/product/image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGenericMainQuotesOnlyOneSourceEditWithoutExtractOrReview(t *testing.T) {
	input := testProductImageExecutionInput()
	input.TargetPlatform = "product"
	input.ImagePolicyContext = &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
	input.Slot.Role = imageagent.SlotRoleMain
	profile := testImageProfile()
	profile.Key = imagepolicy.PolicyKey{Marketplace: "product", Country: "zz", Family: "default", SceneCategory: "general"}
	executor := NewSourceGenerationExecutor(SourceGenerationDependencies{UsageQuoter: testProductUsageQuoter{}, ProfileResolver: &recordingImageProfileResolver{profile: profile}})
	quote, err := executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	require.Len(t, quote.Operations, 1)
	require.Equal(t, "render_source_white_background", quote.Operations[0].Name)
	require.EqualValues(t, 1, quote.Maximum.Images)
	require.EqualValues(t, 1, quote.Maximum.ModelCalls)
}
func TestGenericMainGeneratesOneSourceEditWithoutReviewerOrExtractor(t *testing.T) {
	input := testProductImageExecutionInput()
	input.SourceBytes = testTinyPNG(t)
	digest := sha256.Sum256(input.SourceBytes)
	input.SourceDigest = hex.EncodeToString(digest[:])
	input.TargetPlatform = "product"
	input.ImagePolicyContext = &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
	input.Slot.Role = imageagent.SlotRoleMain
	profile := testImageProfile()
	profile.Key = imagepolicy.PolicyKey{Marketplace: "product", Country: "zz", Family: "default", SceneCategory: "general"}
	profile.Thresholds.WhiteBackgroundReview = 1
	white := &recordingProductWhiteRenderer{candidate: testWhiteCandidate(t, "https://source.example/item.png")}
	white.candidate.Asset.Operations = []string{productimage.SourceWhiteBackgroundOperation}
	executor := NewSourceGenerationExecutor(SourceGenerationDependencies{WhiteBackground: white, UsageQuoter: testProductUsageQuoter{}, ProfileResolver: &recordingImageProfileResolver{profile: profile}})
	quote, err := executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	output, err := executor.GenerateQuotedSlot(context.Background(), input, quote)
	require.NoError(t, err)
	require.Equal(t, input.SourceBytes, white.request.SourceBytes)
	require.Empty(t, white.request.Source.Bytes)
	require.Equal(t, "source-1", white.request.Source.SourceAssetID)
	require.Len(t, output.Assets, 1)
	require.Equal(t, input.AssetCatalog.Assets[0].SourceURL, output.Assets[0].SourceURL)
	require.NotEqual(t, output.Assets[0].URL, output.Assets[0].SourceURL)
	require.Equal(t, []string{productimage.SourceWhiteBackgroundOperation}, output.Assets[0].Operations)
	require.EqualValues(t, 1, output.UsageReceipt.Actual.Images)
	require.EqualValues(t, 1, output.UsageReceipt.Actual.ModelCalls)
	require.Equal(t, 1, white.calls)
	mutated := input
	mutated.SourceBytes = append([]byte(nil), input.SourceBytes...)
	mutated.SourceBytes[0] ^= 1
	_, err = executor.GenerateQuotedSlot(context.Background(), mutated, quote)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	require.Equal(t, imageagent.ProviderRejectedBeforeEffect, imageagent.ProviderDispatchStateOf(err))
	require.Equal(t, 1, white.calls, "changed source bytes must not dispatch under the old intent digest")
	oldQuote := quote
	oldQuote.Fingerprint = "prior-extract-render-review-quote"
	_, err = executor.GenerateQuotedSlot(context.Background(), input, oldQuote)
	require.ErrorIs(t, err, imageagent.ErrRevisionConflict)
	require.Equal(t, 1, white.calls, "changed flow quote must not dispatch again")
	require.Error(t, err, "this flow must not quote a hidden Review on recovery")
	white.err = errors.New("image edit response or redirect outcome unknown")
	output, err = executor.GenerateQuotedSlot(context.Background(), input, quote)
	require.Error(t, err)
	require.Equal(t, imageagent.ProviderDispatchedUnknown, imageagent.ProviderDispatchStateOf(err), "transport refusal after a POST is not proof of no effect")
	require.Empty(t, output.Assets)
	require.Equal(t, 2, white.calls, "the executor does not retry the failed edit internally")
}
func testProductImageExecutionInput() imageagent.SlotExecutionInput {
	return imageagent.SlotExecutionInput{
		RunID: "run-1", TenantID: "tenant-1", UserID: "user-1", TargetPlatform: "shein",
		ImagePolicyContext: &imageagent.ImagePolicyContext{Country: "us", Family: "default", SceneCategory: "shoes"},
		PlanRevision:       1, Attempt: 1, IdempotencyKey: "attempt-1",
		Slot: imageagent.Slot{
			ID: "scene-1", Role: imageagent.SlotRoleScene, Brief: "show the product in use",
			SourceAssetIDs: []string{"source-1"}, IdempotencyKey: "slot-1",
		},
		AssetCatalog: imageagent.AssetCatalog{Assets: []imageagent.AuthorizedAsset{{
			ID: "source-1", Type: imageagent.AuthorizedAssetSource, URL: "https://source.example/item.png",
			SourceURL: "https://source.example/item.png", Width: 1200, Height: 1200,
		}}},
		ProductContext: imageagent.ProductContextRef{
			ProductID: "product-1", Title: "Product", ProductType: "shoe", Attributes: map[string]string{"color": "blue"},
		},
	}
}
func testImageProfile() imagepolicy.ProductImageProfile {
	return imagepolicy.ProductImageProfile{
		Key:           imagepolicy.PolicyKey{Marketplace: "shein", Country: "us", Family: "default", SceneCategory: "shoes"},
		PolicyVersion: "policy-v1",
		SceneDefaults: productimage.PolicySceneDefaults{
			SceneCategory: "shoes", SceneStyle: "lifestyle", BackgroundTone: "warm",
			Composition: "centered", PropsLevel: "light", AudienceHint: "general",
		},
	}
}
func testWhiteCandidate(t *testing.T, sourceURL string) productimage.Candidate {
	t.Helper()
	return productimage.Candidate{Asset: productimage.Asset{
		Bytes: testTinyPNG(t), MediaType: "image/png", SourceURL: sourceURL, SourceAssetID: "source-1",
		Role: productimage.RoleWhiteBackground, Width: 1, Height: 1, Operations: []string{"render_white_background"},
	}}
}
func testTinyPNG(t *testing.T) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 1, 1))
	canvas.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, canvas))
	return encoded.Bytes()
}

type recordingImageProfileResolver struct {
	input   imagepolicy.ProfileInput
	profile imagepolicy.ProductImageProfile
	err     error
}

func (r *recordingImageProfileResolver) Resolve(input imagepolicy.ProfileInput) (imagepolicy.ProductImageProfile, error) {
	r.input = input
	return r.profile, r.err
}

type recordingProductWhiteRenderer struct {
	calls     int
	candidate productimage.Candidate
	request   productimage.SourceWhiteBackgroundRequest
	err       error
}

func (r *recordingProductWhiteRenderer) EditSourceWhiteBackground(_ context.Context, request productimage.SourceWhiteBackgroundRequest) (productimage.Candidate, error) {
	r.calls++
	r.request = request
	return r.candidate, r.err
}

type testProductUsageQuoter struct{}

func (testProductUsageQuoter) QuoteUsage(_ context.Context, request productimage.UsageQuoteRequest) (productimage.UsageQuote, error) {
	return productimage.UsageQuote{
		Operation: request.Operation, Provider: "test", RouteReference: "test-route", Model: "test-model",
		CredentialReference: "test-credential", ConfigurationVersion: "test-config", PricingVersion: "test-pricing",
		Fingerprint: request.Operation + "-quote", MaximumOutputs: request.MaximumOutputs,
		MaximumModelCalls: 1, CostUpperBoundKnown: true,
	}, nil
}

func TestAcquisitionMainFrozenJSONMatchesPreRetirementBaseline(t *testing.T) {
	input := testProductImageExecutionInput()
	input.TargetPlatform = "product"
	input.ImagePolicyContext = &imageagent.ImagePolicyContext{Country: "zz", Family: "default", SceneCategory: "general"}
	input.Slot.ID = "main-1"
	input.Slot.Role = imageagent.SlotRoleMain
	input.SourceBytes = testTinyPNG(t)
	sum := sha256.Sum256(input.SourceBytes)
	input.SourceDigest = hex.EncodeToString(sum[:])
	input.AssetCatalog.ProductContext = input.ProductContext
	profile := testImageProfile()
	profile.Key = imagepolicy.PolicyKey{Marketplace: "product", Country: "zz", Family: "default", SceneCategory: "general"}
	white := &recordingProductWhiteRenderer{candidate: testWhiteCandidate(t, input.AssetCatalog.Assets[0].SourceURL)}
	white.candidate.Asset.Operations = []string{productimage.SourceWhiteBackgroundOperation}
	executor := NewSourceGenerationExecutor(SourceGenerationDependencies{WhiteBackground: white, UsageQuoter: testProductUsageQuoter{}, ProfileResolver: &recordingImageProfileResolver{profile: profile}})
	quote, err := executor.QuoteSlot(context.Background(), input, imageagent.BudgetPolicy{})
	require.NoError(t, err)
	output, err := executor.GenerateQuotedSlot(context.Background(), input, quote)
	require.NoError(t, err)
	owner, err := imageagent.ArtifactOwnerKey(input.UserID)
	require.NoError(t, err)
	published := imageagent.PublishedSlotOutput{SlotID: input.Slot.ID, Attempt: input.Attempt, Assets: []imageagent.PublishedAssetRef{{ObjectKey: fmt.Sprintf("image-agent/public/%s/%s/%s/1/%s/1/0-%s.png", input.TenantID, owner, input.RunID, input.Slot.ID, input.SourceDigest), SHA256: input.SourceDigest, SizeBytes: int64(len(input.SourceBytes)), ContentType: "image/png", Width: 1, Height: 1, SourceAssetID: input.Slot.SourceAssetIDs[0], Operations: output.Assets[0].Operations}}}
	result, err := executor.BuildSlotResult(context.Background(), input, published)
	require.NoError(t, err)
	plan := imageagent.Plan{Revision: 1, SourceAssetIDs: input.Slot.SourceAssetIDs, Slots: []imageagent.Slot{input.Slot}}
	slot := input.Slot
	slot.Status = imageagent.SlotStatusAccepted
	digest, err := imageagent.ResultDigestV3(plan, []imageagent.SlotProjection{{Slot: slot, Candidates: result.Candidates}})
	require.NoError(t, err)
	values := map[string]any{"input": input, "plan": plan, "profile": profile, "quote": quote, "generated": output, "published": published, "result": result, "result_digest": digest}
	hashes := map[string]string{}
	for name, value := range values {
		b, e := json.Marshal(value)
		require.NoError(t, e)
		sum := sha256.Sum256(b)
		hashes[name] = hex.EncodeToString(sum[:])
	}

	// Obtained by executing this same synthetic fixture at approved baseline 4ff3289ad,
	// before extraction. Hash the complete JSON documents, including nil/empty slots.
	expected := map[string]string{
		"generated":     "d61ec26f4f6652b0c9bd934a1a46479d733643296cab2b607a0a81580651215d",
		"input":         "0b6f56da1ae4230bb8536d7a045c923a0750422b6328153a86c657090bb2e9a6",
		"plan":          "51599b85f0ce193ca9a492286b6d8707fa066a42afb4a762745508910e02eb4f",
		"profile":       "9ece596147116a1fac40dbfdb59716aac802425171486702ec74c77d497e3bc2",
		"published":     "1194333ee016701300eca21ca9fe36faf1ea9cb2559b5f40f53beb2902f1a9c4",
		"quote":         "89aa3de5532a4ec03600550b10b4d9843fff0114b4d723a3abfed2ad760fa8bb",
		"result":        "1deee5aca73bf04c94dbb4754dde3a0528ef07625b296862b1de3856060fcb07",
		"result_digest": "35f7e89babe9ce2e09bda4c73c980a3c907acba51d04aa961131eb2b74d6810e",
	}
	require.Equal(t, expected, hashes)
}
