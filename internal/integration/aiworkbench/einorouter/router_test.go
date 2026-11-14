package einorouter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"task-processor/internal/aiworkbench"
	"task-processor/internal/integration/aicapability/einomodel"
)

type controlledText struct {
	output string
	calls  int
}

func (f *controlledText) Generate(_ context.Context, _ aicapability.TextInputIdentity, _ aicapability.TextQuote, validate func(string) error) (einomodel.TextOutput, error) {
	f.calls++
	if err := validate(f.output); err != nil {
		return einomodel.TextOutput{}, err
	}
	return einomodel.TextOutput{Content: f.output}, nil
}

func TestControlledRouteOnlyUsesCurrentQuestionAndClosedOutput(t *testing.T) {
	p := aicapability.ModelProfile{ClientName: "fixture", ProviderID: "fixture", AdapterKind: "openai-compatible", ModelID: "model-a", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "route", AdapterPolicyVersion: "adapter", PromptVersion: PromptVersion, OutputSchemaVersion: SchemaVersion, UsageMappingVersion: "usage", CostPricingVersion: "price", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 10000, MaximumCompletionTokens: 500, MaximumInputBytes: 8192, MaximumOutputBytes: 4096, DeadlineBound: time.Minute, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "model-a", Channel: "fixture", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}}
	prepared, err := Prepare(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", "我有几个店铺", p)
	require.NoError(t, err)
	require.Equal(t, aicapability.OperationAIWorkbenchStoreRoute, prepared.Identity.Operation)
	require.Contains(t, prepared.Identity.Prompt, "我有几个店铺")
	fake := &controlledText{output: `{"kind":"STORE_OVERVIEW","status":"ALL_NON_DELETED"}`}
	result, err := (Router{Text: fake}).Decide(context.Background(), prepared)
	require.NoError(t, err)
	require.Equal(t, "STORE_OVERVIEW", result.Kind)
	fake.output = `{"kind":"STORE_OVERVIEW","status":"ALL_NON_DELETED","organizationId":"other"}`
	_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
	require.Error(t, err)
	require.Equal(t, 2, fake.calls)
}
