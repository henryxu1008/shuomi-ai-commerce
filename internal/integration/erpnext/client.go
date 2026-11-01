package erpnext

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	f "task-processor/internal/fulfillment"
	"time"
)

type Config struct{ Origin, Organization, APIKey, APISecret string }
type Client struct {
	config Config
	http   *http.Client
}

func New(config Config, source *http.Client) (*Client, error) {
	origin, err := url.Parse(config.Origin)
	if err != nil || origin.Scheme != "https" || origin.Host == "" || origin.User != nil || origin.Path != "" || origin.RawQuery != "" || origin.Fragment != "" || origin.Opaque != "" || config.Organization == "" || config.APIKey == "" || config.APISecret == "" || strings.ContainsAny(config.APIKey+config.APISecret, "\r\n:") {
		return nil, f.ErrUnavailable
	}
	client := http.Client{Timeout: 10 * time.Second}
	if source != nil {
		client = *source
	}
	if client.Timeout <= 0 || client.Timeout > 10*time.Second {
		client.Timeout = 10 * time.Second
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{config, &client}, nil
}
func (c *Client) Execute(ctx context.Context, command f.Command) (json.RawMessage, error) {
	if c == nil {
		return nil, f.ErrUnavailable
	}
	if command.Organization != c.config.Organization {
		return nil, f.ErrForbidden
	}
	if f.Permission(command.Command) == "" || !json.Valid(command.Payload) {
		return nil, f.ErrInvalid
	}
	raw, err := json.Marshal(struct {
		Request f.Command `json:"request"`
	}{command})
	if err != nil || len(raw) > 256*1024 {
		return nil, f.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.Origin+"/api/method/commerce_fulfillment.api.execute", bytes.NewReader(raw))
	if err != nil {
		return nil, f.ErrInvalid
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "token "+c.config.APIKey+":"+c.config.APISecret)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, f.ErrUnknown
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024+1))
	if err != nil || len(data) > 2*1024*1024 {
		return nil, f.ErrUnknown
	}
	if response.StatusCode != http.StatusOK {
		// Even Frappe 417/UniqueValidationError does not prove the original key had no effect.
		return nil, f.ErrUnknown
	}
	var envelope struct {
		Message json.RawMessage `json:"message"`
	}
	if json.Unmarshal(data, &envelope) != nil || len(envelope.Message) == 0 || string(envelope.Message) == "null" || !json.Valid(envelope.Message) {
		return nil, f.ErrUnknown
	}
	return envelope.Message, nil
}
