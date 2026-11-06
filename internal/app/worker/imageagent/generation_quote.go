package imageagentworker

import (
	"context"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	"task-processor/internal/shared/aiidentity"
)

type imageGenerationQuoteReader struct{ factory generationProviderFactory }

func NewImageGenerationQuoteReader(price config.ImageAgentGenerationConfig, current d.SharedCategoryReader) (imageagent.ImageSetQuoteReader, error) {
	if current == nil || !price.Configured() {
		return nil, imageagent.ErrBudgetQuoteUnavailable
	}
	return imageGenerationQuoteReader{factory: generationProviderFactory{current: current, price: price}}, nil
}

func (r imageGenerationQuoteReader) ReadImageGenerationQuote(ctx context.Context, identity imageagent.ExecutionIdentity, selection *imageagent.ImageRouteSelection) (imageagent.ImageGenerationQuote, error) {
	if ctx == nil {
		return imageagent.ImageGenerationQuote{}, imageagent.ErrIdentityRequired
	}
	verified, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if ctx == nil || !ok || identity.ScopeProtocol != imageagent.OrganizationScopeProtocol || identity.TenantID == "" || identity.UserID == "" || identity.MemberID == "" || verified.TenantID != identity.TenantID || verified.EffectiveOrganizationID != identity.TenantID || verified.UserID != identity.UserID || verified.EffectiveMemberID != identity.MemberID {
		return imageagent.ImageGenerationQuote{}, imageagent.ErrIdentityRequired
	}
	ctx = aiidentity.WithIdentity(ctx, aiidentity.Identity{TenantID: identity.TenantID, UserID: identity.UserID, BusinessTaskID: identity.BusinessTaskID, AgentRunID: identity.RunID, TraceID: identity.TraceID})
	_, metadata, err := r.factory.resolve(ctx, selection)
	if err != nil {
		return imageagent.ImageGenerationQuote{}, err
	}
	return imageagent.ImageGenerationQuote{Billing: metadata.Billing, Provider: metadata.Provider, Model: metadata.Model, Protocol: metadata.Protocol, Resolution: "1024x1024", Quality: imageagent.GenerationRouteQuality(metadata.Provider, metadata.Model, metadata.Protocol), PriceVersion: metadata.PriceVersion, Points: metadata.Points, RouteReference: metadata.RouteReference, CredentialReference: metadata.CredentialReference, ConfigurationVersion: metadata.ConfigurationVersion}, nil
}

func (r imageGenerationQuoteReader) ReadImageGenerationOptions(ctx context.Context, identity imageagent.ExecutionIdentity) ([]imageagent.ImageGenerationOption, error) {
	q, e := r.ReadImageGenerationQuote(ctx, identity, nil)
	if e != nil {
		return nil, e
	}
	project := func(route *imageagent.ImageRouteSelection, q imageagent.ImageGenerationQuote) imageagent.ImageGenerationOption {
		return imageagent.ImageGenerationOption{ReferencePriceVersion: q.Billing.OfficialTariff.ReferencePriceVersion(), UnitVersion: "ai-micropoint-v1", Route: route, Model: q.Model, Quality: q.Quality, Resolution: q.Resolution, Points: q.Points}
	}
	rows := []imageagent.ImageGenerationOption{project(nil, q)}
	return rows, nil
}
