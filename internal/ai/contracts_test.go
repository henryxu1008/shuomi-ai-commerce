package ai

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestContractsMarshalStableJSON(t *testing.T) {
	for _, test := range []struct {
		value any
		want  string
	}{
		{ImageGenerateRequest{Model: "model", Prompt: "prompt", Size: "1024x1024", Quality: "auto", ResponseFormat: "url", N: 1}, `{"model":"model","prompt":"prompt","size":"1024x1024","quality":"auto","response_format":"url","n":1}`},
		{ImageResponse{Data: []ImageData{{URL: "https://example.test/result.png"}}}, `{"created":0,"data":[{"url":"https://example.test/result.png"}],"usage":{"prompt_tokens":0,"completion_tokens":0,"total_tokens":0}}`},
		{ImageRouteSelection{CredentialReference: "credential", ConfigurationVersion: "version"}, `{"CredentialReference":"credential","ConfigurationVersion":"version"}`},
	} {
		raw, err := json.Marshal(test.value)
		if err != nil || string(raw) != test.want {
			t.Fatalf("current image JSON changed: %s / %v", raw, err)
		}
	}
}
func TestContractsExposeMinimalInterfaces(t *testing.T) {
	var _ RouteBoundImageEditor = contractImageStub{}
	if reflect.TypeOf((*RouteBoundImageEditor)(nil)).Elem().NumMethod() != 1 {
		t.Fatal("route editor contract widened")
	}
}

type contractImageStub struct{}

func (contractImageStub) EditImageWithRoute(context.Context, *ImageEditRequest, ImageRouteSelection) (*ImageResponse, error) {
	return nil, nil
}
