package aicapability

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Snapshots carry only this finite description, never image bytes/base64.
func singleImageQuoteInput(t *testing.T, mutate func(map[string]any)) TextInputIdentity {
	t.Helper()
	p := testTextProfile()
	p.ProviderID, p.ModelID = "openai", "gpt-5.4-mini-2026-03-17"
	p.AdapterPolicyVersion, p.UsageMappingVersion = "openai-mini-vision-v1", "openai-text-v1"
	p.OfficialTariff.Model = p.ModelID
	p.OfficialTariff.Channel = "openai-direct"
	in := TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", AgentRunID: "run", Operation: Operation("product.image.analysis"), System: "fixed readonly image observations", Prompt: "inspect only the frozen image", Profile: p}
	raw, err := json.Marshal(in)
	require.NoError(t, err)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(raw, &envelope))
	envelope["Image"] = map[string]any{"SelectionDigest": strings.Repeat("a", 64), "ContentSHA256": strings.Repeat("b", 64), "MediaType": "image/png", "ByteLength": 256, "Width": 64, "Height": 64, "Detail": "high", "MeteringVersion": "openai-mini-image-high-v1"}
	if mutate != nil {
		mutate(envelope)
	}
	raw, err = json.Marshal(envelope)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, &in))
	return in
}

func TestSingleImageQuoteReservesImageOnceAndBindsSelection(t *testing.T) {
	in := singleImageQuoteInput(t, nil)
	quote, err := QuoteText(in)
	require.NoError(t, err)
	require.True(t, SupportsObservedUsageFailure(in.Operation))
	// The only new billable input component is the fixed conservative 3001
	// image tokens. base64 bytes contribute to wire limits, never text usage.
	larger := singleImageQuoteInput(t, func(e map[string]any) { e["Image"].(map[string]any)["ByteLength"] = 512 })
	other, err := QuoteText(larger)
	require.NoError(t, err)
	require.Equal(t, quote.MaximumTokens, other.MaximumTokens)
	require.NotEqual(t, quote.InputHash, other.InputHash)
	changed := singleImageQuoteInput(t, func(e map[string]any) { e["Image"].(map[string]any)["SelectionDigest"] = strings.Repeat("c", 64) })
	other, err = QuoteText(changed)
	require.NoError(t, err)
	require.NotEqual(t, quote.InputHash, other.InputHash)
	require.Greater(t, quote.MaximumTokens-in.Profile.MaximumCompletionTokens, int64(3001))
	cost, err := in.Profile.CostFor(quote.MaximumTokens-in.Profile.MaximumCompletionTokens, in.Profile.MaximumCompletionTokens)
	require.NoError(t, err)
	require.Equal(t, cost, quote.MaximumCostMicros)
}

func TestSingleImageQuoteRejectsWrongPurposeMissingImageOrUnqualifiedProfile(t *testing.T) {
	for _, mutate := range []func(map[string]any){
		func(e map[string]any) { e["Operation"] = string(OperationProductAgentDecision) },
		func(e map[string]any) { delete(e, "Image") },
		func(e map[string]any) { e["Image"].(map[string]any)["Detail"] = "low" },
		func(e map[string]any) { e["Image"].(map[string]any)["SelectionDigest"] = "" },
		func(e map[string]any) { e["Image"].(map[string]any)["ContentSHA256"] = strings.Repeat("B", 64) },
		func(e map[string]any) { e["Image"].(map[string]any)["MediaType"] = "image/webp" },
		func(e map[string]any) { e["Image"].(map[string]any)["Width"] = 0 },
		func(e map[string]any) { e["Image"].(map[string]any)["Height"] = 8193 },
		func(e map[string]any) {
			e["Image"].(map[string]any)["Width"], e["Image"].(map[string]any)["Height"] = 8192, 8192
		},
		func(e map[string]any) { e["Image"].(map[string]any)["ByteLength"] = 768<<10 + 1 },
		func(e map[string]any) { e["Image"].(map[string]any)["MeteringVersion"] = "other-model" },
		func(e map[string]any) { e["Profile"].(map[string]any)["ModelID"] = "other-model" },
	} {
		_, err := QuoteText(singleImageQuoteInput(t, mutate))
		require.Error(t, err)
	}
}

func TestSingleImageQuoteRejectsWireAndOriginalTokenBudgetOverflow(t *testing.T) {
	in := singleImageQuoteInput(t, nil)
	quote, err := QuoteText(in)
	require.NoError(t, err)
	in.Profile.MaximumPromptTokens = quote.MaximumTokens - in.Profile.MaximumCompletionTokens - 1
	_, err = QuoteText(in)
	require.Error(t, err, "do not raise the original prompt bound to admit an image")
	in = singleImageQuoteInput(t, func(e map[string]any) { e["Image"].(map[string]any)["ByteLength"] = 7000 })
	_, err = QuoteText(in)
	require.Error(t, err, "base64 must fit the original 8192-byte full-wire bound")
}

func TestSingleImageExtensionPreservesExistingTextInputHash(t *testing.T) {
	in := TextInputIdentity{OrganizationID: "org-a", ActorID: "user-a", MemberID: "member-a", Operation: OperationAIWorkbenchChatPlan, System: "system", Prompt: "prompt", Profile: testTextProfile()}
	quote, err := QuoteText(in)
	require.NoError(t, err)
	// Capture the original exact input identity before extending the envelope.
	require.Equal(t, "38ac0d1f2fdad50ca7f6d191ab6da018f5f39e4cc4511efd93b38f99583edd8a", quote.InputHash)
}
