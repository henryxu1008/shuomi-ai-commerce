package openai

import (
	"context"
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
	productimage "task-processor/internal/product/image"
	"testing"
)

func TestSourceEditAdapterSendsExactBundleAndApprovedPromptOnce(t *testing.T) {
	for _, route := range [][2]string{{"grsai", "gpt-image-2.5"}, {"grsai", "gpt-image-2-vip"}, {"bailian", "qwen-image-3.0"}, {"bailian", "qwen-image-3.0-pro"}} {
		t.Run(route[1], func(t *testing.T) { testSourceEditAdapter(t, route[0], route[1]) })
	}
}

func testSourceEditAdapter(t *testing.T, provider, model string) {
	images := &productImageGeneratorStub{response: &ai.ImageResponse{Data: []ai.ImageData{{B64JSON: base64.StdEncoding.EncodeToString(productImagePNG(t, 1024, 1024))}}}}
	config := validProductImageAdapterConfig(images, nil)
	config.Provider = provider
	config.ImageModel = model
	var dimensions [][4]int
	config.DiagnoseOutputDimensions = func(ew, eh, aw, ah int) {
		dimensions = append(dimensions, [4]int{ew, eh, aw, ah})
	}
	adapter, err := NewProductImageAdapter(config)
	require.NoError(t, err)
	capability, err := productimage.NewSourceEditCapability(adapter)
	require.NoError(t, err)
	quote, err := adapter.QuoteUsage(context.Background(), productimage.UsageQuoteRequest{Operation: productimage.SourceEditOperation, InputFingerprint: "approved-input", MaximumOutputs: 1})
	require.NoError(t, err)
	request := productimage.SourceEditRequest{Sources: []productimage.Asset{productImageSource("one", "https://example.com/one.png"), productImageSource("two", "https://example.com/two.png")}, ReferenceBytes: [][]byte{[]byte("first exact bytes"), []byte("second exact bytes")}, Product: productImageContext(), Prompt: "approved purpose and exact real evidence", PromptVersion: "image-set-prompt-v1", Authorization: &quote}
	candidate, err := capability.EditSources(context.Background(), request)
	require.NoError(t, err)
	require.Equal(t, 1, images.editCalls)
	require.Equal(t, request.Prompt, images.lastEdit.Prompt)
	require.Empty(t, images.lastEdit.ImageURL)
	require.Empty(t, images.lastEdit.ImageURLs)
	require.Equal(t, request.ReferenceBytes[0], images.lastEdit.Image)
	require.Equal(t, request.ReferenceBytes[1], images.lastEdit.ReferenceImages[0].Bytes)
	require.NotNil(t, images.lastEdit.MaxRetries)
	require.Zero(t, *images.lastEdit.MaxRetries)
	require.Equal(t, 1, images.lastEdit.N)
	quality := "auto"
	if model == "gpt-image-2-vip" {
		quality = "medium"
	}
	require.Equal(t, quality, images.lastEdit.Quality)
	require.Equal(t, request.PromptVersion, candidate.Metadata.PromptVersion)
	require.Equal(t, []string{productimage.SourceEditOperation}, candidate.Asset.Operations)
	require.Empty(t, dimensions, "matching pixels must not report a mismatch")
	require.Equal(t, 1024, candidate.Asset.Width)
	require.Equal(t, 1024, candidate.Asset.Height)
	request.ReferenceBytes[1][0] = 'X'
	require.Equal(t, []byte("second exact bytes"), images.lastEdit.ReferenceImages[0].Bytes)
	original := productImagePNG(t, 1254, 1254)
	encoded := base64.StdEncoding.EncodeToString(original)
	images.response = &ai.ImageResponse{Data: []ai.ImageData{{B64JSON: encoded}}}
	request.ReferenceBytes[1] = []byte("second exact bytes")
	_, err = capability.EditSources(context.Background(), request)
	require.ErrorIs(t, err, productimage.ErrOutputValidation, "actual output dimensions must satisfy the configured native renderer")
	require.Equal(t, [][4]int{{1024, 1024, 1254, 1254}}, dimensions)
	require.Equal(t, encoded, images.response.Data[0].B64JSON, "diagnostics must not resize or rewrite the original artifact")
	require.Equal(t, 2, images.editCalls, "one dispatch per explicit operation, no mismatch retry")
}
