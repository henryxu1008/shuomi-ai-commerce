package grsai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sync/semaphore"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"task-processor/internal/ai"
	"time"
)

type Config struct {
	APIKey     string
	Model      string
	SubmitURL  string
	Timeout    time.Duration
	HTTPClient *http.Client
	// MaxReferenceMaterializedBytes bounds estimated base64 bytes across edits.
	MaxReferenceMaterializedBytes int64
}
type Client struct {
	cfg                      Config
	httpClient               *http.Client
	referenceMaterialization *referenceMaterializationBudget
}

const (
	maxImageReferenceBytes            int64 = 32 << 20
	maxMaterializedReferenceBytes     int64 = (maxImageReferenceBytes*4 + 2) / 3
	defaultReferenceMaterializedBytes int64 = 512 << 20
	referenceBudgetUnitBytes          int64 = 1 << 20
)

type referenceMaterializationBudget struct {
	bytes     *semaphore.Weighted
	byteUnits int64
}

func newReferenceMaterializationBudget(maxBytes int64) *referenceMaterializationBudget {
	if maxBytes < maxMaterializedReferenceBytes {
		maxBytes = maxMaterializedReferenceBytes
	}
	units := int((maxBytes + referenceBudgetUnitBytes - 1) / referenceBudgetUnitBytes)
	return &referenceMaterializationBudget{
		bytes:     semaphore.NewWeighted(int64(units)),
		byteUnits: int64(units),
	}
}
func (b *referenceMaterializationBudget) acquire(ctx context.Context, bytes int64) (func(), error) {
	if b == nil {
		return func() {}, nil
	}
	units := int((bytes + referenceBudgetUnitBytes - 1) / referenceBudgetUnitBytes)
	if units <= 0 || int64(units) > b.byteUnits {
		return nil, fmt.Errorf("grsai reference materialization budget is too small")
	}
	if err := b.bytes.Acquire(ctx, int64(units)); err != nil {
		return nil, err
	}
	var once sync.Once
	return func() {
		once.Do(func() {
			b.bytes.Release(int64(units))
		})
	}, nil
}

type submitRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Images      []string `json:"images,omitempty"`
	AspectRatio string   `json:"aspectRatio,omitempty"`
	Quality     string   `json:"quality,omitempty"`
	ReplyType   string   `json:"replyType,omitempty"`
}
type submitResponse struct {
	ID      string       `json:"id"`
	Status  string       `json:"status"`
	Results []resultItem `json:"results"`
	Error   string       `json:"error"`
}
type resultPayload struct {
	ID        string
	Results   []resultItem
	Status    string
	RequestID string
}
type resultItem struct {
	URL     string `json:"url"`
	Content string `json:"content"`
}

func NewClient(cfg Config) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 300 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	if cfg.MaxReferenceMaterializedBytes <= 0 {
		cfg.MaxReferenceMaterializedBytes = defaultReferenceMaterializedBytes
	}
	return &Client{cfg: cfg, httpClient: client, referenceMaterialization: newReferenceMaterializationBudget(cfg.MaxReferenceMaterializedBytes)}
}

// exactImageInput never resolves a mutable source URL.
func exactImageInput(req *ai.ImageEditRequest) (string, error) {
	contentType := strings.TrimSpace(req.ImageContentType)
	if contentType == "" {
		contentType = http.DetectContentType(req.Image)
	}
	if !strings.HasPrefix(strings.ToLower(contentType), "image/") {
		return "", errors.New("invalid image data")
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(req.Image), nil
}
func (c *Client) submitGenerationRequestNoPoll(ctx context.Context, submitURL string, req submitRequest) (*resultPayload, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal image generation request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("build image generation request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	client := c.httpClient
	{
		httpReq.GetBody = nil
		local := *client
		local.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
		client = &local
	}
	if strings.TrimSpace(c.cfg.APIKey) != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("submit image generation request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("submit image generation request returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	const unifiedResponseLimit = 1 << 20
	reader := io.LimitReader(resp.Body, unifiedResponseLimit+1)
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("read image generation response: %w", err)
	}
	if len(body) > unifiedResponseLimit {
		return nil, errors.New("image generation response exceeds limit")
	}
	requestID := strings.TrimSpace(resp.Header.Get("X-Request-Id"))
	{
		var result submitResponse
		if err := json.Unmarshal(body, &result); err != nil || result.Status != "succeeded" || result.ID == "" || result.Error != "" {
			return nil, errors.New("synchronous image generation outcome unknown")
		}
		return &resultPayload{ID: result.ID, Status: result.Status, Results: result.Results, RequestID: requestID}, nil
	}
}
func buildSubmitURL(configuredURL string, model string) (string, error) {
	return buildGRSAIURL(configuredURL, "/v1/api/generate")
}
func buildGRSAIURL(configuredURL string, path string) (string, error) {
	trimmed := strings.TrimSpace(configuredURL)
	if trimmed == "" {
		return "", fmt.Errorf("grsai submit url cannot be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse grsai submit url: %w", err)
	}
	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}
