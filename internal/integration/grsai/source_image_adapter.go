package grsai

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"task-processor/internal/ai"
	productimage "task-processor/internal/product/image"
)

// SourceImageProvider is the complete provider-neutral capability surface a
// GRSAI-compatible delegate must implement.
type SourceImageProvider interface {
	productimage.SourceWhiteBackground
	productimage.SourceEditor
	productimage.UsageQuoter
}

type SourceImageAdapterFactory func(ai.RouteBoundImageGenerator) (SourceImageProvider, error)

type SourceImageAdapterConfig struct {
	Client               ai.ImageGenerator
	ImageModel           string
	RouteReference       string
	CredentialReference  string
	ConfigurationVersion string
	Build                SourceImageAdapterFactory
}

// NewSourceImageAdapter pins the concrete GRSAI client before handing it to
// an App-selected compatible capability adapter. The Integration packages do
// not depend on one another; App is the only concrete composition boundary.
func NewSourceImageAdapter(config SourceImageAdapterConfig) (SourceImageProvider, error) {
	if nilGRSAISourceImageClient(config.Client) {
		return nil, fmt.Errorf("grsai product image client is required")
	}
	if config.Build == nil {
		return nil, fmt.Errorf("grsai product image adapter factory is required")
	}
	for name, value := range map[string]string{
		"model": config.ImageModel, "route reference": config.RouteReference,
		"credential reference": config.CredentialReference, "configuration version": config.ConfigurationVersion,
	} {
		if value == "" || strings.TrimSpace(value) != value {
			return nil, fmt.Errorf("grsai product image %s is required and must be canonical", name)
		}
	}
	pinned, err := newPinnedProductImageClient(
		config.Client, config.ImageModel, config.CredentialReference, config.ConfigurationVersion,
	)
	if err != nil {
		return nil, err
	}
	delegate, err := config.Build(pinned)
	if err != nil {
		return nil, fmt.Errorf("build grsai product image delegate: %w", err)
	}
	if nilGRSAISourceImageProvider(delegate) {
		return nil, fmt.Errorf("grsai product image delegate is required")
	}
	return &sourceImageAdapter{
		delegate: delegate, imageModel: config.ImageModel, routeReference: config.RouteReference,
		credentialReference: config.CredentialReference, configurationVersion: config.ConfigurationVersion,
	}, nil
}

type sourceImageAdapter struct {
	delegate             SourceImageProvider
	imageModel           string
	routeReference       string
	credentialReference  string
	configurationVersion string
}

func (a *sourceImageAdapter) EditSourceWhiteBackground(ctx context.Context, request productimage.SourceWhiteBackgroundRequest) (productimage.Candidate, error) {
	return a.delegate.EditSourceWhiteBackground(ctx, request)
}

func (a *sourceImageAdapter) EditSources(ctx context.Context, request productimage.SourceEditRequest) (productimage.Candidate, error) {
	delegate, ok := a.delegate.(productimage.SourceEditor)
	if !ok {
		return productimage.Candidate{}, productimage.ErrCapabilityUnsupported
	}
	return delegate.EditSources(ctx, request)
}

func (a *sourceImageAdapter) QuoteUsage(ctx context.Context, request productimage.UsageQuoteRequest) (productimage.UsageQuote, error) {
	if a == nil || nilGRSAISourceImageProvider(a.delegate) {
		return productimage.UsageQuote{}, productimage.ErrCapabilityUnsupported
	}
	quote, err := a.delegate.QuoteUsage(ctx, request)
	if err != nil {
		return productimage.UsageQuote{}, err
	}
	if quote.Operation != request.Operation || quote.RouteReference != a.routeReference ||
		quote.CredentialReference != a.credentialReference || quote.ConfigurationVersion != a.configurationVersion {
		return productimage.UsageQuote{}, productimage.ErrCapabilityUnsupported
	}
	switch request.Operation {
	case productimage.SourceWhiteBackgroundOperation, productimage.SourceEditOperation:
		if quote.Model != a.imageModel {
			return productimage.UsageQuote{}, productimage.ErrCapabilityUnsupported
		}
	}
	return quote, nil
}

type pinnedProductImageClient struct {
	client               ai.ImageGenerator
	model                string
	credentialReference  string
	configurationVersion string
}

func newPinnedProductImageClient(client ai.ImageGenerator, model, credentialReference, configurationVersion string) (*pinnedProductImageClient, error) {
	if nilGRSAISourceImageClient(client) {
		return nil, fmt.Errorf("grsai product image client is required")
	}
	for name, value := range map[string]string{
		"model": model, "credential reference": credentialReference, "configuration version": configurationVersion,
	} {
		if value == "" || strings.TrimSpace(value) != value {
			return nil, fmt.Errorf("grsai product image %s is required and must be canonical", name)
		}
	}
	if defaultModel := strings.TrimSpace(client.GetDefaultModel()); defaultModel != "" && defaultModel != model {
		return nil, fmt.Errorf("grsai product image model does not match the configured client")
	}
	return &pinnedProductImageClient{
		client: client, model: model, credentialReference: credentialReference, configurationVersion: configurationVersion,
	}, nil
}

func (c *pinnedProductImageClient) EditImageWithRoute(ctx context.Context, request *ai.ImageEditRequest, route ai.ImageRouteSelection) (*ai.ImageResponse, error) {
	if c == nil || request == nil || strings.TrimSpace(request.Model) != c.model ||
		route.CredentialReference != c.credentialReference || route.ConfigurationVersion != c.configurationVersion {
		return nil, fmt.Errorf("grsai product image route changed before dispatch")
	}
	return c.client.EditImage(ctx, request)
}

func (*pinnedProductImageClient) GenerateImage(context.Context, *ai.ImageGenerateRequest) (*ai.ImageResponse, error) {
	return nil, fmt.Errorf("grsai product image generation requires an exact route")
}

func (*pinnedProductImageClient) EditImage(context.Context, *ai.ImageEditRequest) (*ai.ImageResponse, error) {
	return nil, fmt.Errorf("grsai product image edit requires an exact route")
}

func (c *pinnedProductImageClient) GetDefaultModel() string {
	if c == nil {
		return ""
	}
	return c.model
}

func (*pinnedProductImageClient) SupportsAsyncImageGeneration() bool { return false }

func (*pinnedProductImageClient) SubmitImageGeneration(context.Context, *ai.ImageGenerateRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}

func (*pinnedProductImageClient) SubmitImageEdit(context.Context, *ai.ImageEditRequest) (*ai.ImageAsyncSubmitResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}

func (*pinnedProductImageClient) QueryImageGeneration(context.Context, string) (*ai.ImageAsyncQueryResponse, error) {
	return nil, ai.ErrAsyncImageGenerationNotSupported
}

func nilGRSAISourceImageClient(client ai.ImageGenerator) bool {
	return nilGRSAISourceImageValue(client)
}

func nilGRSAISourceImageProvider(provider SourceImageProvider) bool {
	return nilGRSAISourceImageValue(provider)
}

func nilGRSAISourceImageValue(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

var _ ai.RouteBoundImageGenerator = (*pinnedProductImageClient)(nil)
var _ SourceImageProvider = (*sourceImageAdapter)(nil)
