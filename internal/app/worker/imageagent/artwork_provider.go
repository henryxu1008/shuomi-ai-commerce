package imageagentworker

import (
	"bytes"
	"context"
	"image"
	"task-processor/internal/ai"
	"task-processor/internal/imageagent"
	"task-processor/internal/imageagent/tools"
	"task-processor/internal/integration/httpimage"
)

// This adapter reuses the fixed single-dispatch SDK transport and the same
// generation observer. It never constructs an original-image placeholder.
type artworkCreator struct {
	generate func(context.Context, *ai.ImageGenerateRequest) (*ai.ImageResponse, error)
	model    string
	fetch    func(context.Context, string) ([]byte, error)
}

func (a artworkCreator) CreateArtwork(ctx context.Context, prompt string) (imageagent.GeneratedAsset, error) {
	response, err := a.generate(ctx, &ai.ImageGenerateRequest{Model: a.model, Prompt: prompt, N: 1, Size: "1024x1024", Quality: "auto"})
	if err != nil {
		return imageagent.GeneratedAsset{}, err
	}
	if response == nil || len(response.Data) != 1 || response.Data[0].URL == "" {
		return imageagent.GeneratedAsset{}, imageagent.ErrInvalidGeneratedOutput
	}
	var data []byte
	if a.fetch != nil {
		data, err = a.fetch(ctx, response.Data[0].URL)
	} else {
		data, err = httpimage.Download(ctx, nil, response.Data[0].URL, 3<<20)
	}
	if err != nil {
		return imageagent.GeneratedAsset{}, err
	}
	if len(data) == 0 || len(data) > 3<<20 {
		return imageagent.GeneratedAsset{}, imageagent.ErrInvalidGeneratedOutput
	}
	mime, width, height, err := httpimage.InspectGeneratedArtifact(data)
	if err != nil || mime != "image/png" && mime != "image/jpeg" || width != 1024 || height != 1024 {
		return imageagent.GeneratedAsset{}, imageagent.ErrInvalidGeneratedOutput
	}
	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil || decoded.Bounds().Dx() != width || decoded.Bounds().Dy() != height {
		return imageagent.GeneratedAsset{}, imageagent.ErrInvalidGeneratedOutput
	}
	return imageagent.GeneratedAsset{Bytes: append([]byte(nil), data...), ContentType: mime, Type: "artwork", Width: width, Height: height, Operations: []string{"generate_artwork"}, ProviderReceiptID: response.UpstreamJobID}, nil
}

type scopedGenerationExecutor struct {
	product imageagent.BudgetedStagedSlotExecutor
	artwork *tools.ArtworkSlotExecutor
}

func (e scopedGenerationExecutor) selectExecutor(input imageagent.SlotExecutionInput) imageagent.BudgetedStagedSlotExecutor {
	if input.Artwork != nil {
		return e.artwork
	}
	return e.product
}
func (e scopedGenerationExecutor) QuoteSlot(ctx context.Context, in imageagent.SlotExecutionInput, policy imageagent.BudgetPolicy) (imageagent.SlotUsageQuote, error) {
	return e.selectExecutor(in).QuoteSlot(ctx, in, policy)
}
func (e scopedGenerationExecutor) GenerateQuotedSlot(ctx context.Context, in imageagent.SlotExecutionInput, quote imageagent.SlotUsageQuote) (imageagent.SlotGeneratedOutput, error) {
	return e.selectExecutor(in).GenerateQuotedSlot(ctx, in, quote)
}
func (e scopedGenerationExecutor) GenerateSlot(context.Context, imageagent.SlotExecutionInput) (imageagent.SlotGeneratedOutput, error) {
	return imageagent.SlotGeneratedOutput{}, imageagent.ErrCommandBlocked
}
func (e scopedGenerationExecutor) BuildSlotResult(ctx context.Context, in imageagent.SlotExecutionInput, out imageagent.PublishedSlotOutput) (imageagent.SlotExecutionResult, error) {
	return e.selectExecutor(in).BuildSlotResult(ctx, in, out)
}
