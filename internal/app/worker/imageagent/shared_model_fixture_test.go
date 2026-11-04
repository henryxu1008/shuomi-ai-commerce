package imageagentworker

import (
	"context"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/authidentity"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/integration/bailian"
	"time"
)

type sharedImageFixture struct {
	q      d.ModelQualification
	err    error
	digest string
}

func (f *sharedImageFixture) ReadCategory(_ context.Context, org string, category d.ModelCategory) (d.ModelQualification, string, error) {
	q := f.q
	q.OrganizationID = org
	q.Category = category
	return q, f.digest, f.err
}
func syntheticGenerationContext() context.Context {
	return authidentity.WithAuthenticatedIdentity(context.Background(), authidentity.AuthenticatedIdentity{TenantID: "org-1", EffectiveOrganizationID: "org-1", UserID: "actor", EffectiveMemberID: "member", TokenExpiresAt: time.Now().Add(time.Hour)})
}
func syntheticFactoryCurrent(resolver *generationConfigTestResolver, maximum int64) *sharedImageFixture {
	cfg := resolver.resolved.Config
	provider, protocol := "grsai", "grsai-json-sync-v1"
	billing := syntheticImageBilling(cfg.Model, "synthetic-price", maximum)
	if cfg.APIStyle == "dashscope" {
		provider, protocol = "bailian", bailian.Protocol
		billing = aicapability.OfficialModelBilling{BoundEvidenceVersion: "synthetic-count-bound", OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "synthetic-price", FXVersion: "synthetic-fx", Currency: "USD", CNYMicrosPerCurrencyUnit: 10000, Model: cfg.Model, Channel: "synthetic-channel", UsageVersion: "qwen-image-count-v1", Rates: aicapability.OfficialModelRates{OutputImagePerItem: maximum}}, UsageBound: aicapability.BillableModelUsage{MappingVersion: "qwen-image-count-v1", Complete: true, OutputImageItems: 1}}
	}
	image := &d.ImageProfile{Provider: provider, Model: cfg.Model, Protocol: protocol, EndpointDigest: governed.EndpointIdentityDigest(cfg.BaseURL), CredentialReference: resolver.resolved.CacheKey, ConfigurationVersion: "synthetic-config", PriceVersion: billing.OfficialTariff.PriceVersion, Points: maximum, Billing: billing}
	return &sharedImageFixture{q: d.ModelQualification{Reference: d.ModelBindingReference{ID: imageAgentOpenAIClientName, Version: "synthetic-binding"}, Channel: "synthetic-channel", Model: cfg.Model, Ready: true, Price: billing.OfficialTariff, Image: image}, digest: "synthetic-config"}
}
