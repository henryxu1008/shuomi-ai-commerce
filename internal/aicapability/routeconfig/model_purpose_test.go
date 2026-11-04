package routeconfig

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

type purposeQualificationFixture struct {
	sharedCatalogFixture
	p aicapability.ModelProfile
}

func (f *purposeQualificationFixture) QualifyModel(_ context.Context, org string, category ModelCategory, ref ModelBindingReference) (ModelQualification, error) {
	p := f.p
	return ModelQualification{OrganizationID: org, Category: category, Reference: ref, Channel: p.OfficialTariff.Channel, Model: p.ModelID,
		Ready: true, ProfileVersion: "frozen-component-binding", Price: p.OfficialTariff, Text: &p}, nil
}
func TestEinoSharedServiceAdmitsReferenceMultimodalOnlyWithAnImagePurpose(t *testing.T) {
	ref := ModelBindingReference{ID: "configured", Version: "v1"}
	q := syntheticQualifiedModel(ModelQualification{Category: ModelCategoryMultimodal, Reference: ref, Model: "synthetic-gemini",
		Price: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "reference", FXVersion: "fx", Currency: "CNY",
			CNYMicrosPerCurrencyUnit: 1000000, Model: "synthetic-gemini", Channel: "configured-channel", UsageVersion: aicapability.ReferenceTextTotalsV1,
			Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}}})
	p := *q.Text
	p.ProviderID, p.AdapterKind, p.AdapterPolicyVersion = "google", "google-generate-content", aicapability.EinoGeminiImagePolicy
	p.UsageMappingVersion = aicapability.GeminiUsageMapping
	p.ThinkingMode = "low"
	p.Currency = "CNY"
	p.MaximumPromptTokens, p.ImagePromptTokenEstimate = 8000, 3001
	f := &purposeQualificationFixture{p: p}
	s := SharedModelService{Catalog: f}
	qualified, err := s.qualify(context.Background(), "org", ModelCategoryMultimodal, ref)
	require.NoError(t, err)
	require.True(t, qualified.Ready, qualified.Reason)
	f.p.AdapterPolicyVersion = aicapability.EinoGeminiTextPolicy
	qualified, err = s.qualify(context.Background(), "org", ModelCategoryMultimodal, ref)
	require.NoError(t, err)
	require.False(t, qualified.Ready, "a text reference price does not open a vision purpose")
}
