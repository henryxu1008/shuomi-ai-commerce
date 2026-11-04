package imageagentworker

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	credential "task-processor/internal/aicapability/channelcredential"
	d "task-processor/internal/aicapability/routeconfig"
	"task-processor/internal/core/config"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"task-processor/internal/shared/aiidentity"
)

type sharedConnectionFixture struct {
	secret credential.Secret
	reads  int
}

func TestChannelSecretAdmissionRequiresVerifiedMatchingOrganization(t *testing.T) {
	m := (credential.Metadata{OrganizationID: "org-1", ConnectionID: "shared", Revision: 1}).WithIdentity()
	keys := &sharedConnectionFixture{secret: credential.Secret{Metadata: m, APIKey: "synthetic"}}
	gate := organizationCredentialAdmission{resolver: keys}
	for _, ctx := range []context.Context{context.Background(), syntheticGenerationContext(), aiidentity.WithIdentity(syntheticGenerationContext(), aiidentity.Identity{TenantID: "other", UserID: "actor"})} {
		_, e := gate.Resolve(ctx, m.OrganizationID, m.ConnectionID, m.Reference)
		require.Error(t, e)
	}
	require.Zero(t, keys.reads, "unverified/mismatched contexts cannot reach the secret port")
	valid := aiidentity.WithIdentity(syntheticGenerationContext(), aiidentity.Identity{TenantID: "org-1", UserID: "actor"})
	_, e := gate.Resolve(valid, "other", m.ConnectionID, m.Reference)
	require.Error(t, e)
	require.Zero(t, keys.reads)
	_, e = gate.Resolve(valid, m.OrganizationID, m.ConnectionID, m.Reference)
	require.NoError(t, e)
	require.Equal(t, 1, keys.reads)
}

func (s *sharedConnectionFixture) Resolve(_ context.Context, org, id, ref string) (credential.Secret, error) {
	s.reads++
	m := s.secret.Metadata
	if org != m.OrganizationID || id != m.ConnectionID {
		return credential.Secret{}, credential.ErrNotFound
	}
	if ref != m.Reference {
		return credential.Secret{}, credential.ErrChanged
	}
	return s.secret, nil
}

func TestOneConnectionSupportsDistinctTextVisionAndFlareBindings(t *testing.T) {
	m := (credential.Metadata{OrganizationID: "org-1", ConnectionID: "one-openai", Provider: "openai", Channel: "openai-direct", Account: "fixture", Region: "global", BaseURL: "https://api.openai.com/v1", Revision: 1, Enabled: true, KeyPresent: true}).WithIdentity()
	keys := &sharedConnectionFixture{secret: credential.Secret{Metadata: m, APIKey: "synthetic-one-key"}}
	p := aicapability.ModelProfile{ClientName: "mini-text", ConnectionID: m.ConnectionID, ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel, EndpointIdentityDigest: m.EndpointDigest, CredentialVersion: m.Reference, RoutingPolicyVersion: "text-definition", AdapterPolicyVersion: "text-adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "openai-text-v1", CostPricingVersion: "synthetic-price", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 10000, MaximumCompletionTokens: 64, MaximumInputBytes: 8192, MaximumOutputBytes: 16384, DeadlineBound: time.Second}
	p.OfficialTariff = aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "mini-price", FXVersion: "fixture-fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: p.ModelID, Channel: m.Channel, UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, CachedInputText: 1, OutputText: 1}}
	vision := p
	vision.ClientName = "mini-vision"
	vision.RoutingPolicyVersion = "vision-definition"
	vision.AdapterPolicyVersion = aicapability.SingleImageMiniAdapterPolicy
	policy := func(p aicapability.ModelProfile) governed.RoutePolicy {
		return governed.RoutePolicy{Profile: p, AdmittedCredentialVersion: m.Reference, AdmittedEndpointIdentityDigest: m.EndpointDigest}
	}
	resolver, e := governed.NewOrganizationRouteResolver(keys, map[governed.RouteKey]governed.RoutePolicy{{OrganizationID: m.OrganizationID, Operation: aicapability.OperationProductAgentDecision}: policy(p), {OrganizationID: m.OrganizationID, Operation: aicapability.OperationProductImageAnalysis}: policy(vision)})
	require.NoError(t, e)
	text, e := resolver.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: m.OrganizationID, Operation: aicapability.OperationProductAgentDecision})
	require.NoError(t, e)
	see, e := resolver.Resolve(context.Background(), aicapability.TextInputIdentity{OrganizationID: m.OrganizationID, Operation: aicapability.OperationProductImageAnalysis})
	require.NoError(t, e)
	// The image definition is assembled separately; no model comes from keys.
	billing := flareWorkerBilling()
	current := &sharedImageFixture{digest: "flare-definition", q: d.ModelQualification{Ready: true, Price: billing.OfficialTariff, Image: &d.ImageProfile{ConnectionID: m.ConnectionID, DeadlineSeconds: 1, Provider: "openai", Model: aicapability.FlareImageModel, Protocol: aicapability.FlareImageProtocol, EndpointDigest: m.EndpointDigest, CredentialReference: m.Reference, PriceVersion: billing.OfficialTariff.PriceVersion, Points: 300, Billing: billing}}}
	factory := generationProviderFactory{credentials: keys, current: current, price: config.ImageAgentGenerationConfig{Enabled: true, UnitVersion: aicapability.AIPointMicrounitV1, MaximumMicropoints: 300}}
	image, original, e := factory.resolveForSend(syntheticGenerationContext())
	require.NoError(t, e)
	require.Equal(t, text.APIKey, see.APIKey)
	require.Equal(t, text.APIKey, image.APIKey)
	require.Equal(t, text.Profile.CredentialVersion, original.CredentialReference)
	require.Equal(t, p.ModelID, text.Profile.ModelID)
	require.Equal(t, aicapability.FlareImageModel, image.Model)
	a, _ := text.Profile.Digest()
	b, _ := see.Profile.Digest()
	require.NotEqual(t, a, b, "purpose/protocol/definition remain distinct despite one Key")
	require.Equal(t, 3, keys.reads)
	factory.credentials = nil
	_, quoted, e := factory.resolve(syntheticGenerationContext())
	require.NoError(t, e)
	require.Equal(t, original, quoted)
	require.Equal(t, 3, keys.reads, "quote does not read the Key")
	factory.credentials = keys
	keys.secret.Metadata.Revision++
	keys.secret.Metadata = keys.secret.Metadata.WithIdentity()
	require.Equal(t, governed.RouteNeedsConfiguration, resolver.Readiness(context.Background(), aicapability.TextInputIdentity{OrganizationID: m.OrganizationID, Operation: aicapability.OperationProductAgentDecision}))
	_, _, e = factory.resolveForSend(syntheticGenerationContext())
	require.Error(t, e)
	require.Equal(t, m.Reference, original.CredentialReference, "rotation does not rewrite the original binding or price")
	require.Equal(t, billing, original.Billing)
	_, e = keys.Resolve(context.Background(), "other-org", m.ConnectionID, keys.secret.Metadata.Reference)
	require.ErrorIs(t, e, credential.ErrNotFound)
}
