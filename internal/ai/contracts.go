// Package ai contains provider-neutral contracts shared by AI adapters and
// runtime orchestration. Provider-specific configuration and wire payloads
// stay in their respective infrastructure packages.
package ai

import (
	"context"
)

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ImageGenerateRequest struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt"`
	Size           string `json:"size,omitempty"`
	Quality        string `json:"quality,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	N              int    `json:"n,omitempty"`
}

type ImageEditRequest struct {
	// MaxRetries overrides retries for OpenAI multipart and GRSAI edit transports.
	// Explicit zero also forbids redirects that could replay the edit POST.
	// Providers unable to enforce this policy must reject it before dispatch.
	MaxRetries       *int `json:"-"`
	Model            string
	Prompt           string
	Image            []byte
	ImageContentType string
	// ReferenceImages are exact additional source bytes in their approved order.
	// A transport must explicitly support this field or reject before dispatch.
	ReferenceImages []ImageInlineReference `json:"-"`
	ImageURL        string
	ImageURLs       []string
	Mask            []byte
	Size            string
	Quality         string
	ResponseFormat  string
	N               int
}

type ImageInlineReference struct {
	Bytes     []byte `json:"-"`
	MediaType string
}

type ImageData struct {
	B64JSON       string `json:"b64_json,omitempty"`
	URL           string `json:"url,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageResponse struct {
	Created int64       `json:"created"`
	Data    []ImageData `json:"data"`
	Usage   Usage       `json:"usage"`
	// UsageKnown is set only by the image transport after validating explicit
	// input/output/total token counters. False is unknown, never zero usage.
	UsageKnown    bool   `json:"-"`
	RequestID     string `json:"request_id,omitempty"`
	UpstreamJobID string `json:"upstream_job_id,omitempty"`
	RawResponse   string `json:"raw_response,omitempty"`
}

// ImageRouteSelection binds an image call to an already resolved credential
// configuration. It contains no provider-specific client or configuration
// representation.
type ImageRouteSelection struct {
	CredentialReference  string
	ConfigurationVersion string
}

// RouteBoundImageEditor edits only on the exact route resolved before dispatch.
type RouteBoundImageEditor interface {
	EditImageWithRoute(context.Context, *ImageEditRequest, ImageRouteSelection) (*ImageResponse, error)
}
