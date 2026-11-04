package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/stretchr/testify/require"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"task-processor/internal/ai"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

type flareRoundTrip func(*http.Request) (*http.Response, error)

func (f flareRoundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func flareEstimateFixture() FlareImageInputLimits {
	return FlareImageInputLimits{Size: "1024x1024", Quality: "auto", AllowAutoSize: true, MaximumPromptBytes: 65536, MaximumReferenceImages: 3, MaximumReferenceBytes: 16777216, MaximumReferencePixels: 64000000}
}

func TestFlareUnknownResponsesNeverRetryOrInventUsage(t *testing.T) {
	for _, mode := range []string{"transport", "server", "redirect", "missing usage", "wrong model"} {
		t.Run(mode, func(t *testing.T) {
			calls, observed := 0, 0
			client, err := NewFlareImageClient(FlareImageConfig{APIKey: "synthetic-only", Model: aicapability.FlareImageModel, BaseURL: "https://api.openai.com", CredentialReference: "credential", ConfigurationVersion: "config", Timeout: time.Second, Estimate: flareEstimateFixture(), HTTPClient: &http.Client{Transport: flareRoundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				if mode == "transport" {
					return nil, errors.New("controlled response loss")
				}
				status := 200
				body := `{"data":[]}`
				header := make(http.Header)
				if mode == "server" {
					status = 500
				}
				if mode == "redirect" {
					status = 307
					header.Set("Location", "https://api.openai.com/replay")
				}
				if mode == "wrong model" {
					body = `{"model":"different-model","data":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`
				}
				return &http.Response{StatusCode: status, Header: header, Body: io.NopCloser(strings.NewReader(body))}, nil
			})}}, func(context.Context, FlareImageObservation) error { observed++; return nil })
			require.NoError(t, err)
			_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: aicapability.FlareImageModel, Prompt: "test", N: 1, Size: "auto", Quality: "auto"}, ai.ImageRouteSelection{CredentialReference: "credential", ConfigurationVersion: "config"})
			require.ErrorIs(t, err, ErrFlareImageOutcomeUnknown)
			require.Equal(t, 1, calls)
			require.Zero(t, observed)
		})
	}
}
func TestFlareDirectWirePinsRouteAutoBytesAndSinglePOST(t *testing.T) {
	var imageBytes bytes.Buffer
	require.NoError(t, png.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 16, 16))))
	encoded := base64.StdEncoding.EncodeToString(imageBytes.Bytes())
	for _, edit := range []bool{false, true} {
		t.Run(map[bool]string{false: "generate", true: "edit"}[edit], func(t *testing.T) {
			calls, observed := 0, 0
			client, err := NewFlareImageClient(FlareImageConfig{APIKey: "synthetic-only", Model: aicapability.FlareImageModel, BaseURL: "https://api.openai.com/v1", Timeout: time.Second, CredentialReference: "synthetic-credential", ConfigurationVersion: "config-v1", Estimate: flareEstimateFixture(), HTTPClient: &http.Client{Transport: flareRoundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "api.openai.com", r.URL.Host)
				path := "/v1/images/generations"
				if edit {
					path = "/v1/images/edits"
				}
				require.Equal(t, path, r.URL.Path)
				var body map[string]json.RawMessage
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.JSONEq(t, `"gpt-image-2.5-flare-2026-09-08"`, string(body["model"]))
				require.JSONEq(t, `"auto"`, string(body["quality"]))
				require.JSONEq(t, `"auto"`, string(body["size"]))
				require.JSONEq(t, `1`, string(body["n"]))
				require.NotContains(t, body, "response_format")
				require.NotContains(t, body, "stream")
				if edit {
					var refs []map[string]string
					require.NoError(t, json.Unmarshal(body["images"], &refs))
					require.Len(t, refs, 2)
					require.Equal(t, "data:image/png;base64,"+encoded, refs[0]["image_url"])
					require.Equal(t, refs[0], refs[1])
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"X-Request-Id": []string{"synthetic-request"}}, Body: io.NopCloser(bytes.NewBufferString(`{"data":[{"b64_json":"` + encoded + `"}],"usage":{"input_tokens":10,"output_tokens":5,"total_tokens":15}}`))}, nil
			})}}, func(_ context.Context, o FlareImageObservation) error {
				observed++
				require.Equal(t, int64(10), o.Usage.InputTokens)
				require.Equal(t, imageBytes.Bytes(), o.ImageBytes)
				return nil
			})
			require.NoError(t, err)
			route := ai.ImageRouteSelection{CredentialReference: "synthetic-credential", ConfigurationVersion: "config-v1"}
			if edit {
				_, err = client.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: aicapability.FlareImageModel, Prompt: "synthetic edit", Image: imageBytes.Bytes(), ImageContentType: "image/png", ReferenceImages: []ai.ImageInlineReference{{Bytes: imageBytes.Bytes(), MediaType: "image/png"}}, Size: "auto", Quality: "auto", N: 1}, route)
			} else {
				_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: aicapability.FlareImageModel, Prompt: "synthetic generation", Size: "auto", Quality: "auto", N: 1}, route)
			}
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Equal(t, 1, observed)
			route.ConfigurationVersion = "changed"
			_, err = client.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: aicapability.FlareImageModel, Prompt: "synthetic", N: 1}, route)
			require.Error(t, err)
			require.Equal(t, 1, calls)
		})
	}
}
