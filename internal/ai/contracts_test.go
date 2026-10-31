package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestContractsMarshalStableJSON(t *testing.T) {
	async := ImageAsyncSubmitResponse{
		JobID:      "job-1",
		Provider:   "provider-1",
		Status:     "accepted",
		AcceptedAt: time.Unix(1700000000, 0).UTC(),
		Response:   &ImageResponse{Data: []ImageData{{URL: "https://example.test/result.png"}}},
	}
	encoded, err := json.Marshal(async)
	if err != nil {
		t.Fatalf("marshal async response: %v", err)
	}
	const wantAsync = `{"job_id":"job-1","provider":"provider-1","status":"accepted","accepted_at":"2023-11-14T22:13:20Z","response":{"created":0,"data":[{"url":"https://example.test/result.png"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}}`
	if string(encoded) != wantAsync {
		t.Fatalf("unexpected async response JSON: %s", encoded)
	}
}

func TestContractsExposeMinimalInterfaces(t *testing.T) {
	var _ ImageGenerator = contractImageStub{}
	if !errors.Is(ErrAsyncImageGenerationNotSupported, ErrAsyncImageGenerationNotSupported) {
		t.Fatal("async unsupported sentinel must be comparable through errors.Is")
	}
}

type contractImageStub struct{}

func (contractImageStub) GenerateImage(context.Context, *ImageGenerateRequest) (*ImageResponse, error) {
	return nil, nil
}
func (contractImageStub) EditImage(context.Context, *ImageEditRequest) (*ImageResponse, error) {
	return nil, nil
}
func (contractImageStub) GetDefaultModel() string            { return "" }
func (contractImageStub) SupportsAsyncImageGeneration() bool { return false }
func (contractImageStub) SubmitImageGeneration(context.Context, *ImageGenerateRequest) (*ImageAsyncSubmitResponse, error) {
	return nil, nil
}
func (contractImageStub) SubmitImageEdit(context.Context, *ImageEditRequest) (*ImageAsyncSubmitResponse, error) {
	return nil, nil
}
func (contractImageStub) QueryImageGeneration(context.Context, string) (*ImageAsyncQueryResponse, error) {
	return nil, nil
}

func durationPointer(value time.Duration) *time.Duration { return &value }
func intPointer(value int) *int                          { return &value }
