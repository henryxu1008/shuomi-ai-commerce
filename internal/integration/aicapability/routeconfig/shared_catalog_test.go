package routeconfiguration

import (
	"context"
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	d "task-processor/internal/aicapability/routeconfig"
	governed "task-processor/internal/integration/aicapability/einomodel"
	"testing"
	"time"
)

type sharedMetadataFixture struct {
	d.Repository
	credential d.Credential
	prices     d.PlatformPricesSnapshot
	reads      int
}

func (f *sharedMetadataFixture) Credential(_ context.Context, org, id string) (d.Credential, error) {
	f.reads++
	if f.credential.OrganizationID != org || f.credential.ClientName != id {
		return d.Credential{}, d.ErrNotFound
	}
	return f.credential, nil
}
func (f *sharedMetadataFixture) Current(_ context.Context, scope d.Scope) (d.Revision, error) {
	if scope != d.PlatformPricesScope() {
		return d.Revision{}, d.ErrNotFound
	}
	return d.Revision{ID: "prices", Scope: scope, SchemaVersion: 1, Payload: d.Payload{PlatformPrices: &f.prices}}, nil
}
func sharedCatalogImageFixture(t *testing.T) (*SharedCatalog, *sharedMetadataFixture, d.ModelBindingReference) {
	now := time.Date(2026, 11, 3, 8, 0, 0, 0, time.UTC)
	endpoint := "https://workspace.cn-beijing.maas.aliyuncs.com"
	ref := d.ModelBindingReference{ID: "synthetic-image", Version: "synthetic-binding"}
	input := d.PlatformPricesInput{Prices: []d.PublishedModelPrice{{ID: "synthetic-price", Region: "cn-beijing", ServiceType: "image-sync", Image: &d.OfficialImagePriceScope{Width: 1024, Height: 1024, Quality: "auto"}, Provider: "qwen", Model: "qwen-image-3.0", Channel: "bailian-beijing", Currency: "CNY", UsageVersion: "qwen-image-count-v1", Rates: aicapability.OfficialModelRates{OutputImagePerItem: 200000}, SourceURL: "https://help.aliyun.com/zh/model-studio/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}}}
	prices, err := d.FreezePlatformPrices(input, "price-publication", nil, now)
	require.NoError(t, err)
	metadata := &sharedMetadataFixture{prices: prices, credential: d.Credential{OrganizationID: "org", ClientName: ref.ID, Model: "qwen-image-3.0", APIStyle: "dashscope", Version: "synthetic-credential", EndpointDigest: governed.EndpointIdentityDigest(endpoint), Reference: "synthetic-reference", TimeoutSeconds: 30, Enabled: true, KeyPresent: true}}
	def := Definition{Manufacturer: "qwen", Region: "cn-beijing", ServiceType: "image-sync", Scope: d.Scope{OrganizationID: "org", Capability: d.ImageCapability}, Channel: "bailian-beijing", PriceID: "synthetic-price", ReferenceVersion: ref.Version, BoundEvidenceVersion: "synthetic-count-bound", Image: &ImageAdmission{UnitVersion: "ai-micropoint-v1", MaximumMicropoints: 100000000, ClientName: ref.ID, Provider: "bailian", Model: "qwen-image-3.0", Endpoint: endpoint, CredentialVersion: "synthetic-credential", Billing: aicapability.OfficialModelBilling{UsageBound: aicapability.BillableModelUsage{MappingVersion: "qwen-image-count-v1", Complete: true, OutputImageItems: 1}}}}
	catalog, err := NewCatalog([]Definition{def})
	require.NoError(t, err)
	return &SharedCatalog{Catalog: catalog, Credentials: metadata, ImageCredentials: metadata, Prices: &d.PlatformPricesService{Repository: metadata, Now: func() time.Time { return now }}, Now: func() time.Time { return now }}, metadata, ref
}
func TestSharedImageAdmissionUsesPublishedOfficialPriceAndOriginalCredentialOwner(t *testing.T) {
	catalog, metadata, ref := sharedCatalogImageFixture(t)
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, q.Ready, q.Reason)
	require.EqualValues(t, 20000000, q.Image.Points)
	require.Equal(t, q.Price, q.Image.Billing.OfficialTariff)
	require.NotEmpty(t, q.ProfileVersion)
	metadata.prices.Prices[0].Rates.OutputImagePerItem = 250000
	metadata.prices.Prices[0].Version = "new-synthetic-price"
	next, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.True(t, next.Ready, next.Reason)
	require.EqualValues(t, 25000000, next.Image.Points)
	require.NotEqual(t, q.ProfileVersion, next.ProfileVersion)
	metadata.credential.Version = "rotated"
	revoked, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, revoked.Ready)
	require.Equal(t, "CREDENTIAL_NOT_ADMITTED", revoked.Reason)
}
func TestSharedImageAdmissionCannotBorrowTextCredentialsOrMissingBillingBound(t *testing.T) {
	catalog, _, ref := sharedCatalogImageFixture(t)
	catalog.ImageCredentials = nil
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "CREDENTIAL_READER_MISSING", q.Reason)
	catalog, _, ref = sharedCatalogImageFixture(t)
	for scope, defs := range catalog.Catalog.definitions {
		def := defs[ref.ID]
		def.Image.Billing.UsageBound.Complete = false
		catalog.Catalog.definitions[scope][ref.ID] = def
	}
	q, err = catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "COST_BOUND_MISSING", q.Reason)
}
func TestSharedDefaultFlareAndVisionRemainVisibleWithoutFalseReadiness(t *testing.T) {
	catalog, _, _ := sharedCatalogImageFixture(t)
	rows, err := catalog.Options(context.Background(), "org")
	require.NoError(t, err)
	found := false
	for _, row := range rows {
		if row.Reference.ID == "openai-flare" {
			found = true
			require.False(t, row.Ready)
			require.Equal(t, "gpt-image-2.5-flare-2026-09-08", row.Model)
			require.Equal(t, "PROTOCOL_AND_COST_BOUND_MISSING", row.Reason)
		}
	}
	require.True(t, found)
}
func TestTextAdmissionDescriptorDoesNotRequireOrAuthorizeItsOwnCustomerTariff(t *testing.T) {
	profile := aicapability.ModelProfile{ClientName: "synthetic", ProviderID: "synthetic", AdapterKind: "openai-compatible", ModelID: "synthetic-mini", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "admission", AdapterPolicyVersion: "adapter", PromptVersion: "prompt", OutputSchemaVersion: "schema", UsageMappingVersion: "usage", CostPricingVersion: "channel-cost", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 100, MaximumCompletionTokens: 100, MaximumInputBytes: 1024, MaximumOutputBytes: 1024, DeadlineBound: time.Second}
	require.NoError(t, profile.ValidateShape())
	require.Error(t, profile.Validate(), "an admission shape is not an execution or charge permission")
	catalog, err := NewCatalog([]Definition{{Scope: d.Scope{OrganizationID: "org", Capability: string(aicapability.OperationProductAgentDecision)}, Text: &governed.RoutePolicy{Profile: profile, AdmittedCredentialVersion: "credential", AdmittedEndpointIdentityDigest: "endpoint"}, MaximumCostMicros: 100}})
	require.NoError(t, err)
	require.NotNil(t, catalog)
}

func TestRemovedBindingRemainsAnExplicitUnreadyReadInsteadOfDisappearingCurrent(t *testing.T) {
	catalog, _, _ := sharedCatalogImageFixture(t)
	q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryText, d.ModelBindingReference{ID: "removed-binding", Version: "retained-v1"})
	require.NoError(t, err)
	require.False(t, q.Ready)
	require.Equal(t, "UNKNOWN", q.Model)
	require.Equal(t, "removed-binding", q.Reference.ID)
	require.Equal(t, "BINDING_NOT_ADMITTED", q.Reason)
}

func TestOfficialPriceCannotBorrowAnotherManufacturerOrImageScope(t *testing.T) {
	for _, kind := range []string{"manufacturer", "region", "service", "image-size"} {
		t.Run(kind, func(t *testing.T) {
			catalog, metadata, ref := sharedCatalogImageFixture(t)
			price := &metadata.prices.Prices[0]
			switch kind {
			case "manufacturer":
				price.Provider = "openai"
				price.SourceURL = "https://developers.openai.com/api/docs/pricing"
			case "region":
				price.Region = "other-region"
			case "service":
				price.ServiceType = "other-service"
			case "image-size":
				price.Image.Width = 512
			}
			q, err := catalog.QualifyModel(context.Background(), "org", d.ModelCategoryImage, ref)
			require.NoError(t, err)
			require.False(t, q.Ready)
			require.Equal(t, "PRICE_APPLICABILITY_MISSING", q.Reason)
		})
	}
}
