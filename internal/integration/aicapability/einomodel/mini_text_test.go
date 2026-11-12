package einomodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

// The underlying RoundTripper is an in-process fixture; no provider is called.
func TestOpenAIMiniTextUsesBoundedOfficialWireForChatAndContent(t *testing.T) {
	for _, op := range []aicapability.Operation{aicapability.OperationAIWorkbenchChatPlan, aicapability.OperationProductContentDecision} {
		t.Run(string(op), func(t *testing.T) {
			p := testTextProfile("https://api.openai.com/v1")
			p.ProviderID, p.ModelID = "openai", aicapability.SingleImageMiniModel
			p.Currency = "CNY"
			p.AdapterPolicyVersion, p.UsageMappingVersion = "openai-mini-text-v1", "openai-text-v1"
			p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "openai-direct"
			in := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", Operation: op, InvocationID: "fixture", System: "system", Prompt: "prompt", Profile: p}
			if op == aicapability.OperationProductContentDecision {
				in.AgentRunID = "fixture-run"
			}
			q, err := aicapability.QuoteText(in)
			require.NoError(t, err)
			var wire map[string]json.RawMessage
			sends := 0
			executor := &Executor{Ledger: &recordingLedger{}, Admission: NewBoundedAdmission(), Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
				return QualifiedRoute{Profile: p, Endpoint: "https://api.openai.com/v1", APIKey: "fixture-only"}, nil
			}, Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }, BaseTransport: imageFixtureTransport(func(r *http.Request) (*http.Response, error) {
				sends++
				raw, e := io.ReadAll(r.Body)
				require.NoError(t, e)
				require.NoError(t, json.Unmarshal(raw, &wire))
				return singleImageResponse(), nil
			})}
			out, err := executor.Generate(context.Background(), in, q, nil)
			require.NoError(t, err)
			require.Equal(t, 1, sends)
			require.NotNil(t, out.Usage.BillableUsage)
			require.NotContains(t, wire, "max_tokens")
			require.JSONEq(t, fmt.Sprint(p.MaximumCompletionTokens), string(wire["max_completion_tokens"]))
			require.JSONEq(t, `"none"`, string(wire["reasoning_effort"]))
			require.JSONEq(t, `"default"`, string(wire["service_tier"]))
			require.JSONEq(t, `false`, string(wire["store"]))
			require.JSONEq(t, `1`, string(wire["n"]))
		})
	}
}

func TestOpenAIMiniTextUnknownUsageOrTransportKeepsReservation(t *testing.T) {
	for _, tc := range []struct {
		name, old, replacement string
		transportFailure       bool
	}{
		{"missing_cache", `,"prompt_tokens_details":{"cached_tokens":10}`, ``, false},
		{"wrong_model", aicapability.SingleImageMiniModel, "wrong-model", false},
		{"contradictory_total", `"total_tokens":35`, `"total_tokens":36`, false},
		{"excess_completion", `"completion_tokens":5,"total_tokens":35`, `"completion_tokens":10000,"total_tokens":10030`, false},
		{"disconnected", "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := testTextProfile("https://api.openai.com/v1")
			p.ProviderID, p.ModelID, p.Currency = "openai", aicapability.SingleImageMiniModel, "CNY"
			p.AdapterPolicyVersion, p.UsageMappingVersion = OpenAIMiniTextAdapterPolicy, "openai-text-v1"
			p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "openai-direct"
			in := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", Operation: aicapability.OperationAIWorkbenchChatPlan, InvocationID: "fixture", System: "system", Prompt: "prompt", Profile: p}
			quote, err := aicapability.QuoteText(in)
			require.NoError(t, err)
			ledger, sends := &recordingLedger{}, 0
			executor := &Executor{Ledger: ledger, Admission: NewBoundedAdmission(), Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
				return QualifiedRoute{Profile: p, Endpoint: "https://api.openai.com/v1", APIKey: "fixture"}, nil
			}, Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }, BaseTransport: imageFixtureTransport(func(*http.Request) (*http.Response, error) {
				sends++
				if tc.transportFailure {
					return nil, errors.New("fixture disconnect")
				}
				r := singleImageResponse()
				raw, _ := io.ReadAll(r.Body)
				r.Body = io.NopCloser(strings.NewReader(strings.Replace(string(raw), tc.old, tc.replacement, 1)))
				return r, nil
			})}
			_, err = executor.Generate(context.Background(), in, quote, nil)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.True(t, ledger.reserved)
			require.Len(t, ledger.records, 1)
			_, err = executor.Generate(context.Background(), in, quote, nil)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.Equal(t, 1, sends)
		})
	}
}

func TestOpenAIMiniTextFinalAuthorizationRevokedBeforeSend(t *testing.T) {
	p := testTextProfile("https://api.openai.com/v1")
	p.ProviderID, p.ModelID, p.Currency = "openai", aicapability.SingleImageMiniModel, "CNY"
	p.AdapterPolicyVersion, p.UsageMappingVersion = OpenAIMiniTextAdapterPolicy, "openai-text-v1"
	p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "openai-direct"
	in := aicapability.TextInputIdentity{OrganizationID: "org", ActorID: "actor", MemberID: "member", Operation: aicapability.OperationAIWorkbenchChatPlan, InvocationID: "fixture", System: "system", Prompt: "prompt", Profile: p}
	quote, err := aicapability.QuoteText(in)
	require.NoError(t, err)
	checks, sends := 0, 0
	ledger := &recordingLedger{}
	executor := &Executor{Ledger: ledger, Admission: NewBoundedAdmission(), Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
		return QualifiedRoute{Profile: p, Endpoint: "https://api.openai.com/v1", APIKey: "fixture"}, nil
	}, Authorize: func(context.Context, aicapability.TextInputIdentity) error {
		checks++
		if checks > 1 {
			return errors.New("revoked")
		}
		return nil
	}, BaseTransport: imageFixtureTransport(func(*http.Request) (*http.Response, error) { sends++; return singleImageResponse(), nil })}
	_, err = executor.Generate(context.Background(), in, quote, nil)
	require.ErrorIs(t, err, ErrNotDispatched)
	require.Zero(t, sends)
	require.Len(t, ledger.records, 2)
	require.Equal(t, aicapability.InvocationFailed, ledger.records[1].Outcome)
}

func TestOpenAIMiniTextCannotBorrowAnotherPurposeOrProvider(t *testing.T) {
	p := testTextProfile("https://api.openai.com/v1")
	p.ProviderID, p.ModelID = "openai", aicapability.SingleImageMiniModel
	p.Currency = "CNY"
	p.AdapterPolicyVersion, p.UsageMappingVersion = "openai-mini-text-v1", "openai-text-v1"
	for _, op := range []aicapability.Operation{aicapability.OperationAIWorkbenchChatPlan, aicapability.OperationProductContentDecision} {
		require.True(t, ValidRouteShape(p, op))
	}
	require.False(t, ValidRouteShape(p, aicapability.OperationProductAgentDecision))
	wrong := p
	wrong.ProviderID = "other"
	require.False(t, ValidRouteShape(wrong, aicapability.OperationAIWorkbenchChatPlan))
	wrong = p
	wrong.EndpointIdentityDigest = endpointDigest("https://other.invalid/v1")
	require.False(t, ValidRouteShape(wrong, aicapability.OperationAIWorkbenchChatPlan))
	p.OfficialTariff.Model, p.OfficialTariff.Channel = p.ModelID, "openai-direct"
	require.True(t, ValidRouteProfile(p, aicapability.OperationAIWorkbenchChatPlan))
	p.OfficialTariff.Channel = "other-channel"
	require.False(t, ValidRouteProfile(p, aicapability.OperationAIWorkbenchChatPlan))
}

func TestOpenAIMiniTextRejectsUnboundedOrExtraWire(t *testing.T) {
	base := `{"model":"gpt-5.4-mini-2026-03-17","messages":[{"role":"system","content":"system"},{"role":"user","content":"prompt"}],"max_completion_tokens":512,"reasoning_effort":"none","service_tier":"default","n":1,"store":false}`
	require.True(t, validBoundedOpenAITextWire([]byte(base), aicapability.SingleImageMiniModel, 512))
	for name, changed := range map[string]string{
		"other model":  strings.Replace(base, "gpt-5.4-mini-2026-03-17", "gpt-5.4-mini", 1),
		"old limit":    strings.Replace(base, "max_completion_tokens", "max_tokens", 1),
		"larger limit": strings.Replace(base, "512", "513", 1),
		"reasoning":    strings.Replace(base, `"none"`, `"high"`, 1),
		"store":        strings.Replace(base, `"store":false`, `"store":true`, 1),
		"tool":         strings.Replace(base, `"store":false`, `"store":false,"tools":[]`, 1),
		"duplicate":    strings.Replace(base, `"store":false`, `"store":false,"n":1`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			require.False(t, validBoundedOpenAITextWire([]byte(changed), aicapability.SingleImageMiniModel, 512))
		})
	}
}
