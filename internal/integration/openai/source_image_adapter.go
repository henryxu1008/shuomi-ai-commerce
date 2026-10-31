package openai

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"task-processor/internal/ai"
	"task-processor/internal/integration/httpimage"
	productimage "task-processor/internal/product/image"
)

const maxProductImagePromptBytes = 64 << 10
const sourceWhiteBackgroundPromptVersion = "source-white-background-v1"
const sourceWhiteBackgroundPrompt = "Edit only the background of the supplied source image to clean white; preserve shape, proportions, colors, materials, patterns, text and accessory count. Do not add decorations or crop the product."

type SourceImageAdapterConfig struct {
	ImageClient              ai.RouteBoundImageEditor
	Provider                 string
	ImageModel               string
	RouteReference           string
	CredentialReference      string
	ConfigurationVersion     string
	PricingVersion           string
	ImageCostMicrosPerOutput int64
	CostUpperBoundKnown      bool
	GeneratedImageFetcher    func(context.Context, string) ([]byte, error)
	DiagnoseOutputDimensions func(expectedWidth, expectedHeight, actualWidth, actualHeight int)
}
type SourceImageAdapter struct{ config SourceImageAdapterConfig }

const SourceImagePromptVersion = "product-image-prompts-v1"

func NewSourceImageAdapter(config SourceImageAdapterConfig) (*SourceImageAdapter, error) {
	if nilInterface(config.ImageClient) {
		return nil, fmt.Errorf("source image client is required")
	}
	for name, value := range map[string]string{"provider": config.Provider, "image model": config.ImageModel, "route reference": config.RouteReference, "credential reference": config.CredentialReference, "configuration version": config.ConfigurationVersion, "pricing version": config.PricingVersion} {
		if !canonicalProductImageString(value) {
			return nil, fmt.Errorf("source image %s is required and must be canonical", name)
		}
	}
	if config.ImageCostMicrosPerOutput < 0 {
		return nil, fmt.Errorf("source image cost is invalid")
	}
	if config.GeneratedImageFetcher == nil {
		config.GeneratedImageFetcher = func(ctx context.Context, url string) ([]byte, error) {
			return httpimage.Download(ctx, httpimage.NewPublicImageHTTPClient(), url, productimage.MaxInlineArtifactBytes)
		}
	}
	return &SourceImageAdapter{config: config}, nil
}
func (a *SourceImageAdapter) EditSourceWhiteBackground(ctx context.Context, request productimage.SourceWhiteBackgroundRequest) (productimage.Candidate, error) {
	if err := a.authorize(request.Authorization, productimage.SourceWhiteBackgroundOperation, 1); err != nil {
		return productimage.Candidate{}, err
	}
	input := request.Source
	input.Bytes = append([]byte(nil), request.SourceBytes...)
	candidate, err := a.editOne(ctx, input, request.Source, request.Product, productimage.RoleWhiteBackground, productimage.SourceWhiteBackgroundOperation, sourceWhiteBackgroundPrompt)
	if err == nil {
		candidate.Metadata.PromptVersion = sourceWhiteBackgroundPromptVersion
	}
	return candidate, err
}
func (a *SourceImageAdapter) authorize(authorization *productimage.UsageQuote, operation string, maximumOutputs int64) error {
	if authorization == nil {
		return nil
	}
	quote, err := productimage.NormalizeUsageAuthorization(*authorization, operation)
	if err != nil || quote.MaximumOutputs != maximumOutputs {
		return productimage.ErrInputInvalid
	}
	provider, routeReference, credentialReference, configurationVersion := a.config.Provider, a.config.RouteReference, a.config.CredentialReference, a.config.ConfigurationVersion
	model := a.config.ImageModel

	if quote.Provider != provider || quote.RouteReference != routeReference || quote.Model != model ||
		quote.CredentialReference != credentialReference || quote.ConfigurationVersion != configurationVersion ||
		quote.PricingVersion != a.config.PricingVersion {
		return productimage.ErrCapabilityUnsupported
	}

	return nil
}
func (a *SourceImageAdapter) QuoteUsage(_ context.Context, request productimage.UsageQuoteRequest) (productimage.UsageQuote, error) {
	if !canonicalProductImageString(request.Operation) || !canonicalProductImageString(request.InputFingerprint) {
		return productimage.UsageQuote{}, productimage.ErrInputInvalid
	}
	maximumOutputs := request.MaximumOutputs
	model := a.config.ImageModel
	costPerOutput := a.config.ImageCostMicrosPerOutput
	var maximumTokens int64
	switch request.Operation {
	case productimage.SourceEditOperation:
		if !a.supportsSourceEdit() || maximumOutputs != 1 {
			return productimage.UsageQuote{}, productimage.ErrCapabilityUnsupported
		}
	case productimage.SourceWhiteBackgroundOperation:
		maximumOutputs = 1
	default:
		return productimage.UsageQuote{}, productimage.ErrCapabilityUnsupported
	}

	if costPerOutput != 0 && maximumOutputs > (1<<63-1)/costPerOutput {
		return productimage.UsageQuote{}, productimage.ErrInputInvalid
	}
	maximumCost := costPerOutput * maximumOutputs
	provider, routeReference, credentialReference, configurationVersion := a.config.Provider, a.config.RouteReference, a.config.CredentialReference, a.config.ConfigurationVersion

	fingerprintPayload := struct {
		Operation, InputFingerprint, Provider, Model, RouteReference, CredentialReference, ConfigurationVersion, PricingVersion string
		MaximumOutputs, MaximumTokens, MaximumCost                                                                              int64
		CostUpperBoundKnown                                                                                                     bool
		SourcePromptVersion                                                                                                     string `json:",omitempty"`
	}{
		request.Operation, request.InputFingerprint, provider, model, routeReference,
		credentialReference, configurationVersion, a.config.PricingVersion, maximumOutputs, maximumTokens, maximumCost, a.config.CostUpperBoundKnown, "",
	}
	if request.Operation == productimage.SourceWhiteBackgroundOperation {
		fingerprintPayload.SourcePromptVersion = sourceWhiteBackgroundPromptVersion
	}
	encoded, err := json.Marshal(fingerprintPayload)
	if err != nil {
		return productimage.UsageQuote{}, productimage.ErrInputInvalid
	}
	digest := sha256.Sum256(encoded)
	return productimage.UsageQuote{
		Operation: request.Operation, Provider: provider, RouteReference: routeReference, Model: model,
		CredentialReference: credentialReference, ConfigurationVersion: configurationVersion,
		PricingVersion: a.config.PricingVersion, Fingerprint: hex.EncodeToString(digest[:]),
		MaximumOutputs: maximumOutputs, MaximumModelCalls: 1, MaximumTokens: maximumTokens, MaximumCostMicros: maximumCost,
		CostUpperBoundKnown: a.config.CostUpperBoundKnown,
	}, nil
}
func (a *SourceImageAdapter) editOne(ctx context.Context, input, source productimage.Asset, product productimage.ProductContext, role productimage.Role, operation, basePrompt string) (productimage.Candidate, error) {
	prompt, err := productImagePrompt(basePrompt, product, struct {
		Operation string `json:"operation"`
	}{Operation: operation})
	if err != nil {
		return productimage.Candidate{}, err
	}
	request := &ai.ImageEditRequest{
		Model: a.config.ImageModel, Prompt: prompt, Image: append([]byte(nil), input.Bytes...),
		ImageURL: input.URL, ImageContentType: input.MediaType,
		ResponseFormat: "b64_json", N: 1, Size: "auto",
	}
	if a.isVIPImageRoute() {
		request.Size, request.Quality = "1024x1024", "medium"
	}
	if operation == productimage.SourceWhiteBackgroundOperation {
		if len(request.Image) == 0 {
			return productimage.Candidate{}, productimage.ErrInputInvalid
		}
		// The current source-only operation consumes the authorized exact bytes.
		// Keep source URL provenance on the candidate, never send it as an
		// alternate mutable provider input.
		request.ImageURL = ""
		zero := 0
		request.MaxRetries = &zero
	}
	response, err := a.editImage(ctx, request)
	if err != nil {
		return productimage.Candidate{}, err
	}
	candidates, err := a.imageCandidates(ctx, response, source, role, operation, 1)
	if err != nil {
		return productimage.Candidate{}, err
	}
	if len(candidates) != 1 {
		return productimage.Candidate{}, productimage.ErrOutputValidation
	}
	return candidates[0], nil
}
func (a *SourceImageAdapter) editImage(ctx context.Context, request *ai.ImageEditRequest) (*ai.ImageResponse, error) {
	client := a.config.ImageClient
	return client.EditImageWithRoute(ctx, request, ai.ImageRouteSelection{
		CredentialReference: a.config.CredentialReference, ConfigurationVersion: a.config.ConfigurationVersion,
	})
}
func (a *SourceImageAdapter) imageCandidates(ctx context.Context, response *ai.ImageResponse, source productimage.Asset, role productimage.Role, operation string, maximum int) ([]productimage.Candidate, error) {
	if response == nil || len(response.Data) == 0 || len(response.Data) > maximum {
		return nil, productimage.ErrOutputValidation
	}
	candidates := make([]productimage.Candidate, len(response.Data))
	usedArtifactBytes := 0
	for index, item := range response.Data {
		content, err := a.generatedImageBytes(ctx, item)
		if err != nil {
			return nil, err
		}
		usedArtifactBytes, err = consumeProductImageArtifactBudget(usedArtifactBytes, len(content))
		if err != nil {
			return nil, err
		}
		mediaType, width, height, err := httpimage.InspectGeneratedArtifact(content)
		if err != nil {
			return nil, err
		}
		if a.isVIPImageRoute() && (width != 1024 || height != 1024) {
			if a.config.DiagnoseOutputDimensions != nil {
				a.config.DiagnoseOutputDimensions(1024, 1024, width, height)
			}
			return nil, productimage.ErrOutputValidation
		}
		candidates[index] = productimage.Candidate{
			Asset: productimage.Asset{
				Bytes: content, MediaType: mediaType, SourceURL: source.SourceURL, SourceAssetID: source.SourceAssetID,
				Role: role, Width: width, Height: height, Operations: []string{operation},
			},
			Metadata: productimage.GenerationMetadata{
				Capability: operation, ModelFamily: a.config.ImageModel, InvocationID: response.RequestID,
				PromptReference: operation, PromptVersion: SourceImagePromptVersion,
				Values: map[string]string{"provider": a.config.Provider, "configuration_version": a.config.ConfigurationVersion},
			},
		}
	}
	return candidates, nil
}
func consumeProductImageArtifactBudget(used, size int) (int, error) {
	if used < 0 || size <= 0 || size > productimage.MaxInlineArtifactBytes || size > productimage.MaxInlineArtifactAggregateBytes-used {
		return 0, productimage.ErrOutputValidation
	}
	return used + size, nil
}
func (a *SourceImageAdapter) generatedImageBytes(ctx context.Context, item ai.ImageData) ([]byte, error) {
	encoded := strings.TrimSpace(item.B64JSON)
	if encoded != "" {
		if base64.StdEncoding.DecodedLen(len(encoded)) > productimage.MaxInlineArtifactBytes {
			return nil, productimage.ErrOutputValidation
		}
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(decoded) == 0 || len(decoded) > productimage.MaxInlineArtifactBytes {
			return nil, productimage.ErrOutputValidation
		}
		return decoded, nil
	}
	if item.URL == "" || strings.TrimSpace(item.URL) != item.URL {
		return nil, productimage.ErrOutputValidation
	}
	downloaded, err := a.config.GeneratedImageFetcher(ctx, item.URL)
	if err != nil {
		return nil, err
	}
	if len(downloaded) == 0 || len(downloaded) > productimage.MaxInlineArtifactBytes {
		return nil, productimage.ErrOutputValidation
	}
	return append([]byte(nil), downloaded...), nil
}
func productImagePrompt(base string, product productimage.ProductContext, details any) (string, error) {
	payload, err := json.Marshal(struct {
		Product productImagePromptContext `json:"product"`
		Details any                       `json:"details"`
	}{Product: newProductImagePromptContext(product), Details: details})
	if err != nil {
		return "", productimage.ErrInputInvalid
	}
	prompt := base + "\nControlled context: " + string(payload)
	if len(prompt) > maxProductImagePromptBytes {
		return "", productimage.ErrInputInvalid
	}
	return prompt, nil
}

type productImagePromptContext struct {
	ProductKey  string            `json:"product_key"`
	Title       string            `json:"title,omitempty"`
	ProductType string            `json:"product_type,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

func newProductImagePromptContext(product productimage.ProductContext) productImagePromptContext {
	return productImagePromptContext{
		ProductKey: product.ProductKey, Title: product.Title, ProductType: product.ProductType, Attributes: product.Attributes,
	}
}
func canonicalProductImageString(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && len(value) <= maxProductImagePromptBytes
}
func nilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}

var _ productimage.SourceWhiteBackground = (*SourceImageAdapter)(nil)
var _ productimage.SourceEditor = (*SourceImageAdapter)(nil)
var _ productimage.UsageQuoter = (*SourceImageAdapter)(nil)
