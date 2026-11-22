package einorouter

import (
	"context"
	"testing"
	"time"

	"github.com/henryxu1008/shuomi-ai-commerce/internal/aicapability"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/aiworkbench"
	"github.com/henryxu1008/shuomi-ai-commerce/internal/integration/aicapability/einomodel"
	"github.com/stretchr/testify/require"
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

func TestChatIntentFreezesCreatorAndCannotInvokeAnUnavailableTool(t *testing.T) {
	profile := aicapability.ModelProfile{ClientName: "fixture", ProviderID: "fixture", AdapterKind: "openai-compatible", ModelID: "model-a", EndpointIdentityDigest: "endpoint", CredentialVersion: "credential", RoutingPolicyVersion: "route", AdapterPolicyVersion: "adapter", PromptVersion: ChatPromptVersion, OutputSchemaVersion: ChatSchemaVersion, UsageMappingVersion: "usage", CostPricingVersion: "price", Currency: "CNY", InputMicrosPerMillion: 1, OutputMicrosPerMillion: 1, MaximumPromptTokens: 10000, MaximumCompletionTokens: 500, MaximumInputBytes: 65536, MaximumOutputBytes: 4096, DeadlineBound: time.Minute, OfficialTariff: aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: "price", FXVersion: "fx", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1000000, Model: "model-a", Channel: "fixture", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 1, OutputText: 1}}}
	input := aiworkbench.ChatTurnEnvelope{Version: 1, Input: aiworkbench.ChatTurnInput{Content: "创建一个保留品牌型号的标题技能"}, Creator: &aiworkbench.ChatSkillReference{Version: 2, Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, GeneralScope: true}
	prepared, err := PrepareChat(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", input, profile)
	require.NoError(t, err)
	require.Equal(t, aicapability.OperationAIWorkbenchChatIntent, prepared.Identity.Operation)
	require.Contains(t, prepared.Identity.Prompt, input.Input.Content)
	require.Contains(t, prepared.Identity.Prompt, input.Creator.Digest)
	fake := &controlledText{output: `{"kind":"CREATE_TITLE_SKILL","status":""}`}
	decision, err := (Router{Text: fake}).Decide(context.Background(), prepared)
	require.NoError(t, err)
	require.Equal(t, "CREATE_TITLE_SKILL", decision.Kind)
	ordinary := prepared
	input.Input.CreatorOnly = true
	prepared, err = PrepareChat(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", input, profile)
	require.NoError(t, err)
	require.NotEqual(t, ordinary.Quote.InputHash, prepared.Quote.InputHash)
	require.Contains(t, prepared.Identity.Prompt, `"creatorOnly":true`)
	_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
	require.NoError(t, err)
	for _, output := range []string{`{"kind":"STORE_OVERVIEW","status":"ACTIVE"}`, `{"kind":"PRODUCT_TASK","status":""}`} {
		fake.output = output
		_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
		require.ErrorIs(t, err, aiworkbench.ErrInvalid)
	}
	fake.output = `{"kind":"CLARIFY","status":""}`
	_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
	require.NoError(t, err)
	input.Input.CreatorOnly = false
	fake.output = `{"kind":"CREATE_TITLE_SKILL","status":""}`
	input.Creator = nil
	prepared, err = PrepareChat(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", input, profile)
	require.NoError(t, err)
	_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
	require.Error(t, err)
	input.GeneralScope = false
	prepared, err = PrepareChat(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", input, profile)
	require.NoError(t, err)
	fake.output = `{"kind":"STORE_OVERVIEW","status":"ACTIVE"}`
	_, err = (Router{Text: fake}).Decide(context.Background(), prepared)
	require.Error(t, err, "a scoped discussion cannot turn into a general store query")
	profile.PromptVersion = PromptVersion
	_, err = PrepareChat(aiworkbench.Scope{OrganizationID: "org", ActorID: "actor"}, "member", "invocation", input, profile)
	require.Error(t, err, "new turns never silently use the old routing profile")
}
