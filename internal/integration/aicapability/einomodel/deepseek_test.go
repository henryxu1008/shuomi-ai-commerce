package einomodel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
)

func deepSeekTestInput(endpoint string) aicapability.TextInputIdentity {
	profile := testTextProfile(endpoint)
	profile.ProviderID = "DASHSCOPE"
	profile.ModelID = "deepseek-v4.1-flash"
	profile.AdapterPolicyVersion = DashScopeDeepSeekV41FlashNonThinkingPolicy
	return aicapability.TextInputIdentity{OrganizationID: "org-1", ActorID: "user-1", MemberID: "member-1",
		Operation: aicapability.OperationProductContentDecision, AgentRunID: "run-1", InvocationID: "inv-1",
		System: "synthetic system", Prompt: "synthetic product title", Profile: profile}
}

func deepSeekTestExecutor(endpoint string, profile aicapability.ModelProfile, ledger *recordingLedger) Executor {
	return Executor{Ledger: ledger, Admission: NewBoundedAdmission(),
		Resolve: func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
			return QualifiedRoute{Profile: profile, Endpoint: endpoint, APIKey: "synthetic-only"}, nil
		}, Authorize: func(context.Context, aicapability.TextInputIdentity) error { return nil }}
}

func TestDeepSeekContentExecutorSendsFixedNonThinkingPolicy(t *testing.T) {
	wire := make(chan []byte, 1)
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sends.Add(1)
		raw, _ := io.ReadAll(r.Body)
		wire <- raw
		_, _ = fmt.Fprint(w, `{"id":"synthetic","object":"chat.completion","created":1,"model":"deepseek-v4.1-flash","choices":[{"index":0,"message":{"role":"assistant","content":"TITLE"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer server.Close()
	input := deepSeekTestInput(server.URL)
	quote, err := aicapability.QuoteText(input)
	require.NoError(t, err)
	ledger := &recordingLedger{}
	executor := deepSeekTestExecutor(server.URL, input.Profile, ledger)
	output, err := executor.Generate(context.Background(), input, quote, nil)
	require.NoError(t, err)
	require.Equal(t, "TITLE", output.Content)
	var body map[string]any
	require.NoError(t, json.Unmarshal(<-wire, &body))
	require.Equal(t, false, body["enable_thinking"], "the selected model must not inherit the provider thinking default")
	require.NotContains(t, body, "tools")
	require.NotContains(t, body, "reasoning_effort")
	require.Len(t, ledger.records, 2)
	require.Equal(t, aicapability.InvocationSucceeded, ledger.records[1].Outcome)
	require.Equal(t, 15, ledger.records[1].TotalTokens)
	require.Equal(t, quote.InputHash, ledger.records[1].InputHash)
	_, err = executor.Generate(context.Background(), input, quote, nil)
	require.ErrorIs(t, err, ErrOutcomeUnknown)
	require.EqualValues(t, 1, sends.Load(), "the original claim cannot send twice")
}

func TestDeepSeekPolicyQualificationBeforeClaim(t *testing.T) {
	for _, tc := range []struct {
		name, provider, model, policy string
		adapter                       AdapterKind
		valid                         bool
	}{
		{"fixed", "DASHSCOPE", "deepseek-v4.1-flash", DashScopeDeepSeekV41FlashNonThinkingPolicy, AdapterOpenAICompatible, true},
		{"old policy", "DASHSCOPE", "deepseek-v4.1-flash", "adapter-v1", AdapterOpenAICompatible, false},
		{"lowercase provider", "dashscope", "deepseek-v4.1-flash", "adapter-v1", AdapterOpenAICompatible, false},
		{"mixed case provider", "DashScope", "deepseek-v4.1-flash", "adapter-v1", AdapterOpenAICompatible, false},
		{"missing policy", "DASHSCOPE", "deepseek-v4.1-flash", "", AdapterOpenAICompatible, false},
		{"wrong provider", "synthetic", "deepseek-v4.1-flash", DashScopeDeepSeekV41FlashNonThinkingPolicy, AdapterOpenAICompatible, false},
		{"wrong model", "DASHSCOPE", "other-model", DashScopeDeepSeekV41FlashNonThinkingPolicy, AdapterOpenAICompatible, false},
		{"wrong adapter", "DASHSCOPE", "deepseek-v4.1-flash", DashScopeDeepSeekV41FlashNonThinkingPolicy, AdapterClaudeNative, false},
		{"other dashscope model", "DASHSCOPE", "other-model", "adapter-v1", AdapterOpenAICompatible, true},
		{"other provider", "synthetic", "deepseek-v4.1-flash", "adapter-v1", AdapterOpenAICompatible, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := deepSeekTestInput("http://127.0.0.1:1/v1")
			input.Profile.ProviderID, input.Profile.ModelID = tc.provider, tc.model
			input.Profile.AdapterKind, input.Profile.AdapterPolicyVersion = string(tc.adapter), tc.policy
			require.Equal(t, tc.valid, ValidRouteProfile(input.Profile, input.Operation))
			if tc.valid || tc.policy == "" { // An empty version already fails domain quote validation.
				return
			}
			quote, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			ledger := &recordingLedger{}
			executor := deepSeekTestExecutor("http://127.0.0.1:1/v1", input.Profile, ledger)
			_, err = executor.Generate(context.Background(), input, quote, nil)
			require.ErrorIs(t, err, ErrNotDispatched)
			require.False(t, ledger.claimed)
			require.False(t, ledger.reserved)
		})
	}
}

func TestDeepSeekGuardRequiresBooleanFalseBeforeHandoff(t *testing.T) {
	for _, flag := range []string{"", `,"enable_thinking":true`, `,"enable_thinking":null`, `,"enable_thinking":"false"`, `,"enable_thinking":0`, `,"enable_thinking":false`} {
		t.Run(flag, func(t *testing.T) {
			var sends, gates atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				sends.Add(1)
				_, _ = fmt.Fprint(w, `{}`)
			}))
			defer server.Close()
			cfg := GuardConfig{Adapter: AdapterOpenAICompatible, ProviderID: "DASHSCOPE", ModelID: "deepseek-v4.1-flash",
				AdapterPolicyVersion: DashScopeDeepSeekV41FlashNonThinkingPolicy, Endpoint: server.URL,
				MaximumOutputTokens: 64, MaximumRequestBytes: 1024, MaximumResponseBytes: 1024, Timeout: time.Second}
			guard, err := NewGuardedClient(cfg, nil, func(context.Context) error { gates.Add(1); return nil })
			require.NoError(t, err)
			raw := `{"model":"deepseek-v4.1-flash","max_tokens":64` + flag + `}`
			request, err := http.NewRequest(http.MethodPost, server.URL+"/chat/completions", strings.NewReader(raw))
			require.NoError(t, err)
			response, err := guard.Client.Do(request)
			if flag == `,"enable_thinking":false` {
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				require.EqualValues(t, 1, sends.Load())
				require.EqualValues(t, 1, gates.Load())
			} else {
				require.ErrorIs(t, err, ErrNotDispatched)
				require.Zero(t, sends.Load())
				require.Zero(t, gates.Load())
			}
		})
	}
}

func TestDeepSeekGuardRejectsProviderCaseAliases(t *testing.T) {
	for _, provider := range []string{"dashscope", "DashScope"} {
		t.Run(provider, func(t *testing.T) {
			_, err := NewGuardedClient(GuardConfig{Adapter: AdapterOpenAICompatible, ProviderID: provider,
				ModelID: "deepseek-v4.1-flash", AdapterPolicyVersion: "adapter-v1", Endpoint: "http://127.0.0.1:1/v1",
				MaximumOutputTokens: 64, MaximumRequestBytes: 1024, MaximumResponseBytes: 1024, Timeout: time.Second},
				nil, func(context.Context) error { return nil })
			require.ErrorIs(t, err, ErrInvalid, "provider spelling must not select the old thinking default")
		})
	}
}

func TestDeepSeekPolicyQuoteAndFinalRouteMustMatch(t *testing.T) {
	var sends atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { sends.Add(1) }))
	defer server.Close()
	input := deepSeekTestInput(server.URL)
	quote, err := aicapability.QuoteText(input)
	require.NoError(t, err)
	old := input
	old.Profile.AdapterPolicyVersion = "adapter-v1"
	oldQuote, err := aicapability.QuoteText(old)
	require.NoError(t, err)
	require.NotEqual(t, oldQuote.ProfileDigest, quote.ProfileDigest)
	require.NotEqual(t, oldQuote.InputHash, quote.InputHash)
	ledger := &recordingLedger{}
	executor := deepSeekTestExecutor(server.URL, input.Profile, ledger)
	_, err = executor.Generate(context.Background(), input, oldQuote, nil)
	require.ErrorIs(t, err, ErrNotDispatched)
	require.False(t, ledger.claimed)

	resolves := 0
	executor.Resolve = func(context.Context, aicapability.TextInputIdentity) (QualifiedRoute, error) {
		resolves++
		p := input.Profile
		if resolves > 1 {
			p.AdapterPolicyVersion = "adapter-v1"
		}
		return QualifiedRoute{Profile: p, Endpoint: server.URL, APIKey: "synthetic-only"}, nil
	}
	_, err = executor.Generate(context.Background(), input, quote, nil)
	require.ErrorIs(t, err, ErrNoDispatchReleased)
	require.Zero(t, sends.Load())
	require.Len(t, ledger.records, 2)
	require.True(t, ledger.records[1].UsageKnown)
	require.Zero(t, ledger.records[1].TotalTokens)
}

func TestDeepSeekComponentCannotDropFrozenPolicy(t *testing.T) {
	cfg := GuardConfig{Adapter: AdapterOpenAICompatible, ProviderID: "DASHSCOPE", ModelID: "deepseek-v4.1-flash",
		AdapterPolicyVersion: DashScopeDeepSeekV41FlashNonThinkingPolicy, Endpoint: "http://127.0.0.1:1/v1",
		MaximumOutputTokens: 64, MaximumRequestBytes: 1024, MaximumResponseBytes: 1024, Timeout: time.Second}
	guard, err := NewGuardedClient(cfg, nil, func(context.Context) error { return nil })
	require.NoError(t, err)
	for _, field := range []string{"provider", "policy"} {
		t.Run(field, func(t *testing.T) {
			component := ComponentConfig{Adapter: cfg.Adapter, ProviderID: cfg.ProviderID, AdapterPolicyVersion: cfg.AdapterPolicyVersion,
				Endpoint: cfg.Endpoint, APIKey: "synthetic-only", ModelID: cfg.ModelID, MaximumOutputTokens: cfg.MaximumOutputTokens}
			if field == "provider" {
				component.ProviderID = "synthetic"
			} else {
				component.AdapterPolicyVersion = "adapter-v1"
			}
			_, err := NewComponent(context.Background(), component, guard)
			require.ErrorIs(t, err, ErrInvalid)
			require.Zero(t, guard.NetworkSends())
		})
	}
}

func TestDeepSeekStillRejectsReasoningAndPreservesObservedUsageOrUnknown(t *testing.T) {
	for _, tc := range []struct {
		name, reasoning, usage string
		known                  bool
	}{
		{"reasoning returned", `,"reasoning_content":"synthetic thought"`, `{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}`, true},
		{"missing completion usage", "", `{"prompt_tokens":10,"total_tokens":15}`, false},
		{"inconsistent usage", "", `{"prompt_tokens":10,"completion_tokens":5,"total_tokens":20}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sends atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				sends.Add(1)
				_, _ = fmt.Fprint(w, `{"id":"synthetic","object":"chat.completion","choices":[{"index":0,"message":{"role":"assistant","content":"TITLE"`+tc.reasoning+`},"finish_reason":"stop"}],"usage":`+tc.usage+`}`)
			}))
			defer server.Close()
			input := deepSeekTestInput(server.URL)
			quote, err := aicapability.QuoteText(input)
			require.NoError(t, err)
			ledger := &recordingLedger{}
			executor := deepSeekTestExecutor(server.URL, input.Profile, ledger)
			_, err = executor.Generate(context.Background(), input, quote, nil)
			if tc.known {
				require.ErrorIs(t, err, ErrObservedInvalidSettled)
				require.Len(t, ledger.records, 2)
				require.Equal(t, aicapability.InvocationUsageObservedFailed, ledger.records[1].Outcome)
				require.Equal(t, 15, ledger.records[1].TotalTokens)
				require.EqualValues(t, 15, ledger.records[1].EstimatedCostMicros)
			} else {
				require.ErrorIs(t, err, ErrOutcomeUnknown)
				require.Len(t, ledger.records, 1, "unknown usage cannot be recorded as zero or settled")
			}
			_, err = executor.Generate(context.Background(), input, quote, nil)
			require.ErrorIs(t, err, ErrOutcomeUnknown)
			require.EqualValues(t, 1, sends.Load())
		})
	}
}
