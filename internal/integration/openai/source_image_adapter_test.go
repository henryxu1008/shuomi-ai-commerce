package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"task-processor/internal/ai"
	productimage "task-processor/internal/product/image"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProductImageSourceWhiteBackgroundIsOneEditAndRejectsBadOutput(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(map[bool]string{true: "valid", false: "bad_image"}[valid], func(t *testing.T) {
			content := []byte("not an image")
			if valid {
				content = productImagePNG(t, 3, 4)
			}
			images := &productImageGeneratorStub{response: &ai.ImageResponse{Data: []ai.ImageData{{B64JSON: base64.StdEncoding.EncodeToString(content)}}}}
			adapter, err := NewSourceImageAdapter(validSourceImageAdapterConfig(images))
			require.NoError(t, err)
			white, err := productimage.NewSourceWhiteBackgroundCapability(adapter)
			require.NoError(t, err)
			source := productImageSource("source-1", "https://source.example/item.png")
			sourceBytes := productImagePNG(t, 2, 2)
			result, err := white.EditSourceWhiteBackground(context.Background(), productimage.SourceWhiteBackgroundRequest{Source: source, SourceBytes: sourceBytes, Product: productImageContext()})
			require.Equal(t, 1, images.editCalls)
			require.Empty(t, images.lastEdit.ImageURL)
			require.Equal(t, sourceBytes, images.lastEdit.Image, "exact source bytes, no extracted intermediate input")
			if !valid {
				require.ErrorIs(t, err, productimage.ErrOutputValidation)
				require.Empty(t, result.Asset.Bytes)
				return
			}
			require.NoError(t, err)
			require.Equal(t, []string{productimage.SourceWhiteBackgroundOperation}, result.Asset.Operations)
			require.Equal(t, source.SourceAssetID, result.Asset.SourceAssetID)
			require.Contains(t, images.lastEdit.Prompt, "preserve shape, proportions, colors, materials, patterns, text and accessory count")
		})
	}
}
func TestVIPSourceWhiteBackgroundRequiresOriginal1024WithoutTransformation(t *testing.T) {
	for _, size := range []int{1024, 1254} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			content := productImagePNG(t, size, size)
			images := &productImageGeneratorStub{response: &ai.ImageResponse{Data: []ai.ImageData{{B64JSON: base64.StdEncoding.EncodeToString(content)}}}}
			cfg := validSourceImageAdapterConfig(images)
			cfg.Provider, cfg.ImageModel = "grsai", "gpt-image-2-vip"
			diagnostics := 0
			cfg.DiagnoseOutputDimensions = func(ew, eh, aw, ah int) {
				diagnostics++
				require.Equal(t, []int{1024, 1024, size, size}, []int{ew, eh, aw, ah})
			}
			adapter, err := NewSourceImageAdapter(cfg)
			require.NoError(t, err)
			sourceBytes := productImagePNG(t, 2, 2)
			result, err := adapter.EditSourceWhiteBackground(context.Background(), productimage.SourceWhiteBackgroundRequest{Source: productImageSource("source-1", "https://source.example/item.png"), SourceBytes: sourceBytes, Product: productImageContext()})
			require.Equal(t, 1, images.editCalls)
			require.Equal(t, sourceBytes, images.lastEdit.Image)
			require.Equal(t, "1024x1024", images.lastEdit.Size)
			require.Equal(t, "medium", images.lastEdit.Quality)
			if size == 1024 {
				require.NoError(t, err)
				require.Equal(t, content, result.Asset.Bytes)
				require.Zero(t, diagnostics)
			} else {
				require.ErrorIs(t, err, productimage.ErrOutputValidation)
				require.Empty(t, result.Asset.Bytes)
				require.Equal(t, 1, diagnostics)
			}
		})
	}
}
func TestSourceImageAdapterBoundsAggregateInlineOutputBeforeCapabilityReturn(t *testing.T) {
	t.Parallel()

	used, err := consumeProductImageArtifactBudget(0, productimage.MaxInlineArtifactBytes)
	require.NoError(t, err)
	used, err = consumeProductImageArtifactBudget(used, productimage.MaxInlineArtifactBytes)
	require.NoError(t, err)
	require.Equal(t, productimage.MaxInlineArtifactAggregateBytes, used)

	_, err = consumeProductImageArtifactBudget(used, 1)
	require.ErrorIs(t, err, productimage.ErrOutputValidation)
}

func validSourceImageAdapterConfig(images ai.ImageGenerator) SourceImageAdapterConfig {
	return SourceImageAdapterConfig{ImageClient: images, Provider: "openai", ImageModel: "gpt-image-1", RouteReference: "route-a", CredentialReference: "credential-a", ConfigurationVersion: "config-v1", PricingVersion: "pricing-v1", ImageCostMicrosPerOutput: 100, CostUpperBoundKnown: true}
}
func TestSourceImageQuoteAndRouteRejection(t *testing.T) {
	images := &productImageGeneratorStub{}
	adapter, err := NewSourceImageAdapter(validSourceImageAdapterConfig(images))
	require.NoError(t, err)
	first, err := adapter.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: productimage.SourceWhiteBackgroundOperation, InputFingerprint: "input-a", MaximumOutputs: 1})
	require.NoError(t, err)
	second, err := adapter.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: productimage.SourceWhiteBackgroundOperation, InputFingerprint: "input-b", MaximumOutputs: 1})
	require.NoError(t, err)
	require.EqualValues(t, 1, first.MaximumOutputs)
	require.EqualValues(t, 1, first.MaximumModelCalls)
	require.EqualValues(t, 100, first.MaximumCostMicros)
	require.NotEqual(t, first.Fingerprint, second.Fingerprint)
	cfg := validSourceImageAdapterConfig(images)
	cfg.CostUpperBoundKnown = false
	unknown, err := NewSourceImageAdapter(cfg)
	require.NoError(t, err)
	q, err := unknown.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: productimage.SourceWhiteBackgroundOperation, InputFingerprint: "input-a", MaximumOutputs: 1})
	require.NoError(t, err)
	require.NotEqual(t, first.Fingerprint, q.Fingerprint)
	first.ConfigurationVersion = "rotated"
	_, err = adapter.EditSourceWhiteBackground(context.Background(), productimage.SourceWhiteBackgroundRequest{Source: productImageSource("source-1", "https://source.example/item.png"), SourceBytes: []byte{1}, Product: productImageContext(), Authorization: &first})
	require.ErrorIs(t, err, productimage.ErrCapabilityUnsupported)
	require.Nil(t, images.lastEdit)
	for _, operation := range []string{"extract_subject", "render_white_background", "render_scene", "review", "unknown"} {
		_, err = adapter.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: operation, InputFingerprint: "input-a", MaximumOutputs: 1})
		require.ErrorIs(t, err, productimage.ErrCapabilityUnsupported)
	}
}
func TestSourceImageAdapterRequiresCurrentCanonicalDependencies(t *testing.T) {
	for _, mutate := range []func(*SourceImageAdapterConfig){func(c *SourceImageAdapterConfig) { c.ImageClient = nil }, func(c *SourceImageAdapterConfig) { var nilClient *productImageGeneratorStub; c.ImageClient = nilClient }, func(c *SourceImageAdapterConfig) { c.CredentialReference = "" }, func(c *SourceImageAdapterConfig) { c.ConfigurationVersion = "" }, func(c *SourceImageAdapterConfig) { c.Provider = " " }, func(c *SourceImageAdapterConfig) { c.ImageCostMicrosPerOutput = -1 }} {
		cfg := validSourceImageAdapterConfig(&productImageGeneratorStub{})
		mutate(&cfg)
		_, err := NewSourceImageAdapter(cfg)
		require.Error(t, err)
	}
}
func productImageSource(id, url string) productimage.Asset {
	return productimage.Asset{URL: url, MediaType: "image/png", SourceURL: url, SourceAssetID: id, Role: productimage.RoleSource, Width: 1200, Height: 1200, Operations: []string{"ingest"}}
}
func productImageContext() productimage.ProductContext {
	return productimage.ProductContext{ProductKey: "product-1", Title: "Example", ProductType: "shoe", Attributes: map[string]string{"color": "blue"}}
}
func productImagePNG(t *testing.T, width, height int) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, width, height))
	canvas.Set(0, 0, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	var encoded bytes.Buffer
	require.NoError(t, png.Encode(&encoded, canvas))
	return encoded.Bytes()
}

type productImageGeneratorStub struct {
	editCalls int
	response  *ai.ImageResponse
	err       error
	lastEdit  *ai.ImageEditRequest
	lastRoute ai.ImageRouteSelection
}

func (s *productImageGeneratorStub) GenerateImage(context.Context, *ai.ImageGenerateRequest) (*ai.ImageResponse, error) {
	return s.response, s.err
}
func (s *productImageGeneratorStub) EditImage(_ context.Context, request *ai.ImageEditRequest) (*ai.ImageResponse, error) {
	s.editCalls++
	cloned := *request
	cloned.Image = append([]byte(nil), request.Image...)
	cloned.ImageURLs = append([]string(nil), request.ImageURLs...)
	s.lastEdit = &cloned
	return s.response, s.err
}
func (s *productImageGeneratorStub) EditImageWithRoute(ctx context.Context, request *ai.ImageEditRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	s.lastRoute = route
	return s.EditImage(ctx, request)
}
func (*productImageGeneratorStub) GetDefaultModel() string            { return "gpt-image-1" }
func (*productImageGeneratorStub) SupportsAsyncImageGeneration() bool { return false }
func (*productImageGeneratorStub) SubmitImageGeneration(context.Context, *ai.ImageGenerateRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}
func (*productImageGeneratorStub) SubmitImageEdit(context.Context, *ai.ImageEditRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}
func (*productImageGeneratorStub) QueryImageGeneration(context.Context, string) (*ai.ImageAsyncQueryResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}
