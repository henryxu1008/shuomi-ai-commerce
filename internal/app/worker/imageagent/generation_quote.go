package imageagentworker

import (
	"context"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/agentconfig"
	d "github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability/routeconfig"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/authidentity"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/core/config"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/imageagent"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/shared/aiidentity"
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
	return metadata.ImageQuote(), nil
}

func (r imageGenerationQuoteReader) ReadImageGenerationOptions(ctx context.Context, identity imageagent.ExecutionIdentity) ([]imageagent.ImageGenerationOption, error) {
	verified, ok := authidentity.AuthenticatedIdentityFromContext(ctx)
	if !ok || identity.ScopeProtocol != imageagent.OrganizationScopeProtocol || identity.TenantID == "" ||
		verified.TenantID != identity.TenantID || verified.EffectiveOrganizationID != identity.TenantID ||
		identity.UserID == "" || verified.UserID != identity.UserID || identity.MemberID == "" || verified.EffectiveMemberID != identity.MemberID {
		return nil, imageagent.ErrIdentityRequired
	}
	q, e := r.ReadImageGenerationQuote(ctx, identity, nil)
	project := func(route *imageagent.ImageRouteSelection, q imageagent.ImageGenerationQuote) imageagent.ImageGenerationOption {
		unit := "ai-micropoint-v1"
		if q.UsesProviderUsage() {
			unit = agentconfig.ProviderImageUsageUnit
		}
		return imageagent.ImageGenerationOption{ReferencePriceVersion: q.Billing.OfficialTariff.ReferencePriceVersion(), UnitVersion: unit, Route: route, Model: q.Model, Quality: q.Quality, Resolution: q.Resolution, Points: q.Points}
	}
	rows := make([]imageagent.ImageGenerationOption, 0)
	if e == nil {
		rows = append(rows, project(nil, q))
	}
	owned, err := r.factory.ownedProductImageOptions(ctx, identity.TenantID)
	if err != nil {
		return nil, err
	}
	for _, option := range owned {
		route := imageagent.ImageRouteSelection{BindingID: option.BindingID, RevisionID: option.RevisionID}
		quote := ownedProductImageMetadata(option).ImageQuote()
		if !quote.Valid() {
			return nil, imageagent.ErrBudgetQuoteUnavailable
		}
		row := project(&route, quote)
		row.ChannelName = option.ChannelName
		rows = append(rows, row)
	}
	if len(rows) == 0 && e != nil {
		return nil, e
	}
	return rows, nil
}
