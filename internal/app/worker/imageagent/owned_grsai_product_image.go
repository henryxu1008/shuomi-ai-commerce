package imageagentworker

import (
	"context"
	"errors"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	credential "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/channelcredential"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/grsai"
	openai "github.com/henryxu1008/shuomi-ai-commerce/internal/integration/openai"
)

func (f generationProviderFactory) ownedProductImageOptions(ctx context.Context, org string) ([]d.OwnedProductImageOption, error) {
	if !f.price.OwnedGRSaiProductImage || !f.price.Configured() {
		return nil, nil
	}
	reader, ok := f.current.(d.OwnedProductImageReader)
	if !ok {
		return nil, nil
	}
	return reader.ReadOwnedProductImages(ctx, org)
}

func (f generationProviderFactory) resolveOwnedProductImage(ctx context.Context, selection imageagent.ImageRouteSelection, send bool) (*openai.ClientConfig, imageagent.GenerationProviderMetadata, error) {
	reject := func() (*openai.ClientConfig, imageagent.GenerationProviderMetadata, error) {
		return nil, imageagent.GenerationProviderMetadata{}, imageagent.ErrImageRouteChanged
	}
	identity, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.TenantID == "" || identity.EffectiveOrganizationID != identity.TenantID || !selection.Valid() {
		return reject()
	}
	options, err := f.ownedProductImageOptions(ctx, identity.TenantID)
	if err != nil {
		return reject()
	}
	for _, option := range options {
		if option.BindingID != selection.BindingID || option.RevisionID != selection.RevisionID {
			continue
		}
		config := &openai.ClientConfig{Model: aicapability.GRSaiProductImageModel, APIStyle: "grsai", BaseURL: aicapability.GRSaiProductImageBaseURL, Timeout: 25 * time.Second}
		meta := ownedProductImageMetadata(option)
		if !send {
			return config, meta, nil
		}
		if f.credentials == nil {
			return reject()
		}
		secret, err := f.credentials.Resolve(ctx, identity.TenantID, option.ConnectionID, option.CredentialVersion)
		m := secret.Metadata
		if err != nil || secret.APIKey == "" || !m.Enabled || m.OrganizationID != identity.TenantID ||
			m.ConnectionID != option.ConnectionID || m.Reference != option.CredentialVersion || m.EndpointDigest != option.EndpointDigest ||
			m.BaseURL != config.BaseURL || m.Provider != "grsai" || m.Channel != credential.GRSaiImageChannel {
			return reject()
		}
		config.APIKey = secret.APIKey
		return config, meta, nil
	}
	return reject()
}

func ownedProductImageMetadata(option d.OwnedProductImageOption) imageagent.GenerationProviderMetadata {
	return imageagent.GenerationProviderMetadata{Provider: "grsai", Model: aicapability.GRSaiProductImageModel,
		Protocol: aicapability.GRSaiProductImageProtocol, RouteReference: option.BindingID,
		CredentialReference: option.CredentialVersion, ConfigurationVersion: option.RevisionID, PromptVersion: imageagent.ImageSetSchema,
		ConnectionID: option.ConnectionID, BillingMode: aicapability.ModelUsageOnly}
}

type ownedProductImageTasks struct{ *grsai.Client }

func (p ownedProductImageTasks) StartProductImage(ctx context.Context, prompt string, source []byte) (string, error) {
	id, err := p.Client.StartProductImage(ctx, prompt, source)
	if errors.Is(err, grsai.ErrProductImagePreflight) {
		return "", &imageagent.ProviderDispatchError{State: imageagent.ProviderRejectedBeforeEffect, Err: err}
	}
	return id, err
}

func (p ownedProductImageTasks) LookupProductImage(ctx context.Context, id string) (imageagent.ProviderUsageImageResult, error) {
	result, err := p.Client.ReadProductImageResult(ctx, id)
	return imageagent.ProviderUsageImageResult{TaskID: result.TaskID, Status: string(result.State), ResultURL: result.ResultURL, Digest: result.ResultDigest}, err
}
