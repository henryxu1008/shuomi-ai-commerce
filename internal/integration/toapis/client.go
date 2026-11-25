package toapis

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/httpimage"
)

const generations = aicapability.ToAPIsProductImageBaseURL + "/v1/images/generations"
const responseLimit = 1 << 20

var ErrPreflight = errors.New("ToAPIs image request did not start")
var errUnconfirmed = errors.New("ToAPIs image task outcome is unconfirmed")

type Client struct {
	key  string
	http http.Client
}

func NewClient(key string, client *http.Client) *Client {
	if client == nil {
		client = httpimage.NewPublicImageHTTPClient()
	}
	bounded := *client
	bounded.Jar = nil
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{key: key, http: bounded}
}

type Result struct {
	TaskID, Status, URL, Digest string
}

type taskResponse struct {
	ID, Status, Model string
	Result            struct {
		Type string
		Data []struct{ URL string }
	}
}

func (c *Client) Start(ctx context.Context, prompt, reference string) (string, error) {
	if c.key == "" || ctx.Err() != nil || strings.TrimSpace(prompt) == "" || len(prompt) > 16<<10 ||
		!utf8.ValidString(prompt) || !aicapability.ToAPIsProductImageSource(reference) {
		return "", ErrPreflight
	}
	body, err := json.Marshal(struct {
		Model        string   `json:"model"`
		Prompt       string   `json:"prompt"`
		N            int      `json:"n"`
		Quality      string   `json:"quality"`
		Size         string   `json:"size"`
		Resolution   string   `json:"resolution"`
		OutputFormat string   `json:"output_format"`
		References   []string `json:"reference_images"`
	}{Model: aicapability.ToAPIsProductImageModel, Prompt: prompt, N: 1, Quality: "high",
		Size: "1:1", Resolution: "1K", OutputFormat: "png", References: []string{reference}})
	if err != nil {
		return "", ErrPreflight
	}
	ctx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	task, err := c.request(ctx, http.MethodPost, generations, bytes.NewReader(body))
	if err != nil || !aicapability.ValidProductImageTaskID(task.ID) || task.Model != aicapability.ToAPIsProductImageModel {
		return "", errUnconfirmed
	}
	return task.ID, nil
}

func (c *Client) Lookup(ctx context.Context, id string) (Result, error) {
	if c.key == "" || !aicapability.ValidProductImageTaskID(id) {
		return Result{}, errUnconfirmed
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	task, err := c.request(ctx, http.MethodGet, generations+"/"+id, nil)
	if err != nil || task.ID != id || task.Model != "" && task.Model != aicapability.ToAPIsProductImageModel {
		return Result{}, errUnconfirmed
	}
	result := Result{TaskID: id}
	switch task.Status {
	case "pending", "queued", "in_progress":
		if len(task.Result.Data) != 0 {
			return Result{}, errUnconfirmed
		}
		result.Status = "running"
	case "failed":
		if len(task.Result.Data) != 0 {
			return Result{}, errUnconfirmed
		}
		result.Status = "failed"
	case "completed":
		if task.Result.Type != "image" || len(task.Result.Data) != 1 {
			return Result{}, errUnconfirmed
		}
		result.Status, result.URL = "succeeded", task.Result.Data[0].URL
	default:
		return Result{}, errUnconfirmed
	}
	evidence, err := json.Marshal(result)
	if err != nil {
		return Result{}, errUnconfirmed
	}
	digest := sha256.Sum256(evidence)
	result.Digest = hex.EncodeToString(digest[:])
	return result, nil
}

func (c *Client) request(ctx context.Context, method, endpoint string, body io.Reader) (taskResponse, error) {
	var task taskResponse
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return task, errUnconfirmed
	}
	// A lost acknowledgement must never cause an HTTP replay.
	req.GetBody = nil
	req.Header.Set("Authorization", "Bearer "+c.key)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	reply, err := c.http.Do(req)
	if err != nil {
		return task, errUnconfirmed
	}
	defer reply.Body.Close()
	if reply.StatusCode < 200 || reply.StatusCode >= 300 {
		return task, errUnconfirmed
	}
	data, err := io.ReadAll(io.LimitReader(reply.Body, responseLimit+1))
	if err != nil || len(data) > responseLimit || json.Unmarshal(data, &task) != nil {
		return taskResponse{}, errUnconfirmed
	}
	return task, nil
}
