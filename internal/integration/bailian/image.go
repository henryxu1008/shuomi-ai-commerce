// Package bailian implements the admitted synchronous Qwen image route only.
// Product prompts, downloads, accounting and recovery retain their existing owners.
package bailian

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	strictjson "sigs.k8s.io/json"
	"task-processor/internal/ai"
	"task-processor/internal/integration/httpimage"
)

const Model = "qwen-image-3.0"
const ModelPro = "qwen-image-3.0-pro"
const Protocol = "dashscope-image-sync-v1"
const imagePath = "/api/v1/services/aigc/multimodal-generation/generation"

var workspaceHost = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}\.cn-beijing\.maas\.aliyuncs\.com$`)
var errInput = errors.New("Bailian image input or route unavailable")
var errUnknown = errors.New("Bailian image outcome unknown")

type Config struct {
	BaseURL, APIKey, CredentialReference, ConfigurationVersion, Model string
	Timeout                                                           time.Duration
	HTTPClient                                                        *http.Client // Explicit in-process test transports; production uses net/http.
	Diagnose                                                          func(Diagnostic)
}
type Observation struct{ RequestID, ResultURL, ResultDigest string }
type Observer func(context.Context, Observation) error
type Client struct {
	cfg      Config
	endpoint string
	http     *http.Client
	observe  Observer
}

// Endpoint validates a Beijing route; caller-controlled compatible URLs are not guessed.
func Endpoint(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u == nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.RawPath != "" || (u.Port() != "" && u.Port() != "443") {
		return "", errInput
	}
	host := u.Hostname()
	if host != "dashscope.aliyuncs.com" && !workspaceHost.MatchString(host) {
		return "", errInput
	}
	switch u.Path {
	case "", "/", "/api/v1", "/api/v1/", imagePath:
	default:
		return "", errInput
	}
	u.Path = imagePath
	return u.String(), nil
}

func NewClient(cfg Config, observe Observer) (*Client, error) {
	endpoint, err := Endpoint(cfg.BaseURL)
	if err != nil || !SupportsModel(cfg.Model) || observe == nil || strings.TrimSpace(cfg.APIKey) == "" || cfg.Timeout <= 0 || cfg.Timeout > 5*time.Minute || !canonical(cfg.CredentialReference) || !canonical(cfg.ConfigurationVersion) {
		return nil, errInput
	}
	h := http.Client{}
	if cfg.HTTPClient != nil {
		h = *cfg.HTTPClient
	}
	h.Timeout = cfg.Timeout
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return errUnknown }
	return &Client{cfg: cfg, endpoint: endpoint, http: &h, observe: observe}, nil
}
func canonical(s string) bool {
	return s != "" && strings.TrimSpace(s) == s && len(s) <= 192 && !strings.ContainsAny(s, "\r\n\x00")
}
func SupportsModel(model string) bool { return model == Model || model == ModelPro }

func (c *Client) GetDefaultModel() string          { return c.cfg.Model }
func (*Client) SupportsAsyncImageGeneration() bool { return false }
func (*Client) GenerateImage(context.Context, *ai.ImageGenerateRequest) (*ai.ImageResponse, error) {
	return nil, errInput
}
func (*Client) EditImage(context.Context, *ai.ImageEditRequest) (*ai.ImageResponse, error) {
	return nil, errInput
}
func (*Client) SubmitImageGeneration(context.Context, *ai.ImageGenerateRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}
func (*Client) SubmitImageEdit(context.Context, *ai.ImageEditRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}
func (*Client) QueryImageGeneration(context.Context, string) (*ai.ImageAsyncQueryResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}

func (c *Client) matches(route ai.ImageRouteSelection) bool {
	return c != nil && route.CredentialReference == c.cfg.CredentialReference && route.ConfigurationVersion == c.cfg.ConfigurationVersion
}
func (c *Client) GenerateImageWithRoute(ctx context.Context, r *ai.ImageGenerateRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	if !c.matches(route) || r == nil || r.Model != c.cfg.Model || r.N != 0 && r.N != 1 || r.Size != "" && r.Size != "auto" && r.Size != "1024x1024" || r.Quality != "" && r.Quality != "auto" {
		return nil, errInput
	}
	return c.submit(ctx, r.Prompt, nil)
}
func (c *Client) EditImageWithRoute(ctx context.Context, r *ai.ImageEditRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	if !c.matches(route) || r == nil || r.Model != c.cfg.Model || r.ImageURL != "" || len(r.ImageURLs) > 0 || len(r.Mask) > 0 || r.N != 0 && r.N != 1 || r.Size != "" && r.Size != "auto" && r.Size != "1024x1024" || r.Quality != "" && r.Quality != "auto" || len(r.ReferenceImages) > 2 {
		return nil, errInput
	}
	refs := append([]ai.ImageInlineReference{{Bytes: r.Image, MediaType: r.ImageContentType}}, r.ReferenceImages...)
	images := make([]string, 0, len(refs))
	total := 0
	for _, ref := range refs {
		total += len(ref.Bytes)
		if len(ref.Bytes) == 0 || len(ref.Bytes) > 10<<20 || total > 16<<20 || ref.MediaType != "image/png" && ref.MediaType != "image/jpeg" {
			return nil, errInput
		}
		i, format, err := image.DecodeConfig(bytes.NewReader(ref.Bytes))
		if err != nil || format == "png" && ref.MediaType != "image/png" || format == "jpeg" && ref.MediaType != "image/jpeg" || i.Width < 1 || i.Height < 1 || i.Width > 8000 || i.Height > 8000 || i.Width > 10*i.Height || i.Height > 10*i.Width {
			return nil, errInput
		}
		images = append(images, "data:"+ref.MediaType+";base64,"+base64.StdEncoding.EncodeToString(ref.Bytes))
	}
	return c.submit(ctx, r.Prompt, images)
}

func (c *Client) submit(ctx context.Context, prompt string, images []string) (*ai.ImageResponse, error) {
	if ctx == nil || ctx.Err() != nil || !utf8.ValidString(prompt) || strings.TrimSpace(prompt) == "" || len(prompt) > 16<<10 {
		return nil, errInput
	}
	content := make([]map[string]string, 0, len(images)+1)
	for _, source := range images {
		content = append(content, map[string]string{"image": source})
	}
	content = append(content, map[string]string{"text": prompt})
	payload, err := json.Marshal(map[string]any{"model": c.cfg.Model, "input": map[string]any{"messages": []any{map[string]any{"role": "user", "content": content}}}, "parameters": map[string]any{"n": 1, "size": "1024*1024", "watermark": false, "prompt_extend": false, "enable_thinking": false}})
	if err != nil {
		return nil, errInput
	}
	defer clear(payload)
	requestCtx, cancelRequest := context.WithCancel(ctx)
	defer cancelRequest()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, errInput
	}
	req.GetBody = nil // Neither redirects nor transport replay may resubmit an image POST.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, c.unknown("transport", 0, nil, "", transportReason(err))
	}
	defer resp.Body.Close()
	limit := 1 << 20
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Optional failure metadata must not wait for the image-generation
		// deadline. Cancel this request only; it grants no replay permission.
		limit = 64 << 10
		stopRead := time.AfterFunc(time.Second, cancelRequest)
		defer stopRead.Stop()
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, int64(limit)+1))
	defer clear(data)
	if err != nil || len(data) > limit {
		return nil, c.unknown("response_read", resp.StatusCode, nil, resp.Header.Get("X-Request-Id"), "")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if rejection, ok := ParseQuotaRejection(resp.StatusCode, data, resp.Header.Get("X-Request-Id")); ok && rejection.RequestID != c.cfg.APIKey {
			if c.cfg.Diagnose != nil {
				c.cfg.Diagnose(Diagnostic{Stage: "request_rejected", HTTPStatus: resp.StatusCode, ProviderCode: "AllocationQuota.FreeTierOnly", RequestID: rejection.RequestID})
			}
			return nil, rejection
		}
		return nil, c.unknown("http_status", resp.StatusCode, data, resp.Header.Get("X-Request-Id"), "")
	}
	var result struct {
		RequestID string `json:"request_id"`
		Code      string `json:"code"`
		Output    struct {
			Finished   *bool  `json:"finished"`
			TaskID     string `json:"task_id"`
			TaskStatus string `json:"task_status"`
			Choices    []struct {
				Finish  string `json:"finish_reason"`
				Message struct {
					Role    string `json:"role"`
					Content []struct {
						Image string `json:"image"`
						Text  string `json:"text"`
						Type  string `json:"type"`
					} `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		} `json:"output"`
		Usage struct {
			Input  *int `json:"input_image_count"`
			Count  *int `json:"output_image_count"`
			Width  *int `json:"output_width"`
			Height *int `json:"output_height"`
		} `json:"usage"`
	}
	strict, err := strictjson.UnmarshalStrict(data, &result, strictjson.DisallowDuplicateFields)
	if err != nil || len(strict) > 0 {
		return nil, c.unknown("response_json", resp.StatusCode, nil, resp.Header.Get("X-Request-Id"), "")
	}
	if !canonical(result.RequestID) || result.Code != "" || result.Output.Finished != nil && !*result.Output.Finished || result.Output.TaskID != "" || result.Output.TaskStatus != "" || len(result.Output.Choices) != 1 || result.Usage.Input == nil || *result.Usage.Input != len(images) || result.Usage.Count == nil || *result.Usage.Count != 1 || result.Usage.Width == nil || *result.Usage.Width != 1024 || result.Usage.Height == nil || *result.Usage.Height != 1024 {
		return nil, c.unknown("response_contract", resp.StatusCode, data, resp.Header.Get("X-Request-Id"), "")
	}
	choice := result.Output.Choices[0]
	if choice.Finish != "stop" || choice.Message.Role != "assistant" || len(choice.Message.Content) != 1 {
		return nil, c.unknown("response_choice", resp.StatusCode, data, resp.Header.Get("X-Request-Id"), "")
	}
	item := choice.Message.Content[0]
	_, publicErr := httpimage.ValidatePublicHTTPSURL(item.Image)
	u, err := url.Parse(item.Image)
	if err != nil || publicErr != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || len(item.Image) > 4096 || item.Text != "" || item.Type != "" && item.Type != "image" {
		return nil, c.unknown("response_locator", resp.StatusCode, data, resp.Header.Get("X-Request-Id"), "")
	}
	digest := sha256.Sum256(data)
	final, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if c.observe(final, Observation{RequestID: result.RequestID, ResultURL: item.Image, ResultDigest: hex.EncodeToString(digest[:])}) != nil {
		return nil, c.unknown("observation", resp.StatusCode, data, resp.Header.Get("X-Request-Id"), "")
	}
	return &ai.ImageResponse{RequestID: result.RequestID, UpstreamJobID: result.RequestID, Data: []ai.ImageData{{URL: item.Image}}, UsageKnown: false}, nil
}
