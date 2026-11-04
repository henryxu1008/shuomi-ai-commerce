package routeconfiguration

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agentconfig"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
	"time"
)

type visionCategoryFixture struct{ category d.ModelCategory }

func (f *visionCategoryFixture) ReadCategory(_ context.Context, _ string, category d.ModelCategory) (d.ModelQualification, string, error) {
	f.category = category
	return d.ModelQualification{TextProfiles: map[string]aicapability.ModelProfile{string(aicapability.OperationProductImageAnalysis): {ClientName: "mini"}}}, "same-enterprise-current", nil
}
func TestImageAnalysisReadsSharedMultimodalCategory(t *testing.T) {
	f := &visionCategoryFixture{}
	p, err := (SharedTextReader{Service: f}).ReadPolicy(context.Background(), governed.RouteKey{OrganizationID: "org", Operation: aicapability.OperationProductImageAnalysis})
	require.NoError(t, err)
	require.Equal(t, d.ModelCategoryMultimodal, f.category)
	require.Equal(t, "same-enterprise-current", p.Profile.RoutingPolicyVersion)
}

func TestVisionQualificationIsSeparateFromTextConsumersAndUsesOfficialPrice(t *testing.T) {
	now := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	ref := d.ModelBindingReference{ID: "mini", Version: "fixture-binding"}
	p := aicapability.ModelProfile{ClientName: ref.ID, ProviderID: "openai", AdapterKind: "openai-compatible", ModelID: aicapability.SingleImageMiniModel, EndpointIdentityDigest: governed.EndpointIdentityDigest("https://api.openai.com/v1"), CredentialVersion: "credential", RoutingPolicyVersion: "admission", AdapterPolicyVersion: aicapability.SingleImageMiniAdapterPolicy, PromptVersion: agentconfig.ImageAnalysisPromptVersion, OutputSchemaVersion: agentconfig.ImageAnalysisResultVersion, UsageMappingVersion: "openai-text-v1", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 1000000, OutputMicrosPerMillion: 1000000, MaximumPromptTokens: 8000, MaximumCompletionTokens: 1024, MaximumInputBytes: 1 << 20, MaximumOutputBytes: 16 << 10, DeadlineBound: time.Second}
	text := p
	text.AdapterPolicyVersion = "text-v1"
	text.PromptVersion = "prompt"
	text.OutputSchemaVersion = "schema"
	defs := []Definition{}
	for op, profile := range map[aicapability.Operation]aicapability.ModelProfile{aicapability.OperationProductAgentDecision: text, aicapability.OperationProductImageAnalysis: p} {
		defs = append(defs, Definition{Manufacturer: "openai", Region: "global", ServiceType: "standard-sync", Channel: "openai-direct", PriceID: "fixture-price", ReferenceVersion: ref.Version, BoundEvidenceVersion: "fixture-bound", Scope: d.Scope{OrganizationID: "org", Capability: string(op)}, Text: &governed.RoutePolicy{Profile: profile, AdmittedCredentialVersion: "credential", AdmittedEndpointIdentityDigest: p.EndpointIdentityDigest}, MaximumCostMicros: 10000})
	}
	catalog, err := NewCatalog(defs)
	require.NoError(t, err)
	prices, err := d.FreezePlatformPrices(d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: "fixture-price", Provider: "openai", Model: p.ModelID, Channel: "openai-direct", Region: "global", ServiceType: "standard-sync", Currency: "CNY", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 10000, CachedInputText: 1000, OutputText: 20000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}, "fixture-publication", nil, now)
	require.NoError(t, err)
	metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", ClientName: ref.ID, Model: p.ModelID, APIStyle: "openai-compatible", Version: "credential", EndpointDigest: p.EndpointIdentityDigest, Reference: "fixture", TimeoutSeconds: 30, Enabled: true, KeyPresent: true}}
	shared := &SharedCatalog{Catalog: catalog, Credentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}
	for _, category := range []d.ModelCategory{d.ModelCategoryText, d.ModelCategoryMultimodal} {
		q, err := shared.QualifyModel(context.Background(), "org", category, ref)
		require.NoError(t, err)
		require.True(t, q.Ready, q.Reason)
		require.Len(t, q.TextProfiles, 1)
		if category == d.ModelCategoryMultimodal {
			require.True(t, aicapability.SingleImageProfile(q.TextProfiles[string(aicapability.OperationProductImageAnalysis)]))
		} else {
			require.Contains(t, q.TextProfiles, string(aicapability.OperationProductAgentDecision))
		}
	}
}
