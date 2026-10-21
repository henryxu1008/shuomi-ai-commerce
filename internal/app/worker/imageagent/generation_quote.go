package imageagentworker

import (
	"context"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	"task-processor/internal/core/config"
	"task-processor/internal/imageagent"
	openai "task-processor/internal/integration/openai"
	"task-processor/internal/shared/aiidentity"

	"gorm.io/gorm"
)

type imageGenerationQuoteReader struct{ factory generationProviderFactory }

func NewImageGenerationQuoteReader(db *gorm.DB, price config.ImageAgentGenerationConfig, current ...d.CurrentReader) (imageagent.ImageSetQuoteReader, error) {
	if len(current) > 1 || db == nil || !price.Configured() {
		return nil, imageagent.ErrBudgetQuoteUnavailable
	}
	var reader d.CurrentReader
	if len(current) == 1 {
		reader = current[0]
	}
	resolver := openai.NewOrganizationCredentialResolver(db)
	if reader != nil {
		resolver = openai.NewOrganizationOnlyCredentialResolver(db)
	}
	return imageGenerationQuoteReader{factory: generationProviderFactory{current: reader, resolver: organizationCredentialAdmission{resolver: resolver}, price: price}}, nil
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
	return imageagent.ImageGenerationQuote{Provider: metadata.Provider, Model: metadata.Model, Protocol: metadata.Protocol, Resolution: "1024x1024", Quality: imageagent.GenerationRouteQuality(metadata.Provider, metadata.Model, metadata.Protocol), PriceVersion: metadata.PriceVersion, Points: metadata.Points, RouteReference: metadata.RouteReference, CredentialReference: metadata.CredentialReference, ConfigurationVersion: metadata.ConfigurationVersion}, nil
}

func (r imageGenerationQuoteReader) ReadImageGenerationOptions(ctx context.Context, identity imageagent.ExecutionIdentity) ([]imageagent.ImageGenerationOption, error) {
	q, e := r.ReadImageGenerationQuote(ctx, identity, nil)
	if e != nil {
		return nil, e
	}
	project := func(route *imageagent.ImageRouteSelection, q imageagent.ImageGenerationQuote) imageagent.ImageGenerationOption {
		return imageagent.ImageGenerationOption{Route: route, Model: q.Model, Quality: q.Quality, Resolution: q.Resolution, Points: q.Points}
	}
	rows := []imageagent.ImageGenerationOption{project(nil, q)}
	if r.factory.current == nil {
		return rows, nil
	}
	reader, ok := r.factory.current.(d.OptionalReader)
	if !ok {
		return nil, imageagent.ErrBudgetQuoteUnavailable
	}
	options, e := reader.Options(ctx, d.Scope{OrganizationID: identity.TenantID, Capability: d.ImageCapability})
	if e != nil {
		return nil, imageagent.ErrBudgetQuoteUnavailable
	}
	for _, v := range options {
		choice := &imageagent.ImageRouteSelection{BindingID: v.Payload.BindingID, RevisionID: v.ID}
		q, e = r.ReadImageGenerationQuote(ctx, identity, choice)
		if e != nil {
			return nil, e
		}
		rows = append(rows, project(choice, q))
	}
	return rows, nil
}
