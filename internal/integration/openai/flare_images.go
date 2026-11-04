package openai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"task-processor/internal/ai"
	"task-processor/internal/integration/httpimage"

	strict "sigs.k8s.io/json"
)

var ErrFlareImageOutcomeUnknown = errors.New("Flare image outcome unknown")

type FlareImagePreflightError struct{ Digest string }

func (*FlareImagePreflightError) Error() string          { return "Flare image rejected before dispatch" }
func (*FlareImagePreflightError) NoExternalEffect() bool { return true }

type FlareImageConfig struct {
	APIKey, Model, BaseURL, CredentialReference, ConfigurationVersion string
	Timeout                                                           time.Duration
	Estimate                                                          FlareImageInputLimits
	HTTPClient                                                        *http.Client
}

type FlareImageObservation struct {
	Usage                                                  FlareImageUsage
	ResponseID, RequestID, ResultDigest, ResultUnavailable string
	ImageBytes                                             []byte
}

type FlareImageClient struct {
	cfg     FlareImageConfig
	base    string
	http    http.Client
	observe func(context.Context, FlareImageObservation) error
}

// FlareImageEndpoint admits only the approved official origin. A test may inject
// an in-process transport; it cannot turn a different URL into an admitted route.
func FlareImageEndpoint(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "api.openai.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || u.Opaque != "" {
		return "", &FlareImagePreflightError{}
	}
	if u.Path != "" && u.Path != "/" && u.Path != "/v1" && u.Path != "/v1/" {
		return "", &FlareImagePreflightError{}
	}
	return "https://api.openai.com/v1", nil
}

func NewFlareImageClient(cfg FlareImageConfig, observe func(context.Context, FlareImageObservation) error) (*FlareImageClient, error) {
	base, err := FlareImageEndpoint(cfg.BaseURL)
	if err != nil || cfg.Model != FlareImageModel || strings.TrimSpace(cfg.APIKey) == "" || !canonicalProductImageString(cfg.CredentialReference) || !canonicalProductImageString(cfg.ConfigurationVersion) || cfg.Timeout <= 0 || cfg.Timeout > 5*time.Minute || cfg.Estimate.Validate() != nil || observe == nil {
		return nil, &FlareImagePreflightError{}
	}
	h := http.Client{}
	if cfg.HTTPClient != nil {
		h = *cfg.HTTPClient
	}
	h.Timeout = cfg.Timeout
	h.CheckRedirect = func(*http.Request, []*http.Request) error { return ErrFlareImageOutcomeUnknown }
	return &FlareImageClient{cfg: cfg, base: base, http: h, observe: observe}, nil
}

func (c *FlareImageClient) matches(route ai.ImageRouteSelection) bool {
	return c != nil && route.CredentialReference == c.cfg.CredentialReference && route.ConfigurationVersion == c.cfg.ConfigurationVersion
}

type flareImageRequest struct {
	Model        string            `json:"model"`
	Prompt       string            `json:"prompt"`
	N            int               `json:"n"`
	Size         string            `json:"size"`
	Quality      string            `json:"quality"`
	OutputFormat string            `json:"output_format"`
	Images       []flareImageInput `json:"images,omitempty"`
}
type flareImageInput struct {
	ImageURL string `json:"image_url"`
}

func flarePreflight(value any) error {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return &FlareImagePreflightError{Digest: hex.EncodeToString(sum[:])}
}

func (c *FlareImageClient) request(prompt, model, size, quality, format string, n int, route ai.ImageRouteSelection) (flareImageRequest, error) {
	r := flareImageRequest{Model: model, Prompt: prompt, N: 1, Size: size, Quality: quality, OutputFormat: "png"}
	if !c.matches(route) || model != c.cfg.Model || n != 0 && n != 1 || format != "" || strings.TrimSpace(prompt) == "" || !utf8.ValidString(prompt) || len(prompt) > c.cfg.Estimate.MaximumPromptBytes || utf8.RuneCountInString(prompt) > 32000 {
		return r, flarePreflight(r)
	}
	if r.Size == "" {
		r.Size = c.cfg.Estimate.Size
	}
	if r.Quality == "" {
		r.Quality = c.cfg.Estimate.Quality
	}
	if (r.Size != c.cfg.Estimate.Size && (r.Size != "auto" || !c.cfg.Estimate.AllowAutoSize)) || r.Quality != c.cfg.Estimate.Quality {
		return r, flarePreflight(r)
	}
	return r, nil
}

func (c *FlareImageClient) GenerateImageWithRoute(ctx context.Context, input *ai.ImageGenerateRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	if input == nil {
		return nil, flarePreflight(nil)
	}
	r, err := c.request(input.Prompt, input.Model, input.Size, input.Quality, input.ResponseFormat, input.N, route)
	if err != nil {
		return nil, err
	}
	return c.send(ctx, "/images/generations", r)
}

func (c *FlareImageClient) EditImageWithRoute(ctx context.Context, input *ai.ImageEditRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	if input == nil {
		return nil, flarePreflight(nil)
	}
	r, err := c.request(input.Prompt, input.Model, input.Size, input.Quality, input.ResponseFormat, input.N, route)
	if err != nil {
		return nil, err
	}
	if input.ImageURL != "" || len(input.ImageURLs) != 0 || len(input.Mask) != 0 || input.MaxRetries != nil && *input.MaxRetries != 0 {
		return nil, flarePreflight(r)
	}
	refs := append([]ai.ImageInlineReference{{Bytes: input.Image, MediaType: input.ImageContentType}}, input.ReferenceImages...)
	if len(refs) > c.cfg.Estimate.MaximumReferenceImages {
		return nil, flarePreflight(r)
	}
	var totalBytes, pixels int64
	for _, ref := range refs {
		totalBytes += int64(len(ref.Bytes))
		if len(ref.Bytes) == 0 || totalBytes > c.cfg.Estimate.MaximumReferenceBytes {
			return nil, flarePreflight(r)
		}
		mime, width, height, err := httpimage.InspectGeneratedArtifact(ref.Bytes)
		pixels += int64(width) * int64(height)
		if err != nil || len(ref.Bytes) == 0 || ref.MediaType != "" && mime != ref.MediaType || totalBytes > c.cfg.Estimate.MaximumReferenceBytes || pixels > c.cfg.Estimate.MaximumReferencePixels {
			return nil, flarePreflight(r)
		}
		r.Images = append(r.Images, flareImageInput{ImageURL: "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(ref.Bytes)})
	}
	return c.send(ctx, "/images/edits", r)
}

func (c *FlareImageClient) send(ctx context.Context, path string, input flareImageRequest) (*ai.ImageResponse, error) {
	data, err := json.Marshal(input)
	maxRequest := base64.StdEncoding.EncodedLen(int(c.cfg.Estimate.MaximumReferenceBytes)) + 6*c.cfg.Estimate.MaximumPromptBytes + (64 << 10)
	if ctx == nil || err != nil || len(data) > maxRequest {
		return nil, flarePreflight(input)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(data))
	if err != nil {
		return nil, flarePreflight(input)
	}
	// GetBody must not give net/http a replay opportunity for this side effect.
	req.GetBody = nil
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrFlareImageOutcomeUnknown
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, (48<<20)+1))
	if err != nil || len(raw) > 48<<20 || resp.StatusCode != http.StatusOK {
		return nil, ErrFlareImageOutcomeUnknown
	}
	u, err := ObserveFlareImageUsage(raw)
	if err != nil {
		return nil, ErrFlareImageOutcomeUnknown
	}
	var body struct {
		Model   *string        `json:"model"`
		Created int64          `json:"created"`
		Data    []ai.ImageData `json:"data"`
	}
	violations, err := strict.UnmarshalStrict(raw, &body, strict.DisallowDuplicateFields)
	if err != nil || len(violations) != 0 || body.Model != nil && *body.Model != c.cfg.Model {
		return nil, ErrFlareImageOutcomeUnknown
	}
	sum := sha256.Sum256(raw)
	digest := hex.EncodeToString(sum[:])
	requestID := resp.Header.Get("X-Request-Id")
	if !safeFlareID(requestID) {
		requestID = ""
	}
	receipt := requestID
	if receipt == "" {
		receipt = "sha256:" + digest
	}
	observation := FlareImageObservation{Usage: u, ResponseID: receipt, RequestID: requestID, ResultDigest: digest}
	if len(body.Data) == 1 && len(body.Data[0].B64JSON) > 0 && len(body.Data[0].B64JSON) <= base64.StdEncoding.EncodedLen(32<<20) {
		observation.ImageBytes, err = base64.StdEncoding.DecodeString(body.Data[0].B64JSON)
		if err == nil {
			_, _, _, err = httpimage.InspectGeneratedArtifact(observation.ImageBytes)
		}
	} else {
		err = errors.New("invalid image result")
	}
	if err != nil {
		observation.ImageBytes = nil
		observation.ResultUnavailable = "invalid_result"
	}
	final, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	if c.observe(final, observation) != nil {
		return nil, ErrFlareImageOutcomeUnknown
	}
	if observation.ResultUnavailable != "" {
		return nil, ErrFlareImageOutcomeUnknown
	}
	// The original artifact port permits only a bounded key identifier. Keep
	// vendor identity in the generation proof; use its existing response digest
	// as opaque artifact provenance, never a fabricated vendor request ID.
	return &ai.ImageResponse{Created: body.Created, RequestID: requestID, UpstreamJobID: digest, UsageKnown: true,
		Usage: ai.Usage{PromptTokens: int(u.InputTokens), CompletionTokens: int(u.OutputTokens), TotalTokens: int(u.TotalTokens)}, Data: []ai.ImageData{{B64JSON: body.Data[0].B64JSON}}}, nil
}

func safeFlareID(s string) bool {
	if s == "" || len(s) > 192 {
		return false
	}
	for _, r := range s {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}
