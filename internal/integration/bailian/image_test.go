package bailian

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/ai"
)

type imageTransport func(*http.Request) (*http.Response, error)

func (f imageTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func sourcePNG(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	require.NoError(t, png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 8))))
	return b.Bytes()
}
func successBody() string {
	return `{"request_id":"actual-request","output":{"choices":[{"finish_reason":"stop","message":{"role":"assistant","content":[{"image":"https://images.example/result.png?Expires=123"}]}}]},"usage":{"input_image_count":1,"output_image_count":1,"output_width":1024,"output_height":1024}}`
}
func fixtureConfig(transport http.RoundTripper) Config {
	return Config{Model: Model, BaseURL: "https://workspace-1.cn-beijing.maas.aliyuncs.com", APIKey: "fixture-only", CredentialReference: "db:1:1:image_gpt_image_2", ConfigurationVersion: "config-1", Timeout: time.Minute, HTTPClient: &http.Client{Transport: transport}}
}
func fixtureRoute() ai.ImageRouteSelection {
	return ai.ImageRouteSelection{CredentialReference: "db:1:1:image_gpt_image_2", ConfigurationVersion: "config-1"}
}

func TestQwenImageEditSendsFrozenBytesOnceAndObservesBeforeReturning(t *testing.T) {
	for _, model := range []string{Model, ModelPro} {
		t.Run(model, func(t *testing.T) { testQwenImageEditSendsFrozenBytes(t, model) })
	}
}

func testQwenImageEditSendsFrozenBytes(t *testing.T, model string) {
	count, observations := 0, 0
	cfg := fixtureConfig(imageTransport(func(r *http.Request) (*http.Response, error) {
		count++
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/services/aigc/multimodal-generation/generation", r.URL.Path)
		require.Equal(t, "Bearer fixture-only", r.Header.Get("Authorization"))
		require.Nil(t, r.GetBody, "POST must not be replayable")
		var payload struct {
			Model string `json:"model"`
			Input struct {
				Messages []struct {
					Content []map[string]string `json:"content"`
				} `json:"messages"`
			} `json:"input"`
			Parameters struct {
				N        int    `json:"n"`
				Size     string `json:"size"`
				Extend   bool   `json:"prompt_extend"`
				Thinking bool   `json:"enable_thinking"`
			} `json:"parameters"`
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
		require.Equal(t, model, payload.Model)
		require.Equal(t, 1, payload.Parameters.N)
		require.Equal(t, "1024*1024", payload.Parameters.Size)
		require.False(t, payload.Parameters.Extend)
		require.False(t, payload.Parameters.Thinking)
		require.Len(t, payload.Input.Messages, 1)
		require.Len(t, payload.Input.Messages[0].Content, 2)
		require.True(t, strings.HasPrefix(payload.Input.Messages[0].Content[0]["image"], "data:image/png;base64,"))
		require.Equal(t, "preserve exact product and method", payload.Input.Messages[0].Content[1]["text"])
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(successBody()))}, nil
	}))
	cfg.Model = model
	c, err := NewClient(cfg, func(_ context.Context, o Observation) error {
		observations++
		require.Equal(t, "actual-request", o.RequestID)
		require.NotEmpty(t, o.ResultDigest)
		require.Contains(t, o.ResultURL, "result.png")
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, model, c.GetDefaultModel())
	r, err := c.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: model, Prompt: "preserve exact product and method", Image: sourcePNG(t), ImageContentType: "image/png", N: 1}, fixtureRoute())
	require.NoError(t, err)
	require.Len(t, r.Data, 1)
	require.Equal(t, 1, count)
	require.Equal(t, 1, observations)
	require.False(t, r.UsageKnown, "image counts are not token evidence")
}

func TestQwenUnknownAndContradictoryResultsNeverObserveOrRetry(t *testing.T) {
	for _, body := range []string{`{`, strings.Replace(successBody(), `"stop"`, `"length"`, 1), strings.Replace(successBody(), `"input_image_count":1`, `"input_image_count":0`, 1), strings.Replace(successBody(), `"output_image_count":1`, `"output_image_count":2`, 1), strings.Replace(successBody(), `"output_width":1024`, `"output_width":512`, 1), strings.Replace(successBody(), `"request_id":"actual-request",`, ``, 1), strings.Replace(successBody(), "https://images.example", "https://127.0.0.1", 1)} {
		t.Run(body[:min(len(body), 30)], func(t *testing.T) {
			calls, observed := 0, 0
			c, err := NewClient(fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body))}, nil
			})), func(context.Context, Observation) error { observed++; return nil })
			require.NoError(t, err)
			_, err = c.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: Model, Prompt: "exact", Image: sourcePNG(t), ImageContentType: "image/png"}, fixtureRoute())
			require.Error(t, err)
			require.Equal(t, 1, calls)
			require.Zero(t, observed)
		})
	}
}

func TestQwenModelAdmissionRejectsMissingUnknownAndCrossModelBeforeHTTP(t *testing.T) {
	calls := 0
	cfg := fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) {
		calls++
		panic("mismatched model must not dispatch")
	}))
	observe := func(context.Context, Observation) error { return nil }
	for _, invalid := range []string{"", "qwen-image", "qwen-image-3.0-pro ", "qwen-image-2.1-pro"} {
		cfg.Model = invalid
		_, err := NewClient(cfg, observe)
		require.Error(t, err, invalid)
	}
	for _, model := range []string{Model, ModelPro} {
		cfg.Model = model
		c, err := NewClient(cfg, observe)
		require.NoError(t, err)
		other := Model
		if model == Model {
			other = ModelPro
		}
		_, err = c.EditImageWithRoute(context.Background(), &ai.ImageEditRequest{Model: other, Prompt: "exact", Image: sourcePNG(t), ImageContentType: "image/png"}, fixtureRoute())
		require.Error(t, err)
		_, err = c.GenerateImageWithRoute(context.Background(), &ai.ImageGenerateRequest{Model: other, Prompt: "exact"}, fixtureRoute())
		require.Error(t, err)
	}
	require.Zero(t, calls)
}

func TestQwenRejectsWrongRegionRouteOrOversizedReferencesBeforeHTTP(t *testing.T) {
	for _, base := range []string{"https://dashscope-intl.aliyuncs.com", "https://workspace.ap-southeast-1.maas.aliyuncs.com", "https://evil.cn-beijing.maas.aliyuncs.com.evil.test", "https://a.b.cn-beijing.maas.aliyuncs.com", "http://dashscope.aliyuncs.com", "https://dashscope.aliyuncs.com/?key=private"} {
		cfg := fixtureConfig(nil)
		cfg.BaseURL = base
		_, err := NewClient(cfg, func(context.Context, Observation) error { return nil })
		require.Error(t, err, base)
	}
	calls := 0
	c, err := NewClient(fixtureConfig(imageTransport(func(*http.Request) (*http.Response, error) { calls++; panic("must not dispatch") })), func(context.Context, Observation) error { return nil })
	require.NoError(t, err)
	request := &ai.ImageEditRequest{Model: Model, Prompt: "exact", Image: sourcePNG(t), ImageContentType: "image/png"}
	route := fixtureRoute()
	route.CredentialReference = "foreign"
	_, err = c.EditImageWithRoute(context.Background(), request, route)
	require.Error(t, err)
	request.ReferenceImages = []ai.ImageInlineReference{{Bytes: sourcePNG(t), MediaType: "image/png"}, {Bytes: sourcePNG(t), MediaType: "image/png"}, {Bytes: sourcePNG(t), MediaType: "image/png"}}
	_, err = c.EditImageWithRoute(context.Background(), request, fixtureRoute())
	require.Error(t, err)
	require.Zero(t, calls)
}
