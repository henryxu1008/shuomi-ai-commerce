package grsai

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/ai"
)

const (
	ProductImageModel          = "gpt-image-2.6"
	ProductImageBaseURL        = "https://grsaiapi.com"
	ProductImageMaxSourceBytes = 3 << 20

	maxProductImageReplyBytes = 1 << 20
)

type ProductImageState string

const (
	ProductImageRunning   ProductImageState = "running"
	ProductImageSucceeded ProductImageState = "succeeded"
	ProductImageFailed    ProductImageState = "failed"
	ProductImageViolation ProductImageState = "violation"
)

// ProductImageResult is an observation of the exact supplier task. The API
// does not return billing credits on this read. ResultURL is worker-only.
type ProductImageResult struct {
	TaskID       string
	State        ProductImageState
	ResultURL    string
	ResultDigest string
}

var errProductImageOutcome = errors.New("grsai product image outcome is unconfirmed")
var ErrProductImagePreflight = errors.New("grsai product image request did not start")
var productImageTaskID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

// StartProductImage sends a single POST. The caller must persist the original
// dispatch claim first and durably record the returned ID before any lookup.
func (c *Client) StartProductImage(ctx context.Context, prompt string, source []byte) (string, error) {
	if !c.productImageConfigured() || ctx == nil || strings.TrimSpace(prompt) == "" || len(prompt) > 16<<10 || !utf8.ValidString(prompt) || len(source) == 0 || len(source) > ProductImageMaxSourceBytes {
		return "", ErrProductImagePreflight
	}
	media := http.DetectContentType(source)
	if media != "image/jpeg" && media != "image/png" {
		return "", ErrProductImagePreflight
	}
	release, err := c.referenceMaterialization.acquire(ctx, int64(base64EncodedSize(len(source))))
	if err != nil {
		return "", ErrProductImagePreflight
	}
	defer release()
	image, err := exactImageInput(&ai.ImageEditRequest{Image: source, ImageContentType: media})
	if err != nil {
		return "", ErrProductImagePreflight
	}
	body, err := json.Marshal(submitRequest{
		Model: ProductImageModel, Prompt: prompt, Images: []string{image},
		AspectRatio: "1024x1024", Quality: "auto", ReplyType: "async",
	})
	if err != nil {
		return "", ErrProductImagePreflight
	}
	ctx, cancel := context.WithTimeout(ctx, min(c.cfg.Timeout, 5*time.Minute))
	defer cancel()
	if ctx.Err() != nil {
		return "", ErrProductImagePreflight
	}
	response, err := c.readProductImageResponse(ctx, http.MethodPost, ProductImageBaseURL+"/v1/api/generate", bytes.NewReader(body))
	if err != nil || !productImageTaskID.MatchString(response.ID) || response.Status != string(ProductImageRunning) || len(response.Results) != 0 || response.Error != "" {
		return "", errProductImageOutcome
	}
	return response.ID, nil
}

// ReadProductImageResult only reads the known supplier ID. The application
// supplies the original still-valid credential and owns all polling limits.
func (c *Client) ReadProductImageResult(ctx context.Context, taskID string) (ProductImageResult, error) {
	if !c.productImageConfigured() || ctx == nil || !productImageTaskID.MatchString(taskID) {
		return ProductImageResult{}, errors.New("grsai product image lookup is unavailable")
	}
	ctx, cancel := context.WithTimeout(ctx, min(c.cfg.Timeout, 5*time.Second))
	defer cancel()
	query := url.Values{"id": {taskID}}
	response, err := c.readProductImageResponse(ctx, http.MethodGet, ProductImageBaseURL+"/v1/api/result?"+query.Encode(), nil)
	if err != nil || response.ID != taskID {
		return ProductImageResult{}, errProductImageOutcome
	}
	result := ProductImageResult{TaskID: taskID, State: ProductImageState(response.Status)}
	switch result.State {
	case ProductImageRunning, ProductImageFailed, ProductImageViolation:
		if len(response.Results) != 0 {
			return ProductImageResult{}, errProductImageOutcome
		}
	case ProductImageSucceeded:
		if response.Error != "" || len(response.Results) != 1 {
			return ProductImageResult{}, errProductImageOutcome
		}
		result.ResultURL = response.Results[0].URL
	default:
		return ProductImageResult{}, errProductImageOutcome
	}
	evidence, err := json.Marshal(result)
	if err != nil {
		return ProductImageResult{}, errProductImageOutcome
	}
	digest := sha256.Sum256(evidence)
	result.ResultDigest = hex.EncodeToString(digest[:])
	return result, nil
}

func (c *Client) productImageConfigured() bool {
	return c != nil && c.cfg.SubmitURL == ProductImageBaseURL && c.cfg.Model == ProductImageModel && strings.TrimSpace(c.cfg.APIKey) != ""
}

func (c *Client) readProductImageResponse(ctx context.Context, method, endpoint string, body io.Reader) (submitResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return submitResponse{}, errProductImageOutcome
	}
	req.GetBody = nil
	req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	client := *c.httpClient
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := client.Do(req)
	if err != nil {
		return submitResponse{}, errProductImageOutcome
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return submitResponse{}, errProductImageOutcome
	}
	encoded, err := io.ReadAll(io.LimitReader(response.Body, maxProductImageReplyBytes+1))
	var result submitResponse
	if err != nil || len(encoded) > maxProductImageReplyBytes || json.Unmarshal(encoded, &result) != nil {
		return submitResponse{}, errProductImageOutcome
	}
	return result, nil
}
